# xrserver (xRegistry Server)
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
