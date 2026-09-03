package tests

import (
	"fmt"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestVersionCreate(t *testing.T) {
	reg := NewRegistry("TestVersionCreate")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": true
        }
      }
    }
  }
}`

	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{}`,
		201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details", `{}`,
		201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
  "xid": "/dirs/d1/files/f1/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1"
}
`)

	// PUT to an existing version is just an upsert (HTTP has no
	// separate strict-create-only vs upsert distinction like the raw
	// Go API's AddVersion() vs UpsertVersion())
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details", `{}`,
		200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
  "xid": "/dirs/d1/files/f1/versions/v2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.01Z",
  "ancestorid": "v1"
}
`)

	// v2 should now be the default (newest version)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-31T13:40:27.46166805Z",
  "modifiedat": "2026-07-31T13:40:28.46166805Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d2", `{}`, 201, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1/versions/v1$details", `{}`,
		201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/v1$details",
  "xid": "/dirs/d2/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1/versions/v1.1$details", `{}`,
		201, `{
  "fileid": "f1",
  "versionid": "v1.1",
  "self": "http://localhost:8181/dirs/d2/files/f1/versions/v1.1$details",
  "xid": "/dirs/d2/files/f1/versions/v1.1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1"
}
`)

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f1/v1
	//      /d2/f1/v1.1

	// Check basic GET first
	XCheckGet(t, reg, "/dirs/d1/files/f1/versions/v1$details",
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v1"
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/xxx) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/xxx",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/xxx) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/xxx",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/f1/versions/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/xxx/yyy) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/xxx/yyy",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:info:699"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/f1/versions/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1/versions/xxx/yyy) cannot be found.",
  "subject": "/dirs/d1/files/f1/versions/xxx/yyy",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:info:699"
}
`)

	XCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)

	// Delete v2, next default should be auto-computed (back to v1)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2", ``, 204, ``)

	XCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3", `{}`, 201, `*`)

	// Make v3 the sticky default
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
  "defaultversionid": "v3",
  "defaultversionsticky": true
}`, 200, `*`)

	XCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v2":{},"v3":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1":{},"v1.1":{}}}}}}}`)
	XCheckGet(t, reg, "/dirs/d1/files/f1$details?inline=meta", `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 6,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	// Delete v2 (not the sticky default), default should stay v3
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2", ``, 204, ``)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `*`)

	// Delete v3 (the sticky default), next default auto-computed
	// (back to v1, sticky cleared)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", ``, 204, ``)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `*`)

	// Trying to set default to a non-existent version should fail
	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1/meta", `{
  "defaultversionid": "v2",
  "defaultversionsticky": true
}`, 400, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1/versions/v3", `{}`, 201, `*`)

	// Delete v1 in d2, next default auto-computed
	XHTTP(t, reg, "DELETE", "/dirs/d2/files/f1/versions/v1", ``, 204, ``)

	XCheckGet(t, reg, "?inline&oneline",
		`{"dirs":{"d1":{"files":{"f1":{"meta":{},"versions":{"v1":{}}}}},"d2":{"files":{"f1":{"meta":{},"versions":{"v1.1":{},"v3":{}}}}}}}`)

	// Trying to delete a version and set the next default to a
	// non-existent version should fail
	XHTTP(t, reg, "DELETE", "/dirs/d2/files/f1/versions/v1.1?setdefaultversionid=v2",
		``, 400, `*`)

	// Trying to delete a version and set the next default to itself
	// should fail
	XHTTP(t, reg, "DELETE", "/dirs/d2/files/f1/versions/v1.1?setdefaultversionid=v1.1",
		``, 400, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1/versions/v4", `{}`, 201, `*`)

	// Delete v4 and explicitly set the next default to v3
	XHTTP(t, reg, "DELETE", "/dirs/d2/files/f1/versions/v4?setdefaultversionid=v3",
		``, 204, ``)

	XCheckGet(t, reg, "dirs/d2/files?inline=meta",
		`{
  "f1": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d2/files/f1$details",
    "xid": "/dirs/d2/files/f1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "ancestorid": "v1.1",

    "metaurl": "http://localhost:8181/dirs/d2/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d2/files/f1/meta",
      "xid": "/dirs/d2/files/f1/meta",
      "epoch": 6,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:03Z",
      "readonly": false,

      "defaultversionid": "v3",
      "defaultversionurl": "http://localhost:8181/dirs/d2/files/f1/versions/v3$details",
      "defaultversionsticky": true
    },
    "versionsurl": "http://localhost:8181/dirs/d2/files/f1/versions",
    "versionscount": 2
  }
}
`)
}

func TestVersionDefault(t *testing.T) {
	reg := NewRegistry("TestVersionDefault")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	v1, _ := f1.FindVersion("v1", false)
	v2, _ := f1.AddVersion("v2")

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
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
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Doesn't change much, but does make it sticky
	XNoErr(t, f1.SetDefault(v2))

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
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
    "epoch": 2,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	v3, _ := f1.AddVersion("v3")

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
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
    "epoch": 3,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	// Now unstick it and it default should be v3 now
	XNoErr(t, f1.SetDefault(nil))
	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 4,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	v4, _ := f1.AddVersion("v4")
	XNoErr(t, f1.SetDefault(v4))
	v5, _ := f1.AddVersion("v5")

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 5,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`)

	err := v1.DeleteSetNextVersion("")
	XNoErr(t, err)
	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 6,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`)

	err = v3.DeleteSetNextVersion("v1")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"v1\" cannot be found.",
  "detail": "Can't find next default Version \"v1\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "v1",
    "singular": "version"
  },
  "source": "e4e59b8a76c4:registry:version:117"
}`)
	err = v3.DeleteSetNextVersion("v2")
	XNoErr(t, err)
	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 7,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	err = v2.DeleteSetNextVersion("")
	XNoErr(t, err)
	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v4",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 8,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:03Z",
    "readonly": false,

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XNoErr(t, v4.DeleteSetNextVersion(""))
	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "v5",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 9,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,

    "defaultversionid": "v5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XNoErr(t, v5.DeleteSetNextVersion(""))
	XCheckGet(t, reg, "dirs/d1/files/f1$details", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1$details) cannot be found.",
  "subject": "/dirs/d1/files/f1$details",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1730"
}
`)
}

func TestVersionDefaultMaxVersions(t *testing.T) {
	reg := NewRegistry("TestVersionDefaultMaxVersions")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 3, true, true)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	f1.FindVersion("v1", false)
	f1.AddVersion("v2")
	f1.AddVersion("v3")

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:01Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	v4, _ := f1.AddVersion("v4")

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:02Z",
    "modifiedat": "2024-01-01T12:00:01Z",
    "readonly": false,

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	XNoErr(t, f1.SetDefault(v4))
	f1.AddVersion("v5") // v3,v4,v5
	// check def = v4
	f1.AddVersion("v6") // v4*,v5,v6
	f1.AddVersion("v7") // v4*,v6,v7
	f1.AddVersion("v8") // v4*,v7,v8
	// check def = v4    v8, v7, v4

	XCheckGet(t, reg, "dirs/d1/files/f1$details?inline=versions,meta",
		`{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "v4",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 3,
    "createdat": "2024-01-01T12:00:03Z",
    "modifiedat": "2024-01-01T12:00:02Z",
    "readonly": false,

    "defaultversionid": "v4",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v4": {
      "fileid": "f1",
      "versionid": "v4",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4$details",
      "xid": "/dirs/d1/files/f1/versions/v4",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:01Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestorid": "v4"
    },
    "v7": {
      "fileid": "f1",
      "versionid": "v7",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v7$details",
      "xid": "/dirs/d1/files/f1/versions/v7",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestorid": "v7"
    },
    "v8": {
      "fileid": "f1",
      "versionid": "v8",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v8$details",
      "xid": "/dirs/d1/files/f1/versions/v8",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:02Z",
      "modifiedat": "2024-01-01T12:00:02Z",
      "ancestorid": "v7"
    }
  },
  "versionscount": 3
}
`)

}

func TestVersionRequiredFields(t *testing.T) {
	reg := NewRegistry("TestVersionRequiredFields")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	_, err := rm.AddAttribute(&registry.Attribute{
		Name:     "req",
		Type:     STRING,
		Required: true,
	})
	XNoErr(t, err)

	group, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)

	f1, err := group.AddResourceWithObject("files", "f1", "v1",
		Object{"req": "test"}, false)
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	_, err = f1.AddVersion("v2")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f1/versions/v2\" are missing: req.",
  "subject": "/dirs/d1/files/f1/versions/v2",
  "args": {
    "list": "req"
  },
  "source": "e4e59b8a76c4:registry:entity:2150"
}`)
	reg.Rollback()
	reg.Refresh(registry.FOR_WRITE)

	v1, _, err := f1.UpsertVersionWithObject(&registry.VersionUpsert{
		Id:               "v2",
		Obj:              Object{"req": "test"},
		AddType:          registry.ADD_ADD,
		More:             false,
		DefaultVersionID: "",
	})
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	err = v1.SetSave("req", nil)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f1/versions/v2\" are missing: req.",
  "subject": "/dirs/d1/files/f1/versions/v2",
  "args": {
    "list": "req"
  },
  "source": "e4e59b8a76c4:registry:entity:2150"
}`)

	err = v1.SetSave("req", "again")
	XNoErr(t, err)
}

func TestVersionOrdering(t *testing.T) {
	// Make sure that "latest" is based on "createdat" first and then
	// case insensitive "ID"s (smallest == oldest)
	reg := NewRegistry("TestVersionOrdering")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/", `{
      "modelsource": `+MODEL_DIRS_NODOC+`,
      "dirs": {
        "d1": {
          "files": {
            "f1": {
              "versionid": "z5",
              "versions": {
                "z5": { "ancestorid": "z5" },
                "v2": { "ancestorid": "v2" },
                "v9": { "ancestorid": "v9" },
                "V3": { "ancestorid": "V3" },
                "V1": { "ancestorid": "V1" },
                "Z1": { "ancestorid": "Z1" },
                "v5": { "ancestorid": "v5" }
              }
            }
          }
        }
      }
    }`, 200, "*")
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline", "", 200, `{
  "fileid": "f1",
  "versionid": "z5",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-24T19:25:08.643450018Z",
  "modifiedat": "2026-07-24T19:25:08.643450018Z",
  "ancestorid": "z5",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-07-24T19:25:08.643450018Z",
    "modifiedat": "2026-07-24T19:25:08.643450018Z",
    "readonly": false,

    "defaultversionid": "z5",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/z5",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "V1": {
      "fileid": "f1",
      "versionid": "V1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/V1",
      "xid": "/dirs/d1/files/f1/versions/V1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "V1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "v2"
    },
    "V3": {
      "fileid": "f1",
      "versionid": "V3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/V3",
      "xid": "/dirs/d1/files/f1/versions/V3",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "V3"
    },
    "v5": {
      "fileid": "f1",
      "versionid": "v5",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
      "xid": "/dirs/d1/files/f1/versions/v5",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "v5"
    },
    "v9": {
      "fileid": "f1",
      "versionid": "v9",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
      "xid": "/dirs/d1/files/f1/versions/v9",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "v9"
    },
    "Z1": {
      "fileid": "f1",
      "versionid": "Z1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/Z1",
      "xid": "/dirs/d1/files/f1/versions/Z1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "Z1"
    },
    "z5": {
      "fileid": "f1",
      "versionid": "z5",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/z5",
      "xid": "/dirs/d1/files/f1/versions/z5",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-07-24T19:25:08.643450018Z",
      "modifiedat": "2026-07-24T19:25:08.643450018Z",
      "ancestorid": "z5"
    }
  },
  "versionscount": 7
}
`)

	t0 := "2020-01-02T12:00:00Z"
	t1 := "2024-01-02T12:00:00Z"
	t2 := "2023-11-22T01:02:03Z"
	t9 := "2025-01-02T12:00:00Z"
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
	  "versions": {
	    "z5": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v2": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "V3": { "createdat": "`+t0+`","modifiedat":"`+t2+`" },
	    "V1": { "createdat": "`+t9+`","modifiedat":"`+t2+`" },
	    "Z1": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v9": { "createdat": "`+t1+`","modifiedat":"`+t2+`" },
	    "v5": { "createdat": "`+t1+`","modifiedat":"`+t2+`" }
	  }
    }`, 200, `{
  "fileid": "f1",
  "versionid": "V1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "`+t9+`",
  "modifiedat": "`+t2+`",
  "ancestorid": "V1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 7
}
`, NOMASK_TS)
	ids := []string{"V1", "z5", "Z1", "v9", "v5", "v2", "V3"}

	for i, id := range ids {
		XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/"+id, ``, 204, ``)
		if i == len(ids)-1 {
			break
		}

		ct := t1
		if id == "v2" {
			ct = t0
		}

		XHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 200, fmt.Sprintf(`{
  "fileid": "f1",
  "versionid": "%s",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "`+ct+`",
  "modifiedat": "`+t2+`",
  "ancestorid": "%s",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": %d
}
`, ids[i+1], ids[i+1], 6-i), NOMASK_TS)
	}

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1", ``, 404, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/f1) cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1730"
}
`)

}

func TestVersionOrdering2(t *testing.T) {
	// Make sure that "latest" is based on "createdat" first and then
	// case insensitive "ID"s (smallest == oldest)
	reg := NewRegistry("TestVersionOrdering2")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	ts1 := "2020-01-02T12:00:00Z"

	XCheckHTTP(t, reg, &HTTPTest{
		// URL:        "/dirs/d1/files/f1/versions?setdefaultversionid=v5",
		URL:        "/dirs/d1/files/f1",
		Method:     "PUT",
		ReqHeaders: []string{},
		ReqBody: `{
  "versionid": "v1",
  "versions": {
    "v1": { "createdat": "` + ts1 + `","ancestorid":"v1"},
    "v2": { "createdat": "` + ts1 + `","ancestorid":"v2"},
    "v3": { "createdat": "` + ts1 + `","ancestorid":"v3"},
    "v4": { "createdat": "` + ts1 + `","ancestorid":"v4"},
    "v5": { "createdat": "` + ts1 + `","ancestorid":"v5"}
		}}`,

		Code: 201,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "v5",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 5
}
`})

	XCheckHTTP(t, reg, &HTTPTest{
		URL:    "/dirs/d1/files/f1/meta",
		Method: "GET",
		Code:   200,
		ResBody: `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,

  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
  "defaultversionsticky": false
}
`})

	ts2 := "2024-02-02T12:00:00Z"

	XCheckHTTP(t, reg, &HTTPTest{
		URL:        "/dirs/d1/files/f1/versions/v3",
		Method:     "PATCH",
		ReqHeaders: []string{},
		ReqBody: `{
		    "createdat": "` + ts2 + `"
		}`,

		Code: 200,
		ResBody: `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "xid": "/dirs/d1/files/f1/versions/v3",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "ancestorid": "v3"
}
`})
}

