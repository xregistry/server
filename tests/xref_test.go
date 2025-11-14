package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestXrefBasic(t *testing.T) {
	reg := NewRegistry("TestXrefBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", "{}", 201, `*`)
	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	XNoErr(t, err)

	rows := reg.Query("select * from Versions where ResourceSID=?",
		f1.DbSID)
	XEqual(t, "", len(rows), 1) // Just to be sure Query works ok

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"dirs/d1/files/f1"}`, 400, // missing leading /
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx/meta",
  "title": "The request cannot be processed as provided: error parsing 'xref': \"dirs/d1/files/f1\" must start with /"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/foo/dirs/d1/files/f1"}`, 400, // make it bad
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx/meta",
  "title": "The request cannot be processed as provided: error parsing 'xref': \"/foo/dirs/d1/files/f1\" must be of the form: /GROUPS/GID/RESOURCES/RID"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	fx, err := reg.FindResourceByXID("/dirs/d1/files/fx", "/")
	XNoErr(t, err)

	// Grab #createdat so we can make sure it's used when we remove 'xref'
	meta, _ := fx.FindMeta(false, registry.FOR_WRITE)
	oldCreatedAt := meta.Get("#createdat")

	// Make sure the Resource doesn't have any versions in the DB.
	// Use fx.GetVersions() will grab from xref target so don't use that
	rows = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	XEqual(t, "", len(rows), 0)

	XHTTP(t, reg, "GET", "/dirs/d1/files?inline=meta", "", 200, `{
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
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestor": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versionscount": 1
  }
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details?inline=meta",
		`{"description":"testing xref"}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "testing xref",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
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
  "versionscount": 1
}
`)

	f1, err = reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	XNoErr(t, err)

	fx, err = reg.FindResourceByXID("/dirs/d1/files/fx", "/")
	XNoErr(t, err)

	XEqual(t, "", fx.Get("description"), "testing xref")

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v1$details",
		`{"name":"v1 name"}`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 3,
  "name": "v1 name",
  "isdefault": true,
  "description": "testing xref",
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestor": "v1"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
        "xid": "/dirs/d1/files/fx/versions/v1",
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  }
}
`)

	// Now clear xref and make sure a version is created
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null}`, 200, `*`)

	rows = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	XEqual(t, "", len(rows), 1)

	meta, err = reg.FindXIDMeta("/dirs/d1/files/fx/meta", "/")
	XNoErr(t, err)

	if meta.Get("createdat") != oldCreatedAt {
		t.Errorf("CreatedAt has wrong value, should be %q, not %q",
			oldCreatedAt, meta.Get("createdat"))
		t.FailNow()
	}

	XHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:04Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "ancestor": "1",
    "filebase64": "",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 4,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
        "xid": "/dirs/d1/files/fx/versions/1",
        "epoch": 1,
        "isdefault": true,
        "createdat": "2024-01-01T12:00:04Z",
        "modifiedat": "2024-01-01T12:00:04Z",
        "ancestor": "1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  }
}
`)

	// re-Set xref and make sure the version is deleted
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 200, `*`)

	rows = reg.Query("select * from Versions where ResourceSID=?",
		fx.DbSID)
	XEqual(t, "", len(rows), 0)

	XHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "v1": {
        "fileid": "fx",
        "versionid": "v1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/v1$details",
        "xid": "/dirs/d1/files/fx/versions/v1",
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  }
}
`)

	// Now clear xref and set some props at the same time
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details",
		`{"meta":{"xref":null},
		  "name": "fx name",
		  "description": "very cool"}`, 200, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files?inline", "", 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1$details",
    "xid": "/dirs/d1/files/f1",
    "epoch": 3,
    "name": "v1 name",
    "isdefault": true,
    "description": "testing xref",
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "ancestor": "v1",
    "filebase64": "",

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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestor": "v1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/fx$details",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "name": "fx name",
    "isdefault": true,
    "description": "very cool",
    "createdat": "2024-01-01T12:00:04Z",
    "modifiedat": "2024-01-01T12:00:04Z",
    "ancestor": "1",
    "filebase64": "",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 5,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versions": {
      "1": {
        "fileid": "fx",
        "versionid": "1",
        "self": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
        "xid": "/dirs/d1/files/fx/versions/1",
        "epoch": 1,
        "name": "fx name",
        "isdefault": true,
        "description": "very cool",
        "createdat": "2024-01-01T12:00:04Z",
        "modifiedat": "2024-01-01T12:00:04Z",
        "ancestor": "1",
        "filebase64": ""
      }
    },
    "versionscount": 1
  }
}
`)

}

