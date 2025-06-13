# xr (xRegistry CLI)

##  `xr` Command Summary

The `xr` CLI lets you interact with an xRegistry server:

<!-- XR HELP START -->
```yaml
xr [command]
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
      --md2html-no-style          Do not add default styling to html files
  -p, --parallel int              Number of items to download in parallel
                                  (default 10)
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
      --skip-target   Skip 'target' verification for 'xid' attributes

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
      --noepoch       Skip 'epoch' checks
  -p, --patch         Only update specified attributes
```
<!-- XR HELP END -->

## Example Commands

```yaml
# Create a new endpoint group
xr create /endpoints/test1 -d '{"name": "Test Endpoint"}'

# Get all endpoints
xr get /endpoints -o human

# Update an entity
xr update /endpoints/test1 -d '{"description": "Updated description"}'
xr set /endpoints/test1 description="Updated description" name="Test1"

# Delete an entity
xr delete /endpoints/test1

# Import registry contents from a file
xr import / -d @myregistry.json

# Download a registry to static files
xr download ./export-dir
```

## `xr` Command Environment Variables

The following environment variables can be set in the environment in which
the `xr` command is executed:

| Env Var    | Value |
| ---------- | ----- |
| XR_SERVER  | Location of the xRegistry API server (default: localhost:8080) |
