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
