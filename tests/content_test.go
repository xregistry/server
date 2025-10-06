package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestResourceContents(t *testing.T) {
	reg := NewRegistry("TestResourceContents")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("str1", STRING)
	rm.AddAttr("str2", STRING)
	rm.AddAttr("int1", INTEGER)
	rm.AddAttr("int2", INTEGER)
	rm.AddAttr("int3", INTEGER)
	rm.AddAttr("bool1", BOOLEAN)
	rm.AddAttr("bool2", BOOLEAN)
	rm.AddAttr("dec1", DECIMAL)
	rm.AddAttr("dec2", DECIMAL)
	rm.AddAttr("dec3", DECIMAL)

	d1, err := reg.AddGroup("dirs", "d1")
	xNoErr(t, err)
	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)

	f1.SetSaveDefault("name", "file1")
	f1.SetSaveDefault("labels.str1", "foo")
	f1.SetSaveDefault("labels.str2", "")
	// f1.SetSaveDefault("labels.int", 6)
	// f1.SetSaveDefault("labels.bool", true)
	// f1.SetSaveDefault("labels.decimal", 123.456)
	f1.SetSaveDefault("str1", "foo")
	f1.SetSaveDefault("str2", "")
	f1.SetSaveDefault("int1", 6)
	f1.SetSaveDefault("int2", -5)
	f1.SetSaveDefault("int3", 0)
	f1.SetSaveDefault("bool1", true)
	f1.SetSaveDefault("bool2", false)
	f1.SetSaveDefault("dec1", 123.456)
	f1.SetSaveDefault("dec2", -456.876)
	f1.SetSaveDefault("dec3", 0.0)

	f1.SetSaveDefault("file", "Hello there")

	xCheckEqual(t, "", Any2String(f1.Get("file")), "Hello there")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "Hello there",
		Headers: nil,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v1",
		Body:    "Hello there",
		Headers: nil,
	})

	v2, err := f1.AddVersion("v2")
	xNoErr(t, err)
	v2.SetSave("file", "This is version 2")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "This is version 2",
		Headers: nil,
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v2",
		Body:    "This is version 2",
		Headers: nil,
	})

	v3, _ := f1.AddVersion("v3")
	v3.SetSave("fileproxyurl", "http://example.com")

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	CompareContentMeta(t, reg, &Test{
		Code:    200,
		URL:     "dirs/d1/files/f1/versions/v3",
		Body:    "*Example Domain", // contains
		Headers: []string{"Content-Type: text/html"},
	})

	v4, _ := f1.AddVersion("v4")
	v4.SetSave("fileurl", "http://example.com")

	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1",
		Body: "",
		Headers: []string{
			"xRegistry-fileurl: http://example.com",
			"Location: http://example.com",
		},
	})

	CompareContentMeta(t, reg, &Test{
		Code: 303,
		URL:  "dirs/d1/files/f1/versions/v4",
		Body: "",
		Headers: []string{
			"xRegistry-fileurl: http://example.com",
			"Location: http://example.com",
		},
	})

	// v4 = fileURL
	xHTTP(t, reg, "PATCH", "dirs/d1/files/f1/versions/v4$details?inline=file",
		`{"contenttype":null, "description":"hi"}`, 200, `{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
  "xid": "/dirs/d1/files/f1/versions/v4",
  "epoch": 2,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "v3",

  "fileurl": "http://example.com"
}
`)

	xHTTP(t, reg, "GET", "dirs/d1/files/f1$details?doc&inline=file",
		`{"contenttype":null, "description":"hi"}`, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`)

	// Set default to v2
	xHTTP(t, reg, "PATCH", "dirs/d1/files/f1/meta",
		`{"defaultversionid":"v2", "defaultversionsticky":true}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 5,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
  "defaultversionsticky": true
}
`)

	// v2 = file
	xHTTP(t, reg, "PATCH", "dirs/d1/files/f1$details?doc&inline=file",
		`{"contenttype":null, "description":"hi"}`, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`)

}

type Test struct {
	Code    int
	URL     string
	Body    string
	Headers []string
}

