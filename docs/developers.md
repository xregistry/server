# Developers

## Contributing

[Issues](https://github.com/xregistry/server/issues) and
[Pull Requests (PR)](https://github.com/xregistry/server/pulls) are always
welcome from anyone.

### Pull Requests

Simple guidelines for PRs:
- All PRs MUST be DCO signed to be accepted.
- All PRs MUST successfully pass the `make clean all` process.
- PRs do not need an associated issue, stand-alone PRs are fine.
- However, larger PRs would benefit from a discussion prior to doing the work.

## Build Locally

Most common Makefile targets:

```bash
# Build, test and run; will reset the DB & sample Registries:
make

# Build and run w/o testing; will reset the DB & sample Registries:
make run

# Build and run without testing or rebuilding DB & sample Registries:
make start
```

See `misc/Dockefile-dev` for the minimal dependencies required: `golang` and
the packages listed on the `RUN apk add` command.

## Makefile Targets

| Target              | Description |
| ------------------- | ----------- |
| `make`              | Alias for `make all` |
| `make clean`        | Erase all build outputs, clean docker |
| `make all`          | Build all, run test and start server (reset DB) |
| `make run`          | Build and start server (no tests, reset DB) |
| `make start`        | Build and start seerver (no tests, keep DB) |
| `make test`         | Build all + run tests only |
| `make xr`           | Build `xr` CLI only |
| `make xrserver`     | Build `xrserver` executable only |
| `make cmds`         | Build all executables (`xrserver` and` xr`) |
| `make images`       | Build all container images |
| `make push`         | Build all container images & push images to registry |
| `make mysql`        | Start MySQL in a container |
| `make mysql-client` | Run the MySQL client in a container, for debugging |
| `make testdev`      | Build/verify dev image; `make all` using image |

