package registry

// This file implements a generic, byte-level JSON reverse-proxy used by the
// new SPA (registry/ui/app.js) to talk to remote xRegistry servers that
// don't set permissive CORS headers. Unlike the older /proxy endpoint (see
// HTTPProxy in httpStuff.go), which re-renders the old HTML-templated UI
// around fetched remote data, this proxy is a transparent pass-through: it
// forwards the request as-is to the remote server, then rewrites the
// remote's own absolute URLs (self/*url attributes, Location headers, etc.)
// in the response so they point back through this proxy. The SPA never
// needs to know it's talking through a proxy - it just uses the rewritten
// URL as if it were a normal registry root.
//
// URL scheme: /xrproxy/<base64url(remoteOrigin)>/<rest-of-path>[?query]
// The remote origin (e.g. "https://example.com:8080") is base64url-encoded
// (no padding) so it can be embedded as a single path segment without
// colliding with slashes or the rest of the forwarded path/query.

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
)

const XRPROXY_PREFIX = "/xrproxy/"

// hop-by-hop headers that must never be forwarded verbatim (per RFC 7230
// 6.1) - copied from the standard reverse-proxy pattern.
var xrproxyHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
	// Content-Length is recomputed below (the rewritten body can be a
	// different length than the remote's original). Content-Encoding is
	// dropped because Go's http.Transport transparently gunzips the
	// remote's response for us when we don't forward Accept-Encoding
	// ourselves (see below) — relaying a stale "gzip" marker on top of
	// already-decoded bytes would make the browser fail to parse the
	// response. Accept-Encoding itself is excluded from the *outgoing*
	// request for the same reason: setting it ourselves would disable
	// Go's automatic transparent decompression.
	"Content-Length",
	"Content-Encoding",
	"Accept-Encoding",
	// Referer is dropped from the outgoing (proxy -> remote) request:
	// it's the browser's own page URL (e.g. our SPA's http://localhost
	// address), not anything about how the remote itself is being
	// reached. Some remotes (including this server's own code, when
	// deployed behind a TLS-terminating reverse proxy that hides TLS
	// from Go's r.TLS) fall back to sniffing an incoming Referer's
	// scheme to decide their own self/*url scheme - forwarding our SPA's
	// http:// Referer verbatim tricks such a remote into reporting itself
	// as http even though we reached it over real https. Confirmed via
	// live testing against https://xregistry.soaphub.org (reproducible
	// with any curl -H "Referer: http://..." request).
	"Referer",
}

// EncodeXRProxyOrigin base64url-encodes (no padding) a remote origin for use
// as an /xrproxy/ path segment.
func EncodeXRProxyOrigin(origin string) string {
	trimmed := strings.TrimRight(origin, "/")
	return base64.RawURLEncoding.EncodeToString([]byte(trimmed))
}

// decodeXRProxyOrigin reverses EncodeXRProxyOrigin, returning "" if the
// segment isn't valid base64url or doesn't decode to a plausible
// http(s):// origin.
func decodeXRProxyOrigin(seg string) string {
	data, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return ""
	}
	origin := string(data)
	isHTTP := strings.HasPrefix(origin, "http://")
	isHTTPS := strings.HasPrefix(origin, "https://")
	if !isHTTP && !isHTTPS {
		return ""
	}
	return origin
}

