package tests

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestHTTPHasDocumentFalse(t *testing.T) {
	reg := NewRegistry("TestHTTPHasDocumentFalse")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	// plural, singular, versions, verId bool, isDefault bool, hasDocument bool
	_, err = gm.AddResourceModel("bars", "bar", 0, true, true, true)
	rm, err := gm.AddResourceModel("files", "file", 0, true, true, false)
	xNoErr(t, err)
	_, err = rm.AddAttr("*", STRING)
	xNoErr(t, err)

	xHTTP(t, reg, "POST", "/dirs/d1/files$details", `{}`, 400,
		"$details isn't allowed on \"/dirs/d1/files$details\"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions$details", `{}`, 400,
		"$details isn't allowed on \"/dirs/d1/files/f1/versions$details\"\n")

	// Not really a "hasdoc" test, but it has to go someplace :-)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/bars",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-barid: 123",
		},
		ReqBody: `{}`,

		Code: 400,
		ResBody: `Including "xRegistry" headers when "$details" is used is invalid
`,
	})

	// Load up one that has hasdocument=true
	xHTTP(t, reg, "PUT", "/dirs/d1/bars/b1$details", "{}", 201, `{
  "barid": "b1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/bars/b1$details",
  "xid": "/dirs/d1/bars/b1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/bars/b1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions",
  "versionscount": 1
}
`)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files",
		Method: "POST",
		ReqBody: `{
		  "ff1": {
		    "test":"foo"
		  }
		}`,
		Code: 200,
		ResBody: `{
  "ff1": {
    "fileid": "ff1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/ff1",
    "xid": "/dirs/d1/files/ff1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",
    "test": "foo",

    "metaurl": "http://localhost:8181/dirs/d1/files/ff1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/ff1/versions",
    "versionscount": 1
  }
}
`})

	// Make sure that each type of Resource (w/ and w/o hasdoc) has the
	// correct self/defaultversionurl URL (meaing w/ and w/o $details)
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1?inline",
		Method: "GET",

		Code: 200,
		ResBody: `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "barsurl": "http://localhost:8181/dirs/d1/bars",
  "bars": {
    "b1": {
      "barid": "b1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/bars/b1$details",
      "xid": "/dirs/d1/bars/b1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/bars/b1/meta",
      "meta": {
        "barid": "b1",
        "self": "http://localhost:8181/dirs/d1/bars/b1/meta",
        "xid": "/dirs/d1/bars/b1/meta",
        "epoch": 1,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "readonly": false,
        "compatibility": "none",

        "defaultversionid": "1",
        "defaultversionurl": "http://localhost:8181/dirs/d1/bars/b1/versions/1$details",
        "defaultversionsticky": false
      },
      "versionsurl": "http://localhost:8181/dirs/d1/bars/b1/versions",
      "versions": {
        "1": {
          "barid": "b1",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/bars/b1/versions/1$details",
          "xid": "/dirs/d1/bars/b1/versions/1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z",
          "ancestor": "1"
        }
      },
      "versionscount": 1
    }
  },
  "barscount": 1,
  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "ff1": {
      "fileid": "ff1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/ff1",
      "xid": "/dirs/d1/files/ff1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestor": "1",
      "test": "foo",

      "metaurl": "http://localhost:8181/dirs/d1/files/ff1/meta",
      "meta": {
        "fileid": "ff1",
        "self": "http://localhost:8181/dirs/d1/files/ff1/meta",
        "xid": "/dirs/d1/files/ff1/meta",
        "epoch": 1,
        "createdat": "2024-01-01T12:00:02Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "readonly": false,
        "compatibility": "none",

        "defaultversionid": "1",
        "defaultversionurl": "http://localhost:8181/dirs/d1/files/ff1/versions/1",
        "defaultversionsticky": false
      },
      "versionsurl": "http://localhost:8181/dirs/d1/files/ff1/versions",
      "versions": {
        "1": {
          "fileid": "ff1",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/ff1/versions/1",
          "xid": "/dirs/d1/files/ff1/versions/1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "test": "foo"
        }
      },
      "versionscount": 1
    }
  },
  "filescount": 1
}
`,
	})

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"foo":"test"}`, 201,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "foo": "test",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Make sure $details is ok on GET and not in response
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "foo": "test",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// also make sure $details is ok on PUT
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{"foo2":"test2"}`, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "foo2": "test2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1", `{"versionid":"2","foo2":"test2"}`, 201,
		`{
  "fileid": "f1",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/2",
  "xid": "/dirs/d1/files/f1/versions/2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "foo2": "test2"
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"3":{"versionid":"3","foo3":"test3"}}`, 200,
		`{
  "3": {
    "fileid": "f1",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
    "xid": "/dirs/d1/files/f1/versions/3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "2",
    "foo3": "test3"
  }
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/3", `{"foo3_1":"test3.1"}`, 200,
		`{
  "fileid": "f1",
  "versionid": "3",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/3",
  "xid": "/dirs/d1/files/f1/versions/3",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "2",
  "foo3_1": "test3.1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/4", `{"foo4":"test4"}`, 201,
		`{
  "fileid": "f1",
  "versionid": "4",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/4",
  "xid": "/dirs/d1/files/f1/versions/4",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "3",
  "foo4": "test4"
}
`)

}

func TestHTTPReadOnlyResource(t *testing.T) {
	reg := NewRegistry("TestHTTPReadOnlyResource")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	xNoErr(t, err)

	_, err = gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:           "files",
		Singular:         "file",
		MaxVersions:      PtrInt(0),
		SetVersionId:     PtrBool(true),
		SetDefaultSticky: PtrBool(true),
		HasDocument:      PtrBool(true),
	})
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/dir1", "{}", 201, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 0
}
`)

	d1, err := reg.FindGroup("dirs", "dir1", false, registry.FOR_WRITE)
	xNoErr(t, err)
	xCheck(t, d1 != nil, "d1 should not be nil")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	xCheck(t, f1 != nil, "f1 should not be nil")

	xNoErr(t, f1.SetSaveMeta("readonly", true))

	xHTTP(t, reg, "GET", "/dirs/dir1/files?inline=meta", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f1$details",
    "xid": "/dirs/dir1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/dir1/files/f1/meta",
      "xid": "/dirs/dir1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "readonly": true,
      "compatibility": "none",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/dir1/files/f1/versions/v1$details", "", 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1", "", 405,
		"PATCH is not allowed on Resource documents\n")
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", "{}", 400,
		"Write operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1", "", 400,
		"Write operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f1", "", 400,
		"Write operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f1/versions", "{}", 400,
		"Write operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1", "", 405,
		"PATCH is not allowed on Resource documents\n")
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", "{}", 400,
		"Write operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", "", 400,
		"Write operations on read-only resources are not allowed\n")

	xHTTP(t, reg, "PUT", "/",
		`{"dirs":{"dir1":{"files":{"f1":{}}}}}`, 400,
		"Write operations on read-only resources are not allowed\n")

	xHTTP(t, reg, "PATCH", "/",
		`{"dirs":{"dir1":{"files":{"f1":{}}}}}`, 400,
		"Write operations on read-only resources are not allowed\n")

	xHTTP(t, reg, "DELETE", "/dirs/dir1", ``, 400,
		"Delete operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "DELETE", "/dirs/dir1/files", ``, 400,
		"Delete operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "DELETE", "/dirs/dir1/files/f1", "", 400,
		"Delete operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "DELETE", "/dirs/dir1/files/f1/versions", "", 400,
		"Delete operations on read-only resources are not allowed\n")
	xHTTP(t, reg, "DELETE", "/dirs/dir1/files/f1/versions/v1", "", 400,
		"Delete operations on read-only resources are not allowed\n")
}

func TestDefaultVersionThis(t *testing.T) {
	reg := NewRegistry("TestDefaultVersionThis")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "create res?setdefault=request",
		URL:    "/dirs/d1/files/f1?setdefaultversionid=request",
		Method: "PUT",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Location: http://localhost:8181/dirs/d1/files/f1",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "create v2 - no setdef flag",
		URL:    "/dirs/d1/files/f1/versions/2",
		Method: "PUT",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 2",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"xRegistry-xid: /dirs/d1/files/f1/versions/2",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/2",
			"Content-Disposition:f1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1/versions/1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"xRegistry-xid: /dirs/d1/files/f1/versions/1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
	})

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1$details?setdefaultversionid", "{}", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1$details?setdefaultversionid=", "{}", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions?setdefaultversionid=request", "{}", 400, `Can't use 'request' if a version wasn't processed`+"\n")

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?setdefaultversionid", "", 400, `"setdefaultversionid" must not be empty`+"\n")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1?setdefaultversionid=", "", 400, `"setdefaultversionid" must not be empty`+"\n")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1?setdefaultversionid=request",
		Method: "POST",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 3",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/3",
			"xRegistry-xid: /dirs/d1/files/f1/versions/3",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 2",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/3",
			"Content-Disposition:f1",
		},
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "POST",
		Code:   201,
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: 4",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1/versions/4",
			"xRegistry-xid: /dirs/d1/files/f1/versions/4",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: false",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 3",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/4",
			"Content-Disposition:f1",
		},
	})

	// Just move sticky ptr
	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1$details",
		Method:  "PATCH",
		ReqBody: `{"meta":{"defaultversionid":"1","defaultversionsticky":true}}`,
		Code:    200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`,
	})

	// delete version that's default
	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/versions/1",
		Method:  "DELETE",
		Code:    204,
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1$details",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "4",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1/versions/4?setdefaultversionid=2",
		Method:  "DELETE",
		Code:    204,
		ResBody: "",
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "check v1",
		URL:    "/dirs/d1/files/f1$details",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})
}

