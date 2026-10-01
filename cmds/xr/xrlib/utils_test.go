package xrlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpDoFilterNullErrorBody(t *testing.T) {
	var requestURI string
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requestURI = r.URL.RequestURI()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "null")
		}))
	defer server.Close()

	res, xErr := HttpDo(false, http.MethodGet,
		server.URL+"/widgets?filter=widgetid=null", nil, nil)
	if res == nil || res.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected response: %#v", res)
	}
	if xErr == nil {
		t.Fatal("expected a non-nil error")
	}
	if requestURI != "/widgets?filter=widgetid=null" {
		t.Fatalf("unexpected request URI: %q", requestURI)
	}
}
