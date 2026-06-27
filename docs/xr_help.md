# xr (xRegistry CLI)

##  `xr` Command Summary

The `xr` CLI lets you interact with an xRegistry server:

<!-- XR HELP START -->
```yaml
xr [command]
  # Global flags:
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
      --help-all        Help for all commands
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr conform
  # xRegistry Conformance Tester
      --config string   Config file ($HOME/.xrconfig)
  -d, --depth int       Console depth
      --errjson         Print errors as json
      --failfast        stop on first failure
  -?, --help            Help for xr
  -l, --logs            Show logs even on success
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr create XID
  # Create a new entity in the registry
      --config string        Config file ($HOME/.xrconfig)
  -d, --data string          Data, @FILE, @URL, @-(stdin)
      --del stringArray      Delete an attribute: --del NAME
  -m, --details              Data is resource metadata
      --errjson              Print errors as json
  -f, --force                Force an 'update' if exist, no pre-flight checks
  -?, --help                 Help for xr
      --ignore stringArray   Skip certain checks
  -o, --output string        Output format (none*, json) when xReg metadata
  -r, --replace              Replace entire entity (all attributes)
  -s, --server string        xRegistry server URL
      --set stringArray      Set an attribute: --set NAME[=(VALUE | "STRING")]
  -v, --verbose              Be chatty
      --version              Print command version string

xr delete XID...
  # Delete an entity from the registry
      --config string   Config file ($HOME/.xrconfig)
  -d, --data string     Data(json), @FILE, @URL, @-(stdin)
      --errjson         Print errors as json
  -f, --force           Don't error if doesn't exist
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr download DIR [XID...] 
  # Download entities from registry as individual files
  -c, --capabilities              Modify capabilities for static site
      --config string             Config file ($HOME/.xrconfig)
      --errjson                   Print errors as json
  -?, --help                      Help for xr
  -i, --index string              Directory index file name (index.html*)
  -m, --md2html                   Generate HTML files for MD files
      --md2html-css-link string   CSS stylesheet 'link' to add in md2html files
      --md2html-header string     HTML to add in <head> (data,@FILE,@URL,@-)
      --md2html-html string       HTML to add after <head> (data,@FILE,@URL,@-)
      --md2html-no-style          Do not add default styling to html files
  -p, --parallel int              Number of items to download in parallel (10*)
  -s, --server string             xRegistry server URL
  -u, --url string                Host/path to Update xRegistry paths
  -v, --verbose                   Be chatty
      --version                   Print command version string

xr get [XID]
  # Retrieve entities from the registry
      --config string        Config file ($HOME/.xrconfig)
  -m, --details              Show resource metadata
      --doc                  Retieve document view of entities
      --errjson              Print errors as json
  -f, --filter stringArray   Filter: expr[,expr]
  -?, --help                 Help for xr
  -i, --inline stringArray   Inline entities: *, ...
  -o, --output string        Output format: json*, table
  -s, --server string        xRegistry server URL
  -v, --verbose              Be chatty
      --version              Print command version string

xr import [XID]
  # Import entities into the registry
      --config string   Config file ($HOME/.xrconfig)
  -d, --data string     Data(json), @FILE, @URL, @-(stdin)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model [command]
  # Manage a regsitry's model
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model get
  # Retrieve details about the registry's model
  -a, --all             Include default attributes
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -o, --output string   Output format: table*, json
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model group [command]
  # Model Group operations
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model group create PLURAL:SINGULAR...
  # Create a new Model Group type
  -a, --all             Include default attributes in output
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -o, --output string   Output format: none*, table, json
  -r, --resources       Show Resource types in output
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model group delete PLURAL...
  # Delete a Model Group type
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -f, --force           Ignore a "not found" error
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model group get PLURAL
  # Retrieve details about a Model Group type
  -a, --all             Include default attributes
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -o, --output string   Output format: table*, json
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model group list
  # List the Group types defined in the model
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -o, --output string   Output format: table*, json
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model normalize [- | FILE]
  # Parse and resolve 'includes' in an xRegistry model document
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model resource [command]
  # Model Resource operations
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model resource create PLURAL:SINGULAR...
  # Create a new Model Resource type
  -a, --all                        Include default attributes in output
      --config string              Config file ($HOME/.xrconfig)
      --description string         Description text
      --docs string                Documenations URL
      --errjson                    Print errors as json
  -f, --force                      Force an 'update' if exist
  -g, --group string               Group plural name (create with ":SINGULAR")
      --has-doc                    Supports domain doc (true*)
  -?, --help                       Help for xr
      --icon string                Icon URL
      --label stringArray          NAME[=VALUE)]
      --max-versions int           Max versions allowed (0=unlimited*)
      --model-compat-with string   URI of model
      --model-version string       Model version string
      --no-has-doc                 Doesn't support domain doc
      --no-set-version-id          VersionID is not settable
      --no-single-version-root     Allow multiple verson roots (true*)
      --no-strict-validation       Disable strict validation (true*)
      --no-validate-compat         Disable compatibility validation (true*)
      --no-validate-format         Disable format validation (true*)
  -o, --output string              Output format: none*, table, json
  -s, --server string              xRegistry server URL
      --set-version-id             Version ID is settable (true*)
      --single-version-root        Restrict to single root
      --strict-validation          Enforce strict validation
      --type-map stringArray       NAME[=VALUE)]
      --validate-compat            Enable compatibility validation
      --validate-format            Enable format validation
  -v, --verbose                    Be chatty
      --version                    Print command version string
      --version-mode string        Versioning algorithm

xr model resource delete PLURAL...
  # Delete a Model Resource type
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -f, --force           Ignore a "not found" error
  -g, --group string    Group type name
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model resource get PLURAL
  # Retrieve details about a Model Resource type
  -a, --all             Include default attributes
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -g, --group string    Group type plural name
  -?, --help            Help for xr
  -o, --output string   Output format: table*, json
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model resource list
  # List the Resource types in a Group type
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -g, --group string    Group type plural name
  -?, --help            Help for xr
  -o, --output string   Output format: table*, json
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model resource update PLURAL...
  # Update a Model Resource type
  -a, --all                        Include default attributes in output
      --config string              Config file ($HOME/.xrconfig)
      --description string         Description text
      --docs string                Documenations URL
      --errjson                    Print errors as json
  -f, --force                      Force a 'create' if missing
  -g, --group string               Group plural name (create with ":SINGULAR")
      --has-doc                    Supports domain doc (true*)
  -?, --help                       Help for xr
      --icon string                Icon URL
      --label stringArray          NAME[=VALUE)]
      --max-versions int           Max versions allowed (0=unlimited*)
      --model-compat-with string   URI of model
      --model-version string       Model version string
      --no-has-doc                 Doesn't support domain doc
      --no-set-version-id          VersionID is not settable
      --no-single-version-root     Allow multiple verson roots (true*)
      --no-strict-validation       Disable strict validation (true*)
      --no-validate-compat         Disable compatibility validation (true*)
      --no-validate-format         Disable format validation (true*)
  -o, --output string              Output format: none*, table, json
  -s, --server string              xRegistry server URL
      --set-version-id             Version ID is settable (true*)
      --single-version-root        Restrict to single root
      --strict-validation          Enforce strict validation
      --type-map stringArray       NAME[=VALUE)]
      --validate-compat            Enable compatibility validation
      --validate-format            Enable format validation
  -v, --verbose                    Be chatty
      --version                    Print command version string
      --version-mode string        Versioning algorithm

xr model resource upsert PLURAL:SINGULAR...
  # UPdate, or inSERT as appropriate, a Model Resource type
  -a, --all                        Include default attributes in output
      --config string              Config file ($HOME/.xrconfig)
      --description string         Description text
      --docs string                Documenations URL
      --errjson                    Print errors as json
  -g, --group string               Group plural name (create with ":SINGULAR")
      --has-doc                    Supports domain doc (true*)
  -?, --help                       Help for xr
      --icon string                Icon URL
      --label stringArray          NAME[=VALUE)]
      --max-versions int           Max versions allowed (0=unlimited*)
      --model-compat-with string   URI of model
      --model-version string       Model version string
      --no-has-doc                 Doesn't support domain doc
      --no-set-version-id          VersionID is not settable
      --no-single-version-root     Allow multiple verson roots (true*)
      --no-strict-validation       Disable strict validation (true*)
      --no-validate-compat         Disable compatibility validation (true*)
      --no-validate-format         Disable format validation (true*)
  -o, --output string              Output format: none*, table, json
  -s, --server string              xRegistry server URL
      --set-version-id             Version ID is settable (true*)
      --single-version-root        Restrict to single root
      --strict-validation          Enforce strict validation
      --type-map stringArray       NAME[=VALUE)]
      --validate-compat            Enable compatibility validation
      --validate-format            Enable format validation
  -v, --verbose                    Be chatty
      --version                    Print command version string
      --version-mode string        Versioning algorithm

xr model update [- | FILE | -d]
  # Update the registry's model
      --config string   Config file ($HOME/.xrconfig)
  -d, --data string     Data(json), @FILE, @URL, @-(stdin)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty
      --version         Print command version string

xr model verify [- | FILE...]
  # Parse and verify xRegistry model documents
      --config string   Config file ($HOME/.xrconfig)
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
      --skip-target     Skip 'target' verification for 'xid' attributes
  -v, --verbose         Be chatty
      --version         Print command version string

xr serve DIR
  # Run an HTTP file server for a directory
  -a, --address string   address:port of listener (0.0.0.0:8080*)
      --config string    Config file ($HOME/.xrconfig)
      --errjson          Print errors as json
  -?, --help             Help for xr
  -s, --server string    xRegistry server URL
  -v, --verbose          Be chatty
      --version          Print command version string

xr update XID
  # Update an entity in the registry
      --config string        Config file ($HOME/.xrconfig)
  -d, --data string          Data, @FILE, @URL, @-(stdin)
      --del stringArray      Delete an attribute
  -m, --details              Data is resource metadata
      --errjson              Print errors as json
  -f, --force                Force a 'create' if missing, no pre-flight checks
  -?, --help                 Help for xr
      --ignore stringArray   Skip certain checks
  -o, --output string        Output format (none*, json) when xReg metadata
  -r, --replace              Replace entire entity (all attributes)
  -s, --server string        xRegistry server URL
      --set stringArray      Set an attribute
  -v, --verbose              Be chatty
      --version              Print command version string

xr upsert XID
  # UPdate, or inSERT as appropriate, an entity in the registry
      --config string        Config file ($HOME/.xrconfig)
  -d, --data string          Data, @FILE, @URL, @-(stdin)
      --del stringArray      Delete an attribute
  -m, --details              Data is resource metadata
      --errjson              Print errors as json
  -f, --force                Skip pre-flight checks
  -?, --help                 Help for xr
      --ignore stringArray   Skip certain checks
  -o, --output string        Output format (none*, json) when xReg metadata
  -r, --replace              Replace entire entity (all attributes)
  -s, --server string        xRegistry server URL
      --set stringArray      Set an attribute
  -v, --verbose              Be chatty
      --version              Print command version string
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
| XR_SERVER  | Location of the xRegistry API server (localhost:8080*) |

## `xr` Configuration File

- Found in `$HOME/.xrconfig`
- Syntax:
```
# Comment
NAME: VALUE
```

- Supported configuration names:
  - `server.url` - location of the xRegistry server
  - `header.KEY` - an HTTP header (KEY) to add to all xRegistry client
                   requests. For example, for authentication headers

