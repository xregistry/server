package tests

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestTimestampRegistry(t *testing.T) {
	reg := NewRegistry("TestTimestampRegistry")
	defer PassDeleteReg(t, reg)

	// Check basic GET first
	res := XHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	// Should be the same values
	data := res.ToMap()
	regCreate := data["createdat"].(string)
	regMod := data["modifiedat"].(string)
	XEqual(t, "", regCreate, regMod)

	// Test to make sure modify timestamp changes, but created didn't
	res = XHTTP(t, reg, "PATCH", "/", `{"description":"my docs"}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`)

	data = res.ToMap()
	newMod := data["modifiedat"].(string)
	XEqual(t, "", data["createdat"].(string), regCreate)
	XCheck(t, regMod != newMod, "should be new time")

	// Mod should be higher than before
	XCheck(t, newMod > regMod, "Mod should be newer than before")
	regMod = newMod

	XCheck(t, regMod > regCreate, "Mod should be newer than create")

	// Now test with Groups and Resources
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file"
        }
      }
    }
  }
}`, 200, "*")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:03Z",
  "modifiedat": "2024-01-01T12:00:03Z",
  "ancestorid": "v1"
}
`)

	res = XHTTP(t, reg, "GET", "/?inline", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:03Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:03Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:03Z",
          "modifiedat": "2024-01-01T12:00:03Z",
          "ancestorid": "v1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:03Z",
            "modifiedat": "2024-01-01T12:00:03Z",
            "readonly": false,

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:03Z",
              "ancestorid": "v1"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	data = res.ToMap()
	dirs := data["dirs"].(map[string]any)
	d1 := dirs["d1"].(map[string]any)
	dCTime := d1["createdat"].(string)
	dMTime := d1["modifiedat"].(string)

	files := d1["files"].(map[string]any)
	f1 := files["f1"].(map[string]any)
	fCTime := f1["createdat"].(string)
	fMTime := f1["modifiedat"].(string)

	XEqual(t, "", data["createdat"].(string), regCreate)
	// Adding a Group/Resource touches (bumps) the parent Registry's own
	// epoch AND modifiedat, since ancestor propagation re-saves the
	// Registry entity as part of the same commit - so it's expected to
	// now match the new Group/Resource's timestamp, not the old regMod.
	XCheck(t, data["modifiedat"].(string) > regMod,
		"registry modifiedat should be bumped by the child creation")
	XEqual(t, "", data["modifiedat"].(string), dMTime)

	res = XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"description":"myfile"}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "myfile",
  "createdat": "2024-01-01T12:00:03Z",
  "modifiedat": "2024-01-01T12:00:04Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
	newF := res.ToMap()

	dRes := XHTTP(t, reg, "GET", "/dirs/d1", "", 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:03Z",
  "modifiedat": "2024-01-01T12:00:03Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 1
}
`)
	newD := dRes.ToMap()

	XEqual(t, "", dCTime, newD["createdat"].(string))
	XEqual(t, "", dMTime, newD["modifiedat"].(string))
	XEqual(t, "", fCTime, newF["createdat"].(string))
	XCheck(t, fMTime < newF["modifiedat"].(string), "Should not be the same")

	/*
	   	reg = NewRegistry("TestTimestampRegistry2")
	   	defer PassDeleteReg(t, reg)

	   	XCheckHTTP(t, reg, &HTTPTest{
	   		URL:    "/",
	   		Method: "GET",
	   		Code:   200,
	   		ResBody: `{
	     "specversion": "` + SPECVERSION + `",
	     "registryid": "TestTimestampRegistry2",
	     "self": "http://localhost:8181/",
	     "epoch": 1,
	     "createdat": "2024-01-01T12:00:01Z",
	     "modifiedat": "2024-01-01T12:00:01Z"
	   }
	   `})
	*/

	// Test updating registry's times
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	}, NOMASK_TS)

	// Shouldn't need these, but do it anyway
	res = XHTTP(t, reg, "GET", "/", "", 200, "*", NOMASK_TS)
	data = res.ToMap()
	XEqual(t, "", data["createdat"].(string), "1970-01-02T03:04:05Z")
	XEqual(t, "", data["modifiedat"].(string), "2000-05-04T03:02:01Z")

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": null
		}`,
		Code:       200,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	// Test creating a group and setting it's times
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/dirs/d4",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "dirid": "d4",
  "self": "http://localhost:8181/dirs/d4",
  "xid": "/dirs/d4",
  "epoch": 1,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",

  "filesurl": "http://localhost:8181/dirs/d4/files",
  "filescount": 0
}
`,
	})

	gRes := XHTTP(t, reg, "GET", "/dirs/d4", "", 200, "*", NOMASK_TS)
	gData := gRes.ToMap()
	XEqual(t, "", gData["createdat"].(string), "1970-01-02T03:04:05Z")
	XEqual(t, "", gData["modifiedat"].(string), "2000-05-04T03:02:01Z")

	// Test creating a dir/file/version and setting the version's times
	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT reg - set ts",
		URL:        "/dirs/d5/files/f5/versions/v99$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"createdat": "1970-01-02T03:04:05Z",
			"modifiedat": "2000-05-04T03:02:01Z"
		}`,
		Code:       201,
		ResHeaders: []string{"Content-Type:application/json"},
		ResBody: `{
  "fileid": "f5",
  "versionid": "v99",
  "self": "http://localhost:8181/dirs/d5/files/f5/versions/v99$details",
  "xid": "/dirs/d5/files/f5/versions/v99",
  "epoch": 1,
  "isdefault": true,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",
  "ancestorid": "v99"
}
`,
	})

	vRes := XHTTP(t, reg, "GET", "/dirs/d5/files/f5/versions/v99$details", "",
		200, "*", NOMASK_TS)
	vData := vRes.ToMap()
	XEqual(t, "", vData["createdat"].(string), "1970-01-02T03:04:05Z")
	XEqual(t, "", vData["modifiedat"].(string), "2000-05-04T03:02:01Z")
}

