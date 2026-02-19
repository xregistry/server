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
	gm.AddResourceModel("files", "file", 0, true, true, true)
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx) cannot be found.",
  "subject": "/dirs/d1/files/xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx) cannot be found.",
  "subject": "/dirs/d1/files/xxx",
  "source": "e4e59b8a76c4:registry:httpStuff:1395"
}
`)
	XCheckGet(t, reg, "/dirs/d1/files/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx/yyy) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: yyy.",
  "subject": "/dirs/d1/files/xxx/yyy",
  "source": "e4e59b8a76c4:registry:info:651"
}
`)
	XCheckGet(t, reg, "dirs/d1/files/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/d1/files/xxx/yyy) cannot be found.",
  "detail": "Expected \"versions\" or \"meta\", got: yyy.",
  "subject": "/dirs/d1/files/xxx/yyy",
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

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
	rm.AddAttr("ext1", STRING)
	rm.AddAttr("ext2", INTEGER)

	d1, _ := reg.AddGroup("dirs", "d1")
	f1, _ := d1.AddResource("files", "f1", "v1")
	XNoErr(t, reg.SaveModel())

	// /dirs/d1/f1/v1

	XNoErr(t, f1.SetSaveDefault("name", "myName"))
	XNoErr(t, f1.SetSaveDefault("epoch", 68))
	XNoErr(t, f1.SetSaveDefault("ext1", "someext"))
	XNoErr(t, f1.SetSaveDefault("ext2", 123))

	// Make sure the props on the resource weren't set
	XCheck(t, f1.Entity.Get("name") == nil, "name should be nil")
	XCheck(t, f1.Entity.Get("epoch") == nil, "epoch should be nil")
	XCheck(t, f1.Entity.Get("ext1") == nil, "ext1 should be nil")
	XCheck(t, f1.Entity.Get("ext2") == nil, "ext2 should be nil")

	ft, _ := d1.FindResource("files", "f1", false, registry.FOR_WRITE)

	XJSONCheck(t, ft, f1)

	// Make sure the version was set
	vt, _ := ft.GetDefault(registry.FOR_WRITE)
	XEqual(t, "", vt.Get("name"), "myName")
	XEqual(t, "", vt.Get("epoch"), 68)
	XEqual(t, "", vt.Get("ext1"), "someext")
	XEqual(t, "", vt.Get("ext2"), 123)
}

func TestResourceRequiredFields(t *testing.T) {
	reg := NewRegistry("TestResourceRequiredFields")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, true)
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

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	d1, _ := reg.AddGroup("dirs", "d1")
	XNoErr(t, reg.SaveModel())

	_, err = gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:      "files",
		Singular:    "file",
		MaxVersions: PtrInt(-1),
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"maxversions\"(-1) must be >= 0.",
  "subject": "/model",
  "args": {
    "error_detail": "\"maxversions\"(-1) must be >= 0"
  },
  "source": "e4e59b8a76c4:registry:shared_model:1010"
}`)
	// reg.LoadModel()

	// gm = reg.Model.FindGroupModel(gm.Plural)
	rm, err := gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:      "files",
		Singular:    "file",
		MaxVersions: PtrInt(1), // ONLY ALLOW 1 VERSION
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#setdefaultversionsticky_false",
  "title": "The model attribute \"setdefaultversionsticky\" needs to be \"false\" since \"maxversions\" is \"1\".",
  "subject": "/model",
  "source": "e4e59b8a76c4:registry:shared_model:1017"
}`)
	// reg.LoadModel()

	// gm = reg.Model.FindGroupModel(gm.Plural)
	rm, err = gm.AddResourceModelFull(&registry.ResourceModel{
		Plural:           "files",
		Singular:         "file",
		MaxVersions:      PtrInt(1), // ONLY ALLOW 1 VERSION
		SetDefaultSticky: PtrBool(false),
	})
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	f1, err := d1.AddResource("files", "f1", "v1")
	XCheck(t, f1 != nil && err == nil, "Creating f1 failed: %s", err)
	vers, err := f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 1, "Should be just one version")

	defaultV, err := f1.GetDefault(registry.FOR_WRITE)
	XCheck(t, defaultV != nil && err == nil && defaultV.UID == "v1",
		"err: %q default: %s", err, ToJSON(defaultV))

	// Create v2 and bump v1 out of the list
	v2, err := f1.AddVersion("v2")
	XCheck(t, v2 != nil && err == nil, "Creating v2 failed: %s", err)
	defaultV, err = f1.GetDefault(registry.FOR_WRITE)
	XCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q default: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 1 && vers[0].Object["versionid"] == "v2", "Should be v2")

	err = rm.SetMaxVersions(2)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	// Create v3, but keep v2 as default
	XNoErr(t, f1.SetDefault(v2))
	v3, err := f1.AddVersion("v3")
	XCheck(t, v3 != nil && err == nil, "Creating v3 failed: %s", err)
	defaultV, err = f1.GetDefault(registry.FOR_WRITE)
	XCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q defaultV: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 2, "Should be 2")
	XCheck(t, vers[0].Object["versionid"] == "v2", "0=v2")
	XCheck(t, vers[1].Object["versionid"] == "v3", "1=v3")

	// Create v4, which should bump v3 out of the list, not v2 (default)
	v4, err := f1.AddVersion("v4")
	XCheck(t, v4 != nil && err == nil, "Creating v4 failed: %s", err)
	defaultV, err = f1.GetDefault(registry.FOR_WRITE)
	XCheck(t, defaultV != nil && err == nil && defaultV.UID == "v2",
		"err: %q defaultV: %s", err, ToJSON(defaultV))
	vers, err = f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 2, "Should be 2, but is: %d", len(vers))
	XCheck(t, len(vers) == 2, "Should be 2, but is: %s", ToJSON(vers))
	XCheck(t, vers[0].Object["versionid"] == "v2", "0=v2")
	XCheck(t, vers[1].Object["versionid"] == "v4", "1=v4")

	err = rm.SetMaxVersions(0)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	v5, err := f1.AddVersion("v5")
	XNoErr(t, err)
	XNoErr(t, f1.SetDefault(v5))
	_, err = f1.AddVersion("v6")
	XNoErr(t, err)
	_, err = f1.AddVersion("v7")
	XNoErr(t, err)
	_, err = f1.AddVersion("v8")
	XNoErr(t, err)
	_, err = f1.AddVersion("v9")
	XNoErr(t, err)
	vers, err = f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 7, "Should be 7, but is: %d", len(vers))
	XCheck(t, len(vers) == 7, "Should be 7, but is: %s", ToJSON(vers))
	defaultV, err = f1.GetDefault(registry.FOR_WRITE)
	XCheck(t, defaultV != nil && err == nil && defaultV.UID == "v5",
		"err: %q defaultV: %s", err, ToJSON(defaultV))

	// Now set maxVer to 1 and just v5 should remain
	err = rm.SetMaxVersions(1)
	XNoErr(t, err)
	XNoErr(t, reg.SaveModel())

	vers, err = f1.GetVersions()
	XNoErr(t, err)
	XCheck(t, len(vers) == 1, "Should be 1, but is: %d", len(vers))
	XCheck(t, len(vers) == 1, "Should be 1, but is: %s", ToJSON(vers))
	XCheck(t, vers[0].Object["versionid"] == "v5", "0=v5")
}

