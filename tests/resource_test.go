package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestResourceCreate(t *testing.T) {
	reg := NewRegistry("TestResourceCreate")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true)
	d1, _ := reg.AddGroup("dirs", "d1")

	_, err := d1.AddResource("foos", "f1", "v1")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_resource_type",
  "title": "An unknown Resource type (foos) was specified for Group type \"dirs\".",
  "subject": "/dirs/d1",
  "args": {
    "group": "dirs",
    "name": "foos"
  },
  "source": ":registry:group:120"
}`)

	f1, err := d1.AddResource("files", "f1", "v1")
	XNoErr(t, err)
	XCheck(t, f1 != nil && err == nil, "Creating f1 failed")

	ft, err := d1.AddResource("files", "f1", "v1")
	XCheck(t, ft == nil && err != nil, "Dup f1 should have failed")

	v2, err := f1.AddVersion("v2")
	XNoErr(t, err)
	XCheck(t, v2 != nil && err == nil, "Creating v2 failed")

	vt, err := f1.AddVersion("v2")
	XCheck(t, vt == nil && err != nil, "Dup v2 should have failed")

	vt, isNew, err := f1.UpsertVersion("v2")
	XCheck(t, vt != nil && err == nil, "Update v2 should have worked")
	XCheck(t, isNew == false, "Update v2 should have not been new")
	XCheck(t, v2 == vt, "Should not be a new version")

	d2, err := reg.AddGroup("dirs", "d2")
	XNoErr(t, err)
	XCheck(t, d2 != nil && err == nil, "Creating d2 failed")

	f2, _ := d2.AddResource("files", "f2", "v1")
	f2.AddVersion("v1.1")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1

	// Check basic GET first
	XCheckGet(t, reg, "/dirs/d1/files/f1$details",
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
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx) cannot be found.",
  "subject": "/dirs/d1/files/xxx",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx) cannot be found.",
  "subject": "/dirs/d1/files/xxx",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx/yyy) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: yyy.",
  "subject": "/dirs/d1/files/xxx/yyy",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:info:651"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx/yyy) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: yyy.",
  "subject": "/dirs/d1/files/xxx/yyy",
  "instance": "xxx",
  "source": "e4e59b8a76c4:registry:info:651"
}
`)

	ft, err = d1.FindResource("files", "f1", false, registry.FOR_WRITE)
	XNoErr(t, err)
	XCheck(t, ft != nil && err == nil, "Finding f1 failed")
	ft.AccessMode = f1.AccessMode // little cheat
	XJSONCheck(t, ft, f1)

	ft, err = d1.FindResource("files", "xxx", false, registry.FOR_WRITE)
	XCheck(t, ft == nil && err == nil, "Find files/xxx should have failed")

	ft, err = d1.FindResource("xxx", "xxx", false, registry.FOR_WRITE)
	XCheck(t, ft == nil && err == nil, "Find xxx/xxx should have failed")

	ft, err = d1.FindResource("xxx", "f1", false, registry.FOR_WRITE)
	XCheck(t, ft == nil && err == nil, "Find xxx/f1 should have failed")

	err = f1.Delete()
	XNoErr(t, err)

	ft, err = d1.FindResource("files", "f1", false, registry.FOR_WRITE)
	XCheck(t, err == nil && ft == nil, "Finding delete resource failed")
}

