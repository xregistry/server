package registry

import (
	"bytes"
	"embed"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// UIDir, when non-empty, serves the new SPA UI from this directory on disk
// (useful during development — no recompile needed after editing JS/CSS/HTML).
// When empty, the embedded files are used.
var UIDir string

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
	path := strings.TrimPrefix(r.URL.Path, "/ui")
	if path == "" {
		// Redirect /ui → /ui/ so relative paths in index.html resolve correctly.
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
		return
	}

	// For non-root paths that don't exist as files, fall back to index.html
	// so the SPA's client-side router can handle deep links.
	if path != "/" && !uiFileExists(path) {
		path = "/index.html"
	}

	// Prevent browsers from caching UI assets — the embedded files change
	// whenever the binary is rebuilt, and stale cached files cause confusing
	// behaviour (old JS/CSS served after a server restart).
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

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
