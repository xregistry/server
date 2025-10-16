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

## Server Quick Start

Fastest path:

```
$ docker run -ti -p 8080:8080 ghcr.io/xregistry/xrserver-all -vv --samples
```

This will start the xRegistry server, along with a MySQL DB, and load some
sample data into the Registry.

You can then access it via: `http://localhost:8080` for API (e.g. `curl`)
access or use: `http://localhost:8080?ui` with your favorite browser to
examine the sample data.

## Command Line

Download the stand-alone CLI executable from
[here](https://github.com/xregistry/server/releases/tag/dev).

## Configuring MySQL

WIP - Is there anything w.r.t. the schema? I think xrserver just 'does it'
automagically. However, check to see if there are any special admin/config
knobs that might need to be set. I think Azure MySQL required some special
rights to be set.

WIP - configuring MySQL to use a volume outside of the docker container

## Configuring `xrserver` to use an external MySQL Database

See the [`xrserver`](xrserver_help.md) docs for details.

## Adding Authentication to an xRegistry Server

WIP

## Next Steps

See the [`samples/doc-store`](../samples/doc-store) script for a quick setup
with sample data.