func TestXrefErrors(t *testing.T) {
	reg := NewRegistry("TestXrefErrors")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)

	gm2, _ := reg.Model.AddGroupModel("bars", "bar")

	XCheckErr(t, gm2.AddXImportResource("dirs/files"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: 'ximportresources' value \"dirs/files\" must start with /"
}`)
	XCheckErr(t, gm2.AddXImportResource("/dirs/files/versions"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: 'ximportresources' value of \"/dirs/files/versions\" must be of the form: /GROUPS/RESOURCES"
}`)
	XCheckErr(t, gm2.AddXImportResource("/dirs"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: 'ximportresources' value of \"/dirs\" must be of the form: /GROUPS/RESOURCES"
}`)
	XCheckErr(t, gm2.AddXImportResource("//files"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: 'ximportresources' value \"//files\" has an empty part at position 1"
}`)

	// Now a good one
	XNoErr(t, gm2.AddXImportResource("/dirs/files"))

	d, _ := reg.AddGroup("dirs", "d1")
	_, err := d.AddResource("files", "f1", "v1")
	XNoErr(t, err)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","fileid":"f2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The specified \"file\" ID value (f2) needs to be \"f1\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","epoch":5}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The attribute \"epoch\" is not valid: value (5) doesn't match existing value (1)"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx", "modifiedat":"2025-01-01T12:00:00"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: extra attributes (modifiedat) in \"meta\" not allowed when \"xref\" is set"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"foo":"foo","xref": "/dirs/d1/files/fx"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: extra attributes (foo) in \"meta\" not allowed when \"xref\" is set"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: extra attributes (description,epoch) not allowed when \"xref\" is set"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The request cannot be processed as provided: extra attributes (description,epoch) not allowed when \"xref\" is set"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f2", "meta": {"xref":"/dirs/d1/files/f1"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1",
  "title": "The attribute \"fileid\" is not valid: must be set to \"f1\", not \"f2\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"xref":"/dirs/d1/files/f1","epoch":6}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The attribute \"epoch\" is not valid: value (6) doesn't match existing value (1)"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f1", "meta": {"xref":"/dirs/d1/files/f1","modifiedat":"2025-01-01-T:12:00:00"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/f1/meta",
  "title": "The request cannot be processed as provided: extra attributes (modifiedat) in \"meta\" not allowed when \"xref\" is set"
}
`)

	// actually it can point to itself since we just treat it like any other
	// time we point to a Resource that's an xref
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/meta",
		`{"xref":"/dirs/d1/files/f2"}`, 201, `{
  "fileid": "f2",
  "self": "http://localhost:8181/dirs/d1/files/f2/meta",
  "xid": "/dirs/d1/files/f2/meta",
  "xref": "/dirs/d1/files/f2"
}
`)

	XHTTP(t, reg, "PUT", "/bars/b1/files/f1/meta",
		`{"xref":"/bars/b1/files/f1"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/bars/b1/files/f1/meta",
  "title": "The request cannot be processed as provided: 'xref' \"/bars/b1/files/f1\" must point to a Resource of type \"/dirs/files\" not \"/bars/files\""
}
`)

	XHTTP(t, reg, "PUT", "/bars/b1/files/f1/meta",
		`{"xref":"/bars/b1/files/f2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/bars/b1/files/f1/meta",
  "title": "The request cannot be processed as provided: 'xref' \"/bars/b1/files/f2\" must point to a Resource of type \"/dirs/files\" not \"/bars/files\""
}
`)

	// ok even if target is missing
	XHTTP(t, reg, "PUT", "/bars/b1/files/f1/meta",
		`{"xref":"/dirs/dx/files/fx"}`, 201, `{
  "fileid": "f1",
  "self": "http://localhost:8181/bars/b1/files/f1/meta",
  "xid": "/bars/b1/files/f1/meta",
  "xref": "/dirs/dx/files/fx"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/ff", `{}`, 201, `{
  "fileid": "ff",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/ff",
  "xid": "/dirs/d1/files/ff",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/ff/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/ff/versions",
  "versionscount": 1
}
`)

	// Works!
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx", "epoch":1}`,
		200,
		`{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "xref": "/dirs/d1/files/fx"
}
`)

	// ximport Works!
	XHTTP(t, reg, "PUT", "/bars/b1/files/f2?inline=meta",
		`{"meta":{"xref": "/dirs/d1/files/ff"}}`,
		201,
		`{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/bars/b1/files/f2",
  "xid": "/bars/b1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/bars/b1/files/f2/meta",
  "meta": {
    "fileid": "f2",
    "self": "http://localhost:8181/bars/b1/files/f2/meta",
    "xid": "/bars/b1/files/f2/meta",
    "xref": "/dirs/d1/files/ff",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/bars/b1/files/f2/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/bars/b1/files/f2/versions",
  "versionscount": 1
}
`)
}