func TestHTTPContent(t *testing.T) {
	reg := NewRegistry("TestHTTPContent")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")

	// Simple string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"file": "hello"
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
		ResBody: `hello`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "hello",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Escaped string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"file": "\"hel\nlo"
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Length:7",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
			"Content-Type:application/json",
		},
		ResBody: "\"hel\nlo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "\"hel\nlo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Pure JSON - map
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": { "foo": "bar" }
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:18",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
		ResBody: `{
  "foo": "bar"
}`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": {
    "foo": "bar"
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Pure JSON - array
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": [ "hello", null, 5 ]
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:26",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
		ResBody: `[
  "hello",
  null,
  5
]`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": [
    "hello",
    null,
    5
  ],

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Pure JSON - numeric
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "application/json",
	"file": 123
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:5",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f1/versions/1",
			"Content-Disposition:f1",
		},
		ResBody: `123`,
	})
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=file", "", 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": 123,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// base64 - string - with quotes
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "ImhlbGxvIgo="
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "ImhlbGxvIgo=",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// base64 - string - w/o quotes
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "aGVsbG8K"
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 7,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 7,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "aGVsbG8K",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// base64 - json
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"filebase64": "eyAiZm9vIjoiYmFyIjogfQo="
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 8,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 8,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "eyAiZm9vIjoiYmFyIjogfQo=",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// base64 - empty
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"filebase64": ""
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 9,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 9,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// base64 - null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"filebase64": null
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 10,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 10,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// file - null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"file": null
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 11,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details?inline=file",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 11,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// Pure JSON - error
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1$details",
		Method: "PUT",
		ReqBody: `{
	"file": { bad bad json }
`,
		Code: 400,
		ResBody: `path '.file': parsing object key: expected string starting with '"', got 'b' instead
`,
	})

	// New implied json - empty string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11$details?inline=file",
		Method: "PUT",
		ReqBody: `{
	"file": ""
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f11",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f11$details",
  "xid": "/dirs/d1/files/f11",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "",

  "metaurl": "http://localhost:8181/dirs/d1/files/f11/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f11/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f11",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f11",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f11",
			"xRegistry-xid: /dirs/d1/files/f11",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f11/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f11/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:0",
			"Content-Location:http://localhost:8181/dirs/d1/files/f11/versions/1",
			"Content-Disposition:f11",
		},
		ResBody: ``,
	})

	// New implied json - obj
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f12$details",
		Method: "PUT",
		ReqBody: `{
	"file": { "foo": "bar" }
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f12",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f12$details",
  "xid": "/dirs/d1/files/f12",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f12/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f12/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f12",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f12",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f12",
			"xRegistry-xid: /dirs/d1/files/f12",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f12/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f12/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:18",
			"Content-Location:http://localhost:8181/dirs/d1/files/f12/versions/1",
			"Content-Disposition:f12",
		},
		ResBody: `{
  "foo": "bar"
}`,
	})

	// New implied json - numeric
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f13$details",
		Method: "PUT",
		ReqBody: `{
	"file": 123
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f13",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f13$details",
  "xid": "/dirs/d1/files/f13",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f13/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f13/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f13",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f13",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f13",
			"xRegistry-xid: /dirs/d1/files/f13",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f13/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f13/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f13/versions/1",
			"Content-Disposition:f13",
		},
		ResBody: `123`,
	})

	// New implied json - array
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f14$details",
		Method: "PUT",
		ReqBody: `{
	"file": [ 123, 0 ]
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f14",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f14$details",
  "xid": "/dirs/d1/files/f14",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f14/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f14/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f14",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f14",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f14",
			"xRegistry-xid: /dirs/d1/files/f14",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f14/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f14/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:14",
			"Content-Location:http://localhost:8181/dirs/d1/files/f14/versions/1",
			"Content-Disposition:f14",
		},
		ResBody: `[
  123,
  0
]`,
	})

	// New implied json - bool
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f15$details",
		Method: "PUT",
		ReqBody: `{
	"file": true
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f15",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f15$details",
  "xid": "/dirs/d1/files/f15",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f15/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f15/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f15",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f15",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f15",
			"xRegistry-xid: /dirs/d1/files/f15",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f15/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f15/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:4",
			"Content-Location:http://localhost:8181/dirs/d1/files/f15/versions/1",
			"Content-Disposition:f15",
		},
		ResBody: `true`,
	})

	// New implied json - string
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f16$details",
		Method: "PUT",
		ReqBody: `{
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f16",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f16$details",
  "xid": "/dirs/d1/files/f16",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f16/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f16/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f16",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f16",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f16",
			"xRegistry-xid: /dirs/d1/files/f16",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f16/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f16/versions",
			"xRegistry-versionscount:1",
			"Content-Type:application/json",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f16/versions/1",
			"Content-Disposition:f16",
		},
		ResBody: "he\tllo",
	})

	// New unknown type
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f17$details",
		Method: "PUT",
		ReqBody: `{
	"contenttype": "foo/bar",
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f17",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f17$details",
  "xid": "/dirs/d1/files/f17",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "foo/bar",

  "metaurl": "http://localhost:8181/dirs/d1/files/f17/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f17/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f17",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f17",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f17",
			"xRegistry-xid: /dirs/d1/files/f17",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f17/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f17/versions",
			"xRegistry-versionscount:1",
			"Content-Type:foo/bar",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f17/versions/1",
			"Content-Disposition:f17",
		},
		ResBody: "he\tllo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f17$details?inline=file", ``, 200, `{
  "fileid": "f17",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f17$details",
  "xid": "/dirs/d1/files/f17",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "foo/bar",
  "filebase64": "aGUJbGxv",

  "metaurl": "http://localhost:8181/dirs/d1/files/f17/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f17/versions",
  "versionscount": 1
}
`)

	// New unknown type - contenttype:null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18$details",
		Method: "PUT",
		ReqBody: `{
	"contenttype": null,
	"file": "he\tllo"
}
`,
		Code: 201,
		ResBody: `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f18",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-xid: /dirs/d1/files/f18",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f18/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:6",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Disposition:f18",
		},
		ResBody: "he\tllo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18$details?inline=file", ``, 200, `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "filebase64": "aGUJbGxv",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`)

	// patch - contenttype:null
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18$details",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": null,
	"file": "foo"
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f18",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-xid: /dirs/d1/files/f18",
			"xRegistry-epoch:2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f18/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Disposition:f18",
		},
		ResBody: "foo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18$details?inline=file", ``, 200, `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "Zm9v",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`)

	// patch - no ct saved, implied json, set ct
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18$details",
		Method: "PATCH",
		ReqBody: `{
	"file": "foo"
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f18",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-xid: /dirs/d1/files/f18",
			"xRegistry-epoch:3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f18/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Disposition:f18",
			"Content-Type:application/json",
		},
		ResBody: "foo",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18$details?inline=file", ``, 200, `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "application/json",
  "file": "foo",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`)

	// patch - include odd ct
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18$details",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": "foo/bar",
	"file": "bar"
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "foo/bar",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f18",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-xid: /dirs/d1/files/f18",
			"xRegistry-epoch:4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f18/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Disposition:f18",
			"Content-Type:foo/bar",
		},
		ResBody: "bar",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18$details?inline=file", ``, 200, `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "contenttype": "foo/bar",
  "filebase64": "YmFy",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`)

	// patch - has ct, set ct to null, file contents should remain
	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18$details?inline=file",
		Method: "PATCH",
		ReqBody: `{
	"contenttype": null
}
`,
		Code: 200,
		ResBody: `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "YmFy",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f18",
		Method: "GET",
		Code:   200,
		ResHeaders: []string{
			"xRegistry-fileid:f18",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f18",
			"xRegistry-xid: /dirs/d1/files/f18",
			"xRegistry-epoch:5",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:02Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f18/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f18/versions",
			"xRegistry-versionscount:1",
			"Content-Length:3",
			"Content-Location:http://localhost:8181/dirs/d1/files/f18/versions/1",
			"Content-Disposition:f18",
		},
		ResBody: "bar",
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f18$details?inline=file", ``, 200, `{
  "fileid": "f18",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f18$details",
  "xid": "/dirs/d1/files/f18",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",
  "filebase64": "YmFy",

  "metaurl": "http://localhost:8181/dirs/d1/files/f18/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f18/versions",
  "versionscount": 1
}
`)

}

func TestHTTPContent2(t *testing.T) {
	reg := NewRegistry("TestHTTPContent")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	reg.AddGroup("dirs", "d1")

	xCheckHTTP(t, reg, &HTTPTest{
		URL:     "/dirs/d1/files/f1",
		Method:  "PUT",
		ReqBody: `hello world`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:00Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `hello world`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"Content-Type: my/format",
		},
		ReqBody: `hello world2`,
		Code:    200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:2",
			"xRegistry-isdefault: true",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:my/format",
		},
		ResBody: `hello world2`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `hello world3`,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch:3",
			"xRegistry-isdefault: true",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Type:my/format", //Not blank because we PATCH headers
		},
		ResBody: `hello world3`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f2$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"contenttype": "my/format",
			"filebase64": "aGVsbG8gd29ybGQ="
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1",
  "contenttype": "my/format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f2$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"filebase64": "aGVsbG8gd29ybGQ="
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f1": {
		    "file": null
		  },
		  "f2": {
		    "filebase64": null
		  },
		  "f3": {
		    "file": "howdy"
		  },
		  "f4": {
			"filebase64": "aGVsbG8gd29ybGQ="
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 4,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  },
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 3,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 1
  },
  "f3": {
    "fileid": "f3",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f3$details",
    "xid": "/dirs/d1/files/f3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",
    "contenttype": "application/json",

    "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
    "versionscount": 1
  },
  "f4": {
    "fileid": "f4",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f4$details",
    "xid": "/dirs/d1/files/f4",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f4/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f4/versions",
    "versionscount": 1
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/fv/versions?setdefaultversionid=v3",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "v1": {
		    "file": null
		  },
		  "v2": {
		    "filebase64": null
		  },
		  "v3": {
		    "file": "howdy"
		  },
		  "v4": {
			"filebase64": "aGVsbG8gd29ybGQ="
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "fileid": "fv",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fv/versions/v1$details",
    "xid": "/dirs/d1/files/fv/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "fv",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/fv/versions/v2$details",
    "xid": "/dirs/d1/files/fv/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v1"
  },
  "v3": {
    "fileid": "fv",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/fv/versions/v3$details",
    "xid": "/dirs/d1/files/fv/versions/v3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v2",
    "contenttype": "application/json"
  },
  "v4": {
    "fileid": "fv",
    "versionid": "v4",
    "self": "http://localhost:8181/dirs/d1/files/fv/versions/v4$details",
    "xid": "/dirs/d1/files/fv/versions/v4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v3"
  }
}
`})
}

