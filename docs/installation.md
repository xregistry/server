# Installation Options

The xRegistry executables are available as:
- **Container images**: [`https://github.com/orgs/xregistry/packages`](https://github.com/orgs/xregistry/packages)
  - `xrserver` is the xRegistry API server, an external MySQL database will
    need to be configured.
  - `xrserver-all` is the xRegistry API server & an embedded MySQL database.
  - `xr` is the xRegistry CLI tool. `xr` is also in the above images as well.
- **Standalone executables**: [GitHub 'dev' Release](https://github.com/xregistry/server/releases/tag/dev)
  - `xrserver` is the xRegistry API server, an external MySQL database will
    need to be configured.
  - `xr` is the xRegistry CLI tool.

## Configuring MySQL

WIP - Is there anything? I think xrserver just 'does it' automagically.
However, check to see if there are any special admin/config knobs that might
need to be set. I think Azure MySQL required some special rights to be set.

## Configuring `xrserver` to use an external MySQL Database

WIP

To configure the `xrserver` to use a non-local (127.0.0.1:3306) MySQL
instance, set the following environment variables:

| Env Var    | Value |
| ---------- | ----- |
| DBHOST     | Hostname, or IP address, of MySQL instance (default: 127.0.0.1) |
| DBPORT     | Listening port number of MySQL instance (default: 3306) |
| DBUSER     | Admin login for MySQL instance (default: root) |
| DBPASSWORD | Admin password for MySQL instance (default: password) |

## Adding Authentication to an xRegistry Server

WIP

## Next Steps

See the [`samples/doc-store`](../samples/doc-store) script for a quick setup
with sample data.