func TestTimestampParsing(t *testing.T) {
	reg := NewRegistry("TestTimestampParsing")
	defer PassDeleteReg(t, reg)

	// Check basic GET first
	XCheckGet(t, reg, "/",
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestTimestampParsing",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)

	tests := []struct {
		timestamp string
		code      int
		value     string
		utc       string
	}{
		{"xxx", 400, "", ""},
		{"2024-07-04T12:01:02", 200, "2024-07-04T12:01:02Z", ""},
		{"2024-07-04T12:00:01Z", 200, "2024-07-04T12:00:01Z", ""},
		{"2024-07-04T12:00:01+07:00", 200, "2024-07-04T12:00:01+07:00",
			"2024-07-04T05:00:01Z"},
		{"2024-07-04T12:00:01-07:00", 200, "2024-07-04T12:00:01-07:00",
			"2024-07-04T19:00:01Z"},
		{"2024-07-04T12:00:01", 200, "2024-07-04T12:00:01Z", ""},
	}

	for _, test := range tests {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}}
		buf := []byte(`{"modifiedat":"` + test.timestamp + `"}`)
		body := bytes.NewReader(buf)
		req, err := http.NewRequest("PATCH", "http://localhost:8181/", body)
		XNoErr(t, err)

		res, err := client.Do(req)
		if res != nil {
			buf, _ = io.ReadAll(res.Body)
		}

		XNoErr(t, err)
		if res.StatusCode != test.code {
			t.Logf("TS: %#v", test)
			t.Fatalf("Expected status %d, got %d\n%s",
				test.code, res.StatusCode, string(buf))
		}

		if test.code != 200 {
			continue
		}

		reg.Refresh(registry.FOR_WRITE)
		if test.utc != "" {
			XEqual(t, "", reg.Get("modifiedat"), test.utc, NOMASK_TS)
		} else {
			XEqual(t, "", reg.Get("modifiedat"), test.value, NOMASK_TS)
		}
		XNoErr(t, reg.SaveAllAndCommit())
	}
}
