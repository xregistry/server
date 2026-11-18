# xrserver (xRegistry Server)

## `xrserver` Command Summary

The `xrserver` CLI boots and manages the API server and backing database:

The `.xrserver` config file (default location: $HOME) supports:

```
backend: STRING              # Persistence layer (memory, mysql*)
backend: ALIAS,TYPE          # Define alias for backend (TYPE=memory, mysql)
backend.BENAME.OPTION: VALUE # See: xrserver backend list
defaultreg: STRING           # Default xRegistry name (for HOST/xreg path)
http.addr: STRING            # HTTP listen address (":PORT")
http.port: INTEGER           # HTTP listen port
path.defaultreg: STRING      # HTTP path of default xReg ("xreg"*)
path.regcollection: STRING   # HTTP path of xReg collection ("xregs"*)
path.ui: STRING              # HTTP path of UI (eg: "ui"* -> HOST/ui)
rootapp: ui|xreg             # What should HOST/ return, UI* or default xReg?
ui.dir: STRING               # Directory of the UI static web pages
ui.xrui.json: STRING         # Path to xrui.json config file for the UI
```

Example backend settings:

```
backend: mysql
backend.mysql.dbname: egistry

backend: test,mysql          # Define "test" as a backend, reall it's mysql
backend.test.dbname: testregistry
```

<!-- XRSERVER HELP START -->
```yaml
xrserver [command]
  # Global flags:
      --addr string           HTTP listen address (""*)
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
      --dontcreate            Don't create DB/reg if missing
  -?, --help                  Help for commands
      --help-all              Help for all commands
  -p, --port int              HTTP Listen port (8080*)
      --recreatedb            Recreate the DB
      --recreatereg           Recreate registry
  -r, --registry string       Default Registry name
      --rootapp string        Root application (ui,xreg) (default "ui")
      --samples               Load sample registries
      --ui-dir string         Serve new UI from this directory (dev mode)
  -v, --verbose               Be chatty
      --verify                Verify loading and exit
      --version               Print command version string

xrserver backend [command]
  # Manage backends
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver backend list
  # List the available backends and their options
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -o, --output string         Output format: json, table*
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver db [command]
  # Manage databases
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver db create NAME
  # Create a new DB
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -f, --force                 Delete existing DB first
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver db delete NAME
  # Delete a DB
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -f, --force                 Ignore DB missing error
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver db get NAME
  # Get details about a DB
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver db list
  # List the databases
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -o, --output string         Output format: json, table*
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver help [command]
  # Help about any command

xrserver registry [command]
  # Manage xRegistries
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver registry create ID...
  # Create one or more xRegistry
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -f, --force                 Ignore existing registry
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver registry delete ID...
  # Delete one or more registries
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -f, --force                 Ignore missing registry
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver registry get ID
  # Get details about a registry
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver registry list
  # List the registries
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
  -?, --help                  Help for commands
  -v, --verbose               Be chatty
      --version               Print command version string

xrserver run
  # Run server (the default command)
      --addr string           HTTP Listen address (""*)
      --backend stringArray   Backend and its options: [NAME,][OPT=VALUE,]...
      --config string         Config file ($HOME/.xrserver*)
      --cset stringArray      Override configFile property: --cset NAME[:VALUE]
      --dontcreate            Don't create DB/reg if missing
  -?, --help                  Help for commands
  -p, --port int              HTTP Listen port (8080*)
      --recreatedb            Recreate the DB
      --recreatereg           Recreate registry
  -r, --registry string       Default Registry name(xRegistry*)
      --rootapp string        Root application (ui*,xreg)
      --samples               Load sample registries
  -v, --verbose               Be chatty
      --verify                Verify loading and exit
      --version               Print command version string
```
<!-- XRSERVER HELP END -->

## Example Commands
```yaml
# Start server on port 8080 and load sample data ('run' is optional)
xrserver --samples
xrserver run --samples

# Drop & recreate the database, then run the server ('run' is optional)
xrserver --recreatedb
xrserver run --recreatedb

# Create a new registry named "myregistry"
xrserver registry create myregistry

# List all registries
xrserver registry list
```

## `xrserver` Environment Variables

The following environment variables can be set in the environment in which
the `xrserver` command is executed:

| Env Var    | Value |
| ---------- | ----- |
| XR_ADDR    | Listening address of the `xrserver` API server (""*) |
| XR_BACKEND | `xrserver`'s backend driver and options (mysql*, memory) |
| XR_BACKEND_SECRET | Same as `XR_BACKEND` but processed after it and hidden when env vars are dumped (e.g. `password=...`) |
| XR_PORT    | Listening port of the `xrserver` API server (8080*) |
| XR_MODEL_PATH | Where to find the sample's model files |
| XR_LOAD_LARGE | If set, a very large default sample Registry will be loaded |
| XR_VERBOSE | Chatty level - 0=none, 1=start-up info, 2=HTTP requests*, 3+=debug |

Backend options are set via `--backend`, `XR_BACKEND`, `XR_BACKEND_SECRET`  or
the config file using the syntax `[NAME,]OPT=VALUE,...`. For example, to
configure the `xrserver` to use a non-local (127.0.0.1:3306) MySQL instance:

```
XR_BACKEND=mysql,host=db.example.com,port=3307,user=admin
XR_BACKEND_SECRET=password=secret
```

or in the config file:

```
backend: mysql
backend.mysql.host: db.example.com
backend.mysql.port: 3307
backend.mysql.user: admin
backend.mysql.password: secret
```

Use `xrserver backend list` to see the available backends and their options.
