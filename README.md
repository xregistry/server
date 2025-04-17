[![CI](https://github.com/xregistry/server/actions/workflows/ci.yaml/badge.svg)](https://github.com/xregistry/server/actions/workflows/ci.yaml)

# xRegistry Implementation

Implementation of the [xRegistry](https://xregistry.io) spec.
A live version is available at
[https://xregistry.soaphub.org?ui](https://xregistry.soaphub.org?ui) too.

Still a work-in-progress so check the [todo](todo) list for what's missing,
but let us know via [issues](https://github.com/xregistry/server/issues)
if something seems wrong.

To run the official image (with an embedded mysql DB):
```
# You need to have Docker installed

docker run -ti -p 8080:8080 ghcr.io/xregistry/xreg-server-all
```

Please see the simple [doc-store](samples/doc-store) script to see how
to quickly stand-up and load an xRegistry server for experimentation.
Lots more samples/docs are needed...

To build and run it locally:
```
# You need to have Docker installed

# Build, test and run the xreg server (creates a new DB each time):
$ make

or to run the server w/o creating a new DB (no testing either):
$ make start
```

Then explore it with:
```
# In a browser go to:  http://localhost:8080?ui

# Or access the APIs programmatically, e.g.:
$ curl http://localhost:8080
$ curl http://localhost:8080?inline
```

# Developers

See `misc/Dockefile-dev` for the minimal things you'll need to install.
Useful Makefile targets:
```
- make              : build all, test and run the server (alias for 'all')
- make all          : build all, test and run the server (reset the DB)
- make run          : build server and run it (no tests, reset the DB)
- make start        : build server and run it (no tests, do not reset the DB)
- make test         : build all, images and run tests, don't run server
- make clean        : erase all build artifacts, stop mysql. Basically, reset
- make server       : build the server
- make cmds         : build the exes (server and CLIs)
- make image        : build the all Docker images
- make push         : push the Docker images to DockerHub
- make mysql        : just start mysql as a Docker container
- make mysql-client : run the mysql client, for testing
- make testdev      : build a dev docker image, and build/test/run everything
                      to make sure the minimal dev install requirements
                      haven't changed
```
