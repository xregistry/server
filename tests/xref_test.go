package tests

import (
	"fmt"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestXrefBasic(t *testing.T) {
	reg := NewRegistry("TestXrefBasic")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", "{}", 201, `*`)
	f1, err := reg.FindResourceByXID("/dirs/d1/files/f1", "/")
	XNoErr(t, err)

	rows := reg.Query("select * from Versions where ResourceSID=?",
		f1.DbSID)
	XEqual(t, "", len(rows), 1) // Just to be sure Query works ok

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"dirs/d1/files/f1"}`, 400, // missing leading /
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_xref",
  "title": "The specified xref value (dirs/d1/files/f1) is malformed: \"dirs/d1/files/f1\" must start with /.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "error_detail": "\"dirs/d1/files/f1\" must start with /",
    "xref": "dirs/d1/files/f1"
  },
  "source": "e4e59b8a76c4:registry:resource:589"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/foo/dirs/d1/files/f1"}`, 400, // make it bad
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_xref",
  "title": "The specified xref value (/foo/dirs/d1/files/f1) is malformed: \"/foo/dirs/d1/files/f1\" must be of the form: /GROUPS/GID/RESOURCES/RID.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "error_detail": "\"/foo/dirs/d1/files/f1\" must be of the form: /GROUPS/GID/RESOURCES/RID",
    "xref": "/foo/dirs/d1/files/f1"
  },
  "source": "e4e59b8a76c4:registry:resource:589"
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
    "ancestorid": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "readonly": false,

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
    "ancestorid": "v1",

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
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

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
  "ancestorid": "v1"
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
    "ancestorid": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestorid": "v1"
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
    "ancestorid": "v1",

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
        "ancestorid": "v1"
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
    "ancestorid": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestorid": "v1"
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
    "ancestorid": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 4,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "readonly": false,

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
        "ancestorid": "1"
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
    "ancestorid": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestorid": "v1"
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
    "ancestorid": "v1",

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
        "ancestorid": "v1"
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
    "ancestorid": "v1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:01Z",
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
        "epoch": 3,
        "name": "v1 name",
        "isdefault": true,
        "description": "testing xref",
        "createdat": "2024-01-01T12:00:01Z",
        "modifiedat": "2024-01-01T12:00:02Z",
        "ancestorid": "v1"
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
    "ancestorid": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
    "meta": {
      "fileid": "fx",
      "self": "http://localhost:8181/dirs/d1/files/fx/meta",
      "xid": "/dirs/d1/files/fx/meta",
      "epoch": 5,
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:04Z",
      "readonly": false,

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
        "ancestorid": "1"
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
	gm.AddResourceModel("files", "file", 0, true, false)

	gm2, _ := reg.Model.AddGroupModel("bars", "bar")

	XCheckErr(t, gm2.AddXImportResource("dirs/files"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: 'ximportresources' value \"dirs/files\" must start with /.",
  "subject": "/model",
  "args": {
    "error_detail": "'ximportresources' value \"dirs/files\" must start with /"
  },
  "source": "e4e59b8a76c4:registry:shared_model:1061"
}`)
	XCheckErr(t, gm2.AddXImportResource("/dirs/files/versions"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: 'ximportresources' value of \"/dirs/files/versions\" must be of the form: /GROUPS/RESOURCES.",
  "subject": "/model",
  "args": {
    "error_detail": "'ximportresources' value of \"/dirs/files/versions\" must be of the form: /GROUPS/RESOURCES"
  },
  "source": "e4e59b8a76c4:registry:shared_model:1065"
}`)
	XCheckErr(t, gm2.AddXImportResource("/dirs"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: 'ximportresources' value of \"/dirs\" must be of the form: /GROUPS/RESOURCES.",
  "subject": "/model",
  "args": {
    "error_detail": "'ximportresources' value of \"/dirs\" must be of the form: /GROUPS/RESOURCES"
  },
  "source": "e4e59b8a76c4:registry:shared_model:1065"
}`)
	XCheckErr(t, gm2.AddXImportResource("//files"),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: 'ximportresources' value \"//files\" has an empty part at position 1.",
  "subject": "/model",
  "args": {
    "error_detail": "'ximportresources' value \"//files\" has an empty part at position 1"
  },
  "source": "e4e59b8a76c4:registry:shared_model:1061"
}`)

	// Now a good one
	XNoErr(t, gm2.AddXImportResource("/dirs/files"))

	d, _ := reg.AddGroup("dirs", "d1")
	_, err := d.AddResource("files", "f1", "v1")
	XNoErr(t, err)

	// bad xrefs
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/zoos/d1/files/fx"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_xref",
  "title": "The specified xref value (/zoos/d1/files/fx) is malformed: points to a non-existing Group Type: zoos.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "points to a non-existing Group Type: zoos",
    "xref": "/zoos/d1/files/fx"
  },
  "source": "49a49fc034c5:registry:resource:666"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/zoos/fx"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#malformed_xref",
  "title": "The specified xref value (/dirs/d1/zoos/fx) is malformed: points to a non-existing Resource Type: zoos.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "points to a non-existing Resource Type: zoos",
    "xref": "/dirs/d1/zoos/fx"
  },
  "source": "49a49fc034c5:registry:resource:675"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","fileid":"f2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (f2) for \"/dirs/d1/files/f1/meta\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "expected_id": "f1",
    "invalid_id": "f2",
    "singular": "file"
  },
  "source": "0018b4bbf02e:registry:resource:487"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx","epoch":5}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (5) for \"/dirs/d1/files/f1/meta\" does not match its current value (1).",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "bad_epoch": "5",
    "epoch": "1"
  },
  "source": "e4e59b8a76c4:registry:entity:1005"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"xref": "/dirs/d1/files/fx", "modifiedat":"2025-01-01T12:00:00"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"modifiedat\" is not allowed to be present since the Resource (/dirs/d1/files/f1/meta) uses \"xref\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "name": "modifiedat"
  },
  "source": "0018b4bbf02e:registry:resource:746"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"foo":"foo","xref": "/dirs/d1/files/fx"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"foo\" is not allowed to be present since the Resource (/dirs/d1/files/f1/meta) uses \"xref\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "name": "foo"
  },
  "source": "0018b4bbf02e:registry:resource:746"
}
`)

	// XHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 200, ``)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"description\" is not allowed to be present since the Resource (/dirs/d1/files/f1) uses \"xref\".",
  "detail": "Full list: description,epoch.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "description"
  },
  "source": "0018b4bbf02e:registry:group:411"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"fileid":"f1", "xref":"/dirs/d1/files/f1"},"epoch":5, "description": "x"}`,
		400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"description\" is not allowed to be present since the Resource (/dirs/d1/files/f1) uses \"xref\".",
  "detail": "Full list: description,epoch.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "description"
  },
  "source": "0018b4bbf02e:registry:group:411"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f2", "meta": {"xref":"/dirs/d1/files/f1"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_id",
  "title": "The specified \"fileid\" value (f2) for \"/dirs/d1/files/f1\" needs to be \"f1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "expected_id": "f1",
    "invalid_id": "f2",
    "singular": "file"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:2166"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta": {"xref":"/dirs/d1/files/f1","epoch":6}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_epoch",
  "title": "The specified epoch value (6) for \"/dirs/d1/files/f1/meta\" does not match its current value (1).",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "bad_epoch": "6",
    "epoch": "1"
  },
  "source": "e4e59b8a76c4:registry:entity:1005"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"fileid": "f1", "meta": {"xref":"/dirs/d1/files/f1","modifiedat":"2025-01-01-T:12:00:00"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"modifiedat\" is not allowed to be present since the Resource (/dirs/d1/files/f1/meta) uses \"xref\".",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "name": "modifiedat"
  },
  "source": "0018b4bbf02e:registry:resource:746"
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
		`{"xref":"/bars/b1/files/f1"}`, 201,
		`{
  "fileid": "f1",
  "self": "http://localhost:8181/bars/b1/files/f1/meta",
  "xid": "/bars/b1/files/f1/meta",
  "xref": "/bars/b1/files/f1"
}
`)

	XHTTP(t, reg, "PUT", "/bars/b1/files/f1/meta",
		`{"xref":"/bars/b1/files/f2"}`, 200,
		`{
  "fileid": "f1",
  "self": "http://localhost:8181/bars/b1/files/f1/meta",
  "xid": "/bars/b1/files/f1/meta",
  "xref": "/bars/b1/files/f2"
}
`)

	// ok even if target is missing
	XHTTP(t, reg, "PUT", "/bars/b1/files/f1/meta",
		`{"xref":"/dirs/dx/files/fx"}`, 200, `{
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
  "ancestorid": "1",

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
  "ancestorid": "1",

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
	gm.AddResourceModel("files", "file", 0, true, false)
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
  "ancestorid": "v9"
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
    "ancestorid": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f0/meta",
    "meta": {
      "fileid": "f0",
      "self": "http://localhost:8181/dirs/d1/files/f0/meta",
      "xid": "/dirs/d1/files/f0/meta",
      "epoch": 1,
      "createdat": "YYYY-MM-DDTHH:MM:01Z",
      "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
      "readonly": false,

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
    "ancestorid": "v9",

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
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,

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
  "ancestorid": "v9",

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
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 3,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,

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
  "ancestorid": "v9",

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
  "versionid": "v1",
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
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 4,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:01Z",
    "readonly": false,

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
  "ancestorid": "v9",

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
  "versionid": "z1",
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
  "ancestorid": "a1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 5,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

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
  "ancestorid": "v9",

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
  "versionid": "z2",
  "versions": { "z2": {}, "b3": {} }
} `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/fx/meta\", the \"version\" with a \"versionid\" value of \"bb\" cannot be found.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "id": "bb",
    "singular": "version"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:2628"
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
  "ancestorid": "b3",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 6,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,

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
  "ancestorid": "v9",

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
  "versionid": "z2",
  "versions": { "z2": {}, "b3": {} }
} `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/fx/meta\", the \"version\" with a \"versionid\" value of \"bb\" cannot be found.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "id": "bb",
    "singular": "version"
  },
  "source": "e4e59b8a76c4:registry:resource:852"
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
  "ancestorid": "b3",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "epoch": 7,
    "createdat": "2025-01-01T12:00:01Z",
    "modifiedat": "2025-01-01T12:00:02Z",
    "readonly": false,

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

  "defaultversionid": "v9",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v9",
  "defaultversionsticky": false
}
`)

	// defaultversionid is bad
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":null,
          "defaultversionid": "bb",
		  "defaultversionsticky": true }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/fx/meta\", the \"version\" with a \"versionid\" value of \"bb\" cannot be found.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "id": "bb",
    "singular": "version"
  },
  "source": "e4e59b8a76c4:registry:resource:852"
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
	gm.AddResourceModel("files", "file", 0, true, true)

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
  "ancestorid": "1",

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
  "ancestorid": "1",

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
			"xRegistry-fileid: fx",
			"xRegistry-versionid: 1",
			"xRegistry-self: http://localhost:8181/dirs/d1/files/fx",
			"xRegistry-xid: /dirs/d1/files/fx",
			"xRegistry-epoch: 1",
			"xRegistry-isdefault: true",
			"xRegistry-createdat: 2024-01-01T12:00:01Z",
			"xRegistry-modifiedat: 2024-01-01T12:00:01Z",
			"xRegistry-ancestorid: 1",
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
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx$details", `{"versions":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx?setdefaultversionid=2", `{}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx$details?setdefaultversionid=2",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details?setdefaultversionid=2",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"defaultversionid\" is not allowed to be present since the Resource (/dirs/d1/files/fx/meta) uses \"xref\".",
  "detail": "Full list: defaultversionid,defaultversionsticky.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "name": "defaultversionid"
  },
  "source": "396100315a6e:registry:resource:796"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx$details?setdefaultversionid=1",
		`{}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"defaultversionid\" is not allowed to be present since the Resource (/dirs/d1/files/fx/meta) uses \"xref\".",
  "detail": "Full list: defaultversionid,defaultversionsticky.",
  "subject": "/dirs/d1/files/fx/meta",
  "args": {
    "name": "defaultversionid"
  },
  "source": "396100315a6e:registry:resource:796"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx?setdefaultversionid=2",
		``, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions", "{}", 200,
		"{}\n")
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions", `{"vv":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/1", "hi", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/1$details", "{}", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/fx/versions/1", "hi", 405,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#action_not_supported",
  "title": "The specified action (POST) is not supported for: /dirs/d1/files/fx/versions/1.",
  "detail": "POST not allowed on a version.",
  "subject": "/dirs/d1/files/fx/versions/1",
  "args": {
    "action": "POST"
  },
  "source": "e4e59b8a76c4:registry:httpStuff:1878"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/2", "hi", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/versions/2$details", "{}", 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\".",
  "subject": "/dirs/d1/files/fx",
  "args": {
    "error_detail": "Cannot update Resource \"/dirs/d1/files/fx\" in this way since it uses \"xref\""
  },
  "source": "396100315a6e:registry:resource:1026"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy$details?doc&inline",
		`{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"versions\" is not allowed to be present since the Resource (/dirs/d1/files/fy) uses \"xref\".",
  "subject": "/dirs/d1/files/fy",
  "args": {
    "name": "versions"
  },
  "source": "396100315a6e:registry:group:479"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy$details?doc&inline",
		`{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{"2":{},"3":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"versions\" is not allowed to be present since the Resource (/dirs/d1/files/fy) uses \"xref\".",
  "subject": "/dirs/d1/files/fy",
  "args": {
    "name": "versions"
  },
  "source": "396100315a6e:registry:group:479"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/",
		`{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"versions\" is not allowed to be present since the Resource (/dirs/d1/files/fy) uses \"xref\".",
  "subject": "/dirs/d1/files/fy",
  "args": {
    "name": "versions"
  },
  "source": "396100315a6e:registry:group:479"
}
`)
	XHTTP(t, reg, "POST", "/dirs/d1/files/",
		`{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{"2":{},"3":{}}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"versions\" is not allowed to be present since the Resource (/dirs/d1/files/fy) uses \"xref\".",
  "subject": "/dirs/d1/files/fy",
  "args": {
    "name": "versions"
  },
  "source": "396100315a6e:registry:group:479"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d2",
		`{"files":{"fy":{"meta":{"xref":"/dirs/d1/files/f1"},"versions":{}}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#extra_xref_attribute",
  "title": "Attribute \"versions\" is not allowed to be present since the Resource (/dirs/d2/files/fy) uses \"xref\".",
  "subject": "/dirs/d2/files/fy",
  "args": {
    "name": "versions"
  },
  "source": "396100315a6e:registry:group:479"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/1", ``,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't delete \"versions\" of a Resource (/dirs/d1/files/fx) that uses \"xref\".",
  "subject": "/dirs/d1/files/fx/versions/1",
  "args": {
    "error_detail": "Can't delete \"versions\" of a Resource (/dirs/d1/files/fx) that uses \"xref\""
  },
  "source": "e4e59b8a76c4:registry:version:56"
}
`)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/x", ``,
		404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/fx/versions/x) cannot be found.",
  "subject": "/dirs/d1/files/fx/versions/x",
  "source": "e4e59b8a76c4:registry:httpStuff:2775"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fx/versions/", `{"1":{}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Can't delete \"versions\" of a Resource (/dirs/d1/files/fx) that uses \"xref\".",
  "subject": "/dirs/d1/files/fx/versions/1",
  "args": {
    "error_detail": "Can't delete \"versions\" of a Resource (/dirs/d1/files/fx) that uses \"xref\""
  },
  "source": "e4e59b8a76c4:registry:version:56"
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
  "ancestorid": "1",

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
  "ancestorid": "1",

  "fileproxyurl": "http://localhost:8282/EMPTY-Proxy",
  "filebase64": "aGVsbG8tUHJveHkK",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// Test transitive ximportresources and xref's
func TestXrefXImportTransitive(t *testing.T) {
	reg := NewRegistry("TestXrefXImportTransitive")
	defer PassDeleteReg(t, reg)

	// Make sure they're not alphabetically ordered
	XHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "bars":{"singular":"bar","ximportresources":["/foos/files"]},
        "foos":{"singular":"foo","resources":{"files":{"singular":"file"}}},
        "zoos":{"singular":"zoo","ximportresources":["/bars/files"]}
      }
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/foos/f1/files/f1", ``, 201, `*`)

	for _, test := range [][2]string{
		{"/foos/f1/files/f2", "/foos/f1/files/f1"},

		{"/bars/b1/files/f1", "/foos/f1/files/f1"},
		{"/bars/b1/files/f2", "/foos/f1/files/f2"},
		{"/bars/b1/files/f3", "/bars/b1/files/f1"},

		{"/zoos/z1/files/f1", "/foos/f1/files/f1"},
		{"/zoos/z1/files/f2", "/foos/f1/files/f2"},
		{"/zoos/z1/files/f3", "/bars/b1/files/f1"},
		{"/zoos/z1/files/f4", "/bars/b1/files/f2"},
		{"/zoos/z1/files/f5", "/bars/b1/files/f3"},

		{"/zoos/z1/files/f6", "/zoos/z1/files/f1"},
		{"/zoos/z1/files/f7", "/zoos/z1/files/f2"},
		{"/zoos/z1/files/f8", "/zoos/z1/files/f3"},
		{"/zoos/z1/files/f9", "/zoos/z1/files/f4"},
		{"/zoos/z1/files/fA", "/zoos/z1/files/f5"},
	} {
		XHTTP(t, reg, "PUT", test[0]+"$details", `{
          "meta": {
            "xref": "`+test[1]+`"
          }
        }`, 201, `*`)
	}
}

// TestXrefClearAfterMultipleTouches covers a real repro :
// create a Resource, touch its Meta several times (bumping
// its epoch), set an xref on it, then clear the xref again via a
// full-replace update that simply omits "xref" (as opposed to
// explicitly setting it to null). This used to:
//   - corrupt Props with a PRIMARY KEY collision (Duplicate
//     entry ... "ancestorid") when SaveXrefCascade() ran eagerly
//     before the Resource's own real Version rows were deleted - fixed
//     by deferring it into Tx.runResourceCascade().
//   - then, once that was fixed, panic with "No versions" in
//     Resource.EnsureLatest() because clearing an xref via a
//     full-replace body that omits "xref" wasn't recognized as
//     "clearing" (only an explicit "xref":null/false was) - fixed in
//     Resource.UpsertMeta() ("hasXref || !IsNil(meta.Object["xref"])")
//     and, for the Resource-level ("?inline=meta") path, in
//     Group.UpsertResource() (the "hasMeta && !okObj && metaAddType
//     != ADD_PATCH" clause).
//
// It also locks in the epoch math: when xref is set, epoch mirrors the
// target's epoch (dropping down, even if lower than our own last
// value); when xref is cleared, epoch must become greater than
// max(our own last real epoch, the target's epoch) - not just greater
// than one of them - per spec.
func TestXrefClearAfterMultipleTouches(t *testing.T) {
	reg := NewRegistry("TestXrefClearAfterMultipleTouches")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)
	d, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)

	// ----------------------------------------------------------------
	// Scenario A: drive it all through the Meta endpoint directly
	// ("/dirs/d1/files/fx/meta") - this is the path fixed in
	// Resource.UpsertMeta().
	// ----------------------------------------------------------------
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", "{}", 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx", "{}", 201, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Touch fx's Meta 4 times (no xref yet) - epoch just increments:
	// 1 (from create) -> 2 -> 3 -> 4 -> 5.
	for epoch := 2; epoch <= 5; epoch++ {
		XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta", "{}", 200,
			fmt.Sprintf(`{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": %d,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:01:00Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": false
}
`, epoch))
	}

	// Set xref -> f1. fx's Meta.epoch mirrors f1's (1), even though
	// fx's own epoch was already at 5 - it's a full mirror of the
	// target, not a max().
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	// Clear xref via a full-replace Meta PUT that simply omits "xref"
	// (as opposed to explicitly "xref":null). New epoch must be
	// greater than max(previous real epoch [5], xref target's epoch
	// [1]) i.e. 6, and a new real Version must get created since fx
	// had none of its own left.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta", "{}", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 6,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:01:00Z",
  "readonly": false,

  "defaultversionid": "2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/2",
  "defaultversionsticky": false
}
`)

	fx, err := d.FindResource("files", "fx", false, registry.FOR_WRITE)
	XNoErr(t, err)
	numVers, xErr := fx.GetNumberOfVersions()
	XNoErr(t, xErr)
	XCheck(t, numVers == 1, "Expected 1 Version for fx, got %d", numVers)

	fxMeta, xErr := fx.FindMeta(false, registry.FOR_WRITE)
	XNoErr(t, xErr)
	XCheck(t, IsNil(fxMeta.Get("xref")), "fx.xref should be cleared: %v",
		fxMeta.Get("xref"))
	epochAny := fxMeta.Get("epoch")
	XCheck(t, NotNilInt(&epochAny) == 6,
		"Expected fx epoch 6, got %v", epochAny)

	// ----------------------------------------------------------------
	// Scenario B: same dance, but this time driven through the
	// Resource-level endpoint ("/dirs/d1/files/fy?inline=meta") -
	// exercising Group.UpsertResource()'s own (separate) xref-clear
	// detection logic, which had the same gap and needed its own fix.
	// ----------------------------------------------------------------
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", "{}", 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy?inline=meta", "{}", 201, `{
  "fileid": "fy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fy",
  "xid": "/dirs/d1/files/fy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fy/meta",
  "meta": {
    "fileid": "fy",
    "self": "http://localhost:8181/dirs/d1/files/fy/meta",
    "xid": "/dirs/d1/files/fy/meta",
    "epoch": 1,
    "createdat": "2025-01-01T00:00:00Z",
    "modifiedat": "2025-01-01T00:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fy/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fy/versions",
  "versionscount": 1
}
`)

	// Touch fy (via the Resource endpoint, "meta":{}) 4 times -
	// epoch: 1 (from create) -> 2 -> 3 -> 4 -> 5.
	for epoch := 2; epoch <= 5; epoch++ {
		XHTTP(t, reg, "PUT", "/dirs/d1/files/fy?inline=meta",
			`{"meta":{}}`, 200, fmt.Sprintf(`{
  "fileid": "fy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fy",
  "xid": "/dirs/d1/files/fy",
  "epoch": %d,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:01:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fy/meta",
  "meta": {
    "fileid": "fy",
    "self": "http://localhost:8181/dirs/d1/files/fy/meta",
    "xid": "/dirs/d1/files/fy/meta",
    "epoch": %d,
    "createdat": "2025-01-01T00:00:00Z",
    "modifiedat": "2025-01-01T00:01:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fy/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fy/versions",
  "versionscount": 1
}
`, epoch, epoch))
	}

	// Set xref -> f2 (Resource-level). fy's Meta.epoch mirrors f2's
	// (1), same as scenario A.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f2"}}`, 200, `{
  "fileid": "fy",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fy",
  "xid": "/dirs/d1/files/fy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fy/meta",
  "meta": {
    "fileid": "fy",
    "self": "http://localhost:8181/dirs/d1/files/fy/meta",
    "xid": "/dirs/d1/files/fy/meta",
    "xref": "/dirs/d1/files/f2",
    "epoch": 1,
    "createdat": "2025-01-01T00:00:00Z",
    "modifiedat": "2025-01-01T00:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fy/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fy/versions",
  "versionscount": 1
}
`)

	// Clear xref via a full-replace Resource-level PUT whose "meta"
	// sub-object simply omits "xref". New epoch must be 6 (same math
	// as scenario A) and a new real Version must get created.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy?inline=meta", `{"meta":{}}`,
		200, `{
  "fileid": "fy",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/fy",
  "xid": "/dirs/d1/files/fy",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "2",

  "metaurl": "http://localhost:8181/dirs/d1/files/fy/meta",
  "meta": {
    "fileid": "fy",
    "self": "http://localhost:8181/dirs/d1/files/fy/meta",
    "xid": "/dirs/d1/files/fy/meta",
    "epoch": 6,
    "createdat": "2025-01-01T00:00:01Z",
    "modifiedat": "2025-01-01T00:00:00Z",
    "readonly": false,

    "defaultversionid": "2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fy/versions/2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fy/versions",
  "versionscount": 1
}
`)

	fy, err := d.FindResource("files", "fy", false, registry.FOR_WRITE)
	XNoErr(t, err)
	numVers, xErr = fy.GetNumberOfVersions()
	XNoErr(t, xErr)
	XCheck(t, numVers == 1, "Expected 1 Version for fy, got %d", numVers)

	fyMeta, xErr := fy.FindMeta(false, registry.FOR_WRITE)
	XNoErr(t, xErr)
	XCheck(t, IsNil(fyMeta.Get("xref")), "fy.xref should be cleared: %v",
		fyMeta.Get("xref"))
	epochAny = fyMeta.Get("epoch")
	XCheck(t, NotNilInt(&epochAny) == 6,
		"Expected fy epoch 6, got %v", epochAny)
}

