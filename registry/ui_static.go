package registry

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

//go:embed ui
var uiEmbedded embed.FS

// uiFileSystem returns an fs.FS rooted at the ui/ subtree.
func uiFileSystem(xrsConfig *Config) http.FileSystem {
	uiDir := xrsConfig.GetAsString("ui.dir")
	if uiDir != "" {
		return http.Dir(uiDir)
	}
	sub, err := fs.Sub(uiEmbedded, "ui")
	if err != nil {
		panic("ui embed sub: " + err.Error())
	}
	return http.FS(sub)
}

// ServeUIStatic handles all requests under /ui/ as well as any other
// unmatched top-level path when --root=ui (so that relative asset requests -
// e.g. /style.css requested by a page that was itself served at some bad
// path - still resolve). It determines for itself whether the original
// request path was actually "/" or "/<UISegment>" or "/<UISegment>/" (a
// legitimate UI page request) as opposed to some other path that only landed
// here because it's the catch-all, or a deeper /<UISegment>/xxx path that
// doesn't correspond to a real asset file; when it's not recognized and this
// turns out to be an index.html fallback (not a real asset file), it serves
// the SPA shell but marks it (via HTTP status + an injected JS flag) as an
// unrecognized path so app.js can render a 404 instead of the normal home
// page.
func ServeUIStatic(server *Server, uuid string, w http.ResponseWriter, r *http.Request) {
	defer log.Trace("tx: %s", uuid)()

	uiSegment := server.XRSConfig.GetAsString("path.ui")

	reqPath := strings.TrimPrefix(r.URL.Path, "/")
	isRecognizedPath := (reqPath == "" || reqPath == uiSegment || reqPath == uiSegment+"/")

	path := strings.TrimPrefix(r.URL.Path, "/"+uiSegment)
	if path == "" {
		// Redirect /ui → /ui/ so relative paths in index.html resolve correctly
		http.Redirect(w, r, "/"+uiSegment+"/", http.StatusMovedPermanently)
		return
	}

	// Prevent browsers from caching UI assets — the embedded files change
	// whenever the binary is rebuilt, and stale cached files cause confusing
	// behaviour (old JS/CSS served after a server restart).
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// xrui.json is special-cased so it can be overridden, at runtime, by an
	// external file (config:ui.xrui.json) without needing a rebuild/restart.
	if path == "/xrui.json" {
		serveXRUIJSON(uuid, server.XRSConfig, w, r)
		return
	}

	// For non-root paths that don't exist as files, fall back to index.html
	// so the SPA's client-side router can handle deep links.
	//
	// NOTE: we deliberately do NOT do this by setting path="/index.html" and
	// handing it to http.FileServer below - the stdlib's FileServer has a
	// built-in special case that 301-redirects any request whose path ends
	// in "/index.html" to "./" (to avoid two URLs serving the same content).
	// Since "./" is relative, the browser just re-requests the exact same
	// (still-unmapped) deep-link URL, which falls back to index.html again,
	// which redirects again, forever - an infinite 301 loop. So instead we
	// serve the index.html bytes directly here, without ever touching
	// http.FileServer for this case.
	//
	// The exact root ("/" or "/<UISegment>/") ALSO goes through
	// serveIndexHTML() rather than the plain http.FileServer path below -
	// it needs the same injected window.__XR_API_BASE__ flag (see
	// serveIndexHTML()) that tells app.js where the xRegistry API actually
	// lives when rootApp=="ui" (at "/<DefaultRegSegment>", not "/") so
	// "this server"/DEFAULT_SERVER_ORIGIN resolves correctly instead of
	// treating the UI's own root as the API root.
	if path == "/" || !uiFileExists(server.XRSConfig, path) {
		serveIndexHTML(uuid, server, w, r, isRecognizedPath)
		return
	}

	fileServer := http.FileServer(uiFileSystem(server.XRSConfig))
	r2 := r.Clone(r.Context())
	r2.URL.Path = path
	// Strip conditional-request headers so http.FileServer always returns 200,
	// not 304. Without this a browser can get a 304 and use a stale cached copy
	// even though we set no-store above.
	r2.Header.Del("If-Modified-Since")
	r2.Header.Del("If-None-Match")
	r2.Header.Del("If-Range")
	fileServer.ServeHTTP(w, r2)
}