func TestResourceSet(t *testing.T) {
	reg := NewRegistry("TestResourceSet")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "attributes": {
            "ext1": {
              "type": "string"
            },
            "ext2": {
              "type": "integer"
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Make sure fields are empty first
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Delete it to start over
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1", ``, 204, ``)

	//Create + some fields (built-in and custom) on the default Version only -
	// make sure they don't leak onto the Resource/meta entity itself.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "name": "myName",
  "epoch": 68,
  "ext1": "someext",
  "ext2": 123
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "myName",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestorid": "1",
  "ext1": "someext",
  "ext2": 123,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Check Resource and its meta to make sure they didn't pick-up stuff
	// they shouldn't have (like version-only fields)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?doc&inline=meta", ``, 200, `{
  "fileid": "f1",
  "self": "#/",
  "xid": "/dirs/d1/files/f1",

  "metaurl": "#/meta",
  "meta": {
    "fileid": "f1",
    "self": "#/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-07-31T16:32:37.606013676Z",
    "modifiedat": "2026-07-31T16:32:37.606013676Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)
}

func TestResourceRequiredFields(t *testing.T) {
	reg := NewRegistry("TestResourceRequiredFields")
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
	reg.SaveAllAndCommit()

	_, err = group.AddResource("files", "f1", "v1")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f1/versions/v1\" are missing: req.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "list": "req"
  },
  "source": "e4e59b8a76c4:registry:entity:2149"
}`)
	reg.Rollback()
	reg.Refresh(registry.FOR_WRITE)

	f1, err := group.AddResourceWithObject("files", "f1", "v1",
		Object{"req": "test"}, false)
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	f1.Refresh(registry.FOR_WRITE)
	err = f1.SetSaveDefault("req", nil)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1/files/f1/versions/v1\" are missing: req.",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "list": "req"
  },
  "source": "e4e59b8a76c4:registry:entity:2149"
}`)

	err = f1.SetSaveDefault("req", "again")
	XNoErr(t, err)
}

