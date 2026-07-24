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
//   - corrupt FullTreeTable with a PRIMARY KEY collision (Duplicate
//     entry ... "ancestorid") when fullSaveXrefCascade() ran eagerly
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