func TestResourceDeprecated(t *testing.T) {
	reg := NewRegistry("TestResourceDeprecated")
	defer PassDeleteReg(t, reg)

	gm, err := reg.Model.AddGroupModel("dirs", "dir")
	XNoErr(t, err)
	_, err = gm.AddResourceModelSimple("files", "file")

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
  "compatibility": "none",
  "deprecated": {},

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/1$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta", `{
      "deprecated": {
        "effective": "2123-01-01T12:00:00+07:00",
        "removal": "2000-01-01T12:01:00+07",
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
  "compatibility": "none",
  "deprecated": {
    "effective": "2123-01-01T12:00:00.3Z",
    "removal": "2000-01-01T12:01:00.4Z",
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
  "source": "e4e59b8a76c4:registry:entity:2561"
}
`)

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
  "source": "e4e59b8a76c4:registry:entity:2561"
}
`)
}

func TestResourceSamples(t *testing.T) {
	reg := NewRegistry("TestResourceSamples")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false) //hasdoc=false
	rm.SetVersionMode("createdat")

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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:39:34.121124603Z",
    "modifiedat": "2026-02-04T20:39:34.121124603Z",
    "readonly": false,
    "compatibility": "none",

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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:39:34.121124603Z",
    "modifiedat": "2026-02-04T20:39:34.121124603Z",
    "readonly": false,
    "compatibility": "none",

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
    "ancestor": "1",

    "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
    "meta": {
      "fileid": "f1",
      "self": "http://localhost:8181/dirs/d1/files/f1/meta",
      "xid": "/dirs/d1/files/f1/meta",
      "epoch": 1,
      "createdat": "2026-02-04T20:45:06.826527109Z",
      "modifiedat": "2026-02-04T20:45:06.826527109Z",
      "readonly": false,
      "compatibility": "none",

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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v2"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:54:11.336910391Z",
    "modifiedat": "2026-02-04T20:54:11.336910391Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v2"
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
  "ancestor": "v3",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T20:59:37.791374128Z",
    "modifiedat": "2026-02-04T20:59:37.791374128Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v3"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T21:04:14.240631033Z",
    "modifiedat": "2026-02-04T21:04:14.240631033Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
  "ancestor": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-04T21:06:12.940653748Z",
    "modifiedat": "2026-02-04T21:06:12.940653748Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v0"
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
  "ancestor": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2021-01-01T12:00:00Z",
    "modifiedat": "2021-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v0"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2021-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v0"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v0"
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
  "ancestor": "v0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
      "ancestor": "v0"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T12:15:38.555135964Z",
    "modifiedat": "2026-02-11T12:15:38.555135964Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T12:54:32.492627638Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v2"
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
      "ancestor": "v2"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:45:16.34043263Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v2"
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
      "ancestor": "v2"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:52:51.187504012Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v2"
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
      "ancestor": "v2"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:57:59.428348579Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T15:57:59.428348579Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "compatibility": "none",

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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T16:46:02.002688282Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T17:02:26.753415624Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T17:06:48.948313868Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2025-01-01T12:00:00Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
  "ancestor": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "2025-01-01T12:00:00Z",
    "modifiedat": "2026-02-11T18:07:21.627075803Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v2"
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
      "ancestor": "v2"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:11:11.509170139Z",
    "modifiedat": "2026-02-11T18:11:11.509170139Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:13:29.243111729Z",
    "modifiedat": "2026-02-11T18:13:29.243111729Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:15:05.715666555Z",
    "modifiedat": "2026-02-11T18:15:05.715666555Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-11T18:15:05.715666555Z",
    "modifiedat": "2026-02-11T18:15:05.715666555Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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
    "ancestor": "v1"
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
    "ancestor": "v1"
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
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "2026-02-12T14:11:41.527893268Z",
    "modifiedat": "2026-02-12T14:11:41.527893268Z",
    "readonly": false,
    "compatibility": "none",

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
      "ancestor": "v1"
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
      "ancestor": "v1"
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

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true, false) //hasdoc=false
	rm.SetVersionMode("createdat")

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
  "source": "55cdbec617b8:registry:entity:1451"
}
`)

}
