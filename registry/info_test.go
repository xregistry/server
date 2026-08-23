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
  "title": "For \"/?ignore=foo\", an error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly.",
  "subject": "/?ignore=foo",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly",
    "value": "foo"
  },
  "source": "4a51b174cf4e:registry:info:660"
}`,
		},
		{"?ignore&ignore=foo", nil, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_ignore",
  "title": "For \"/?ignore&ignore=foo\", an error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly.",
  "subject": "/?ignore&ignore=foo",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly",
    "value": "foo"
  },
  "source": "4a51b174cf4e:registry:info:660"
}`,
		},
		{"?ignore=*,foo", nil, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_ignore",
  "title": "For \"/?ignore=*,foo\", an error was found in \"ignore\" value (foo): value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly.",
  "subject": "/?ignore=*,foo",
  "args": {
    "error_detail": "value not supported; allowed values: capabilities,defaultversionid,defaultversionsticky,epoch,id,modelsource,readonly",
    "value": "foo"
  },
  "source": "4a51b174cf4e:registry:info:660"
}`},
	} {
		t.Logf("URL: %s", test.URL)
		info := &RequestInfo{
			Registry: &Registry{
				Capabilities: DefaultCapabilities.Clone(),
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
			XCheckErr(t, xErr.ToJSON(""), test.err)
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