func TestHTTPResourcesBulk(t *testing.T) {
	reg := NewRegistry("TestHTTPResourcesBulk")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.AddGroup("dirs", "dir1")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - {}",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - one, just id",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f22": {
		    "fileid": "f22"
		  }
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f22": {
    "fileid": "f22",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f22$details",
    "xid": "/dirs/dir1/files/f22",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f22/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f22/versions",
    "versionscount": 1
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - one, just id",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f23": {
		    "fileid": "bad f23"
		  }
        }`,
		Code: 400,
		ResBody: `The "fileid" attribute must be set to "f23", not "bad f23"
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - one, empty",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f2$details",
    "xid": "/dirs/dir1/files/f2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions",
    "versionscount": 1
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - one, update",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          }
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f2$details",
    "xid": "/dirs/dir1/files/f2",
    "epoch": 2,
    "isdefault": true,
    "description": "foo",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions",
    "versionscount": 1
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - two, update+create, bad ext",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "hello"
          },
          "f3": {
            "foo": "bar"
          }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Invalid extension(s): foo
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources - two, update+create",
		URL:        "/dirs/dir1/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          },
		  "f3": {
		    "labels": {
			  "l1": "hello"
			}
          }
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f2$details",
    "xid": "/dirs/dir1/files/f2",
    "epoch": 3,
    "isdefault": true,
    "description": "foo",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions",
    "versionscount": 1
  },
  "f3": {
    "fileid": "f3",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f3$details",
    "xid": "/dirs/dir1/files/f3",
    "epoch": 1,
    "isdefault": true,
    "labels": {
      "l1": "hello"
    },
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f3/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f3/versions",
    "versionscount": 1
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f1$details - two, bad metadata",
		URL:        "/dirs/dir1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f2": {
            "description": "foo"
          },
		  "f3": {
		    "labels": {
			  "l1": "hello"
			}
          }
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Invalid extension(s): f2,f3
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4$details - new resource - bad id",
		URL:        "/dirs/dir1/files/f4$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "fileid": "f5",
          "description": "my f5"
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `The "fileid" attribute must be set to "f4", not "f5"
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4$details - new resource",
		URL:        "/dirs/dir1/files/f4$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "fileid": "f4",
          "description": "my f4",
          "file": "hello"
        }`,
		Code: 201,
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/dir1/files/f4$details",
		},
		ResBody: `{
  "fileid": "f4",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/dir1/files/f4$details",
  "xid": "/dirs/dir1/files/f4",
  "epoch": 1,
  "isdefault": true,
  "description": "my f4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f4/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f4/versions",
  "versionscount": 1
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET resources/f4 - check doc",
		URL:        "/dirs/dir1/files/f4",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-fileid:f4",
			"xRegistry-versionid: 1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f4",
			"xRegistry-xid: /dirs/dir1/files/f4",
			"xRegistry-epoch:1",
			"xRegistry-description:my f4",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/dir1/files/f4/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/dir1/files/f4/versions",
			"xRegistry-versionscount:1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f4/versions/1",
			"Content-Disposition:f4",
			"Content-Type:application/json",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f4$details - new resource - no id",
		URL:        "/dirs/dir1/files/f5$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "my f5"
        }`,
		Code: 201,
		ResHeaders: []string{
			"Content-Type:application/json",
			"Location:http://localhost:8181/dirs/dir1/files/f5$details",
		},
		ResBody: `{
  "fileid": "f5",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/dir1/files/f5$details",
  "xid": "/dirs/dir1/files/f5",
  "epoch": 1,
  "isdefault": true,
  "description": "my f5",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f5/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f5/versions",
  "versionscount": 1
}
`})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f6 - new res,version - no id",
		URL:    "/dirs/dir1/files/f6",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f6",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f6",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f6/versions/1",
			"xRegistry-xid: /dirs/dir1/files/f6/versions/1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f6",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f6/versions/1",
			"Content-Disposition:f6",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f61 - new res,version - no id+setdef=null",
		URL:    "/dirs/dir1/files/f61?setdefaultversionid=null",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f61",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f61",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f61/versions/1",
			"xRegistry-xid: /dirs/dir1/files/f61/versions/1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f61",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f61/versions/1",
			"Content-Disposition:f61",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f62 - new res,version - no id+setdef=request",
		URL:    "/dirs/dir1/files/f62?setdefaultversionid=request",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f62",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f62",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f62/versions/1",
			"xRegistry-xid: /dirs/dir1/files/f62/versions/1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f62",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f62/versions/1",
			"Content-Disposition:f62",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f63 - new res,version - no id+setdef=1",
		URL:    "/dirs/dir1/files/f63?setdefaultversionid=1",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f63",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f63",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f63/versions/1",
			"xRegistry-xid: /dirs/dir1/files/f63/versions/1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f63",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f63/versions/1",
			"Content-Disposition:f63",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f7 - new res,version - with id",
		URL:    "/dirs/dir1/files/f7",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f7",
			"xRegistry-versionid: v1",
		},
		ReqBody: `hello`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f7",
			"xRegistry-versionid:v1",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f7/versions/v1",
			"xRegistry-xid: /dirs/dir1/files/f7/versions/v1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f7",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"Content-Length:5",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f7/versions/v1",
			"Content-Disposition:f7",
		},
		ResBody: `hello`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST resources/f8$details - new res,version - extra headers",
		URL:    "/dirs/dir1/files/f8$details",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description: my f8",
		},
		ReqBody:    `hello`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Including "xRegistry" headers when "$details" is used is invalid
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9$details - new res,version - empty",
		URL:        "/dirs/dir1/files/f9$details",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f9",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/dir1/files/f9/versions/1$details",
  "xid": "/dirs/dir1/files/f9/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9$details - new res,version - empty",
		URL:        "/dirs/dir1/files/f9$details",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f9",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/dir1/files/f9/versions/2$details",
  "xid": "/dirs/dir1/files/f9/versions/2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f99/versions - new res,version - empty",
		URL:        "/dirs/dir1/files/f99/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Set of Versions to add can't be empty
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f9/versions - new res,version-v1",
		URL:        "/dirs/dir1/files/f9/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "fileid": "f9",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f9/versions/v1$details",
    "xid": "/dirs/dir1/files/f9/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "2"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10/versions?setdefaultversionid=null",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `?setdefaultversionid can not be 'null'
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10/versions?setdefaultversionid=request",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `?setdefaultversionid can not be 'request'
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10a/versions - new res,version-2v,err",
		URL:        "/dirs/dir1/files/f10a/versions?setdefaultversionid=request",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v1": {
    "fileid": "f10a",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f10a/versions/v1$details",
    "xid": "/dirs/dir1/files/f10a/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f99/versions - new res,version-2v,newest=alphabetical",
		URL:        "/dirs/dir1/files/f99/versions",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v9": {},
          "v2": {}
        }`,
		Code: 200,
		ResBody: `{
  "v1": {
    "fileid": "f99",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f99/versions/v1$details",
    "xid": "/dirs/dir1/files/f99/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2025-01-04T13:31:22.013338763Z",
    "modifiedat": "2025-01-04T13:31:22.013338763Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "f99",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/f99/versions/v2$details",
    "xid": "/dirs/dir1/files/f99/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2025-01-04T13:31:22.013338763Z",
    "modifiedat": "2025-01-04T13:31:22.013338763Z",
    "ancestor": "v1"
  },
  "v9": {
    "fileid": "f99",
    "versionid": "v9",
    "self": "http://localhost:8181/dirs/dir1/files/f99/versions/v9$details",
    "xid": "/dirs/dir1/files/f99/versions/v9",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2025-01-04T13:31:22.013338763Z",
    "modifiedat": "2025-01-04T13:31:22.013338763Z",
    "ancestor": "v2"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions - new res,version-1v",
		URL:        "/dirs/dir1/files/f10/versions?setdefaultversionid=v3",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 400,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `Version "v3" not found
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f10/versions - new res,version-1v",
		URL:        "/dirs/dir1/files/f10/versions?setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "fileid": "f10",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f10/versions/v1$details",
    "xid": "/dirs/dir1/files/f10/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f11/versions - new res,version-2v",
		URL:        "/dirs/dir1/files/f11/versions?setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "fileid": "f11",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f11/versions/v1$details",
    "xid": "/dirs/dir1/files/f11/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "f11",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/f11/versions/v2$details",
    "xid": "/dirs/dir1/files/f11/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions - new res,version-2v",
		URL:        "/dirs/dir1/files/f12/versions?setdefaultversionid=v2",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v1": {},
          "v2": {}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "fileid": "f12",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v1$details",
    "xid": "/dirs/dir1/files/f12/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "f12",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v2$details",
    "xid": "/dirs/dir1/files/f12/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions - update,add v",
		URL:        "/dirs/dir1/files/f12/versions?setdefaultversionid=v1",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v3": { "description": "my v3"},
          "v1": { "description": "my v1"},
          "v2": { "description": "my v2"}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v1": {
    "fileid": "f12",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v1$details",
    "xid": "/dirs/dir1/files/f12/versions/v1",
    "epoch": 2,
    "isdefault": true,
    "description": "my v1",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "f12",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v2$details",
    "xid": "/dirs/dir1/files/f12/versions/v2",
    "epoch": 2,
    "isdefault": false,
    "description": "my v2",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1"
  },
  "v3": {
    "fileid": "f12",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v3$details",
    "xid": "/dirs/dir1/files/f12/versions/v3",
    "epoch": 1,
    "isdefault": false,
    "description": "my v3",
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v2"
  }
}
`,
	})

	// Make sure you can point to an existing version
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions - default=existing",
		URL:        "/dirs/dir1/files/f12/versions?setdefaultversionid=v2",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v4": { "description": "my v4"}
        }`,
		Code: 200,
		ResHeaders: []string{
			"Content-Type:application/json",
		},
		ResBody: `{
  "v4": {
    "fileid": "f12",
    "versionid": "v4",
    "self": "http://localhost:8181/dirs/dir1/files/f12/versions/v4$details",
    "xid": "/dirs/dir1/files/f12/versions/v4",
    "epoch": 1,
    "isdefault": false,
    "description": "my v4",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v3"
  }
}
`,
	})

	// Make sure we error if versionid isn't there
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST resources/f12/versions - default=bad",
		URL:        "/dirs/dir1/files/f12/versions?setdefaultversionid=vx",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
          "v4": { "description": "my v4"}
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Version "vx" not found
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources/f13/versions - new v+doc, err",
		URL:    "/dirs/dir1/files/f13/versions/3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 33",
		},
		ReqBody:    `v3`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `The "versionid" attribute must be set to "3", not "33"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT resources/f13/versions - new v+doc+id",
		URL:    "/dirs/dir1/files/f13/versions/3",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-versionid: 3",
		},
		ReqBody: `v3`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f13",
			"xRegistry-versionid:3",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/3",
			"xRegistry-xid: /dirs/dir1/files/f13/versions/3",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 3",
			"Content-Length:2",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/3",
			"Content-Disposition:f13",
		},
		ResBody: `v3`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions - new v+doc+no id",
		URL:        "/dirs/dir1/files/f13/versions/4",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `v4`,
		Code:       201,
		ResHeaders: []string{
			"xRegistry-fileid:f13",
			"xRegistry-versionid:4",
			"xRegistry-self:http://localhost:8181/dirs/dir1/files/f13/versions/4",
			"xRegistry-xid: /dirs/dir1/files/f13/versions/4",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 3",
			"Content-Length:2",
			"Content-Location:http://localhost:8181/dirs/dir1/files/f13/versions/4",
			"Content-Disposition:f13",
		},
		ResBody: `v4`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - empty",
		URL:        "/dirs/dir1/files/f13/versions/5$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f13",
  "versionid": "5",
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/5$details",
  "xid": "/dirs/dir1/files/f13/versions/5",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "4"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - {}",
		URL:        "/dirs/dir1/files/f13/versions/6$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f13",
  "versionid": "6",
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/6$details",
  "xid": "/dirs/dir1/files/f13/versions/6",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "5"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - {} again",
		URL:        "/dirs/dir1/files/f13/versions/6$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f13",
  "versionid": "6",
  "self": "http://localhost:8181/dirs/dir1/files/f13/versions/6$details",
  "xid": "/dirs/dir1/files/f13/versions/6",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "5"
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT resources/f13/versions + meta - bad id update",
		URL:        "/dirs/dir1/files/f13/versions/7$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "versionid": "77" }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `The "versionid" attribute must be set to "7", not "77"
