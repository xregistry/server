package xrlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestHttpDoNullErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "null")
		}))
	defer server.Close()

	res, xErr := HttpDo(&HttpRequest{
		Verb:  "GET",
		URL:   server.URL + "/",
		Debug: false,
	})

	if res == nil || res.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected response: %#v", res)
	}
	if xErr == nil {
		t.Fatal("expected a non-nil error")
	}
	if string(res.Body) != "null" {
		t.Fatalf("unexpected response body: %q", res.Body)
	}
}