func TestResourceMaxVersions(t *testing.T) {
	reg := NewRegistry("TestResourceMaxVersions")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir"
    }
  }
}`

	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z"
}
`)

	// -1 is not a valid maxversions
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": -1
        }
      }
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"maxversions\"(-1) must be >= 0.",
  "subject": "/model",
  "args": {
    "error_detail": "\"maxversions\"(-1) must be >= 0"
  },
  "instance": "xxx",
  "source": "abc04c6d0dd6:registry:shared_model:2513"
}
`)

	// ONLY ALLOW 1 VERSION, no sticky
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 1,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false,
              "enum": [
                false
              ]
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1"
}
`)

	// Make sure it only has one version and its v1
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta,versions", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2024-01-01T12:00:00.00Z",
    "modifiedat": "2024-01-01T12:00:00.00Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:00.00Z",
      "modifiedat": "2024-01-01T12:00:00.00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 1
}
`)

	// Create v2 and bump v1 out of the list
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "xid": "/dirs/d1/files/f1/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v2"
}
`)

	// Verify everything - just one version, v2
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta,versions", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.02Z",
  "modifiedat": "2024-01-01T12:00:00.02Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2024-01-01T12:00:00.01Z",
    "modifiedat": "2024-01-01T12:00:00.02Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:00.02Z",
      "modifiedat": "2024-01-01T12:00:00.02Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 1
}
`)

	// Set maxversion=2
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 2,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Create v3, but keep v2 as default (sticky)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
  "defaultversionid": "v2",
  "defaultversionsticky": true
}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.01Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": true
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "xid": "/dirs/d1/files/f1/versions/v3",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v2"
}
`)

	// Verison v2 is default,sticky and both v2 and v3 exist
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta,versions", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.01Z",
  "modifiedat": "2024-01-01T12:00:00.01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 4,
    "createdat": "2024-01-01T12:00:00.00Z",
    "modifiedat": "2024-01-01T12:00:00.02Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:00.01Z",
      "modifiedat": "2024-01-01T12:00:00.01Z",
      "ancestorid": "v2"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:00.02Z",
      "modifiedat": "2024-01-01T12:00:00.02Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 2
}
`)

	// Create v4, which should bump v3 out of the list, not v2 (default)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v4", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v4",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4",
  "xid": "/dirs/d1/files/f1/versions/v4",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",
  "ancestorid": "v4"
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta,versions", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.01Z",
  "modifiedat": "2024-01-01T12:00:00.01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 5,
    "createdat": "2024-01-01T12:00:00.00Z",
    "modifiedat": "2024-01-01T12:00:00.02Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2024-01-01T12:00:00.01Z",
      "modifiedat": "2024-01-01T12:00:00.01Z",
      "ancestorid": "v2"
    },
    "v4": {
      "fileid": "f1",
      "versionid": "v4",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4",
      "xid": "/dirs/d1/files/f1/versions/v4",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2024-01-01T12:00:00.02Z",
      "modifiedat": "2024-01-01T12:00:00.02Z",
      "ancestorid": "v4"
    }
  },
  "versionscount": 2
}
`)

	// back to unlimited versions
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 0,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Create v5 and then set it to be default/sticky
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v5", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
  "xid": "/dirs/d1/files/f1/versions/v5",
  "epoch": 1,
  "isdefault": false,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.00Z",
  "ancestorid": "v4"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
  "defaultversionid": "v5",
  "defaultversionsticky": true
}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 7,
  "createdat": "2026-07-31T21:17:00.531802523Z",
  "modifiedat": "2026-07-31T21:17:00.747828766Z",
  "readonly": false,

  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
  "defaultversionsticky": true
}
`)

	// Create a bunch more versions
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v6", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v7", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v8", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v9", `{}`, 201, `*`)

	// Make sure there's 7 of them and v5 is the default
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", ``, 200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:20.971064343Z",
    "modifiedat": "2026-07-31T13:42:20.971064343Z",
    "ancestorid": "v2"
  },
  "v4": {
    "fileid": "f1",
    "versionid": "v4",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4",
    "xid": "/dirs/d1/files/f1/versions/v4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:21.194838059Z",
    "modifiedat": "2026-07-31T13:42:21.194838059Z",
    "ancestorid": "v4"
  },
  "v5": {
    "fileid": "f1",
    "versionid": "v5",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
    "xid": "/dirs/d1/files/f1/versions/v5",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-07-31T13:42:21.303275078Z",
    "modifiedat": "2026-07-31T13:42:21.303275078Z",
    "ancestorid": "v4"
  },
  "v6": {
    "fileid": "f1",
    "versionid": "v6",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v6",
    "xid": "/dirs/d1/files/f1/versions/v6",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:21.379015939Z",
    "modifiedat": "2026-07-31T13:42:21.379015939Z",
    "ancestorid": "v5"
  },
  "v7": {
    "fileid": "f1",
    "versionid": "v7",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v7",
    "xid": "/dirs/d1/files/f1/versions/v7",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:21.428627935Z",
    "modifiedat": "2026-07-31T13:42:21.428627935Z",
    "ancestorid": "v6"
  },
  "v8": {
    "fileid": "f1",
    "versionid": "v8",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v8",
    "xid": "/dirs/d1/files/f1/versions/v8",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:21.48199951Z",
    "modifiedat": "2026-07-31T13:42:21.48199951Z",
    "ancestorid": "v7"
  },
  "v9": {
    "fileid": "f1",
    "versionid": "v9",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
    "xid": "/dirs/d1/files/f1/versions/v9",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-07-31T13:42:21.537276505Z",
    "modifiedat": "2026-07-31T13:42:21.537276505Z",
    "ancestorid": "v8"
  }
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", ``, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 11,
  "createdat": "2026-07-31T13:42:24.626557604Z",
  "modifiedat": "2026-07-31T13:42:25.183426687Z",
  "readonly": false,

  "defaultversionid": "v5",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
  "defaultversionsticky": true
}
`)

	// Trying to set maxversions=1 now should fail since f1 is still
	// pinned sticky to v5
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 1,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionsticky_false",
  "title": "For \"/dirs/d1/files/f1/meta\", setting \"defaultversionsticky\" to \"true\" is not allowed since \"maxversions\" is \"1\".",
  "subject": "/dirs/d1/files/f1/meta",
  "instance": "xxx",
  "source": "4a51b174cf4e:registry:entity:2244"
}
`)

	// Now clear the sticky flag,notice default is now v9
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
  "defaultversionid": null
}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 12,
  "createdat": "2024-01-01T12:00:00.00Z",
  "modifiedat": "2024-01-01T12:00:00.01Z",
  "readonly": false,

  "defaultversionid": "v9",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
  "defaultversionsticky": false
}
`)

	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 1,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false,
              "enum": [
                false
              ]
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// After clearing sticky and setting maxversions=1, the newest
	// version (v9) survives - not v5 (v5 was only sticky, not newest)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", ``, 200, `{
  "v9": {
    "fileid": "f1",
    "versionid": "v9",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
    "xid": "/dirs/d1/files/f1/versions/v9",
    "epoch": 2,
    "isdefault": true,
    "createdat": "2024-01-01T12:00:00.00Z",
    "modifiedat": "2024-01-01T12:00:00.01Z",
    "ancestorid": "v9"
  }
}
`)

	// Back to maxversions=2
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 2,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Make v9 sticky, and the PUT should bump its epoch
	XHTTP(t, reg, "PUT",
		"/dirs/d1/files/f1$details?inline=meta&setdefaultversionid=v9",
		`{}`, 200, `{
  "fileid": "f1",
  "versionid": "v9",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "2024-01-01T12:00:00.01Z",
  "modifiedat": "2024-01-01T12:00:00.02Z",
  "ancestorid": "v9",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 14,
    "createdat": "2024-01-01T12:00:00.00Z",
    "modifiedat": "2024-01-01T12:00:00.02Z",
    "readonly": false,

    "defaultversionid": "v9",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v9",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Now set maxversions and we should get an error because sticky is set
	// but it's not allowed to be when maxversions=1
	model = `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 1,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionsticky_false",
  "title": "For \"/dirs/d1/files/f1/meta\", setting \"defaultversionsticky\" to \"true\" is not allowed since \"maxversions\" is \"1\".",
  "subject": "/dirs/d1/files/f1/meta",
  "instance": "xxx",
  "source": "4a51b174cf4e:registry:resource:1896"
}
`)
}