`,
	})

}

func TestHTTPRegistryPatch(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatch")
	defer PassDeleteReg(t, reg)

	reg.Model.AddAttr("regext", STRING)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddAttr("gext", STRING)

	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("rext", STRING)

	g, _ := reg.AddGroup("dirs", "dir1")
	f, err := g.AddResource("files", "f1", "v1")

	xNoErr(t, err)

	reg.SaveAllAndCommit()
	reg.Refresh(registry.FOR_WRITE)
	regCre := reg.GetAsString("createdat")
	regMod := reg.GetAsString("modifiedat")

	// Test PATCHing the Registry

	// skip timestamp masking (the "--TS--")
	xHTTP(t, reg, "GET", "/", ``, 200, `--TS--{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "`+regCre+`",
  "modifiedat": "`+regMod+`",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	reg.Refresh(registry.FOR_WRITE)
	xCheckEqual(t, "", reg.GetAsString("createdat"), regCre)
	xCheckNotEqual(t, "", reg.GetAsString("modifiedat"), regMod)

	xHTTP(t, reg, "PATCH", "/", `{
	  "description": "testing"
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "description": null
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": {}
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "labels": null
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 7,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "regext": "str"
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "regext": "str",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "regext": null
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRegistryPatch",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 9,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Group
	// //////////////////////////////////////////////////////

	gmod := g.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs", `{}`, 200,
		`{}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	g.Refresh(registry.FOR_WRITE)
	xCheck(t, g.GetAsString("modifiedat") != gmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "description": "testing"
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 3,
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "description": null
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 5,
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": {}
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 6,
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "labels": null
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 7,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "gext": "str"
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 8,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "gext": "str",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "gext": null
	}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 9,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Resource
	// //////////////////////////////////////////////////////

	f.Refresh(registry.FOR_WRITE)
	v, _ := f.GetDefault(registry.FOR_WRITE)
	vmod := v.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files", `{}`, 200,
		`{}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1", ``, 405,
		`PATCH is not allowed on Resource documents
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	v.Refresh(registry.FOR_WRITE)
	xCheck(t, v.GetAsString("modifiedat") != vmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "description": "testing"
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "description": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "labels": {}
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "labels": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 7,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "rext": "str"
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 8,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "rext": "str",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "rext": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 9,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test PATCHing a Version
	// //////////////////////////////////////////////////////

	f.Refresh(registry.FOR_WRITE)
	v, _ = f.GetDefault(registry.FOR_WRITE)
	vmod = v.GetAsString("modifiedat")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions", `{}`, 200,
		`{}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1", ``, 405,
		`PATCH is not allowed on Resource documents
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 10,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	v.Refresh(registry.FOR_WRITE)
	xCheck(t, v.GetAsString("modifiedat") != vmod, "Should be diff")

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "description": "testing"
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 11,
  "isdefault": true,
  "description": "testing",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "description": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 12,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "labels": {
	    "foo": "bar"
	  },
	  "createdat": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 13,
  "isdefault": true,
  "labels": {
    "foo": "bar"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "labels": {}
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 14,
  "isdefault": true,
  "labels": {},
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "labels": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 15,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "rext": "str"
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 16,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "rext": "str"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "rext": null
	}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 17,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{
	  "badext": "str"
	}`, 400, `Invalid extension(s): badext
`)

	// Test that PATCH can be used to create stuff too

	xHTTP(t, reg, "PATCH", "/dirs/dir2", `{}`, 201, `{
  "dirid": "dir2",
  "self": "http://localhost:8181/dirs/dir2",
  "xid": "/dirs/dir2",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filesurl": "http://localhost:8181/dirs/dir2/files",
  "filescount": 0
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir2/files/f2$details", `{}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/dir2/files/f2$details",
  "xid": "/dirs/dir2/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/dir2/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/dir2/files/f2/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir2/files/f2/versions/v2$details", `{}`, 201, `{
  "fileid": "f2",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/dir2/files/f2/versions/v2$details",
  "xid": "/dirs/dir2/files/f2/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir3/files/f3/versions/v3$details", `{}`, 201, `{
  "fileid": "f3",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/dir3/files/f3/versions/v3$details",
  "xid": "/dirs/dir3/files/f3/versions/v3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v3"
}
`)

}

func TestHTTPEpoch(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatch")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details", `{}`,
		201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1",
		`{"epoch":null}`, 200, `{
  "dirid": "dir1",
  "self": "http://localhost:8181/dirs/dir1",
  "xid": "/dirs/dir1",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/dir1/files",
  "filescount": 1
}
`)
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details",
		`{"epoch":null}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1$details",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details",
		`{"epoch":null}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

}

func TestHTTPRegistryPatchNoDoc(t *testing.T) {
	reg := NewRegistry("TestHTTPRegistryPatchNoDoc")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	g, _ := reg.AddGroup("dirs", "dir1")
	_, err := g.AddResource("files", "f1", "v1")

	xNoErr(t, err)

	// Just double check $details on PATCH at the same time
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1$details",
		`{"description": "desc"}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "desc",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1",
		`{"description": null}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1",
  "xid": "/dirs/dir1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
  "versionscount": 1
}
`)

	// check $details on request too at the same time
	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1$details",
		`{"description": "desc"}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 4,
  "isdefault": true,
  "description": "desc",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PATCH", "/dirs/dir1/files/f1/versions/v1",
		`{"description": null}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1"
}
`)

}

func TestHTTPResourceCollections(t *testing.T) {
	reg := NewRegistry("TestHTTPResourceCollections")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	// Files + empty
	xHTTP(t, reg, "POST", "/dirs/dir1/files", `{
	  "f1": {},
	  "f2": {}
	}`, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f1",
    "xid": "/dirs/dir1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f1/versions",
    "versionscount": 1
  },
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f2",
    "xid": "/dirs/dir1/files/f2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f2/versions",
    "versionscount": 1
  }
}
`)

	// Files + IDs
	xHTTP(t, reg, "POST", "/dirs/dir1/files", `{
	  "f3": { "fileid": "f3" },
	  "f4": { "fileid": "f4" }
	}`, 200, `{
  "f3": {
    "fileid": "f3",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f3",
    "xid": "/dirs/dir1/files/f3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f3/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f3/versions",
    "versionscount": 1
  },
  "f4": {
    "fileid": "f4",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/dir1/files/f4",
    "xid": "/dirs/dir1/files/f4",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/dir1/files/f4/meta",
    "versionsurl": "http://localhost:8181/dirs/dir1/files/f4/versions",
    "versionscount": 1
  }
}
`)

	// Files + Bad IDs
	xHTTP(t, reg, "POST", "/dirs/dir1/files", `{
	  "f5": { "fileid": "f5" },
	  "f6": { "fileid": "ef6" }
	}`, 400, `The "fileid" attribute must be set to "f6", not "ef6"
`)

	// via file, Versions + empty - new file
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f7?setdefaultversionid=v1", `{
	  "versionid": "v1"
	}`, 201, `{
  "fileid": "f7",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f7/versions/v1",
  "xid": "/dirs/dir1/files/f7/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	// via file, Versions + empty + existing file
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f7?setdefaultversionid=v2", `{
	  "versionid": "v2"
	}`, 201, `{
  "fileid": "f7",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/dir1/files/f7/versions/v2",
  "xid": "/dirs/dir1/files/f7/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	// via file, Versions + empty + existing file + this
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f7?setdefaultversionid=request", `{
	  "versionid": "v3"
	}`, 201, `{
  "fileid": "f7",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/dir1/files/f7/versions/v3",
  "xid": "/dirs/dir1/files/f7/versions/v3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2"
}
`)

	// via file, Versions + empty + existing file + this + existing v
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f7?setdefaultversionid=request", `{
	  "versionid": "v2"
	}`, 200, `{
  "fileid": "f7",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/dir1/files/f7/versions/v2",
  "xid": "/dirs/dir1/files/f7/versions/v2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "v1"
}
`)

	// via file, Versions + empty + existing file + bad def + existing v
	xHTTP(t, reg, "POST", "/dirs/dir1/files/f7?setdefaultversionid=xxx", `{
	  "versionid": "v2"
	}`, 400, `Version "xxx" not found
`)

	// Versions + empty
	xHTTP(t, reg, "POST", "/dirs/dir1/files/ff1/versions?setdefaultversionid=v2", `{
	  "v1": {  },
	  "v2": {  }
	}`, 200, `{
  "v1": {
    "fileid": "ff1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/ff1/versions/v1",
    "xid": "/dirs/dir1/files/ff1/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "ff1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/ff1/versions/v2",
    "xid": "/dirs/dir1/files/ff1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`)

	// Versions + IDs
	xHTTP(t, reg, "POST", "/dirs/dir1/files/ff8/versions?setdefaultversionid=v2", `{
	  "v1": { "versionid": "v1" },
	  "v2": { "versionid": "v2" }
	}`, 200, `{
  "v1": {
    "fileid": "ff8",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/dir1/files/ff8/versions/v1",
    "xid": "/dirs/dir1/files/ff8/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  },
  "v2": {
    "fileid": "ff8",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/dir1/files/ff8/versions/v2",
    "xid": "/dirs/dir1/files/ff8/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1"
  }
}
`)

	// Versions + bad IDs
	xHTTP(t, reg, "POST", "/dirs/dir1/files/ff9/versions?setdefaultversionid=v2", `{
	  "v1": { "versionid": "v1" },
	  "v2": { "versionid": "ev2" }
	}`, 400, `The "versionid" attribute must be set to "v2", not "ev2"
`)
}

func TestHTTPmeta(t *testing.T) {
	reg := NewRegistry("TestHTTPmeta")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1$details", `{}`, 201,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1$details",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "$details", `{}`, 400, `$details isn't allowed on "/$details"
`)
	xHTTP(t, reg, "PUT", "/$details", `{}`, 400, `$details isn't allowed on "/$details"
`)
	xHTTP(t, reg, "PUT", "/dirs$details", `{}`, 400, `$details isn't allowed on "/dirs$details"
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1$details", `{}`, 400, `$details isn't allowed on "/dirs/dir1$details"
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/$details", `{}`, 400,
		`$details isn't allowed on "/dirs/dir1/$details"
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/files$details", `{}`, 400,
		`$details isn't allowed on "/dirs/dir1/files$details"
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/$details", `{}`, 400,
		`"fileid" can't be an empty string
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions$details", `{}`, 400,
		`$details isn't allowed on "/dirs/dir1/files/f1/versions$details"
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/$details", `{}`, 400,
		`Version id in URL can't be blank
`)
	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1/$details", `{}`, 404,
		`URL is too long
`)
}