func TestXrefRevert(t *testing.T) {
	reg := NewRegistry("TestXrefRevert")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, false)
	d, _ := reg.AddGroup("dirs", "d1")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v9",
		`{"description":"hi"}`, 201, `{
  "fileid": "f1",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
  "xid": "/dirs/d1/files/f1/versions/v9",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-09T15:59:29.22249886Z",
  "modifiedat": "2025-01-09T15:59:29.22249886Z",
  "ancestor": "v9"
}
`)

	// Revert with no versions (create 2 files so we can grab the TS from f0)
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "POST", "/dirs/d1/files/?inline=meta",
		`{"f0":{}, "fx":{"meta":{"xref":"/dirs/d1/files/f1"}}}`, 200, `{
  "f0": {
    "fileid": "f0",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f0",
    "xid": "/dirs/d1/files/f0",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f0/meta",
    "meta": {
      "fileid": "f0",
      "self": "http://localhost:8181/dirs/d1/files/f0/meta",
      "xid": "/dirs/d1/files/f0/meta",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f0/versions/1",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f0/versions",
    "versionscount": 1
  },
  "fx": {
    "fileid": "fx",
    "versionid": "v9",
    "self": "http://localhost:8181/dirs/d1/files/fx",
    "xid": "/dirs/d1/files/fx",
    "epoch": 1,
    "isdefault": true,
    "description": "hi",
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestor": "v9",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "xref": "/dirs/d1/files/f1",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:02Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
      "readonly": false,
      "compatibility": "none",

      "defaultversionid": "v9",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
    "versionscount": 1
  }
}
`)

	// Grab F0's timestamp so we can compare later
	f0, err := d.FindResource("files", "f0", false, registry.FOR_WRITE)
	XNoErr(t, err)
	f0TS := f0.Get("createdat").(string)
	XCheck(t, f0TS > "2024", "bad ts: %s", f0TS)

	// Notice epoch will be 2 not 1 since it's max(0,fx.epoch)+1
	// Notice meta.createat == f0's createdat, others are now()
	// Make sure we pick up def ver attrs
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "hello",
  "meta":{"xref":null}
} `, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hello",
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
	fx, err := d.FindResource("files", "fx", false, registry.FOR_WRITE)
	XNoErr(t, err)
	fxMeta, err := fx.FindMeta(false, registry.FOR_WRITE)
	XNoErr(t, err)
	fxMetaTS := fxMeta.Get("createdat").(string)
	XCheck(t, f0TS == fxMetaTS, "Bad ts: %s/%s", f0TS, fxMetaTS)

	// Revert with empty versions
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestor": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null},
  "versions": {}
} `, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 3,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with one version
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestor": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Notice "description:bye" is ignored
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "bye",
  "meta":{"xref":null},
  "versions": { "v1": { "description": "ver1" } }
} `, 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "ver1",
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 4,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - no default
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestor": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// "description:bye" is ignored
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "description": "bye",
  "meta":{"xref":null},
  "versions": { "z1": {}, "a1": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "z1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "a1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 5,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "z1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/z1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - w/default query param
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestor": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Not 100% this is legal per the spec, we should probably reject the
	// query parameter since I think it's only allowed on 'POST /versions'
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta&setdefaultversionid=bb", `{
  "meta":{"xref":null },
  "versions": { "z2": {}, "b3": {} }
} `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: Version \"bb\" not found"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta&setdefaultversionid=b3", `{
  "meta":{"xref":null },
  "versions": { "z2": {}, "b3": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "b3",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "b3",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 6,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "b3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/b3",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert with two versions - w/default in meta
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 200, `{
  "fileid": "fx",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "description": "hi",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestor": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null,
          "defaultversionid": "bb",
          "defaultversionsticky": true },
  "versions": { "z2": {}, "b3": {} }
} `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "subject": "http://localhost:8181/dirs/d1/files/fx/meta",
  "title": "The \"Version\" with the ID \"bb\" cannot be found"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
  "meta":{"xref":null,
          "defaultversionid": "b3",
          "defaultversionsticky": true },
  "versions": { "z2": {}, "b3": {} }
} `, 200, `{
  "fileid": "fx",
  "versionid": "b3",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:02Z",
  "modifiedat": "2025-01-01T12:00:02Z",
  "ancestor": "b3",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 7,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,
    "compatibility": "none",

    "defaultversionid": "b3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/b3",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

	// Revert via meta + default
	////////////////////////////////////////////////////////
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v9",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
  "defaultversionsticky": false
}
`)

	// defaultversionid is bad because we're not sticky
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
          "defaultversionid": "bb"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx/meta",
  "title": "The request cannot be processed as provided: attribute \"defaultversionid\" must be \"1\" since \"defaultversionsticky\" is \"false\""
}
`)

	// defaultversionid is bad
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
          "defaultversionid": "bb",
		  "defaultversionsticky": true }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "subject": "http://localhost:8181/dirs/d1/files/fx/meta",
  "title": "The \"Version\" with the ID \"bb\" cannot be found"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
		  "defaultversionsticky": true}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 8,
  "createdat": "2025-01-09T23:16:04.619269627Z",
  "modifiedat": "2025-01-09T23:16:05.273949318Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": true
}
`)

	XNoErr(t, fxMeta.Refresh(registry.FOR_WRITE))
	XNoErr(t, fx.Refresh(registry.FOR_WRITE))
	XEqual(t, "ts check", f0TS, fxMeta.Get("createdat").(string))
	XCheckGreater(t, "ts check", fx.Get("createdat").(string), f0TS)

}

func TestXrefDocs(t *testing.T) {
	reg := NewRegistry("TestXrefRevert")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "hello world", 201, "hello world")
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details?inline=file",
		`{"fileurl":"http://localhost:8282/EMPTY-URL"}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2$details",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "fileurl": "http://localhost:8282/EMPTY-URL",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f3$details?inline=file",
		`{"fileproxyurl":"http://localhost:8282/EMPTY-Proxy"}`, 201, `{
  "fileid": "f3",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f3$details",
  "xid": "/dirs/d1/files/f3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
  "filebase64": "aGVsbG8tUHJveHkK",

  "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1", "", 200, `hello world`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx", "", 200, `hello world`)

	XCheckHTTP(t, reg, &HTTPTest{
		Name:       "check xref header",
		URL:        "/dirs/d1/files/fx",
		Method:     "GET",
		ReqHeaders: []string{},
		ReqBody:    "",

		Code: 200,
		ResHeaders: []string{
			"Content-Type:text/plain; charset=utf-8",
			"xRegistry-fileid: fx",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/fx",
			"xRegistry-xid: /dirs/d1/files/fx",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestor: 1",
			"xRegistry-metaurl: http://localhost:8181/dirs/d1/files/fx/meta",
			"xRegistry-versionsurl: http://localhost:8181/dirs/d1/files/fx/versions",
			"xRegistry-versionscount: 1",
			"Content-Location: http://localhost:8181/dirs/d1/files/fx/versions/1",
			"Content-Disposition: fx",
			"Content-Length: 11",
		},
		ResBody: `hello world`,
	})

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions/1", "", 200, `hello world`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/versions/1", "", 200, `hello world`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/fx", `{"versions":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx$details", `{"versions":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx?setdefaultversionid=2", `{}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx$details?setdefaultversionid=2",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details?setdefaultversionid=2",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: Version \"2\" not found"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details?setdefaultversionid=1",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"defaultversionid\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx?setdefaultversionid=2",
		``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions", "{}", 200,
		"{}\n")
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions", `{"vv":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/1", "hi", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/1$details", "{}", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions/1", "hi", 405,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "subject": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "title": "The specified action (POST) is not supported",
  "detail": "POST not allowed on a version"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/2", "hi", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/2$details", "{}", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy$details?doc&inline",
		`{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fy",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy$details?doc&inline",
		`{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{"2":{},"3":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fy",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/",
		`{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fy",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/",
		`{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{"2":{},"3":{}}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fy",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d2",
		`{"files":{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d2/files/fy",
  "title": "The request cannot be processed as provided: can't update \"versions\" of a Resource that uses \"xref\""
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/1", ``,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "title": "The request cannot be processed as provided: can't delete \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/x", ``,
		404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "subject": "http://localhost:8181/dirs/d1/files/fx/versions/x",
  "title": "The specified entity cannot be found: /dirs/d1/files/fx/versions/x"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/", `{"1":{}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "title": "The request cannot be processed as provided: can't delete \"versions\" of a Resource that uses \"xref\""
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx", ``, 204, ``)

	// Now test stuff that use fileurl and fileproxy

	// fileurl
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/fx/meta?doc",
		`{"xref":"/dirs/d1/files/f2"}`, 201, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f2"
}
`)
	XCheckHTTP(t, reg, &HTTPTest{
		Name:   "",
		URL:    "/dirs/d1/files/fx",
		Method: "GET",

		Code:       303,
		ResHeaders: []string{"Location: http://localhost:8282/EMPTY-URL"},
		ResBody:    ``})
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "fileurl": "http://localhost:8282/EMPTY-URL",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// fileProxyURL
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/fx/meta?doc",
		`{"xref":"/dirs/d1/files/f3"}`, 200, `{
  "fileid": "fx",
  "self": "#/",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f3"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx", ``, 200, "hello-Proxy\n")
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details?inline=file", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "1",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
  "filebase64": "aGVsbG8tUHJveHkK",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}