func TestVersionExtensions(t *testing.T) {
	reg := NewRegistry("TestVersionExtensions")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "attributes": {
            "*": {
              "type": "any"
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
      "meta": "ads"
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"meta\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: meta.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: meta",
    "name": "meta"
  },
  "instance": "xxx",
  "source": "a3d56ce41e09:registry:entity:2236"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
      "metaurl": {}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"metaurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: metaurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: metaurl",
    "name": "metaurl"
  },
  "instance": "xxx",
  "source": "a3d56ce41e09:registry:entity:2236"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
     "v1": {
       "metaurl": {}
      }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"metaurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: metaurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: metaurl",
    "name": "metaurl"
  },
  "instance": "xxx",
  "source": "a3d56ce41e09:registry:entity:2236"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
     "versions": {
       "v1": {
         "metaurl": {}
        }
      }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"metaurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: metaurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: metaurl",
    "name": "metaurl"
  },
  "instance": "xxx",
  "source": "a3d56ce41e09:registry:entity:2236"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
         "versions": ""
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versions\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versions.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versions",
    "name": "versions"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
         "versionsurl": ""
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versionsurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versionsurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versionsurl",
    "name": "versionsurl"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {
         "versions": ""
       }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versions\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versions.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versions",
    "name": "versions"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {
         "versionsurl": ""
       }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versionsurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versionsurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versionsurl",
    "name": "versionsurl"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
      "versions": {
        "v1": {
           "versions": ""
         }
      }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versions\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versions.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versions",
    "name": "versions"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
      "versions": {
        "v1": {
           "versionsurl": ""
         }
      }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"versionsurl\" for \"/dirs/d1/files/f1/versions/v1\" is not valid: Versions can't define an extension called: versionsurl.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "error_detail": "Versions can't define an extension called: versionsurl",
    "name": "versionsurl"
  },
  "instance": "xxx",
  "source": "74880ecd28f6:registry:entity:2241"
}
`)

}

// Deleting the current (non-sticky) default Version must recompute the
// default to the next-newest remaining Version.
func TestVerisonCascadeDeferDeleteNonStickyDefault(t *testing.T) {
	reg := NewRegistry("TestVerisonCascadeDeferDeleteNonStickyDefault")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	// v3 is the newest so it's the (non-sticky) default
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "defaultversionsticky": false
}
`)

	// Delete the current default with no ?setdefaultversionid - must
	// fall back to the next-newest remaining Version (v2).
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)
}