// ourOrigin computes the scheme+host this server is being reached as,
// mirroring the logic in NewRequestInfo (BaseURL) so links we generate use
// the same scheme the browser used to reach us.
func ourOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if tmp := r.Header.Get("Referer"); strings.HasPrefix(tmp, "https:") {
		scheme = "https"
	} else if tmp := r.Header.Get("Forwarded"); strings.Contains(tmp, "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

var xrproxyClient = &http.Client{
	Timeout: 30 * time.Second,
	// Don't auto-follow redirects - relay them as-is (with rewritten
	// Location headers) so the browser follows them through the proxy too.
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// HTTPXRProxy handles requests under /xrproxy/<encodedOrigin>/<path...>. It
// forwards the request to the remote origin, then rewrites any occurrence
// of that origin in the response body and Location/Content-Location headers
// to point back through this proxy, so the SPA can treat the proxied
// registry exactly like a normal same-origin one.
func HTTPXRProxy(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, XRPROXY_PREFIX)
	seg, remainder, _ := strings.Cut(rest, "/")

	remoteOrigin := decodeXRProxyOrigin(seg)
	if remoteOrigin == "" {
		http.Error(w, "invalid /xrproxy/ URL - bad or missing encoded origin",
			http.StatusBadRequest)
		return
	}

	targetPath := "/" + remainder
	targetURL := remoteOrigin + targetPath
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	log.VPrintf(3, "xrproxy: %s %s -> %s", r.Method, r.URL.Path, targetURL)

	var body io.Reader
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "error reading request body: "+err.Error(),
				http.StatusBadRequest)
			return
		}
		body = bytes.NewReader(data)
	}

	outReq, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		http.Error(w, "error building proxied request: "+err.Error(),
			http.StatusBadGateway)
		return
	}

	copyXRProxyHeaders(outReq.Header, r.Header)

	resp, err := xrproxyClient.Do(outReq)
	if err != nil {
		msg := "error contacting remote registry (" + remoteOrigin + "): " +
			err.Error()
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "error reading remote response: "+err.Error(),
			http.StatusBadGateway)
		return
	}

	ourPrefix := ourOrigin(r) + XRPROXY_PREFIX + seg

	// Rewrite any occurrence of the remote's own origin so embedded
	// absolute URLs (self, *url attributes, Location/Link headers, etc.)
	// route back through this proxy. We match BOTH the http:// and
	// https:// variants of the remote's host (not just the scheme we
	// actually connected with) because some real-world remotes are
	// inconsistent about which scheme they report in their own self
	// URLs (e.g. a CDN/cache in front of the origin occasionally serving
	// a response body that was generated via an internal http hop, even
	// though we always connect to the origin over https) - confirmed via
	// live testing against a public xRegistry server. Rewriting both
	// variants is harmless when the remote is perfectly consistent.
	remoteHost := strings.TrimPrefix(strings.TrimPrefix(remoteOrigin,
		"https://"), "http://")
	remoteOriginHTTP := "http://" + remoteHost
	remoteOriginHTTPS := "https://" + remoteHost
	data = bytes.ReplaceAll(data, []byte(remoteOriginHTTPS), []byte(ourPrefix))
	data = bytes.ReplaceAll(data, []byte(remoteOriginHTTP), []byte(ourPrefix))

	outHeader := w.Header()
	copyXRProxyHeaders(outHeader, resp.Header)
	for _, h := range []string{"Location", "Content-Location", "Link"} {
		if v := outHeader.Get(h); v != "" {
			v = strings.ReplaceAll(v, remoteOriginHTTPS, ourPrefix)
			v = strings.ReplaceAll(v, remoteOriginHTTP, ourPrefix)
			outHeader.Set(h, v)
		}
	}
	// Same-origin now (browser only ever talks to us), but set these
	// anyway in case something still probes cross-origin during dev.
	outHeader.Set("Access-Control-Allow-Origin", "*")
	outHeader.Set("Access-Control-Allow-Methods", "GET, PATCH, POST, PUT, DELETE")
	outHeader.Set("Content-Length", strconv.Itoa(len(data)))

	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}

// copyXRProxyHeaders copies all headers from src to dst except hop-by-hop
// ones that must never be forwarded/relayed as-is.
func copyXRProxyHeaders(dst, src http.Header) {
	for k, vv := range src {
		if isXRProxyHopHeader(k) {
			continue
		}
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func isXRProxyHopHeader(name string) bool {
	for _, h := range xrproxyHopHeaders {
		if strings.EqualFold(h, name) {
			return true
		}
	}
	return false
}
