package registry

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed ui
var uiEmbedded embed.FS

// uiFileSystem returns an fs.FS rooted at the ui/ subtree.
func uiFileSystem() http.FileSystem {
	if UIDir != "" {
		return http.Dir(UIDir)
	}
	sub, err := fs.Sub(uiEmbedded, "ui")
	if err != nil {
		panic("ui embed sub: " + err.Error())
	}
	return http.FS(sub)
}

// ServeUIStatic handles all requests under /ui/.
// It strips the /ui prefix and serves from the embedded (or disk) fs.
func ServeUIStatic(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/"+UISegment)
	if path == "" {
		// Redirect /ui → /ui/ so relative paths in index.html resolve correctly
		http.Redirect(w, r, "/"+UISegment+"/", http.StatusMovedPermanently)
		return
	}

	// Prevent browsers from caching UI assets — the embedded files change
	// whenever the binary is rebuilt, and stale cached files cause confusing
	// behaviour (old JS/CSS served after a server restart).
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// xrui.json is special-cased so it can be overridden, at runtime, by an
	// external file (registry.XRUIJSON) without needing a rebuild/restart.
	if path == "/xrui.json" {
		serveXRUIJSON(w, r)
		return
	}

	// For non-root paths that don't exist as files, fall back to index.html
	// so the SPA's client-side router can handle deep links.
	if path != "/" && !uiFileExists(path) {
		path = "/index.html"
	}

	// Files under /xreg/ get $HOST substituted with the request's scheme+host.
	if strings.HasPrefix(path, "/xreg/") {
		serveXregFile(w, r, path)
		return
	}

	fileServer := http.FileServer(uiFileSystem())
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
func serveXRUIJSON(w http.ResponseWriter, r *http.Request) {
	if XRUIJSON != "" {
		content, err := os.ReadFile(XRUIJSON)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(content)
			return
		}
		log.Printf("Error reading xrui.json file %q: %s", XRUIJSON, err)
		// fall through to default handling below
	}

	var content []byte
	var err error
	if UIDir != "" {
		content, err = os.ReadFile(UIDir + "/xrui.json")
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

// serveXregFile reads a file from the ui/xreg directory, replaces $HOST with
// the incoming request's scheme://host, and writes the result.
func serveXregFile(w http.ResponseWriter, r *http.Request, path string) {
	// If path is a directory (trailing slash or stat confirms it), serve index.html
	if strings.HasSuffix(path, "/") {
		path += "index.html"
	} else if isXregDir(path) {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	// Determine scheme
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS != nil {
		scheme = "https"
	}
	host := scheme + "://" + r.Host

	// Read the file
	var content []byte
	var err error
	if UIDir != "" {
		content, err = os.ReadFile(UIDir + path)
	} else {
		var f fs.File
		sub, subErr := fs.Sub(uiEmbedded, "ui")
		if subErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		f, err = sub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			content, err = io.ReadAll(f)
			f.Close()
		}
	}
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Replace $HOST
	content = bytes.ReplaceAll(content, []byte("$HOST"), []byte(host))

	// Set Content-Type based on file extension
	ext := filepath.Ext(path)
	ct := mime.TypeByExtension(ext)
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func isXregDir(path string) bool {
	if UIDir != "" {
		info, err := os.Stat(UIDir + path)
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

func uiFileExists(path string) bool {
	if UIDir != "" {
		_, err := os.Stat(UIDir + path)
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