func TestHTTPURLs(t *testing.T) {
	reg := NewRegistry("TestHTTPURLs")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	// Just simple tests to make sure the most basic tests against the APIs
	// work

	// GET /
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /",
		URL:        "/",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`})

	// PUT /
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
			"description": "a test"
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "a test",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`})

	// PATCH /
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /",
		URL:        "/",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
			"labels": {"l1": "foo"}
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPURLs",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "a test",
  "labels": {
    "l1": "foo"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`})

	// GET /GROUPS
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS",
		URL:        "/dirs",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{}
`})

	// PUT /GROUPS
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS",
		URL:        "/dirs",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `PUT not allowed on collections
`})

	// PATCH /GROUPS
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS",
		URL:        "/dirs",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{}
`})

	// POST /GROUPS
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS",
		URL:        "/dirs",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "d1": {}
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0
  }
}
`})

	// GET /GROUPS/GID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID",
		URL:        "/dirs/d1",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`})

	// PUT /GROUPS/GID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID",
		URL:        "/dirs/d2",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "foo"
        }`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "description": "foo",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`})

	// PATCH /GROUPS/GID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID",
		URL:        "/dirs/d2",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		  "labels": {"l1":"v1"}
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 2,
  "description": "foo",
  "labels": {
    "l1": "v1"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`})

	// GET /GROUPS/GID/RESOURCES
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES",
		URL:        "/dirs/d2/files",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{}
`})

	// PUT /GROUPS/GID/RESOURCES
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID/REOURCES",
		URL:        "/dirs/d2/files",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `PUT not allowed on collections
`})

	// PATCH /GROUPS/GID/RESOURCES
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID/RESOURCES",
		URL:        "/dirs/d2/files",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{}
`})

	// POST /GROUPS/GID/RESOURCES
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS/GID/REOURCES",
		URL:        "/dirs/d2/files",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
		  "f1": {
		    "description": "my f1",
            "file": "hello world"
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d2/files/f1$details",
    "xid": "/dirs/d2/files/f1",
    "epoch": 1,
    "isdefault": true,
    "description": "my f1",
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "1",
    "contenttype": "application/json",

    "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
    "versionscount": 1
  }
}
`})

	// GET /GROUPS/GID/RESOURCES/RID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES/RID",
		URL:        "/dirs/d2/files/f1",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/d2/files/f1",
			"xRegistry-xid: /dirs/d2/files/f1",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:my f1",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:00Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d2/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d2/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Length:11",
			"Content-Location:http://localhost:8181/dirs/d2/files/f1/versions/1",
			"Content-Disposition:f1",
			"Content-Type:application/json",
		},
		ResBody: `hello world`,
	})

	// PUT /GROUPS/GID/RESOURCES/RID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT /GROUPS/GID/REOURCES/RID",
		URL:    "/dirs/d2/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			"xRegistry-description:new f1",
		},
		ReqBody: `Everybody wants to rule the world`,
		Code:    200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid:1",
			"xRegistry-self:http://localhost:8181/dirs/d2/files/f1",
			"xRegistry-xid: /dirs/d2/files/f1",
			"xRegistry-epoch:2",
			"xRegistry-isdefault:true",
			"xRegistry-description:new f1",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl:http://localhost:8181/dirs/d2/files/f1/meta",
			"xRegistry-versionsurl:http://localhost:8181/dirs/d2/files/f1/versions",
			"xRegistry-versionscount:1",
			"Content-Length:33",
			"Content-Location:http://localhost:8181/dirs/d2/files/f1/versions/1",
			"Content-Disposition:f1",
			"Content-Type:application/json",
		},
		ResBody: `Everybody wants to rule the world`,
	})

	// PATCH /GROUPS/GID/RESOURCES/RID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PATCH /GROUPS/GID/REOURCES/RID",
		URL:    "/dirs/d2/files/f1",
		Method: "PATCH",
		ReqHeaders: []string{
			"xRegistry-description:foo",
		},
		ReqBody:    `Everybody wants to rule the world`,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `PATCH is not allowed on Resource documents
`,
	})

	// POST /GROUPS/GID/RESOURCES/RID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST /GROUPS/GID/REOURCES/RID",
		URL:    "/dirs/d2/files/f1",
		Method: "POST",
		ReqHeaders: []string{
			"xRegistry-description:new v",
		},
		ReqBody: `this is a new version`,
		Code:    201,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid:2",
			"xRegistry-self:http://localhost:8181/dirs/d2/files/f1/versions/2",
			"xRegistry-xid: /dirs/d2/files/f1/versions/2",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-description:new v",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:00Z",
			"xRegistry-ancestor: 1",
			"Content-Location:http://localhost:8181/dirs/d2/files/f1/versions/2",
			"Content-Disposition:f1",
			"Content-Type:text/plain; charset=utf-8",
		},
		ResBody: `this is a new version`,
	})

	// GET /GROUPS/GID/RESOURCES/RID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES/RID$details",
		URL:        "/dirs/d2/files/f1$details",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d2/files/f1$details",
  "xid": "/dirs/d2/files/f1",
  "epoch": 1,
  "isdefault": true,
  "description": "new v",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
  "versionscount": 2
}
`})

	// PUT /GROUPS/GID/RESOURCES/RID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID/REOURCES/RID$details",
		URL:        "/dirs/d2/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "description": "update 2" }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d2/files/f1$details",
  "xid": "/dirs/d2/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "update 2",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
  "versionscount": 2
}
`})

	// PATCH /GROUPS/GID/RESOURCES/RID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID/REOURCES/RID$details",
		URL:        "/dirs/d2/files/f1$details",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody:    `{ "labels": {"l1":"v1"} }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d2/files/f1$details",
  "xid": "/dirs/d2/files/f1",
  "epoch": 3,
  "isdefault": true,
  "description": "update 2",
  "labels": {
    "l1": "v1"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
  "versionscount": 2
}
`})

	// POST /GROUPS/GID/RESOURCES/RID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS/GID/REOURCES/RID$details",
		URL:        "/dirs/d2/files/f1$details",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody: `{
			"description": "be 3!",
			"file": "should be 3"
		}`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "3",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/3$details",
  "xid": "/dirs/d2/files/f1/versions/3",
  "epoch": 1,
  "isdefault": true,
  "description": "be 3!",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "2",
  "contenttype": "application/json"
}
`})

	// GET /GROUPS/GID/RESOURCES/RID/versions
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES/RID$details",
		URL:        "/dirs/d2/files/f1/versions",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d2/files/f1/versions/1$details",
    "xid": "/dirs/d2/files/f1/versions/1",
    "epoch": 2,
    "isdefault": false,
    "description": "new f1",
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",
    "contenttype": "application/json"
  },
  "2": {
    "fileid": "f1",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d2/files/f1/versions/2$details",
    "xid": "/dirs/d2/files/f1/versions/2",
    "epoch": 3,
    "isdefault": false,
    "description": "update 2",
    "labels": {
      "l1": "v1"
    },
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "ancestor": "1"
  },
  "3": {
    "fileid": "f1",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d2/files/f1/versions/3$details",
    "xid": "/dirs/d2/files/f1/versions/3",
    "epoch": 1,
    "isdefault": true,
    "description": "be 3!",
    "createdat": "2024-01-01T12:00:04Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "ancestor": "2",
    "contenttype": "application/json"
  }
}
`})

	// PUT /GROUPS/GID/RESOURCES/RID/versions
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID/REOURCES/RID/versions",
		URL:        "/dirs/d2/files/f1/versions",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `PUT not allowed on collections
`})

	// PATCH /GROUPS/GID/RESOURCES/RID/versions
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID/REOURCES/RID/versions",
		URL:        "/dirs/d2/files/f1/versions",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody:    `{}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{}
`})

	// POST /GROUPS/GID/RESOURCES/RID/versions
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS/GID/REOURCES/RID/versions",
		URL:        "/dirs/d2/files/f1/versions?setdefaultversionid=v5",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `{ "v4":{}, "v5":{"file":"hello"}}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "v4": {
    "fileid": "f1",
    "versionid": "v4",
    "self": "http://localhost:8181/dirs/d2/files/f1/versions/v4$details",
    "xid": "/dirs/d2/files/f1/versions/v4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "3"
  },
  "v5": {
    "fileid": "f1",
    "versionid": "v5",
    "self": "http://localhost:8181/dirs/d2/files/f1/versions/v5$details",
    "xid": "/dirs/d2/files/f1/versions/v5",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "ancestor": "v4",
    "contenttype": "application/json"
  }
}
`})

	// GET /GROUPS/GID/RESOURCES/RID/versions/vID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES/RID/versions/vID",
		URL:        "/dirs/d2/files/f1/versions/v5",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid:v5",
			"xRegistry-self:http://localhost:8181/dirs/d2/files/f1/versions/v5",
			"xRegistry-xid: /dirs/d2/files/f1/versions/v5",
			"xRegistry-epoch:1",
			"xRegistry-isdefault:true",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:00Z",
			"xRegistry-ancestor: v4",
			"Content-Type:application/json",
		},
		ResBody: `hello`,
	})

	// PUT /GROUPS/GID/RESOURCES/RID/versions/vID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID/REOURCES/RID/versions/vID",
		URL:        "/dirs/d2/files/f1/versions/v5",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `test doc`,
		Code:       200,
		ResHeaders: []string{
			"xRegistry-fileid:f1",
			"xRegistry-versionid:v5",
			"xRegistry-self:http://localhost:8181/dirs/d2/files/f1/versions/v5",
			"xRegistry-xid: /dirs/d2/files/f1/versions/v5",
			"xRegistry-epoch:2",
			"xRegistry-isdefault:true",
			"xRegistry-createdat:2024-01-01T12:00:00Z",
			"xRegistry-modifiedat:2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v4",
			"Content-Type:application/json",
		},
		ResBody: `test doc`,
	})

	// PATCH /GROUPS/GID/RESOURCES/RID/versions/vID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID/REOURCES/RID/versions/vID",
		URL:        "/dirs/d2/files/f1/versions/v5",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody:    `test doc`,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `PATCH is not allowed on Resource documents
`,
	})

	// POST /GROUPS/GID/RESOURCES/RID/versions/vID
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS/GID/REOURCES/RID/versions/vID",
		URL:        "/dirs/d2/files/f1/versions/v5",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    `test doc`,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `POST not allowed on a version
