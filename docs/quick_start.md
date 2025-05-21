# Quick Start

Fastest way to get started is to have [Docker](https://www.docker.com/)
installed, and then run the xRegistry server with an embedded MySQL database:

```yaml
$ docker run -ti -p 8080:8080 ghcr.io/xregistry/xrserver-all --samples
```

> Note: the `--samples` flag will preload a set of sample Registries for you to
> explore. See the `Loading: /...` lines of the output to see each Registry's
> URL path. Leave this option off to run with an empty Registry.

When ready, the API server will be available at: `http://localhost:8080` by
any HTTP client, such as `curl`:

```yaml
$ curl localhost:8080

{
  "specversion": "1.0-rc1",
  "registryid": "xRegistry",
  "self": "http://ubuntu:8080/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2025-05-20T16:06:00.652061965Z",
  "modifiedat": "2025-05-20T16:06:00.652061965Z"
}
```

Which shows some top-level metadata about the default Registry, which is in
this case is empty. Append one of the sample Registry URL paths to explore
another one:

```yaml
$ curl localhost:8080/reg-DocStore

{
  "specversion": "1.0-rc1",
  "registryid": "DocStore",
  "self": "http://localhost:8080/reg-DocStore/",
  "xid": "/",
  "epoch": 1,
  "name": "DocStore Registry",
  "description": "A doc store Registry",
  "documentation": "https://github.com/xregistry/server",
  "createdat": "2025-05-20T18:01:45.609185559Z",
  "modifiedat": "2025-05-20T18:01:45.609185559Z",

  "documentsurl": "http://localhost:8080/reg-DocStore/documents",
  "documentscount": 2
}
```

You can also access one of the Registries via a browser-based explorer tool by
adding the `?ui` query parameter to the URL and putting it into a browser:

```yaml
http://localhost:8080?ui
http://localhost:8080/reg-DocStore?ui
```

## Try the Sample

See the [`samples/doc-store`](../samples/doc-store) script for a quick setup
with sample data.