// Deleting a xref TARGET's current default Version must fan out
// correctly to any xref SOURCE(s) pointing at it - this ties
// Resource.SetDefault()'s cascade mark together with
// SaveXrefFanOutForTarget in the same deferred drain.
func TestCascadeDeferDeleteDefaultWithXrefFanOut(t *testing.T) {
	reg := NewRegistry("TestCascadeDeferDeleteDefaultWithXrefFanOut")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Delete the target's current (non-sticky) default - v3 - with no
	// explicit next. Target's new default becomes v2.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)

	// The xref source must mirror the SAME final state - v2 as default -
	// not a stale v3 reference nor an intermediate/partial cascade
	// result.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2",
  "defaultversionsticky": false
}
`)
}

// TestXrefSurvivesModelAttributeRemoval (gaps item 6) verifies that
// removing an attribute definition from the model (that an existing xref
// TARGET's real data still uses) doesn't break or lose the xref'd mirror -
// the value should simply become an untyped extension attribute, still
// present and still correctly mirrored.
func TestXrefSurvivesModelAttributeRemoval(t *testing.T) {
	reg := NewRegistry("TestXrefSurvivesModelAttributeRemoval")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "extra": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"extra":"value1"}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:46.350994676Z",
  "modifiedat": "2026-07-27T00:25:46.350994676Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
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
  "createdat": "2026-07-27T00:25:48.418456014Z",
  "modifiedat": "2026-07-27T00:25:48.418456014Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:50.468764805Z",
  "modifiedat": "2026-07-27T00:25:50.468764805Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now apply a new model that no longer defines "extra" explicitly,
	// but keeps a "*" wildcard so it survives as an untyped extension
	// attribute (a model without ANY wildcard would instead reject
	// existing "extra" data as an unknown_attribute during revalidation -
	// that's a model-authoring choice, not what this test is about).
	newModelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "*": { "type": "any" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, newModelSrc, true))

	// The target's data survives (now as an untyped extension attr) ...
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:52.495299534Z",
  "modifiedat": "2026-07-27T00:25:52.495299534Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// ... and so does the xref mirror.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:54.526697612Z",
  "modifiedat": "2026-07-27T00:25:54.526697612Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// ---- Moved from xref_order_test.go ----
//
// These tests focus on a gap not covered by xref_test.go/ancestor_test.go/
// constraints_test.go: what happens when the xref TARGET resource is
// mutated (batch version create, default-version change) AFTER one or
// more xref SOURCES already point at it. This exercises the interaction
// between the version/default-version cascade and the xref fan-out logic
// (see plan.md "Backend / SQL re-architecture" item (b)) - specifically
// whether the fan-out correctly reflects the target's final state,
// including when multiple xref sources point at the same target, and
// when the target's default-version cascade and its xref-fan-out happen
// in the same request/transaction.
//
// IMPORTANT: These tests intentionally only use the public HTTP API
// (XHTTP) and don't reach into FullTree*-specific internals, so they can
// be ported to the pre-FullTree codebase (if needed) to distinguish
// pre-existing bugs from new regressions introduced by any future
// cascade-deferral work.

// Adding multiple versions (in one batch request) to an xref TARGET,
// after one or more xref SOURCES already exist, must be reflected
// correctly in all of the sources' mirrored data (versions, versioncount,
// default version).
func TestXrefOrderMultiVersionAfterXref(t *testing.T) {
	reg := NewRegistry("TestXrefOrderMultiVersionAfterXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Target starts with just one version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	// Two different xref sources pointing at the same target
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Now batch-add two more versions to the target in a single request
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": {},
      "v3": {}
    }`, 200, `*`)

	// Target itself should show all 3 versions, v3 as default (newest)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta", "", 200, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	// Both xref sources must mirror the target's final state - 3 versions,
	// v3 as the default - not some stale intermediate state from the
	// batch's individual per-version saves.
	for _, rid := range []string{"fx", "fy"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"?inline=meta", "", 200, `{
  "fileid": "`+rid+`",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`",
  "xid": "/dirs/d1/files/`+rid+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "meta": {
    "fileid": "`+rid+`",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
    "xid": "/dirs/d1/files/`+rid+`/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v3",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions",
  "versionscount": 3
}
`)

		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"/versions", "", 200, `{
  "v1": {
    "fileid": "`+rid+`",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v1",
    "xid": "/dirs/d1/files/`+rid+`/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  },
  "v2": {
    "fileid": "`+rid+`",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "xid": "/dirs/d1/files/`+rid+`/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "v1"
  },
  "v3": {
    "fileid": "`+rid+`",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v3",
    "xid": "/dirs/d1/files/`+rid+`/versions/v3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "v2"
  }
}
`)
	}
}

// Several oddities that can only show up when MULTIPLE resources xref
// the same target: batch-creating several sources pointing at the same
// target in a single request (order-independence within one Tx),
// deleting one source must not affect the target or the remaining
// sources, and re-pointing one source's xref away from the target must
// stop it from being fanned-out to while the other sources keep mirroring
// the original target correctly.
func TestXrefOrderMultipleSourcesSameTarget(t *testing.T) {
	reg := NewRegistry("TestXrefOrderMultipleSourcesSameTarget")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Target with 2 versions, and a second (unrelated) target to re-point
	// to later.
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}
    }`, 200, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/w1", `{}`, 201, `*`)

	// Batch-create 3 xref sources in ONE request, all pointing at f1.
	XHTTP(t, reg, "POST", "/dirs/d1/files/?inline=meta", `{
      "fa": {"meta": {"xref": "/dirs/d1/files/f1"}},
      "fb": {"meta": {"xref": "/dirs/d1/files/f1"}},
      "fc": {"meta": {"xref": "/dirs/d1/files/f1"}}
    }`, 200, `*`)

	// All three must mirror f1's current default version (v2) - order of
	// creation within the batch must not matter.
	for _, rid := range []string{"fa", "fb", "fc"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"?inline=meta", "", 200, `{
  "fileid": "`+rid+`",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`",
  "xid": "/dirs/d1/files/`+rid+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "meta": {
    "fileid": "`+rid+`",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
    "xid": "/dirs/d1/files/`+rid+`/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions",
  "versionscount": 2
}
`)
	}

	// Deleting one source (fb) must not affect the target or the other
	// two sources.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fb", ``, 204, ``)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta", "", 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	for _, rid := range []string{"fa", "fc"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"/meta", "", 200, `{
  "fileid": "`+rid+`",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "xid": "/dirs/d1/files/`+rid+`/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
  "defaultversionsticky": false
}
`)
	}

	// Re-point fa's xref away from f1 to f2. fa must now mirror f2, while
	// fc keeps mirroring f1 correctly (no cross-contamination).
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fa/meta",
		`{"xref":"/dirs/d1/files/f2"}`, 200, `*`)

	// Now add a new version to f1 - fa (re-pointed to f2) must NOT pick
	// this up, but fc (still pointing at f1) must.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3", `{}`, 201, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fa/meta", "", 200, `{
  "fileid": "fa",
  "self": "http://localhost:8181/dirs/d1/files/fa/meta",
  "xid": "/dirs/d1/files/fa/meta",
  "xref": "/dirs/d1/files/f2",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "w1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fa/versions/w1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fc/meta", "", 200, `{
  "fileid": "fc",
  "self": "http://localhost:8181/dirs/d1/files/fc/meta",
  "xid": "/dirs/d1/files/fc/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fc/versions/v3",
  "defaultversionsticky": false
}
`)
}

