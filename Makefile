all: mysql cmds test images run

MAKEFLAGS  += --no-print-directory

# Notes:
# export XR_VERBOSE=[0-9]
# Override these env vars as needed:
export GIT_ORG        ?= xregistry
export GIT_REPO       ?= $(shell basename `git rev-parse --show-toplevel`)
# DOCKERHUB must end with /, if it's set at all
export DOCKERHUB      ?=
export DBHOST         ?= 127.0.0.1
export DBPORT         ?= 3306
export DBUSER         ?= root
export DBPASSWORD     ?= password
export XR_IMAGE       ?= $(DOCKERHUB)xr
export XRSERVER_IMAGE ?= $(DOCKERHUB)xrserver
export XR_SPEC        ?= $(HOME)/go/src/github.com/xregistry/spec
export GIT_COMMIT     ?= $(shell git rev-list -1 HEAD)
export BUILDFLAGS     := -ldflags -X=main.GitCommit=$(GIT_COMMIT)

TESTDIRS := $(shell find . -name *_test.go -exec dirname {} \; | sort -u | grep -v save)
UTESTDIRS := $(shell find . -path ./tests -prune -o -name *_test.go -exec dirname {} \; | sort -u)

export XR_MODEL_PATH=.:./spec:$(XR_SPEC)

cmds: .cmds
.cmds: xrserver xr xrconform
	@touch .cmds

qtest: .test

utest: .utest
.utest: export TESTING=1
.utest: .cmds */*test.go
	@make mysql waitformysql
	@echo
	@echo "# Unit Testing"
	@go clean -testcache
	@echo "go test -failfast $(UTESTDIRS)"
	@for s in $(UTESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	@echo
	@touch .utest

test: .test .testimages
.test: export TESTING=1
.test: .cmds */*test.go
	@make mysql waitformysql
	@echo
	@echo "# Testing"
	@! grep -P '\t' registry/init.sql || (echo "Remove tabs in init.db";exit 1)
	@go clean -testcache
	@echo "go test -failfast $(TESTDIRS)"
	@for s in $(TESTDIRS); do if ! go test -failfast $$s; then exit 1; fi; done
	@# go test -failfast $(TESTDIRS)
	@echo
	@echo "# Run again w/o deleting the Registry after each one"
	@go clean -testcache
	NO_DELETE_REGISTRY=1 go test -failfast $(TESTDIRS)
	@touch .test

unittest:
	go test -failfast ./registry