`,
	})

	// GET /GROUPS/GID/RESOURCES/RID/versions/vID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "GET /GROUPS/GID/REOURCES/RID/versions/vID$details",
		URL:        "/dirs/d2/files/f1/versions/v5$details",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/v5$details",
  "xid": "/dirs/d2/files/f1/versions/v5",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v4",
  "contenttype": "application/json"
}
`})

	// PUT /GROUPS/GID/RESOURCES/RID/versions/vID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /GROUPS/GID/REOURCES/RID/versions/vID$details",
		URL:        "/dirs/d2/files/f1/versions/v5$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "cool one"
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/v5$details",
  "xid": "/dirs/d2/files/f1/versions/v5",
  "epoch": 3,
  "isdefault": true,
  "description": "cool one",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v4"
}
`})

	// PATCH /GROUPS/GID/RESOURCES/RID/versions/vID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /GROUPS/GID/REOURCES/RID/versions/vID$details",
		URL:        "/dirs/d2/files/f1/versions/v5$details",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		  "labels": {"l1": "v1"}
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/v5$details",
  "xid": "/dirs/d2/files/f1/versions/v5",
  "epoch": 4,
  "isdefault": true,
  "description": "cool one",
  "labels": {
    "l1": "v1"
  },
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v4"
}
`})

	// POST /GROUPS/GID/RESOURCES/RID/versions/vID$details
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "POST /GROUPS/GID/REOURCES/RID/versions/vID$details",
		URL:        "/dirs/d2/files/f1/versions/v5$details",
		Method:     "POST",
		ReqHeaders: []string{},
		ReqBody:    ``,
		Code:       405,
		ResHeaders: []string{},
		ResBody: `POST not allowed on a version
`,
	})

}

func TestHTTPGroupResources(t *testing.T) {
	reg := NewRegistry("TestHTTPGroupResources")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	gm.AddResourceModel("datas", "data", 0, true, true, true)

	// Upload some resources into a preexisting group
	// First create the group
	xHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2025-04-03T11:56:00.456981454Z",
  "modifiedat": "2025-04-03T11:56:00.456981454Z",

  "datasurl": "http://localhost:8181/dirs/d1/datas",
  "datascount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	// Now do some "upload resources" tests
	xHTTP(t, reg, "POST", "/dirs/d1", `{"files":{}}`, 200, `{
  "files": {}
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1", `{"files":{"f1":{}}}`, 200, `{
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
      "versionscount": 1
    }
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1", `{"files":{"f1":{},"f2":{}}}`, 200, `{
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
      "versionscount": 1
    },
    "f2": {
      "fileid": "f2",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f2$details",
      "xid": "/dirs/d1/files/f2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
      "versionscount": 1
    }
  }
}
`)

	// Make sure epoch is right
	xHTTP(t, reg, "GET", "/dirs/d1", ``, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "datasurl": "http://localhost:8181/dirs/d1/datas",
  "datascount": 0,
  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1", `{"files":{"f3":{}},"datas":{}}`, 200, `{
  "datas": {},
  "files": {
    "f3": {
      "fileid": "f3",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f3$details",
      "xid": "/dirs/d1/files/f3",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
      "versionscount": 1
    }
  }
}
`)

	xHTTP(t, reg, "POST", "/dirs/d1", `{"files":{"f4":{}},"datas":{"d1":{},"d2":{}}}`, 200, `{
  "datas": {
    "d1": {
      "dataid": "d1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/datas/d1$details",
      "xid": "/dirs/d1/datas/d1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/datas/d1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/datas/d1/versions",
      "versionscount": 1
    },
    "d2": {
      "dataid": "d2",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/datas/d2$details",
      "xid": "/dirs/d1/datas/d2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/datas/d2/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/datas/d2/versions",
      "versionscount": 1
    }
  },
  "files": {
    "f4": {
      "fileid": "f4",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f4$details",
      "xid": "/dirs/d1/files/f4",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f4/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/f4/versions",
      "versionscount": 1
    }
  }
}
`)

	// Make sure epoch is right
	xHTTP(t, reg, "GET", "/dirs/d1", ``, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",

  "datasurl": "http://localhost:8181/dirs/d1/datas",
  "datascount": 2,
  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 4
}
`)

}

func TestHTTPNestedRegistry(t *testing.T) {
	reg := NewRegistry("TestHTTPNestedRegistry")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	// Registry + Nested Groups
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / no groups",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "description": "myreg" }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "myreg",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 0
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + groups",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "myreg2",
		  "dirs": {
		    "d1": {}
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "myreg2",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs", ``, 200,
		`{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 0
  }
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + groups+resources",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "myreg3",
		  "dirs": {
		    "d1": {
			  "description": "d1",
			  "files": {
			    "f1": {}
			  }
			}
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "description": "myreg3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/?inline", ``, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "description": "myreg3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 2,
      "description": "d1",
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",

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

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1"
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

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + groups+resources+versions",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "myreg4",
		  "dirs": {
		    "d1": {
			  "description": "d1.1",
			  "files": {
			    "f1": {
				  "description": "f1",
				  "versions": {
				    "1": {
					  "description": "f1-1"
					}
				  }
                }
			  }
			}
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "description": "myreg4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/?inline", ``, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "description": "myreg4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 3,
      "description": "d1.1",
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 2,
          "isdefault": true,
          "description": "f1-1",
          "createdat": "2024-01-01T12:00:04Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:04Z",
            "modifiedat": "2024-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 2,
              "isdefault": true,
              "description": "f1-1",
              "createdat": "2024-01-01T12:00:04Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1"
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

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + groups+resources+versions*2",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "myreg4",
		  "dirs": {
		    "d1": {
			  "description": "d1.1",
			  "files": {
			    "f1": {
				  "description": "f1",

                  "meta": {
				    "defaultversionsticky": true,
				    "defaultversionid": "2"
                  },
				  "versions": {
				    "1": {
					  "description": "f1-1.1"
					},
				    "2": {
					  "description": "f1-2.1"
					}
				  }
                }
			  }
			}
		  }
		}`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "description": "myreg4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/?inline", ``, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPNestedRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "description": "myreg4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 4,
      "description": "d1.1",
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "2",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "description": "f1-2.1",
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2024-01-01T12:00:04Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "2",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/2$details",
            "defaultversionsticky": true
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "1": {
              "fileid": "f1",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
              "xid": "/dirs/d1/files/f1/versions/1",
              "epoch": 3,
              "isdefault": false,
              "description": "f1-1.1",
              "createdat": "2024-01-01T12:00:04Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1"
            },
            "2": {
              "fileid": "f1",
              "versionid": "2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/2$details",
              "xid": "/dirs/d1/files/f1/versions/2",
              "epoch": 1,
              "isdefault": true,
              "description": "f1-2.1",
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1"
            }
          },
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

}

func TestHTTPNestedResources(t *testing.T) {
	reg := NewRegistry("TestHTTPNestedResources")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	// Registry + Nested Groups
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + new",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    `{ "description": "f1" }`,
		Code:       201,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "description": "f1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "description": "f1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "isdefault": true,
      "description": "f1",
      "createdat": "2024-01-01T12:00:00Z",
      "modifiedat": "2024-01-01T12:00:00Z",
      "ancestor": "1"
    }
  },
  "versionscount": 1
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + bad defaultversionid",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.1",
          "meta": {
		    "defaultversionsticky": true,
		    "defaultversionid": "v2"
		  }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Version "v2" not found
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky not bool",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.1",
		  "meta": {
		    "defaultversionsticky": "hi"
		  }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Attribute "defaultversionsticky" must be a boolean
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky not bool and wrong vid",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.1",
		  "meta": {
		    "defaultversionsticky": "hi",
		    "defaultversionid": "v2"
		  }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Attribute "defaultversionsticky" must be a boolean
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky null",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.2",
		  "meta": {
		    "defaultversionsticky": null,
		    "defaultversionid": "v3"
		  }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Attribute "defaultversionid" must be "1" since "defaultversionsticky" is "false"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky null",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.2",
		  "meta": {
		    "defaultversionsticky": null,
		    "defaultversionid": "1"
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "f1.2",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky null",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.2",
		  "meta": {
		    "defaultversionsticky": null
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "description": "f1.2",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky null",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.2",
		  "meta": {
		    "defaultversionsticky": false
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "description": "f1.2",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + missing sticky",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.3",
		  "meta": {
		    "defaultversionid": "v3"
		  }
        }`,
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Attribute "defaultversionid" must be "1" since "defaultversionsticky" is "false"
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + just sticky",
		URL:        "/dirs/d1/files/f1$details",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f1.3",
		  "meta": {
		    "defaultversionsticky": true
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "description": "f1.3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": true
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + sticky",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		  "meta": {
            "defaultversionsticky": true
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "description": "f1.3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 6,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`,
	})

	// Epoch is bumped
	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + new version",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
          "versions": {
		    "v2": {}
          }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 7,
  "isdefault": true,
  "description": "f1.3",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 7,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`,
	})

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 7,
  "isdefault": true,
  "description": "f1.3",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 7,
    "createdat": "2024-01-01T12:00:00Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 7,
      "isdefault": true,
      "description": "f1.3",
      "createdat": "2024-01-01T12:00:00Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1"
    }
  },
  "versionscount": 2
}
`)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + new version, non-sticky",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
          "meta": {
		    "defaultversionsticky": false
          },
          "versions": {
		    "v3": {}
          }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 8,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + sticky old ver, add newV",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "f2.4",
          "meta": {
		    "defaultversionsticky": true,
		    "defaultversionid": "v2"
          },
		  "versions": {
		    "v4": { "description": "v4.1" }
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "f2.4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 9,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT /RID + defaultversionid=newV",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
          "description": "fx",
          "meta": {
		    "defaultversionsticky": true,
		    "defaultversionid": "v5"
          },
		  "versions": {
		    "v5": { "description": "v5.1" }
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "description": "v5.1",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v4",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 10,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + defaultversionid=oldV",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		  "meta": {
		    "defaultversionid": "v2"
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "description": "f2.4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 11,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + sticky-nochange",
		URL:        "/dirs/d1/files/f1$details?inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
          "meta": {
		    "defaultversionsticky": true
          }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 4,
  "isdefault": true,
  "description": "f2.4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 12,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + ignoredefaultversionsticky",
		URL:        "/dirs/d1/files/f1$details?ignoredefaultversionsticky&inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
          "meta": {
            "defaultversionsticky": null
          }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 5,
  "isdefault": true,
  "description": "f2.4",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 13,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PATCH /RID + ignoredefaultversionid",
		URL:        "/dirs/d1/files/f1$details?ignoredefaultversionid&inline=meta",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		  "meta": {
		    "defaultversionid": "badone.ignored"
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 6,
  "isdefault": true,
  "description": "f2.4",
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 14,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`,
	})

}

func TestHTTPExport(t *testing.T) {
	reg := NewRegistry("TestHTTPExport")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + init load",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
		  "description": "my reg",
		  "dirs": {
		    "d1": {
			  "files": {
			    "d1-f1": {},
				"d1-f2": {}
			  }
			},
			"d2": {
			  "files": {
			    "d2-f1": {},
				"d2-f2": {}
			  }
			}
		  }
        }`,
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPExport",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "my reg",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`,
	})

	res, err := http.Get("http://localhost:8181/")
	xNoErr(t, err)
	body, err := io.ReadAll(res.Body)
	xNoErr(t, err)

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + re-load, ok epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    string(body),
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPExport",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "description": "my reg",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + re-load, bad epoch",
		URL:        "/",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    string(body),
		Code:       400,
		ResHeaders: []string{},
		ResBody: `Attribute "epoch"(2) doesn't match existing value (3)
`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:       "PUT / + re-load, ignore epoch",
		URL:        "/?ignoreepoch",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody:    string(body),
		Code:       200,
		ResHeaders: []string{},
		ResBody: `{
  "specversion": "` + SPECVERSION + `",
  "registryid": "TestHTTPExport",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "description": "my reg",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 2
}
`,
	})

}

