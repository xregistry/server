package registry

import (
	// "maps"
	"net/http"
	"net/url"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

func TestInfoIgnore(t *testing.T) {
	for _, test := range []struct {
		URL string
		exp map[string]bool
		err string
	}{
		{"", map[string]bool{}, ""},
		{"?ignore", map[string]bool{"*": true}, ""},
		{"?ignore=", map[string]bool{"*": true}, ""},
		{"?ignore=*", map[string]bool{"*": true}, ""},
		{"?ignore=,", map[string]bool{}, ""},
		{"?ignore=epoch", map[string]bool{"epoch": true}, ""},
		{"?ignore=epoch,", map[string]bool{"epoch": true}, ""},
		{"?ignore=,epoch,", map[string]bool{"epoch": true}, ""},
		{"?ignore=epoch,modelsource", map[string]bool{"epoch": true, "modelsource": true}, ""},
		{"?ignore=epoch&ignore=modelsource", map[string]bool{"epoch": true, "modelsource": true}, ""},
		{"?ignore=modelsource&ignore=epoch,capabilities", map[string]bool{"epoch": true, "modelsource": true, "capabilities": true}, ""},

		// errors
		{"?ignore=foo", nil, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_ignore",
  "title": "An error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly.",
  "subject": "/",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly",
    "value": "foo"
  },
  "source": "0e8077782f41:registry:info:465"
}`,
		},
		{"?ignore&ignore=foo", nil, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_ignore",
  "title": "An error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly.",
  "subject": "/",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly",
    "value": "foo"
  },
  "source": "0e8077782f41:registry:info:465"
}`,
		},
		{"?ignore=*,foo", nil, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_ignore",
  "title": "An error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly.",
  "subject": "/",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,modelsource,readonly",
    "value": "foo"
  },
  "source": "0e8077782f41:registry:info:465"
}`},
	} {
		t.Logf("URL: %s", test.URL)
		info := &RequestInfo{
			Registry: &Registry{
				Capabilities: DefaultCapabilities,
			},
			OriginalRequest: &http.Request{},
		}
		info.OriginalRequest.URL, _ = url.Parse(test.URL)
		xErr := info.ParseRequestURL()

		// Allow "*" to mean ANY error, but not "no error"
		if xErr != nil || test.err != "" {
			if test.err == "*" {
				continue
			}
			XCheckErr(t, xErr, test.err)
			continue
		}

		if strings.Contains(test.URL, "ignore") {
			if _, ok := info.Flags["ignore"]; !ok {
				t.Fatalf("URL: %q should have 'ignore' in flags, but doesn't",
					test.URL)
			}
		} else {
			if _, ok := info.Flags["ignore"]; ok {
				t.Fatalf("URL: %q should NOT have 'ignore' in flags, but does",
					test.URL)
			}
		}

		XEqual(t, "URL: "+test.URL, info.Ignores, test.exp)
	}
}
