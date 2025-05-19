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

<!-- XR HELP START -->
```bash
xr [command]
  # xRegistry CLI
  # Global flags:
  -?, --help            Help for xr
      --help-all        Help for all commands
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty

xr conform
  # xRegistry Conformance Tester
  -c, --config string   Location of config file
  -d, --depth count     Console depth
  -l, --logs            Show logs on success
  -t, --tdDebug         td debug

xr create [ XID ]
  # Create a new entity in the registry
  -d, --data string   Data(json), @FILE, @URL, @-(stdin)
  -m, --details       Data is resource metadata
  -f, --force         Force an 'update' if already exist, skip pre-flight
                      checks
  -p, --patch         Only 'update' specified attributes when -f is applied

xr delete [ XID ... ]
  # Delete an entity from the registry
  -d, --data string   Data(json), @FILE, @URL, @-(stdin)
  -f, --force         Don't error if doesn't exist

xr download DIR [ XID...]
  # Download entities from registry as individual files
  -c, --capabilities              Modify capabilities for static site
  -i, --index string              Directory index file name (default
                                  "index.html")
  -m, --md2html                   Generate HTML files for MD files
      --md2html-css-link string   CSS stylesheet 'link' to add in md2html files
      --md2html-header string     HTML to add in <head> of md2html files
                                  (data,@FILE,@URL,@-)
      --md2html-html string       HTML to add after <head> in md2html
                                  files (data,@FILE,@URL,@-)
  -u, --url string                Host/path to Update xRegistry paths

xr get [ XID ]
  # Retrieve entities from the registry
  -m, --details         Show resource metadata
  -o, --output string   Output format(json,human) (default "json")

xr import [ XID ]
  # Import entities into the registry
  -d, --data string   Data(json), @FILE, @URL, @-(stdin)

xr model get
  # Get the registry's model
  -a, --all             Show all data
  -o, --output string   output: table, json (default "table")

xr model group create PLURAL:SINGULAR...
  # Create a new Model Group type

xr model group delete PLURAL...
  # Delete a Model Group type
  -f, --force   Ignore a "not found" error

xr model normalize [ - | FILE ]
  # Parse and resolve 'includes' in an xRegistry model document

xr model resource create PLURAL:SINGULAR...
  # Create a new Model Resource type
  -g, --group string   Group type name

xr model resource delete PLURAL...
  # Delete a Model Resource type
  -f, --force          Ignore a "not found" error
  -g, --group string   Group type name

xr model update [ - | FILE | -d ]
  # Update the registry's model
  -d, --data string   Data(json), @FILE, @URL, @-(stdin)

xr model verify [ - | FILE ... ]
  # Parse and verify xRegistry model documents

xr serve DIR
  # Run an HTTP file server for a directory
  -a, --address string   address:port of listener (default "0.0.0.0:8080")

xr set XID NAME[=(VALUE | "STRING")]
  # Update an entity's xRegistry metadata
  -m, --details         Show resource metadata
  -o, --output string   Output format(json,human) (default "json")

xr update [ XID ]
  # Update an entity in the registry
  -d, --data string   Data(json), @FILE, @URL, @-(stdin)
  -m, --details       Data is resource metadata
  -f, --force         Force a 'create' if doesnt exist, skip pre-flight checks
  -p, --patch         Only update specified attributes
```
<!-- XR HELP END -->

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
```bash
xrserver [default-registry-name] [command]
  # xRegistry server
  # Global flags:
      --db string     DB name (default "registry")
      --dontcreate    Don't create DB/reg if missing
  -?, --help          Help for commands
      --help-all      Help for all commands
  -p, --port int      Listen port (default 8080)
      --recreatedb    Recreate the DB
      --recreatereg   Recreate registry
      --samples       Load sample registries
  -v, --verbose       Be chatty - can specify multiple (-v=0 to turn off)
      --verify        Verify loading and exit

xrserver db create NAME
  # Create a new mysql DB
  -f, --force   Delete existing DB first

xrserver db delete NAME
  # Delete a mysql DB
  -f, --force   Ignore DB missing error

xrserver db get NAME
  # Get details about a mysql DB

xrserver help [command]
  # Help about any command

xrserver registry create ID...
  # Create one or more xRegistry
  -f, --force   Ignore existing registry

xrserver registry delete ID...
  # Delete one or more registry
  -f, --force   Ignore missing registry

xrserver registry get ID
  # Get details about a registry

xrserver registry list
  # List the registries

xrserver run [default-registry-name]
  # Run server (the default command)
      --db string     DB name (default "registry")
      --dontcreate    Don't create DB/reg if missing
  -p, --port int      Listen port (default 8080)
      --recreatedb    Recreate the DB
      --recreatereg   Recreate registry
      --samples       Load sample registries
      --verify        Verify loading and exit
```
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