func TestHTTPVersionIDs(t *testing.T) {
	reg := NewRegistry("TestHTTPVersionIDs")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{}`, 201,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:00Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "fileid": "f1",
	  "versionid": "v1"
	}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "fileid": "f1"
	}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 4,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "versionid": "v1"
	}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/dir1/files/f1/versions/v1",
  "xid": "/dirs/dir1/files/f1/versions/v1",
  "epoch": 5,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "fileid": "fx",
	  "versionid": "v1"
	}`, 400, "The \"fileid\" attribute must be set to \"f1\", not \"fx\"\n")

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "fileid": "fx"
	}`, 400, "The \"fileid\" attribute must be set to \"f1\", not \"fx\"\n")

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "fileid": "f1",
	  "versionid": "vx"
	}`, 400, "The \"versionid\" attribute must be set to \"v1\", not \"vx\"\n")

	xHTTP(t, reg, "PUT", "/dirs/dir1/files/f1/versions/v1", `{
	  "versionid": "vx"
	}`, 400, "The \"versionid\" attribute must be set to \"v1\", not \"vx\"\n")

}

func TestHTTPRecursiveData(t *testing.T) {
	reg := NewRegistry("TestHTTPRecursiveData")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModelSimple("files", "file")

	// Converting the RESOURCE attributes from JSON into byte array tests

	xHTTP(t, reg, "PATCH", "/", `{
  "dirs": {
    "d1": {
      "files": {
        "f1": {
	      "file": { "foo": "bar" },
          "versions": {
		    "v1": {
			  "file": { "bar": "foo" }
			}
          }
	    },
	    "f2": {
	      "file": "string"
	    },
	    "f3": {
	      "file": 42
	    },
	    "f4": {
	      "file": [ "foo" ]
	    },
	    "f5": {
	      "filebase64": "YmluYXJ5Cg=="
	    }
      }
	}
  }
}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRecursiveData",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-11-07T15:53:55.28040091Z",
  "modifiedat": "2024-11-07T15:53:55.294594572Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRecursiveData",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
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
          "contenttype": "application/json",
          "file": {
            "bar": "foo"
          },

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
              "contenttype": "application/json",
              "file": {
                "bar": "foo"
              }
            }
          },
          "versionscount": 1
        },
        "f2": {
          "fileid": "f2",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f2$details",
          "xid": "/dirs/d1/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "contenttype": "application/json",
          "file": "string",

          "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs/d1/files/f2/meta",
            "xid": "/dirs/d1/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
          "versions": {
            "1": {
              "fileid": "f2",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
              "xid": "/dirs/d1/files/f2/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1",
              "contenttype": "application/json",
              "file": "string"
            }
          },
          "versionscount": 1
        },
        "f3": {
          "fileid": "f3",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f3$details",
          "xid": "/dirs/d1/files/f3",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "contenttype": "application/json",
          "file": 42,

          "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
          "meta": {
            "fileid": "f3",
            "self": "http://localhost:8181/dirs/d1/files/f3/meta",
            "xid": "/dirs/d1/files/f3/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f3/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
          "versions": {
            "1": {
              "fileid": "f3",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f3/versions/1$details",
              "xid": "/dirs/d1/files/f3/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1",
              "contenttype": "application/json",
              "file": 42
            }
          },
          "versionscount": 1
        },
        "f4": {
          "fileid": "f4",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f4$details",
          "xid": "/dirs/d1/files/f4",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "contenttype": "application/json",
          "file": [
            "foo"
          ],

          "metaurl": "http://localhost:8181/dirs/d1/files/f4/meta",
          "meta": {
            "fileid": "f4",
            "self": "http://localhost:8181/dirs/d1/files/f4/meta",
            "xid": "/dirs/d1/files/f4/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f4/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f4/versions",
          "versions": {
            "1": {
              "fileid": "f4",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f4/versions/1$details",
              "xid": "/dirs/d1/files/f4/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1",
              "contenttype": "application/json",
              "file": [
                "foo"
              ]
            }
          },
          "versionscount": 1
        },
        "f5": {
          "fileid": "f5",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f5$details",
          "xid": "/dirs/d1/files/f5",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "filebase64": "YmluYXJ5Cg==",

          "metaurl": "http://localhost:8181/dirs/d1/files/f5/meta",
          "meta": {
            "fileid": "f5",
            "self": "http://localhost:8181/dirs/d1/files/f5/meta",
            "xid": "/dirs/d1/files/f5/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f5/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f5/versions",
          "versions": {
            "1": {
              "fileid": "f5",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f5/versions/1$details",
              "xid": "/dirs/d1/files/f5/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1",
              "filebase64": "YmluYXJ5Cg=="
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 5
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "PATCH", "/", `{
  "dirs": {
    "d1": {
      "files": {
        "f1": {
	      "file": { "foo": "bar" },
          "versions": {
		    "v1": {
			  "file": { "bar": "foo" }
			}
          }
	    },
	    "f2": {
	      "file": "string"
	    }
      }
	}
  }
}`, 200,
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRecursiveData",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-11-07T15:53:55.28040091Z",
  "modifiedat": "2024-11-07T15:53:55.294594572Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	xHTTP(t, reg, "GET", "/?inline", ``, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestHTTPRecursiveData",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
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
          "contenttype": "application/json",
          "file": {
            "bar": "foo"
          },

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
              "contenttype": "application/json",
              "file": {
                "bar": "foo"
              }
            }
          },
          "versionscount": 1
        },
        "f2": {
          "fileid": "f2",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f2$details",
          "xid": "/dirs/d1/files/f2",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:02Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "1",
          "contenttype": "application/json",
          "file": "string",

          "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
          "meta": {
            "fileid": "f2",
            "self": "http://localhost:8181/dirs/d1/files/f2/meta",
            "xid": "/dirs/d1/files/f2/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:02Z",
            "modifiedat": "2024-01-01T12:00:02Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
          "versions": {
            "1": {
              "fileid": "f2",
              "versionid": "1",
              "self": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
              "xid": "/dirs/d1/files/f2/versions/1",
              "epoch": 1,
              "isdefault": true,
              "createdat": "2024-01-01T12:00:02Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "1",
              "contenttype": "application/json",
              "file": "string"
            }
          },
          "versionscount": 1
        }
      },
      "filescount": 2
    }
  },
  "dirscount": 1
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "POST", "/dirs", `{
  "d1": {
    "files": {
      "f1": {
	    "file": { "foo": "bar" },
        "versions": {
		  "v1": {
			"file": { "bar": "foo" }
		  }
        }
	  },
	  "f2": {
	    "file": "string"
	  }
    }
  }
}`, 200,
		`{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs?inline", ``, 200, `{
  "d1": {
    "dirid": "d1",
    "self": "http://localhost:8181/dirs/d1",
    "xid": "/dirs/d1",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",

    "filesurl": "http://localhost:8181/dirs/d1/files",
    "files": {
      "f1": {
        "fileid": "f1",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/f1$details",
        "xid": "/dirs/d1/files/f1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "ancestor": "v1",
        "contenttype": "application/json",
        "file": {
          "bar": "foo"
        },

        "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
        "meta": {
          "fileid": "f1",
          "self": "http://localhost:8181/dirs/d1/files/f1/meta",
          "xid": "/dirs/d1/files/f1/meta",
          "epoch": 1,
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z",
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
            "createdat": "2024-01-01T12:00:01Z",
            "modifiedat": "2024-01-01T12:00:01Z",
            "ancestor": "v1",
            "contenttype": "application/json",
            "file": {
              "bar": "foo"
            }
          }
        },
        "versionscount": 1
      },
      "f2": {
        "fileid": "f2",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/f2$details",
        "xid": "/dirs/d1/files/f2",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "ancestor": "1",
        "contenttype": "application/json",
        "file": "string",

        "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
        "meta": {
          "fileid": "f2",
          "self": "http://localhost:8181/dirs/d1/files/f2/meta",
          "xid": "/dirs/d1/files/f2/meta",
          "epoch": 1,
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z",
          "readonly": false,
          "compatibility": "none",

          "defaultversionid": "1",
          "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
          "defaultversionsticky": false
        },
        "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
        "versions": {
          "1": {
            "fileid": "f2",
            "versionid": "1",
            "self": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
            "xid": "/dirs/d1/files/f2/versions/1",
            "epoch": 1,
            "isdefault": true,
            "createdat": "2024-01-01T12:00:01Z",
            "modifiedat": "2024-01-01T12:00:01Z",
            "ancestor": "1",
            "contenttype": "application/json",
            "file": "string"
          }
        },
        "versionscount": 1
      }
    },
    "filescount": 2
  }
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "PUT", "/dirs/d1", `{
  "files": {
    "f1": {
	  "file": { "foo": "bar" },
      "versions": {
	    "v1": {
	      "file": { "bar": "foo" }
	    }
      }
	},
	"f2": {
	  "file": "string"
    }
  }
}`, 201,
		`{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 2
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1?inline", ``, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v1",
      "contenttype": "application/json",
      "file": {
        "bar": "foo"
      },

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "meta": {
        "fileid": "f1",
        "self": "http://localhost:8181/dirs/d1/files/f1/meta",
        "xid": "/dirs/d1/files/f1/meta",
        "epoch": 1,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
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
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z",
          "ancestor": "v1",
          "contenttype": "application/json",
          "file": {
            "bar": "foo"
          }
        }
      },
      "versionscount": 1
    },
    "f2": {
      "fileid": "f2",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f2$details",
      "xid": "/dirs/d1/files/f2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "1",
      "contenttype": "application/json",
      "file": "string",

      "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
      "meta": {
        "fileid": "f2",
        "self": "http://localhost:8181/dirs/d1/files/f2/meta",
        "xid": "/dirs/d1/files/f2/meta",
        "epoch": 1,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "readonly": false,
        "compatibility": "none",

        "defaultversionid": "1",
        "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
        "defaultversionsticky": false
      },
      "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
      "versions": {
        "1": {
          "fileid": "f2",
          "versionid": "1",
          "self": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
          "xid": "/dirs/d1/files/f2/versions/1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "2024-01-01T12:00:01Z",
          "modifiedat": "2024-01-01T12:00:01Z",
          "ancestor": "1",
          "contenttype": "application/json",
          "file": "string"
        }
      },
      "versionscount": 1
    }
  },
  "filescount": 2
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "POST", "/dirs/d1/files", `{
  "f1": {
	"file": { "foo": "bar" },
    "versions": {
	  "v1": {
	    "file": { "bar": "foo" }
	  }
    }
  },
  "f2": {
	"file": "string"
  }
}`, 200,
		`{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",
    "contenttype": "application/json",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  },
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",
    "contenttype": "application/json",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files?inline", ``, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",
    "contenttype": "application/json",
    "file": {
      "bar": "foo"
    },

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
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
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "ancestor": "v1",
        "contenttype": "application/json",
        "file": {
          "bar": "foo"
        }
      }
    },
    "versionscount": 1
  },
  "f2": {
    "fileid": "f2",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "1",
    "contenttype": "application/json",
    "file": "string",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "meta": {
      "fileid": "f2",
      "self": "http://localhost:8181/dirs/d1/files/f2/meta",
      "xid": "/dirs/d1/files/f2/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versions": {
      "1": {
        "fileid": "f2",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/f2/versions/1$details",
        "xid": "/dirs/d1/files/f2/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:01Z",
        "ancestor": "1",
        "contenttype": "application/json",
        "file": "string"
      }
    },
    "versionscount": 1
  }
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "file": { "foo": "bar" },
  "versions": {
	"v1": {
	  "file": { "bar": "foo" }
	}
  }
}`, 201,
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "application/json",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestor": "v1",
  "contenttype": "application/json",
  "file": {
    "bar": "foo"
  },

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
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
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "ancestor": "v1",
      "contenttype": "application/json",
      "file": {
        "bar": "foo"
      }
    }
  },
  "versionscount": 1
}
`)

	xHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
  "v1": {
	"file": { "bar": "foo" }
  }
}`, 200,
		`{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",
    "contenttype": "application/json"
  }
}
`)

	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?inline", ``, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",
    "contenttype": "application/json",
    "file": {
      "bar": "foo"
    }
  }
}
`)

}

