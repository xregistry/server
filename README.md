[![CI](https://github.com/xregistry/server/actions/workflows/master-ci.yaml/badge.svg)](https://github.com/xregistry/server/actions/workflows/ci.yaml)
[![FOSSA Status](https://app.fossa.com/api/projects/custom%2B162%2Fgithub.com%2Fxregistry%2Fserver.svg?type=small)](https://app.fossa.com/projects/custom%2B162%2Fgithub.com%2Fxregistry%2Fserver?ref=badge_small)

<img src="https://github.com/cncf/artwork/raw/main/projects/xregistry/horizontal/color/xregistry-horizontal-color.svg" alt="xRegistry"></img><br>

# xRegistry Implementation

A full implementation of the [xRegistry specification](https://xregistry.io),
providing a metadata registry for discovering, validating, and versioning
resources in distributed systems. This also includes a CLI (`xr`) that can
be used to manage data in Registries.

> **Note:**
This project is a work-in-progress. See the [todo](todo) list for upcoming
features or report [issues](https://github.com/xregistry/server/issues) if you
encounter problems.

### Where to go from here...

- [Try it online now!](http://xregistry.soaphub.org)
- [Quick Start](docs/quick_start.md) (requires Docker)
- [Installation Options](docs/installation.md)
- [`xr` Command](docs/xr_help.md)
- [`xrserver` Command](docs/xrserver_help.md)
- [Developers/Contributing](docs/developers.md)
- [Community (and additional) Information](docs/community.md)

### Implementation Details/Choices

See the [Design Doc](DESIGN.md)


### Storage Backends

- `mysql` (default): persistent storage in a MySQL database.
- `memory`: pure in-memory storage, nothing is persisted. Named DBs
  live in a process-wide map and are lost when the process exits. Useful for
  quick demos and for running tests without MySQL.

Select a backend with `xrserver --backend memory`, the `XR_BACKEND`
environment variable, or the `backend` config property. To run the test
suite against the memory backend (no MySQL needed): `make mtest`.

Backend options are set with the same syntax: `--backend [NAME,]OPT=VALUE,...`
(e.g. `--backend mysql,host=db1,port=3307,dbname=foo`), `XR_BACKEND`, or in
the config file as `backend.<name>.<option>` (e.g. `backend.mysql.host: db1`).
If the name is omitted then the options apply to the current backend. The flag
can be repeated, and a comma in a value can be escaped as `\,`. Run
`xrserver backend list [-o json]` to see the backends and their options.
`XR_BACKEND_SECRET` is parsed like `XR_BACKEND`, after it, but isn't shown when
env vars are dumped (e.g. by `misc/start`), so use it for passwords.
