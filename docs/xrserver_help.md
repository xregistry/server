# xrserver (xRegistry Server)

## `xrserver` Command Summary

The `xrserver` CLI boots and manages the API server and backing database:

<!-- XRSERVER HELP START -->
```yaml
xrserver [command]
  # Global flags:
      --db string         DB name (default "registry")
      --dontcreate        Don't create DB/reg if missing
  -?, --help              Help for commands
      --help-all          Help for all commands
  -p, --port int          Listen port (default 8080)
      --recreatedb        Recreate the DB
      --recreatereg       Recreate registry
  -r, --registry string   Default Registry name (default "xRegistry")
      --samples           Load sample registries
  -v, --verbose           Be chatty - can specify multiple (-v=0 to turn off)
      --verify            Verify loading and exit

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

xrserver run
  # Run server (the default command)
      --db string         DB name (default "registry")
      --dontcreate        Don't create DB/reg if missing
  -p, --port int          Listen port (default 8080)
      --recreatedb        Recreate the DB
      --recreatereg       Recreate registry
  -r, --registry string   Default Registry name (default "xRegistry")
      --samples           Load sample registries
      --verify            Verify loading and exit
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
| XR_PORT    | Listening port of the `xrserver` API server (default: 8080) |
| XR_MODEL_PATH | Where to find the sample's model files |
| XR_LOAD_LARGE | If set, a very large default sample Registry will be loaded |
| XR_VERBOSE | Chatty level - 0=none, 1=start-up info, 2=HTTP requests, 3+=debug (default: 2) |

To configure the `xrserver` to use a non-local (127.0.0.1:3306) MySQL
instance, set the following environment variables:

| Env Var    | Value |
| ---------- | ----- |
| DBHOST     | Hostname, or IP address, of MySQL instance (default: 127.0.0.1) |
| DBPORT     | Listening port of MySQL instance (default: 3306) |
| DBUSER     | Admin login for MySQL instance (default: root) |
| DBPASSWORD | Admin password for MySQL instance (default: password) |