func TestResourceDeprecated(t *testing.T) {
	reg := NewRegistry("TestResourceDeprecated")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": false,
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {}
    }  `, 201, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "2025-06-12T15:43:53.756277894Z",
  "modifiedat": "2025-06-12T15:43:53.756277894Z",
  "readonly": false,
  "deprecated": {},

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	// All sub-fields, with removal >= effective (valid)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12:00:00+07:00",
        "removal": "2124-01-01T12:01:00Z",
        "alternative": "some-url",
        "documentation": "another-url",
        "dep_zzz": "zzz",
        "dep_aaa": "foo"
      }
    }  `, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-06-12T15:43:53.1Z",
  "modifiedat": "2025-06-12T15:43:53.2Z",
  "readonly": false,
  "deprecated": {
    "effective": "2123-01-01T05:00:00Z",
    "removal": "2124-01-01T12:01:00Z",
    "alternative": "some-url",
    "documentation": "another-url",
    "dep_aaa": "foo",
    "dep_zzz": "zzz"
  },

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	// removal before effective must be rejected (spec MUST NOT)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12:00:00Z",
        "removal": "2000-01-01T12:00:00Z"
      }
    }  `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"deprecated.removal\" for \"/dirs/d1/files/f1/meta\" is not valid: must not be sooner than deprecated.effective.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "must not be sooner than deprecated.effective",
    "name": "deprecated.removal"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	// removal equal to effective is valid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12:00:00Z",
        "removal": "2123-01-01T12:00:00Z"
      }
    }  `, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:01Z",
  "readonly": false,
  "deprecated": {
    "effective": "2123-01-01T12:00:00Z",
    "removal": "2123-01-01T12:00:00Z"
  },

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	// malformed effective timestamp
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12"
      }
    }  `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"deprecated.effective\" for \"/dirs/d1/files/f1/meta\" is not valid: is a malformed timestamp.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "is a malformed timestamp",
    "name": "deprecated.effective"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	// malformed removal timestamp
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12:00:00",
        "removal": "2123-01-01T12"
      }
    }  `, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"deprecated.removal\" for \"/dirs/d1/files/f1/meta\" is not valid: is a malformed timestamp.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "error_detail": "is a malformed timestamp",
    "name": "deprecated.removal"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	// set deprecated, verify it appears on direct resource read
	// (via $details?inline=meta)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": { "effective": "2099-06-01T00:00:00Z" }
    }`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 4,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:01Z",
  "readonly": false,
  "deprecated": {
    "effective": "2099-06-01T00:00:00Z"
  },

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details?inline=meta", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T00:00:00Z",
  "modifiedat": "2025-01-01T00:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 4,
    "createdat": "2025-01-01T00:00:00Z",
    "modifiedat": "2025-01-01T00:00:01Z",
    "readonly": false,
    "deprecated": {
      "effective": "2099-06-01T00:00:00Z"
    },

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// clear deprecated by setting it to null
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": null
    }`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 5,
  "createdat": "2025-06-12T15:43:53.1Z",
  "modifiedat": "2025-06-12T15:43:53.2Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)
}