xrserver: cmds/xrserver/* registry/*
	@echo
	@echo "# Building xrserver"
	go build $(BUILDFLAGS) -o $@ cmds/xrserver/*.go

xr: cmds/xr/* registry/*
	@echo
	@echo "# Building xr (cli)"
	go build $(BUILDFLAGS) -o $@ cmds/xr/*.go

xrconform: cmds/xrconform/* registry/*
	@echo
	@echo "# Building xrconform (compliance checker)"
	go build $(BUILDFLAGS) -o $@ cmds/xrconform/*.go

images: .images
.images: xr xrserver misc/waitformysql \
		misc/Dockerfile-xr misc/Dockerfile-xrserver misc/Dockerfile-all \
		misc/start
	@echo
	@echo "# Building the container images"
	@rm -rf .spec
	@mkdir -p .spec
ifdef XR_SPEC
	@! test -d "$(XR_SPEC)" || \
		(echo "# Copy xReg spec files so 'docker build' gets them" && \
		cp -r "$(XR_SPEC)/"* .spec/  )
endif
	@misc/errOutput docker build -f misc/Dockerfile-xr \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(XR_IMAGE) --no-cache .
	@misc/errOutput docker build -f misc/Dockerfile-xrserver \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(XRSERVER_IMAGE) --no-cache .
	@misc/errOutput docker build -f misc/Dockerfile-all \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) -t $(XRSERVER_IMAGE)-all \
		--no-cache .
	@rm -rf .spec
	@touch .images

testimages: .testimages
.testimages: .images
	@echo
	@echo "# Verifying the images"
	@make mysql waitformysql
	@misc/errOutput docker run --network host $(XR_IMAGE)
	@misc/errOutput docker run --network host \
		$(XRSERVER_IMAGE) run -v --recreatedb --samples --verify
	@misc/errOutput docker run --network host \
		-e DBHOST=$(DBHOST) -e DBPORT=$(DBPORT) -e DBUSER=$(DBUSER) \
		$(XRSERVER_IMAGE) run -v --recreatedb --samples --verify
	@touch .testimages

push: .push
.push: .images
	docker push $(XR_IMAGE)
	docker push $(XRSERVER_IMAGE)
	docker push $(XRSERVER_IMAGE)-all
	@touch .push

start: mysql cmds waitformysql
	@echo
	@echo "# Starting xrserver"
	./xrserver run -v $(VERIFY)

notest run local: mysql cmds waitformysql
	@echo
	@echo "# Starting xrserver from scratch"
	./xrserver run -v --recreatedb --samples $(VERIFY)

docker-all: images
	docker run -ti -p 8080:8080 $(XRSERVER_IMAGE)-all -v --recreatedb --samples

large:
	# Run the xrserver with a ton of data
	@XR_LOAD_LARGE=1 make run

docker: mysql images waitformysql
	@echo
	@echo "# Starting xrserver in Docker from scratch"
	docker run -ti --network host $(XRSERVER_IMAGE) -v --recreatedb \
		--samples $(VERIFY)

mysql:
	@docker container inspect mysql > /dev/null 2>&1 || \
	(echo && echo "# Starting mysql" && \
	docker run -d --rm -ti -e MYSQL_ROOT_PASSWORD="$(DBPASSWORD)" \
	    --network host --name mysql mysql --port $(DBPORT) > /dev/null )
		@ # -e MYSQL_USER=$(DBUSER) \

waitformysql:
	@while ! docker run --network host mysql mysqladmin \
		-h $(DBHOST) -P $(DBPORT) -s ping ;\
	do \
		echo "Waiting for mysql" ; \
		sleep 2 ; \
	done

mysql-client: mysql waitformysql
	@(docker container inspect mysql-client > /dev/null 2>&1 && \
		echo "Attaching to existing client... (press enter for prompt)" && \
		docker attach mysql-client) || \
	docker run -ti --rm --network host --name mysql-client mysql \
		mysql --host $(DBHOST) --port $(DBPORT) \
		--user $(DBUSER) --password="$(DBPASSWORD)" \
		--protocol tcp || \
		echo "If it failed, make sure mysql is ready"

k3d: misc/mysql.yaml
	@k3d cluster list | grep xreg > /dev/null || \
		(creating k3d cluster || \
		k3d cluster create xreg --wait \
			-p $(DBPORT):32002@loadbalancer  \
			-p 8080:32000@loadbalancer ; \
		while ((kubectl get nodes 2>&1 || true ) | \
		grep -e "E0727" -e "forbidden" > /dev/null 2>&1  ) ; \
		do echo -n . ; sleep 1 ; done ; \
		kubectl apply -f misc/mysql.yaml )

k3dserver: k3d images
	-kubectl delete -f misc/deploy.yaml 2> /dev/null
	k3d image import $(XRSERVER_IMAGE) -c xreg
	kubectl apply -f misc/deploy.yaml
	sleep 2 ; kubectl logs -f xrserver

prof: xrserver
	@# May need to install: apt-get install graphviz
	NO_DELETE_REGISTRY=1 \
		go test -cpuprofile cpu.prof -memprofile mem.prof -bench . \
		github.com/$(GIT_ORG)/$(GIT_REPO)/tests
	@# go tool pprof -http:0.0.0.0:9999 cpu.prof
	@go tool pprof -top -cum cpu.prof | sed -n '0,/flat/p;/xreg/p' | more
	@rm -f cpu.prof mem.prof tests.test

devimage:
	@# See the misc/Dockerfile-dev for more info
	@echo
	@echo "# Build the dev image"
	@misc/errOutput docker build -t $(DOCKERHUB)xreg-dev --no-cache \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) -f misc/Dockerfile-dev .

testdev: devimage
	@# See the misc/Dockerfile-dev for more info
	@echo
	@echo "# Make sure mysql isn't running"
	-docker rm -f mysql > /dev/null 2>&1
	@echo
	@echo "## Build, test and run the xrserver all within the dev image"
	docker run -ti -v /var/run/docker.sock:/var/run/docker.sock \
		-e VERIFY=--verify --network host $(DOCKERHUB)xreg-dev make clean all
	@echo "## Done! Exit the dev image testing"

clean:
	@echo
	@echo "# Cleaning"
	@rm -f cpu.prof mem.prof
	@rm -f xrserver xr xrconform
	@rm -f .test .images .push
	@go clean -cache -testcache
	@-! which k3d > /dev/null || k3d cluster delete xreg > /dev/null 2>&1
	@-docker rm -f mysql mysql-client > /dev/null 2>&1
	@# do "sleep" so that "docker system prune" won't delete the mysql image
	@-docker run -d -ti --rm mysql sleep 5 > /dev/null 2>&1
	@-docker system prune -f -a --volumes > /dev/null