// Batch-create an xref TARGET and two xref SOURCES pointing at it, all
// in a single request/Tx (rather than the target pre-existing before
// the sources, as in TestXrefOrderMultipleSourcesSameTarget). This
// means the target and both sources start out simultaneously "pending
// validation" in the same Tx, so - depending on the randomized order
// Registry.Validate()'s drain loop happens to process them in - one or
// both sources' own runCascade() may run before the target's, letting
// the target's own (still-pending) fan-out cover them instead of each
// source doing its own redundant xref-cascade-insert (see runCascade()'s
// "skip our own insert if the target is still pending" optimization).
// Named f1/f2/f3 (target f2 alphabetically between the two sources) to
// maximize the odds of that ordering actually being exercised across
// runs, since Go's map iteration order is randomized per-run - but the
// end result asserted below must be correct regardless of which order
// actually happened.
func TestXrefOrderBatchCreateTargetAndSourcesTogether(t *testing.T) {
	reg := NewRegistry("TestXrefOrderBatchCreateTargetAndSourcesTogether")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/?inline=meta", `{
      "f1": {"meta": {"xref": "/dirs/d1/files/f2"}},
      "f2": {"versions": {"v1": {}, "v2": {}}},
      "f3": {"meta": {"xref": "/dirs/d1/files/f2"}}
    }`, 200, `*`)

	// Target itself: 2 versions, v2 as default (newest)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f2?inline=meta", "", 200, `{
  "fileid": "f2",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "meta": {
    "fileid": "f2",
    "self": "http://localhost:8181/dirs/d1/files/f2/meta",
    "xid": "/dirs/d1/files/f2/meta",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f2/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 2
}
`)

	// Both sources must mirror the target's final state (v2, 2 versions),
	// no matter which order the batch's 3 entries happened to be
	// processed/validated in.
	for _, rid := range []string{"f1", "f3"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"?inline=meta", "", 200, `{
  "fileid": "`+rid+`",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`",
  "xid": "/dirs/d1/files/`+rid+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "meta": {
    "fileid": "`+rid+`",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
    "xid": "/dirs/d1/files/`+rid+`/meta",
    "xref": "/dirs/d1/files/f2",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions",
  "versionscount": 2
}
`)

		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"/versions", "", 200, `{
  "v1": {
    "fileid": "`+rid+`",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v1",
    "xid": "/dirs/d1/files/`+rid+`/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  },
  "v2": {
    "fileid": "`+rid+`",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "xid": "/dirs/d1/files/`+rid+`/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  }
}
`)
	}
}

// Explicitly changing the xref TARGET's default version (via
// setdefaultversionid, making it sticky) after xref SOURCES already
// exist must fan-out correctly to all sources.
func TestXrefOrderTargetDefaultChangeAfterXref(t *testing.T) {
	reg := NewRegistry("TestXrefOrderTargetDefaultChangeAfterXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Sanity: fx currently mirrors v3 (newest) as default, non-sticky
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v3",
  "defaultversionsticky": false
}
`)

	// Now stick the TARGET's default version to v1 (older, non-newest)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?setdefaultversionid=v1", `{}`,
		200, `*`)

	// Target itself should now be sticky on v1
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "defaultversionsticky": true
}
`)

	// The xref SOURCE must mirror the target's new sticky default (v1),
	// not the old newest-wins default (v3).
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 3
}
`)

	// Adding a brand new version (v4) to the target should NOT move the
	// sticky default off of v1, and the source must keep mirroring v1 too.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v4", `{}`, 201, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 3,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 4
}
`)
}

// Combines the "defaultversionsticky=true-by-default" edge case (see
// TestHTTPSticky in tests/http1_test.go ~line 10142, where setting
// defaultversionid on a Meta whose model default for
// defaultversionsticky is true causes it to stick even though a plain
// PUT would normally just be ignored) with xref: reverting an xref (so
// the resource goes back to owning its own versions) and then setting
// defaultversionid must still honor a true default-sticky model setting.
func TestXrefOrderRevertWithStickyDefault(t *testing.T) {
	reg := NewRegistry("TestXrefOrderRevertWithStickyDefault")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "dirs": {
          "singular": "dir",
          "resources": {
            "files": {
              "singular": "file",
              "hasdocument": false,
              "metaattributes": {
                "defaultversionsticky": {
                  "type": "boolean",
                  "required": true,
                  "enum": [ true ],
                  "default": true
                }
              }
            }
          }
        }
      }
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Revert the xref and add a couple of versions of its own
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
      "meta": {"xref": null},
      "versions": {"a1": {}, "a2": {}}
    }`, 200, `*`)

	// Normally setting defaultversionid=a1 (not the newest) would be
	// ignored, but since defaultversionsticky defaults to true for this
	// resource type, it should stick.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"defaultversionid":"a1"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "a1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/a1",
  "defaultversionsticky": true
}
`)
}

// A dangling xref (pointing at a target Resource that doesn't exist yet)
// must self-resolve once the target is later created - even in a
// completely separate request/Tx from the one that set the xref. This
// used to silently never populate the source's mirror, because
// Metas.xRefSID was a point-in-time-resolved SID with no retry
// mechanism; it's now Metas.xRefPath (a plain path string), resolved
// live via a join against Resources on every read/cascade, so there's
// nothing to go stale or need re-resolving.
func TestXrefOrderDanglingTargetCreatedLater(t *testing.T) {
	reg := NewRegistry("TestXrefOrderDanglingTargetCreatedLater")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Dangling xref: target "f1" doesn't exist yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta",
		`{"meta":{"xref":"/dirs/d1/files/f1"}}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	// Now create the target for real, in a completely separate request.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	// The source should now mirror the target.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

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

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// If an xref target Resource is deleted and a brand-new Resource is
// later created at that exact same Path (reusing the same UID, but a
// fresh SID under the hood), any existing xref source pointing at that
// Path must pick up the NEW target - not be permanently orphaned, and
// not accidentally still reflect the deleted target's stale data. This
// exercises the same "resolve xref by Path at query time, never cache
// a SID" fix as TestXrefOrderDanglingTargetCreatedLater, but via
// delete+recreate instead of dangling-then-created.
func TestXrefOrderTargetDeletedThenRecreatedAtSamePath(t *testing.T) {
	reg := NewRegistry("TestXrefOrderTargetDeletedThenRecreatedAtSamePath")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Original target, and a source xref'ing it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Delete the target entirely.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1", ``, 204, ``)

	// The source is now dangling (target gone) - its mirror should be
	// empty (no versions of its own, no defaultversionid). Note: like
	// any dangling xref (even one that never had a target - see
	// TestXrefOrderDanglingTargetCreatedLater before its target is
	// created), the mirrored meta.* attrs (epoch/createdat/modifiedat/
	// readonly) are simply absent, not reverted to some "own" value -
	// setting xref replaces (not shadows) those own rows.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)

	// Recreate a brand-new "f1" (same Path, new underlying SID) with a
	// different version.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{}`, 201, `*`)

	// The source should now mirror the NEW target, not remain
	// orphaned or show any trace of the deleted one.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

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

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// Bulk-delete path: deleting an entire Group (which cascades to
// deleting its Resources at the DB layer via GroupTrigger, bypassing
// Go-level Resource.Delete() entirely) must still clean up any xref
// source's stale mirror - even when that source lives in a DIFFERENT
// Group. This exercises ResourcesTrigger's own DELETE...JOIN cleanup
// (init.sql), not any Go-level call site, since Group deletion never
// calls Resource.Delete() per-row.
func TestXrefOrderTargetDeletedViaGroupBulkDelete(t *testing.T) {
	reg := NewRegistry("TestXrefOrderTargetDeletedViaGroupBulkDelete")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Target "f1" lives in group "d1".
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	// Source "fx" lives in a DIFFERENT group ("d2") and xrefs "f1".
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Sanity check: source currently mirrors the target.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d2/files/fx/meta",
    "xid": "/dirs/d2/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)

	// Delete the ENTIRE "d1" group - this cascades f1's deletion at
	// the DB layer, bypassing Go-level Resource.Delete() entirely.
	XHTTP(t, reg, "DELETE", "/dirs/d1", ``, 204, ``)

	// The source (in the other group) must correctly go dangling -
	// same shape as a direct single-Resource delete.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d2/files/fx/meta",
    "xid": "/dirs/d2/files/fx/meta",
    "xref": "/dirs/d1/files/f1"
  }
}
`)
}

// ---- Moved from xref_usesxref_test.go ----
//
// These tests specifically exercise the Registries.UsesXref internal
// fast-path flag itself (registry/init.sql, Resource.runCascade() in
// resource.go) - not general xref functional behavior (already covered
// by xref_test.go/xref_order_test.go). Since UsesXref is a raw,
// internal-only DB column (never exposed via the HTTP API), these
// tests reach into the registry package directly to check its value,
// unlike the HTTP-only convention used elsewhere in this package.

func getUsesXref(t *testing.T, reg *registry.Registry) bool {
	t.Helper()
	tx := reg.GetTx()
	results := registry.Query(tx, `SELECT UsesXref FROM Registries WHERE SID=?`,
		reg.DbSID)
	defer results.Close()
	row := results.NextRow()
	if row == nil {
		t.Fatalf("no Registries row found for %q", reg.UID)
	}
	return NotNilBoolDef(row[0], false)
}

// A fresh Registry that has never used xref must have UsesXref=false.
// Creating the first xref must flip it to true (and stay true across
// unrelated saves). Deleting that xref (going back to zero xrefs in
// the Registry) must eventually flip it back to false. Creating a new
// xref again afterward must flip it back to true - the flag's
// lifecycle must be fully reversible, not "sticky" once cleared.
func TestUsesXrefLifecycle(t *testing.T) {
	reg := NewRegistry("TestUsesXrefLifecycle")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Brand-new Registry, never touched xref.
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false on a fresh Registry")
	}

	// An unrelated save (no xref involved) must not flip it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after an unrelated save")
	}

	// Create the first (and only) xref in this Registry.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true right after creating an xref")
	}

	// Another unrelated save must not clear it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v1", `{}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref to stay true after an unrelated save")
	}

	// Clear the (only) xref in the Registry - flag should go back to
	// false since a rescan finds nothing left.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta", `{}`, 200, `*`)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after clearing the last xref")
	}

	// Create a new xref again - must correctly flip back to true, not
	// stay stuck false.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy/meta",
		`{"xref":"/dirs/d1/files/f2"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after re-creating an xref")
	}

	// Deleting the xref SOURCE Resource entirely (rather than just
	// clearing its xref attribute) must also correctly rescan/clear.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fy", ``, 204, ``)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after deleting the xref source")
	}
}

// Deleting an entire Group containing an xref source (bulk-delete path
// that bypasses Go-level Resource.Delete()) must still correctly
// rescan and clear UsesXref - this is the whole reason the clearing
// side of the flag is handled by DB triggers rather than only in Go.
func TestUsesXrefClearedByGroupBulkDelete(t *testing.T) {
	reg := NewRegistry("TestUsesXrefClearedByGroupBulkDelete")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after creating an xref")
	}

	// Bulk-delete the SOURCE's entire group ("d2").
	XHTTP(t, reg, "DELETE", "/dirs/d2", ``, 204, ``)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after bulk-deleting the only xref source's group")
	}
}

// Deleting the entire Registry while it still has an active xref must
// not error out (regression test for MySQL error 1442 - "Can't update
// table 'Registries' in stored function/trigger because it is already
// used by statement which invoked this stored function/trigger" - hit
// during development because ResourcesTrigger/FullTreeXref's own
// UsesXref-clearing UPDATE against Registries collides with the
// outermost DELETE FROM Registries statement that's cascading through
// them). PassDeleteReg() (deferred above) already deletes the
// Registry, so this test's real assertion is simply that it doesn't
// panic.
func TestUsesXrefRegistryDeleteWithActiveXref(t *testing.T) {
	reg := NewRegistry("TestUsesXrefRegistryDeleteWithActiveXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after creating an xref")
	}
}