// serveXRUIJSON handles GET requests for /xrui.json. If registry.XRUIJSON is
// set, it's (re-)read from disk on every request (no caching, so edits take
// effect immediately). If it's unset, can't be found, or fails to load, we
// log the error (only when XRUIJSON was actually set - i.e. an explicit
// override was attempted and failed) and fall back to serving the default
// xrui.json from UIDir/the embedded fs, exactly as if no override existed.
func serveXRUIJSON(uuid string, xrsConfig *Config, w http.ResponseWriter, r *http.Request) {
	xrUIJSON := xrsConfig.GetAsString("ui.xrui.json")
	if xrUIJSON != "" {
		content, err := os.ReadFile(xrUIJSON)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(content)
			return
		}
		log.Printf("tx: %s Error reading xrui.json file %q: %s",
			uuid, xrUIJSON, err)
		// fall through to default handling below
	}

	var content []byte
	var err error
	uiDir := xrsConfig.GetAsString("ui.dir")
	if uiDir != "" {
		content, err = os.ReadFile(uiDir + "/xrui.json")
	} else {
		var f fs.File
		sub, subErr := fs.Sub(uiEmbedded, "ui")
		if subErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		f, err = sub.Open("xrui.json")
		if err == nil {
			content, err = io.ReadAll(f)
			f.Close()
		}
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// serveIndexHTML writes the ui/index.html bytes directly (bypassing
// http.FileServer - see the comment where this is called from ServeUIStatic
// for why: FileServer 301-redirects any "/index.html" path to "./", which
// would infinite-loop for SPA deep-link fallback requests).
//
// When found is false, the requested path wasn't a recognized UI path (e.g.
// a stale /xreg-XXX URL, or some other unmatched top-level path) - we still
// serve the SPA shell (so relative asset loading keeps working regardless of
// what page they were requested from), but respond with an actual HTTP 404
// status and inject a small flag (window.__XR_NOT_FOUND__) that app.js
// checks at startup to render a "Not Found" view instead of the normal UI.
func serveIndexHTML(uuid string, server *Server, w http.ResponseWriter, r *http.Request, found bool) {
	var content []byte
	var err error

	uiDir := server.XRSConfig.GetAsString("ui.dir")
	uiSegment := server.XRSConfig.GetAsString("path.ui")
	rootApp := server.XRSConfig.GetAsString("rootapp")

	if uiDir != "" {
		content, err = os.ReadFile(uiDir + "/index.html")
	} else {
		var f fs.File
		sub, subErr := fs.Sub(uiEmbedded, "ui")
		if subErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		f, err = sub.Open("index.html")
		if err == nil {
			content, err = io.ReadAll(f)
			f.Close()
		}
	}
	if err != nil {
		log.Printf("tx: %s Error loading %q: %s", uuid, "inden.html", err)
		http.NotFound(w, r)
		return
	}
	// index.html's assets (style.css, app.js, etc.) are referenced with
	// relative paths and there's no <base> tag baked into the static file
	// (UISegment is only known/configurable at runtime). That's fine when
	// the browser's current URL is exactly "/" or "/<UISegment>/" (one path
	// segment deep), but for any deeper/multi-segment fallback path (e.g.
	// "/ui/rrrr/qweqwe", or "/foo/bar/baz" when rootApp=="ui"), the browser
	// would resolve "app.js" against the wrong directory and 404, breaking
	// the page (and silently preventing app.js from ever running its
	// __XR_NOT_FOUND__ check). Inject an explicit <base> tag anchored at
	// "/<UISegment>/" so relative asset paths always resolve correctly
	// regardless of how deep/bad the requested URL was.
	base := []byte(`<base href="/` + uiSegment + `/">`)
	content = bytes.Replace(content, []byte("<head>"), append([]byte("<head>"), base...), 1)

	// Tell app.js where the xRegistry API for "this server" (the UI's own
	// hosting origin, DEFAULT_SERVER_ORIGIN) actually lives. When
	// rootApp=="ui", the UI occupies the site root, so the API is offset
	// under "/<DefaultRegSegment>" instead (e.g. "/xreg") - without this,
	// the client assumed "this server"'s API root was "/" itself, which
	// under rootApp=="ui" actually serves the UI shell, not JSON, making
	// "this server" show up as a broken/invalid registry on Home. When
	// rootApp=="xreg", the site root already IS the API root, so no offset
	// is needed. Injected unconditionally (both found/not-found responses)
	// right alongside the <base> tag above, so it's always in place before
	// app.js's own <script src="app.js"> tag (later in <head>) runs.
	apiBase := ""
	if rootApp == "ui" {
		apiBase = "/" + server.XRSConfig.GetAsString("path.defaultreg")
	}
	apiBaseFlag := []byte(`<script>window.__XR_API_BASE__=` + strconv.Quote(apiBase) + `;</script>`)
	content = bytes.Replace(content, []byte("<head>"), append([]byte("<head>"), apiBaseFlag...), 1)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !found {
		// "Home" for the UI is "/" when it's the site root (rootApp=="ui"),
		// but "/<UISegment>/" when the UI instead lives under its own
		// segment (rootApp=="xreg") - inject the right one so app.js's
		// "Go to Home" link on the 404 page doesn't send the user back into
		// the registry API root.
		homeHref := "/"
		if rootApp != "ui" {
			homeHref = "/" + uiSegment + "/"
		}
		flag := []byte(`<script>window.__XR_NOT_FOUND__=true;window.__XR_HOME__=` +
			strconv.Quote(homeHref) + `;</script>`)
		content = bytes.Replace(content, []byte("</head>"), append(flag, []byte("</head>")...), 1)
		w.WriteHeader(http.StatusNotFound)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write(content)
}

func isXregDir(xrsConfig *Config, path string) bool {
	uiDir := xrsConfig.GetAsString("ui.dir")

	if uiDir != "" {
		info, err := os.Stat(uiDir + path)
		return err == nil && info.IsDir()
	}
	sub, err := fs.Sub(uiEmbedded, "ui")
	if err != nil {
		return false
	}
	f, err := sub.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		return false
	}
	defer f.Close()
	info, err := f.Stat()
	return err == nil && info.IsDir()
}

func uiFileExists(xrsConfig *Config, path string) bool {
	uiDir := xrsConfig.GetAsString("ui.dir")

	if uiDir != "" {
		_, err := os.Stat(uiDir + path)
		return err == nil
	}
	sub, err := fs.Sub(uiEmbedded, "ui")
	if err != nil {
		return false
	}
	f, err := sub.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		return false
	}
	f.Close()
	return true
}