// Deleting the current STICKY default Version, with no explicit
// ?setdefaultversionid, must un-stick and recompute the default to the
// newest remaining Version (Resource.SetDefault(nil) path).
func TestVersionCascadeDeferDeleteStickyDefaultUnsticks(t *testing.T) {
	reg := NewRegistry("TestVersionCascadeDeferDeleteStickyDefaultUnsticks")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	// Explicitly stick the default to the oldest Version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"defaultversionid":"v1","defaultversionsticky":true}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "defaultversionsticky": true
}
`)

	// Delete the sticky default with no ?setdefaultversionid - must
	// un-stick and pick the newest remaining Version (v3).
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v1", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "defaultversionsticky": false
}
`)
}

// Deleting the current sticky default Version WITH an explicit
// ?setdefaultversionid must keep the result sticky and pointed at the
// requested Version (Resource.SetDefault(nextVersion) path).
func TestVerisonCascadeDeferDeleteStickyDefaultExplicitNext(t *testing.T) {
	reg := NewRegistry("TestVerisonCascadeDeferDeleteStickyDefaultExplicitNext")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"defaultversionid":"v1","defaultversionsticky":true}`, 200, `*`)

	// Delete the sticky default, explicitly choosing v2 as the new
	// (still sticky) default.
	XHTTP(t, reg, "DELETE",
		"/dirs/d1/files/f1/versions/v1?setdefaultversionid=v2", "",
		204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": true
}
`)
}