func TestHTTPDefVer(t *testing.T) {
	reg := NewRegistry("TestHTTPDefVer")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModelSimple("files", "file")

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "PUT file + versionid header",
		URL:    "/dirs/d1/files/f1",
		Method: "PUT",
		ReqHeaders: []string{
			`xRegistry-versionid: v1`,
		},
		ReqBody:     `pick me`,
		Code:        201,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"xRegistry-fileid: f1",
			"xRegistry-versionid: v1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/f1",
			"xRegistry-xid: /dirs/d1/files/f1",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: v1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/f1/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/f1/versions",
			"xRegistry-versionscount: 1",
			"Content-Length: 7",
			"Content-Location: http://localhost:8181/dirs/d1/files/f1/versions/v1",
			"Content-Disposition:f1",
			"Content-Type: text/plain; charset=utf-8",
			"Location: http://localhost:8181/dirs/d1/files/f1",
		},
		ResBody: `pick me`,
	})

	xCheckHTTP(t, reg, &HTTPTest{
		Name:   "POST file + versionid header",
		URL:    "/dirs/d1/files",
		Method: "POST",
		ReqBody: `{
		  "f2": {
		    "versionid": "v1"
		  }
        }`,
		Code:        200,
		HeaderMasks: []string{},
		ResHeaders: []string{
			"Content-Type: application/json",
		},
		ResBody: `{
  "f2": {
    "fileid": "f2",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f2$details",
    "xid": "/dirs/d1/files/f2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-12-12T21:53:22.592492247Z",
    "modifiedat": "2024-12-12T21:53:22.592492247Z",
    "ancestor": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
    "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
    "versionscount": 1
  }
}
`,
	})

}

func TestHTTPInvalidID(t *testing.T) {
	reg := NewRegistry("TestHTTPInvalidID")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModelSimple("files", "file")

	match := `^[a-zA-Z0-9_][a-zA-Z0-9_.\-~:@]{0,127}$`

	xHTTP(t, reg, "PUT", "/", `{"registryid": "*" }`, 400,
		`Invalid ID "*", must match: `+match+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1*", `{}`, 400,
		`Invalid ID "d1*", must match: `+match+"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1", `{"dirid": "d1*" }`, 400,
		`Invalid ID "d1*", must match: `+match+"\n")
	xHTTP(t, reg, "POST", "/dirs/", `{"d1*":{}}`, 400,
		`Invalid ID "d1*", must match: `+match+"\n")
	xHTTP(t, reg, "POST", "/dirs/", `{"d1*":{"dirid": "d1*" }}`, 400,
		`Invalid ID "d1*", must match: `+match+"\n")
	xHTTP(t, reg, "POST", "/dirs/", `{"d1":{"dirid": "d2*" }}`, 400,
		`Invalid ID "d2*", must match: `+match+"\n")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1*$details", `{}`, 400,
		`Invalid ID "f1*", must match: `+match+"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{"fileid":"f1*"}`, 400,
		"The \"fileid\" attribute must be set to \"f1\", not \"f1*\"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{"versionid":"v1*"}`,
		400,
		`Invalid ID "v1*", must match: `+match+"\n")

	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{"v1*":{}}`, 400,
		`Invalid ID "v1*", must match: `+match+"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1*", `{}`, 400,
		`Invalid ID "v1*", must match: `+match+"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"versionid": "v1*"}`, 400,
		`Invalid ID "v1*", must match: `+match+"\n")
	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details",
		`{"fileid": "f1*"}`, 400,
		`Invalid ID "f1*", must match: `+match+"\n")
}

func TestHTTPSpecVersion(t *testing.T) {
	reg := NewRegistry("TestHTTPSpecVersion")
	defer PassDeleteReg(t, reg)

	xHTTP(t, reg, "GET", "?specversion="+SPECVERSION, "", 200, "*")
	xHTTP(t, reg, "GET", "?specversion=0.x", "", 400,
		"Unsupported xRegistry spec version: 0.x\n")
}

func TestHTTPMissingBody(t *testing.T) {
	reg := NewRegistry("TestHTTPMissingBody")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	_, err = gm.AddResourceModel("files", "file", 0, true, true, true)  //doc
	_, err = gm.AddResourceModel("datas", "data", 0, true, true, false) //nodoc
	xNoErr(t, err)

	// Just check for an error about a missing body based on the path + method

	msg := "An HTTP body must be specified\n"

	xHTTP(t, reg, "PUT", "/", "", 400, msg)
	xHTTP(t, reg, "POST", "/", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs", "", 405, "*")
	xHTTP(t, reg, "POST", "/dirs", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1", "", 400, msg)
	xHTTP(t, reg, "POST", "/dirs/d1", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/files", "", 405, "*")
	xHTTP(t, reg, "POST", "/dirs/d1/files", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "", 201, "")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1", "", 201, "")
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", "", 405, "*")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", "", 400, msg)
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1$details", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions", "", 405, "*")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/1", "", 200, "*")
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions/1", "", 405, "*")
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/1", "", 405, "*")

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/1$details", "", 400, msg)
	xHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions/1$details", "", 405, "*")
	xHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/1$details", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/datas", "", 405, "*")
	xHTTP(t, reg, "POST", "/dirs/d1/datas", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/datas", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/datas/d1", "", 400, msg)
	xHTTP(t, reg, "POST", "/dirs/d1/datas/d1", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/datas/d1", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/datas/d1/versions", "", 405, "*")
	xHTTP(t, reg, "POST", "/dirs/d1/datas/d1/versions", "", 400, msg)
	xHTTP(t, reg, "PATCH", "/dirs/d1/datas/d1/versions", "", 400, msg)

	xHTTP(t, reg, "PUT", "/dirs/d1/datas/d1/versions/1", "", 400, msg)
	xHTTP(t, reg, "POST", "/dirs/d1/datas/d1/versions/1", "", 405, "*")
	xHTTP(t, reg, "PATCH", "/dirs/d1/datas/d1/versions/1", "", 400, msg)
}

func TestHTTPJsonParsingErrors(t *testing.T) {
	reg := NewRegistry("TestHTTPJsonParsingErrors")
	defer PassDeleteReg(t, reg)

	tests := []struct {
		body string
		msg  string
	}{
		{`{1`, `path '': parsing object key: expected string starting with '"', got '1' instead`},
		{`{"}`, `path '': parsing object key: unterminated string`},
		{`{},"}`, `Syntax error at line 1: invalid character ',' looking for beginning of value; possibly near position 2`},
		{`{}[]`, `Error parsing json: extra data possibly near position 3: [`},
		{`{}{}`, `Error parsing json: extra data possibly near position 3: {`},
		{`[]`, `path '': expected "map", got "array"`},
		{``, `An HTTP body must be specified`},
		{`,`, `path '': unexpected character ','`},
		{`"`, `path '': unterminated string`},
		{`1`, `path '': expected "map", got "number"`},
		{`{"foo":{"bar":"asd}}`, `path '.foo.bar': unterminated string`},
	}

	for _, test := range tests {
		body := bytes.NewReader([]byte(test.body))

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}}

		req, err := http.NewRequest("PUT", "http://localhost:8181/", body)
		xNoErr(t, err)
		res, err := client.Do(req)
		xNoErr(t, err)
		xCheckEqual(t, test.body, res.StatusCode, 400)

		data, err := io.ReadAll(res.Body)
		xNoErr(t, err)
		xCheckEqual(t, test.body, string(data), test.msg+"\n")
	}
}

func TestHTTPCollectionsFlag(t *testing.T) {
	reg := NewRegistry("TestHTTPCollectionsFlag")
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", "", 201, "*")

	// Test the 2 valid cases
	xHTTP(t, reg, "GET", "?collections", "", 200, `{
  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "isdefault": true,
          "createdat": "YYYY-MM-DDTHH:MM:01Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
          "ancestor": "v1",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "YYYY-MM-DDTHH:MM:01Z",
            "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
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
              "createdat": "YYYY-MM-DDTHH:MM:01Z",
              "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
              "ancestor": "v1"
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

	xHTTP(t, reg, "GET", "/dirs/d1?collections", "", 200, `{
  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "f1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1$details",
      "xid": "/dirs/d1/files/f1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "ancestor": "v1",

      "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
      "meta": {
        "fileid": "f1",
        "self": "http://localhost:8181/dirs/d1/files/f1/meta",
        "xid": "/dirs/d1/files/f1/meta",
        "epoch": 1,
        "createdat": "YYYY-MM-DDTHH:MM:01Z",
        "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
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
          "createdat": "YYYY-MM-DDTHH:MM:01Z",
          "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
          "ancestor": "v1"
        }
      },
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)

	// And now the errors
	xHTTP(t, reg, "GET", "/dirs?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")
	xHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/v1$details?collections", "", 400,
		"?collections is only allow on the Registry or Group instance level\n")

	xHTTP(t, reg, "GET", "/dirsx?collections", "", 404,
		"Unknown Group type: dirsx\n")
	xHTTP(t, reg, "GET", "/dirs/d1x?collections", "", 404,
		"Not found\n")
	xHTTP(t, reg, "GET", "/dirs/d1/filesx?collections", "", 404,
		"Unknown Resource type: filesx\n")
}