func TestResourceSamples(t *testing.T) {
	reg := NewRegistry("TestResourceSamples")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "cREATEdAt",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Create single Resource with empty content - PUT
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=meta", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T20:39:34.121124603Z",
  "modifiedat": "2026-02-04T20:39:34.121124603Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:39:34.121124603Z",
    "modifiedat": "2026-02-04T20:39:34.121124603Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create single Resource with empty content - PATCH
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=meta", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T20:39:34.121124603Z",
  "modifiedat": "2026-02-04T20:39:34.121124603Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:39:34.121124603Z",
    "modifiedat": "2026-02-04T20:39:34.121124603Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource via the "files" collection
	XHTTP(t, reg, "POST", "/dirs/d1/files?inline=meta", `{
  "f1": {
    "name": "my file"
  }
}
`, 200, `{
  "f1": {
    "fileid": "f1",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f1",
    "xid": "/dirs/d1/files/f1",
    "epoch": 1,
    "name": "my file",
    "isdefault": true,
    "createdat": "2026-02-04T20:45:06.826527109Z",
    "modifiedat": "2026-02-04T20:45:06.826527109Z",
    "ancestorid": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2026-02-04T20:45:06.826527109Z",
      "modifiedat": "2026-02-04T20:45:06.826527109Z",
      "readonly": false,

      "defaultversionid": "1",
      "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "defaultversionsticky": false
    },
    "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
    "versionscount": 1
  }
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions, no defaultversionid - PUT
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "versions": {
    "v1": {},
    "v2": {}
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T20:54:11.336910391Z",
  "modifiedat": "2026-02-04T20:54:11.336910391Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions, no defaultversionid - PATCH
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "versions": {
    "v1": {},
    "v2": {}
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T20:54:11.336910391Z",
  "modifiedat": "2026-02-04T20:54:11.336910391Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions and unique defaultversionid - PUT
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionid": "v1"
  },
  "versions": {
    "v2": {},
    "v3": {}
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T20:54:11.336910391Z",
  "modifiedat": "2026-02-04T20:54:11.336910391Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "foo",
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions and unique defaultversionid - PATCH
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionid": "v1"
  },
  "versions": {
    "v2": {},
    "v3": {}
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "foo",
  "isdefault": true,
  "createdat": "2026-02-04T20:54:11.336910391Z",
  "modifiedat": "2026-02-04T20:54:11.336910391Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "foo",
      "isdefault": true,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v1"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:54:11.336910391Z",
      "modifiedat": "2026-02-04T20:54:11.336910391Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions and defaultversionid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionid": "v1"
  },
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {
      "createdat": "3030-01-01T12:00:00"
    },
    "v3": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "3030-01-01T12:00:00Z",
  "modifiedat": "2026-02-04T20:59:37.791374128Z",
  "ancestorid": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:59:37.791374128Z",
    "modifiedat": "2026-02-04T20:59:37.791374128Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-04T20:59:37.791374128Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "3030-01-01T12:00:00Z",
      "modifiedat": "2026-02-04T20:59:37.791374128Z",
      "ancestorid": "v3"
    },
    "v3": {
      "fileid": "f1",
      "versionid": "v3",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
      "xid": "/dirs/d1/files/f1/versions/v3",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-04T20:59:37.791374128Z",
      "modifiedat": "2026-02-04T20:59:37.791374128Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with defaultversionid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionid": "v1"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "foo",
  "isdefault": true,
  "createdat": "2026-02-04T21:04:14.240631033Z",
  "modifiedat": "2026-02-04T21:04:14.240631033Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T21:04:14.240631033Z",
    "modifiedat": "2026-02-04T21:04:14.240631033Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "foo",
      "isdefault": true,
      "createdat": "2026-02-04T21:04:14.240631033Z",
      "modifiedat": "2026-02-04T21:04:14.240631033Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with versionid and Versions
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v0",
  "name": "foo",
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-04T21:06:12.940653748Z",
  "modifiedat": "2026-02-04T21:06:12.940653748Z",
  "ancestorid": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T21:06:12.940653748Z",
    "modifiedat": "2026-02-04T21:06:12.940653748Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v0": {
      "fileid": "f1",
      "versionid": "v0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
      "xid": "/dirs/d1/files/f1/versions/v0",
      "epoch": 1,
      "name": "foo",
      "isdefault": false,
      "createdat": "2026-02-04T21:06:12.940653748Z",
      "modifiedat": "2026-02-04T21:06:12.940653748Z",
      "ancestorid": "v1"
    },
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-04T21:06:12.940653748Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-04T21:06:12.940653748Z",
      "modifiedat": "2026-02-04T21:06:12.940653748Z",
      "ancestorid": "v0"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with new Versions and sticky default Version
	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v0",
  "createdat": "2021-01-01T12:00:00",
  "modifiedat": "2021-01-01T12:00:00",
  "meta": {
    "createdat": "2021-01-01T12:00:00",
    "modifiedat": "2021-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2021-01-01T12:00:00Z",
  "modifiedat": "2021-01-01T12:00:00Z",
  "ancestorid": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2021-01-01T12:00:00Z",
    "modifiedat": "2021-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v0",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v0": {
      "fileid": "f1",
      "versionid": "v0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
      "xid": "/dirs/d1/files/f1/versions/v0",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2021-01-01T12:00:00Z",
      "modifiedat": "2021-01-01T12:00:00Z",
      "ancestorid": "v0"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	// Note that v0 will be updated with Resource level attributes because
	// v0 is the current default version, we don't use meta.defaultvid to
	// determine what the current default version is when the Resource already
	// exists
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "createdat": "2021-01-01T12:00:00",
  "meta": {
    "defaultversionid": "v1",
    "defaultversionsticky": true
  },
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {}
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2020-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T12:15:38.555135964Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2021-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v0": {
      "fileid": "f1",
      "versionid": "v0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
      "xid": "/dirs/d1/files/f1/versions/v0",
      "epoch": 2,
      "name": "foo",
      "isdefault": false,
      "createdat": "2021-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v0"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with Versions and sticky default Version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v0",
  "name": "foo",
  "meta": {
    "defaultversionid": "v1",
    "defaultversionsticky": true
  },
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2020-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T12:15:38.555135964Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v0": {
      "fileid": "f1",
      "versionid": "v0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
      "xid": "/dirs/d1/files/f1/versions/v0",
      "epoch": 1,
      "name": "foo",
      "isdefault": false,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v0"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with versionid and defaultversionid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v0",
  "name": "foo",
  "meta": {
    "defaultversionid": "v1"
  },
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-11T12:15:38.555135964Z",
  "modifiedat": "2026-02-11T12:15:38.555135964Z",
  "ancestorid": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v0": {
      "fileid": "f1",
      "versionid": "v0",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
      "xid": "/dirs/d1/files/f1/versions/v0",
      "epoch": 1,
      "name": "foo",
      "isdefault": false,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v0"
    }
  },
  "versionscount": 3
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with sticky defaultversionid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "v1",
    "defaultversionsticky": true
  },
  "versions": {
    "v1": {
      "createdat": "2020-01-01T12:00:00"
    },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2020-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T12:15:38.555135964Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-11T12:15:38.555135964Z",
      "modifiedat": "2026-02-11T12:15:38.555135964Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with non-sticky bad defaultversionid

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v1",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00",
    "defaultversionid": "v1",
    "defaultversionsticky": true
  },
  "versions": {
    "v2": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionid": "abc"
  },
  "versions": {
    "v2": {
      "createdat": "2020-01-01T12:00:00"
    }
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "foo",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T12:54:32.492627638Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T12:54:32.492627638Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "name": "foo",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:54:32.492627638Z",
      "ancestorid": "v2"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T12:54:32.492627638Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with sticky non-specified defaultversionid

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v2",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "v1": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    },
    "v2": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionsticky": true
  },
  "versions": {
    "v2": {
      "createdat": "2020-01-01T12:00:00"
    }
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:45:16.34043263Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:45:16.34043263Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:45:16.34043263Z",
      "ancestorid": "v2"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:45:16.34043263Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource with Versions and defaultversionsticky

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v2",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "v1": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    },
    "v2": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "meta": {
    "defaultversionsticky": true
  },
  "versions": {
    "v2": {
      "createdat": "2020-01-01T12:00:00"
    }
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2020-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:52:51.187504012Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:52:51.187504012Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:52:51.187504012Z",
      "ancestorid": "v2"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2020-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:52:51.187504012Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with empty content

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource with empty content

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with new description

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "description": "very cool"
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "description": "very cool",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "isdefault": true,
      "description": "very cool",
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource's description field

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "description": "very cool"
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "my file",
  "isdefault": true,
  "description": "very cool",
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "name": "my file",
      "isdefault": true,
      "description": "very cool",
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with non-specified defaultversionsticky

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionsticky": true
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:57:59.428348579Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource with defaultversionsticky

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "name": "my file",
  "createdat": "2025-01-01T12:00:00",
  "modifiedat": "2025-01-01T12:00:00",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionsticky": true
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T15:57:59.428348579Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:57:59.428348579Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T15:57:59.428348579Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource with sticky defaultversionid

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v2",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "v1": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    },
    "v2": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/meta?inline=*", `{
  "defaultversionid": "v1",
  "defaultversionsticky": true
}
`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T16:45:28.200011152Z",
  "readonly": false,

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "defaultversionsticky": true
}
`)

	// Check all data
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=*", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T16:46:02.002688282Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Patch Resource with bad defaultversionid

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "1": {
      "name": "my file",
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "foo"
  }
}
`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1/meta\", the \"version\" with a \"versionid\" value of \"foo\" cannot be found.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "id": "foo",
    "singular": "version"
  },
  "instance": "xxx",
  "source": "396100315a6e:registry:entity:1442"
}
`)

	// Bonus points - make sure PUT would have worked though
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "foo"
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T17:02:26.753415624Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T17:02:26.753415624Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T17:02:26.753415624Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with bad sticky defaultversionid

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "1",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "1": {
      "name": "my file",
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "my file",
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 1,
      "name": "my file",
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "foo",
    "defaultversionsticky": true
  }
}
`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1/meta\", the \"version\" with a \"versionid\" value of \"foo\" cannot be found.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "id": "foo",
    "singular": "version"
  },
  "instance": "xxx",
  "source": "396100315a6e:registry:entity:1442"
}
`)

	// Bonus - make sure sticky=false would have worked
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "foo",
    "defaultversionsticky": false
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T17:06:48.948313868Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T17:06:48.948313868Z",
    "readonly": false,

    "defaultversionid": "1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "1": {
      "fileid": "f1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/1",
      "xid": "/dirs/d1/files/f1/versions/1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T17:06:48.948313868Z",
      "ancestorid": "1"
    }
  },
  "versionscount": 1
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Update Resource with non-specified sticky default Version

	// First the set-up
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v1",
  "meta": {
    "createdat": "2025-01-01T12:00:00",
    "modifiedat": "2025-01-01T12:00:00"
  },
  "versions": {
    "v1": {
      "createdat": "2025-01-01T12:00:00",
      "modifiedat": "2025-01-01T12:00:00"
    }
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:00Z",
  "modifiedat": "2025-01-01T12:00:00Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2025-01-01T12:00:00Z",
      "modifiedat": "2025-01-01T12:00:00Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 1
}
`)

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "name": "foo",
  "createdat": "1999-01-01T12:00:00",
  "meta": {
    "defaultversionsticky": true
  },
  "versions": {
    "v2": {
      "createdat": "1998-01-01T12:00:00"
    }
  }
}
`, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "name": "foo",
  "isdefault": true,
  "createdat": "1999-01-01T12:00:00Z",
  "modifiedat": "2026-02-11T18:07:21.627075803Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T18:07:21.627075803Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 2,
      "name": "foo",
      "isdefault": true,
      "createdat": "1999-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T18:07:21.627075803Z",
      "ancestorid": "v2"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "1998-01-01T12:00:00Z",
      "modifiedat": "2026-02-11T18:07:21.627075803Z",
      "ancestorid": "v2"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with conflicting default Version attributes - variant 1

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v1",
  "name": "foo",
  "meta": {
    "defaultversionsticky": true
  },
  "versions": {
    "v1": { "name": "abc" },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-11T18:11:11.509170139Z",
  "modifiedat": "2026-02-11T18:11:11.509170139Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:11:11.509170139Z",
    "modifiedat": "2026-02-11T18:11:11.509170139Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "abc",
      "isdefault": false,
      "createdat": "2026-02-11T18:11:11.509170139Z",
      "modifiedat": "2026-02-11T18:11:11.509170139Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-11T18:11:11.509170139Z",
      "modifiedat": "2026-02-11T18:11:11.509170139Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with conflicting default Version attributes - variant 2

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "meta": {
    "defaultversionid": "v1"
  },
  "versions": {
    "v1": { "name": "abc" },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-11T18:13:29.243111729Z",
  "modifiedat": "2026-02-11T18:13:29.243111729Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:13:29.243111729Z",
    "modifiedat": "2026-02-11T18:13:29.243111729Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "abc",
      "isdefault": false,
      "createdat": "2026-02-11T18:13:29.243111729Z",
      "modifiedat": "2026-02-11T18:13:29.243111729Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-11T18:13:29.243111729Z",
      "modifiedat": "2026-02-11T18:13:29.243111729Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with conflicting default Version attributes - variant 3

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=*", `{
  "versionid": "v1",
  "versions": {
    "v1": { "name": "abc" },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-02-11T18:15:05.715666555Z",
  "modifiedat": "2026-02-11T18:15:05.715666555Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:15:05.715666555Z",
    "modifiedat": "2026-02-11T18:15:05.715666555Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "abc",
      "isdefault": false,
      "createdat": "2026-02-11T18:15:05.715666555Z",
      "modifiedat": "2026-02-11T18:15:05.715666555Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-02-11T18:15:05.715666555Z",
      "modifiedat": "2026-02-11T18:15:05.715666555Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with SetDefaultVersionID flag

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?setdefaultversionid=v1&inline=*", `{
  "versions": {
    "v1": { "name": "abc" },
    "v2": {}
  }
}
`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "abc",
  "isdefault": true,
  "createdat": "2026-02-11T18:15:05.715666555Z",
  "modifiedat": "2026-02-11T18:15:05.715666555Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:15:05.715666555Z",
    "modifiedat": "2026-02-11T18:15:05.715666555Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "abc",
      "isdefault": true,
      "createdat": "2026-02-11T18:15:05.715666555Z",
      "modifiedat": "2026-02-11T18:15:05.715666555Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-11T18:15:05.715666555Z",
      "modifiedat": "2026-02-11T18:15:05.715666555Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

	// Clean-up
	XHTTP(t, reg, "DELETE", "/dirs/d1", "", 204, ``)

	// Create Resource with SetDefaultVersionID flag via /versions

	// Now the real test
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions?setdefaultversionid=v1&inline=*", `{
  "v1": { "name": "abc" },
  "v2": {}
}
`, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "name": "abc",
    "isdefault": true,
    "createdat": "2026-02-12T14:10:25.20755952Z",
    "modifiedat": "2026-02-12T14:10:25.20755952Z",
    "ancestorid": "v1"
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "2026-02-12T14:10:25.20755952Z",
    "modifiedat": "2026-02-12T14:10:25.20755952Z",
    "ancestorid": "v1"
  }
}
`)

	// Verify full resource
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=*", ``, 200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "name": "abc",
  "isdefault": true,
  "createdat": "2026-02-12T14:11:41.527893268Z",
  "modifiedat": "2026-02-12T14:11:41.527893268Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-12T14:11:41.527893268Z",
    "modifiedat": "2026-02-12T14:11:41.527893268Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versions": {
    "v1": {
      "fileid": "f1",
      "versionid": "v1",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
      "xid": "/dirs/d1/files/f1/versions/v1",
      "epoch": 1,
      "name": "abc",
      "isdefault": true,
      "createdat": "2026-02-12T14:11:41.527893268Z",
      "modifiedat": "2026-02-12T14:11:41.527893268Z",
      "ancestorid": "v1"
    },
    "v2": {
      "fileid": "f1",
      "versionid": "v2",
      "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
      "xid": "/dirs/d1/files/f1/versions/v2",
      "epoch": 1,
      "isdefault": false,
      "createdat": "2026-02-12T14:11:41.527893268Z",
      "modifiedat": "2026-02-12T14:11:41.527893268Z",
      "ancestorid": "v1"
    }
  },
  "versionscount": 2
}
`)

}

// More tests that are kind of related to the previous func's
func TestResourceFlow(t *testing.T) {
	reg := NewRegistry("TestResourceFlow")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "createdat",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// NOT PART OF THE resource.md doc

	// Invalid meta.defaultversionid

	// Now the real test
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "name": "foo",
  "meta": {
    "defaultversionid": "v2",
    "defaultversionsticky": true
  },
  "versions": {
    "v3": {}
  }
}
`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1/meta\", the \"version\" with a \"versionid\" value of \"v2\" cannot be found.",
  "subject": "/dirs/d1/files/f1/meta",
  "args": {
    "id": "v2",
    "singular": "version"
  },
  "instance": "xxx",
  "source": "55cdbec617b8:registry:entity:1451"
}
`)

}
