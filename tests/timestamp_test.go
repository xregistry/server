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
	XCheckGet(t, reg, "/",
		`{
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
	regCreate := reg.Get("createdat")
	regMod := reg.Get("modifiedat")
	XEqual(t, "", regCreate, regMod)
	reg.SaveAllAndCommit()
	reg.Refresh(registry.FOR_WRITE)

	// Test to make sure modify timestamp changes, but created didn't
	XNoErr(t, reg.SetSave("description", "my docs"))
	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`})

	XEqual(t, "", reg.Get("createdat"), regCreate)
	XCheck(t, regMod != reg.Get("modifiedat"), "should be new time")

	// Mod should be higher than before
	XCheck(t, ToJSON(reg.Get("modifiedat")) > ToJSON(regMod),
		"Mod should be newer than before")

	reg.Refresh(registry.FOR_WRITE)
	regMod = reg.Get("modifiedat")

	XCheck(t, ToJSON(regMod) > ToJSON(regCreate),
		"Mod should be newer than create")

	// Now test with Groups and Resources
	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, err)

	d, _ := reg.AddGroup("dirs", "d1")
	f, _ := d.AddResource("files", "f1", "v1")

	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/?inline",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestTimestampRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "my docs",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

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
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
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
`})
	dCTime := d.Get("createdat")
	dMTime := d.Get("modifiedat")

	fCTime := f.Get("createdat")
	fMTime := f.Get("modifiedat")

	XEqual(t, "", reg.Get("createdat"), regCreate)
	XEqual(t, "", reg.Get("modifiedat"), regMod)

	XNoErr(t, f.SetSaveDefault("description", "myfile"))

	XEqual(t, "", dCTime, d.Get("createdat"))
	XEqual(t, "", dMTime, d.Get("modifiedat"))
	XEqual(t, "", fCTime, f.Get("createdat"))
	XCheck(t, ToJSON(fMTime) < ToJSON(f.Get("modifiedat")),
		"Should not be the same")

	// Close out any lingering tx
	XNoErr(t, reg.SaveAllAndCommit())

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
  "epoch": 4,
  "createdat": "1970-01-02T03:04:05Z",
  "modifiedat": "2000-05-04T03:02:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	}, NOMASK_TS)
	reg.Refresh(registry.FOR_WRITE)
	// Shouldn't need these, but do it anyway
	XEqual(t, "", reg.Get("createdat"), "1970-01-02T03:04:05Z")
	XEqual(t, "", reg.Get("modifiedat"), "2000-05-04T03:02:01Z")

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
  "epoch": 5,
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

	g, err := reg.FindGroup("dirs", "d4", false, registry.FOR_WRITE)
	XNoErr(t, err)
	XEqual(t, "", g.Get("createdat"), "1970-01-02T03:04:05Z")
	XEqual(t, "", g.Get("modifiedat"), "2000-05-04T03:02:01Z")

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
  "ancestor": "v99"
}
`,
	})

	g, err = reg.FindGroup("dirs", "d5", false, registry.FOR_WRITE)
	XNoErr(t, err)
	r, err := g.FindResource("files", "f5", false, registry.FOR_WRITE)
	XNoErr(t, err)
	v, err := r.FindVersion("v99", false, registry.FOR_WRITE)
	XNoErr(t, err)
	XEqual(t, "", v.Get("createdat"), "1970-01-02T03:04:05Z")
	XEqual(t, "", v.Get("modifiedat"), "2000-05-04T03:02:01Z")
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
