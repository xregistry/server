# Developers

## Contributing

[Issues](https://github.com/xregistry/server/issues) and
[Pull Requests (PR)](https://github.com/xregistry/server/pulls) are always
welcome from anyone.

### Codin

- Wrap lines at 80 columns (or try hard)
- `gofmt` all go files
- If it feels like it's taking too long code something, or the code you're
  writing feels more complex/harder than it should be, stop and ask the team
  about it. There may be a simple way to do what you're trying to do and it's
  better to ask a question than to waste your time rat-holing. Use the
  [#xregistry CNCF slack
  channel](https://cloud-native.slack.com/archives/C03GJK3MCMD)  appropriately.

### Testing

- Tests should check the expected output byte-for-byte. We want to make sure
  that every character (even spaces) are exactly as we expect. So, avoid
  expected outputs of "*" and regular expressions (ie. one- that start with
  `^`).
- There may be times when that rule is too strict due to output varying too
  much across runs/environments - try to use masking first, but as a last
  resort regexp matching of the output is ok.
  - The tooling utils (e.g. `XHTTP`) should mask most of the fields already
    by default (e.g. timestamps, error's `source` field).

### Pull Requests

Simple guidelines for PRs:
- All PRs MUST be DCO signed to be accepted.
- All PRs MUST successfully pass the `make clean all` process.
- PRs do not need an associated issue, stand-alone PRs are fine.
- However, larger PRs would benefit from a discussion prior to doing the work.

### READ THE SPECS!

Make sure you understand the
[xRegistry specifications](https://github.com/xregistry/spec/tree/main/core).
Some semantics are not obvious, and the docs should help explain why some
decisions were made.

## Build Locally

Most common Makefile targets:

```yaml
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
| `make test`         | Build all & run all tests |
| `make qtest`        | Build all & run just main tests |
| `make benchmark`    | Run some basic speed bencharks (ftest/largeload) |
| `make xr`           | Build `xr` CLI only |
| `make xrserver`     | Build `xrserver` executable only |
| `make cmds`         | Build all executables (`xrserver` and` xr`) |
| `make images`       | Build all container images |
| `make push`         | Build all container images & push images to registry |
| `make mysql`        | Start MySQL in a container |
| `make mysql-client` | Run the MySQL client in a container, for debugging |
| `make testdev`      | Build/verify dev image; `make all` using image |

