FROM golang:alpine
RUN apk add make

ARG GIT_COMMIT

WORKDIR /go/src/
COPY . /go/src/

# Erase executables that were copied from the COPY cmd above
RUN find . -maxdepth 1 -type f -executable -exec rm {} \;

# Force static builds
ENV GO_EXTLINK_ENABLED=0
ENV CGO_ENABLED=0
ENV BUILDFLAGS -ldflags \
	\"-w -extldflags -static -X=main.GitCommit=$GIT_COMMIT \" \
	-tags netgo -installsuffix netgo

RUN make cmds

FROM alpine
# FROM mysql
# ENV MYSQL_ROOT_PASSWORD=password

LABEL org.opencontainers.image.source https://github.com/xregistry/server

COPY --from=0 /etc/ssl/certs/ca-certificates.crt \
	/etc/ssl/certs/ca-certificates.crt

COPY --from=0 /go/src/server /server
COPY --from=0 /go/src/xr /xr
COPY misc/repo.tar /misc/repo.tar

# If local copy of spec is found, copy it into the image so we can use it
COPY .spec* /spec/
COPY misc/waitformysql /
COPY misc/start /

ENTRYPOINT [ "/start" ]