func CompareContentMeta(t *testing.T, reg *registry.Registry, test *Test) {
	t.Helper()
	xNoErr(t, reg.SaveAllAndCommit())

	u := test.URL

	t.Logf("Testing: URL: %s", test.URL)
	metaResp, err := http.Get("http://localhost:8181/" + u + "$details")
	xNoErr(t, err)
	if metaResp == nil {
		t.Fatalf("metaResp is nil")
	}
	metaBody, err := io.ReadAll(metaResp.Body)
	xNoErr(t, err)
	if metaResp.StatusCode/100 != 2 {
		t.Fatalf("Bad response: %s\n%s", metaResp.Status, metaBody)
	}
	metaProps := map[string]any{}
	err = json.Unmarshal(metaBody, &metaProps)
	if err != nil {
		t.Fatalf("Err: %s\n Body:\n%s", err, string(metaBody))
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}

	res, err := client.Get("http://localhost:8181/" + u)
	xNoErr(t, err)

	resBody, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	xCheck(t, res.StatusCode == test.Code,
		fmt.Sprintf("\nTest: %s\nBad http code: %d should be %d\n%s", u,
			res.StatusCode, test.Code, string(resBody)))

	// Make sure any headers have the expected text someplace
	for _, header := range test.Headers {
		name, value, _ := strings.Cut(header, ":")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		h := res.Header.Get(name)
		xCheck(t, strings.Contains(h, value),
			fmt.Sprintf("Test %s\nHeader %q(%s) should be %q",
				u, name, h, value))
	}

	if test.Body == "" || test.Body[0] != '*' {
		xCheckEqual(t, "body", string(resBody), test.Body)
	} else {
		if !strings.Contains(string(resBody), test.Body[1:]) {
			t.Fatalf("Unexpected body for %q\nGot:\n%s\nExpected:\n%s",
				u, string(resBody), test.Body[1:])
		}
	}

	headerLabels := map[string]string{}

	for name, value := range res.Header {
		name = strings.ToLower(name)
		// t.Logf("Header: %s", name)

		// Special!  TODO make this less special
		if strings.ToLower(name) == "content-type" {
			// Only check if metaProps has it
			if metaProps["contenttype"] == nil {
				continue
			}
			name = "xregistry-contenttype"
		}

		if !strings.HasPrefix(name, "xregistry-") {
			continue
		}
		name = name[len("xregistry-"):]

		if strings.HasPrefix(name, "labels-") {
			headerLabels[name[7:]] = value[0]
			continue
		}

		// t.Logf("metaProps:\n%s\n", ToJSON(metaProps))

		foundIt := false
		for propName, propValue := range metaProps {
			if strings.ToLower(propName) != name {
				continue
			}

			delete(metaProps, propName)
			foundIt = true

			// TODO will need to sort the maps before diff'ing
			str := ""
			str = fmt.Sprintf("%v", propValue)
			if name == "self" || name == "defaultversionurl" {
				xCheckEqual(t, propName, value[0]+"$details", str)
				break
			}

			xCheckEqual(t, propName+"(exp:header, got:$details)", value[0], str)
			break
		}
		if !foundIt {
			t.Fatalf("Missing %q in $details version(%s)", name, u)
		}
	}

	var metaLabels map[string]interface{}
	if tmp := metaProps["labels"]; tmp != nil {
		metaLabels = tmp.(map[string]interface{})
	}

	for k, v := range headerLabels {
		metaVal, ok := metaLabels[k]
		if !ok {
			t.Fatalf("metaLabel %v is missing: %s", k, u)
			continue
		}

		metaStr := fmt.Sprintf("%v", metaVal)
		if v != metaStr {
			t.Fatalf("metaLabel %v value mismatch(%q vs %q): %s",
				k, v, metaStr, u)
		}
		delete(metaLabels, k)
	}
	if len(metaLabels) != 0 {
		t.Fatalf("Extra metaLabels: %v for url: %q", metaLabels, u)
	}

	for propName, _ := range metaProps {
		if propName == "labels" {
			continue
		}
		t.Fatalf("Extra prop %q in $details, not in header: %s", propName, u)
	}
}
