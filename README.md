[![CI](https://github.com/xregistry/server/actions/workflows/ci.yaml/badge.svg)](https://github.com/xregistry/server/actions/workflows/ci.yaml)

# xRegistry Implementation

A full implementation of the [xRegistry specification](https://xregistry.io),
providing a metadata registry for discovering, validating, and versioning
resources in distributed systems.

> **Try it online:**
A live demo is available at [https://xregistry.soaphub.org?ui](https://xregistry.soaphub.org?ui)

> **Note:**
This project is a work-in-progress. See the [todo](todo) list for upcoming
features or report [issues](https://github.com/xregistry/server/issues) if you
encounter problems.

## Quick Start

### Run with Docker (recommended)

```bash
# Run the server, with embedded MySQL DB, on port 8080
docker run -ti -p 8080:8080 ghcr.io/xregistry/xrserver-all
```

### Installation Options

The xRegistry tools are available as:
- **Container images**: [`ghcr.io/xregistry/xrserver-all`](https://github.com/orgs/xregistry/packages?repo_name=server)
- **Standalone executables**: [GitHub Releases](https://github.com/xregistry/server/releases/tag/dev)
  - Note: The standalone `xrserver` requires an external MySQL database

### Try the Sample

See the [`samples/doc-store`](samples/doc-store) script for a quick setup with
sample data.

### Build Locally

```bash
# Build, test and run with a fresh DB:
make

# Build and run without testing or rebuilding DB (faster):
make start
```

### Explore the API

```bash
# View the UI in your browser:
open http://localhost:8080?ui

# Access the API programmatically
$ curl http://localhost:8080
$ curl http://localhost:8080?inline
```

## Command Line Tools

### xr (xRegistry CLI)

The `xr` CLI lets you interact with an xRegistry server:

<!-- XR CLI HELP START -->
<!-- XR CLI HELP END -->

#### Example Commands

```bash
# Create a new endpoint group
xr create /endpoints/test1 -d '{"name": "Test Endpoint"}'
# Get all endpoints
xr get /endpoints -o human
# Update an entity
xr update /endpoints/test1 -d '{"description": "Updated description"}'
# Delete an entity
xr delete /endpoints/test1
# Import registry contents from a file
xr import / -d @myregistry.json
# Download a registry to static files
xr download ./export-dir
```

### xrserver (xRegistry Server)
The `xrserver` CLI boots and manages the API server and backing database:

<!-- XRSERVER HELP START -->
<!-- XRSERVER HELP END -->

#### Example Commands
```bash
# Start server on port 8080 and load sample data
xrserver run --samples
# Drop & recreate the database, then run the server
xrserver run --recreatedb
# Create a new registry named "myregistry"
xrserver registry create myregistry
# List all registries
xrserver registry list
```

# Developers

See `misc/Dockefile-dev` for the minimal dependencies required.

## Makefile Targets

| Target              | Description |
| ------------------- | ----------- |
| `make`              | Alias for `make all` |
| 'make all`          | Build all, run test and start server (reset DB) |
| `make run`          | Build and start server (no tests, reset DB) |
| `make start`        | Build and start seerver (no tests, keep DB) |
| `make test`         | Build all + run tests only |
| `make xr`           | Build `xr` CLI only |
| `make xrserver`     | Build `xrserver` executable only |
| `make cmds`         | Build all executables (`xrserver` and` xr`) |
| `make images`       | Build all container images |
| `make push`         | Build all container images + push images to registry |
| `make mysql`        | Start MySQL in a container |
| `make mysql-client` | Run the MySQL client in a container, for debugging |
| `make testdev`      | Build dev container + `make all` using container |

