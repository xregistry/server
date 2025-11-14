package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestSetAttributeNames(t *testing.T) {
	reg := NewRegistry("TestSetAttributeName")
	defer PassDeleteReg(t, reg)

	type test struct {
		name string
		msg  string
	}

	sixty := "a23456789012345678901234567890123456789012345678901234567890"

	tests := []test{
		{sixty + "12", ""},
		{sixty + "123", ""},
		{"_123", ""},
		{"_12_3", ""},
		{"_123_", ""},
		{"_123_", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: attribute \"_123_\" already exists"
}`},
		{"_", ""},
		{"__", ""},
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"\" is not valid: attribute name \"\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{sixty + "1234", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"a234567890123456789012345678901234567890123456789012345678901234\" is not valid: attribute name \"a234567890123456789012345678901234567890123456789012345678901234\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"1234", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"1234\" is not valid: attribute name \"1234\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"A\" is not valid: attribute name \"A\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"aA", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"aA\" is not valid: attribute name \"aA\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"_A", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"_A\" is not valid: attribute name \"_A\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"_ _", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"_ _\" is not valid: attribute name \"_ _\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
		{"#abc", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/",
  "title": "The attribute \"#abc\" is not valid: attribute name \"#abc\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}`},
	}

	for _, test := range tests {
		t.Logf("test: %q", test.name)
		_, xErr := reg.Model.AddAttr(test.name, STRING)

		if test.msg == "" && xErr != nil {
			t.Fatalf("Name: %q failed: %s", test.name, xErr)
		}
		if test.msg != "" && (xErr == nil || xErr.String() != test.msg) {
			XCheckErr(t, xErr, test.msg)
		}

	}
	XNoErr(t, reg.SaveModel())
}

func TestSetResource(t *testing.T) {
	reg := NewRegistry("TestSetResource")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	XNoErr(t, reg.SaveModel())

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Default and res
	namePP := NewPP().P("name").UI()
	file.SetSaveDefault(namePP, "myName")
	ver, _ := file.FindVersion("v1", false, registry.FOR_WRITE)
	val := ver.Get(namePP)
	if val != "myName" {
		t.Errorf("ver.Name is %q, should be 'myName'", val)
	}

	name := file.Get(namePP).(string)
	XEqual(t, "", name, "myName")

	// Verify that nil and "" are treated differently
	ver.SetSave(namePP, nil)
	ver2, _ := file.FindVersion(ver.UID, false, registry.FOR_WRITE)
	XJSONCheck(t, ver2, ver)
	val = ver.Get(namePP)
	XCheck(t, val == nil, "Setting to nil should return nil")

	ver.SetSave(namePP, "")
	ver2, _ = file.FindVersion(ver.UID, false, registry.FOR_WRITE)
	XJSONCheck(t, ver2, ver)
	val = ver.Get(namePP)
	XCheck(t, val == "", "Setting to '' should return ''")
}

func TestSetVersion(t *testing.T) {
	reg := NewRegistry("TestSetVersion")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.SaveModel()

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1", false, registry.FOR_WRITE)

	// /dirs/d1/f1/v1

	// Make sure setting it on the version is seen by res.Default and res
	namePP := NewPP().P("name").UI()
	ver.SetSave(namePP, "myName")
	file, _ = dir.FindResource("files", "f1", false, registry.FOR_WRITE)
	l, xErr := file.GetDefault(registry.FOR_WRITE)
	XNoErr(t, xErr)
	XCheck(t, l != nil, "default is nil")
	val := l.Get(namePP)
	if val != "myName" {
		t.Errorf("resource.default.Name is %q, should be 'myName'", val)
	}
	val = file.Get(namePP)
	if val != "myName" {
		t.Errorf("resource.Name is %q, should be 'myName'", val)
	}

	// Make sure we can also still see it from the version itself
	ver, _ = file.FindVersion("v1", false, registry.FOR_WRITE)
	val = ver.Get(namePP)
	if val != "myName" {
		t.Errorf("version.Name is %q, should be 'myName'", val)
	}
}

func TestSetDots(t *testing.T) {
	reg := NewRegistry("TestSetDots")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)
	reg.SaveModel()

	// check some dots in the prop names - and some labels stuff too
	dir, _ := reg.AddGroup("dirs", "d1")
	labels := NewPP().P("labels")

	XNoErr(t, reg.SaveAllAndCommit())
	dir.Refresh(registry.FOR_WRITE)

	xErr := dir.SetSave(labels.UI(), "xxx")
	XCheck(t, xErr != nil, "labels=xxx should fail")

	// Nesting under labels should fail
	xErr = dir.SetSave(labels.P("xxx").P("yyy").UI(), "xy")
	XCheckErr(t, xErr, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/dirs/d1",
  "title": "The attribute \"labels.xxx\" is not valid: must be a string"
}`)

	// dots are ok as tag names
	xErr = dir.SetSave(labels.P("abc.def").UI(), "ABC")
	XNoErr(t, xErr)
	XEqual(t, "", dir.Get(labels.P("abc.def").UI()), "ABC")

	XCheckGet(t, reg, "/dirs/d1", `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "labels": {
    "abc.def": "ABC"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	dir.Refresh(registry.FOR_WRITE)

	xErr = dir.SetSave("labels", nil)
	XJSONCheck(t, xErr, nil)
	XCheckGet(t, reg, "/dirs/d1", `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	xErr = dir.SetSave(NewPP().P("labels").P("xxx/yyy").UI(), nil)
	XCheckErr(t, xErr, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/dirs/d1",
  "title": "The request cannot be processed as provided: Unexpected / in \"labels.xxx/yyy\" at pos 11"
}`)

	xErr = dir.SetSave(NewPP().P("labels").P("").P("abc").UI(), nil)
	XCheckErr(t, xErr, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/dirs/d1",
  "title": "The request cannot be processed as provided: Unexpected . in \"labels..abc\" at pos 8"
}`)

	xErr = dir.SetSave(NewPP().P("labels").P("xxx.yyy").UI(), "xxx")
	XJSONCheck(t, xErr, nil)

	xErr = dir.SetSave(NewPP().P("xxx.yyy").UI(), nil)
	XCheckErr(t, xErr, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/dirs/d1",
  "title": "The request cannot be processed as provided: invalid extension(s): xxx"
}`)
	XCheck(t, xErr != nil, "xxx.yyy=nil should fail")
	xErr = dir.SetSave("xxx.", "xxx")
	XCheck(t, xErr != nil, "xxx.=xxx should fail")
	xErr = dir.SetSave(".xxx", "xxx")
	XCheck(t, xErr != nil, ".xxx=xxx should fail")
	xErr = dir.SetSave(".xxx.", "xxx")
	XCheck(t, xErr != nil, ".xxx.=xxx should fail")
}

func TestSetLabels(t *testing.T) {
	reg := NewRegistry("TestSetLabels")
	defer PassDeleteReg(t, reg)
	reg.SaveAllAndCommit()
	reg.Refresh(registry.FOR_WRITE)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	dir, _ := reg.AddGroup("dirs", "d1")
	file, _ := dir.AddResource("files", "f1", "v1")
	ver, _ := file.FindVersion("v1", false, registry.FOR_WRITE)
	ver2, _ := file.AddVersion("v2")

	reg.SaveAllAndCommit()
	reg.Refresh(registry.FOR_WRITE)
	dir.Refresh(registry.FOR_WRITE)
	file.Refresh(registry.FOR_WRITE)
	ver.Refresh(registry.FOR_WRITE)
	ver2.Refresh(registry.FOR_WRITE)

	// /dirs/d1/f1/v1
	labels := NewPP().P("labels")
	err := reg.SetSave(labels.P("r2").UI(), "123.234")
	XNoErr(t, err)
	reg.Refresh(registry.FOR_WRITE)
	// But it's a string here because labels is a map[string]string
	XEqual(t, "", reg.Get(labels.P("r2").UI()), "123.234")
	err = reg.SetSave("labels.r1", "foo")
	XNoErr(t, err)
	reg.Refresh(registry.FOR_WRITE)
	XEqual(t, "", reg.Get(labels.P("r1").UI()), "foo")
	err = reg.SetSave(labels.P("r1").UI(), nil)
	XNoErr(t, err)
	reg.Refresh(registry.FOR_WRITE)
	XEqual(t, "", reg.Get(labels.P("r1").UI()), nil)

	err = dir.SetSave(labels.P("d1").UI(), "bar")
	XNoErr(t, err)
	dir.Refresh(registry.FOR_WRITE)
	XEqual(t, "", dir.Get(labels.P("d1").UI()), "bar")
	// test override
	err = dir.SetSave(labels.P("d1").UI(), "foo")
	XNoErr(t, err)
	dir.Refresh(registry.FOR_WRITE)
	XEqual(t, "", dir.Get(labels.P("d1").UI()), "foo")
	err = dir.SetSave(labels.P("d1").UI(), nil)
	XNoErr(t, err)
	dir.Refresh(registry.FOR_WRITE)
	XEqual(t, "", dir.Get(labels.P("d1").UI()), nil)

	err = file.SetSaveDefault(labels.P("f1").UI(), "foo")
	XNoErr(t, err)
	file.Refresh(registry.FOR_WRITE)
	XEqual(t, "", file.Get(labels.P("f1").UI()), "foo")
	err = file.SetSaveDefault(labels.P("f1").UI(), nil)
	XNoErr(t, err)
	file.Refresh(registry.FOR_WRITE)
	XEqual(t, "", file.Get(labels.P("f1").UI()), nil)

	// Set before we refresh to see if creating v2 causes issues
	// see comment below too
	err = ver.SetSave(labels.P("v1").UI(), "foo")
	XNoErr(t, err)
	ver.Refresh(registry.FOR_WRITE)
	XEqual(t, "", ver.Get(labels.P("v1").UI()), "foo")
	err = ver.SetSave(labels.P("v1").UI(), nil)
	XNoErr(t, err)
	ver.Refresh(registry.FOR_WRITE)
	XEqual(t, "", ver.Get(labels.P("v1").UI()), nil)

	dir.SetSave(labels.P("dd").UI(), "dd.foo")
	file.SetSaveDefault(labels.P("ff").UI(), "ff.bar")

	file.SetSaveDefault(labels.P("dd-ff").UI(), "dash")
	file.SetSaveDefault(labels.P("dd-ff-ff").UI(), "dashes")
	file.SetSaveDefault(labels.P("dd_ff").UI(), "under")
	file.SetSaveDefault(labels.P("dd.ff").UI(), "dot")

	ver2.Refresh(registry.FOR_WRITE) // very important since ver2 is not stale
	err = ver.SetSave(labels.P("vv").UI(), 987.234)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "title": "The attribute \"labels.vv\" is not valid: must be a string"
}`)
	// ver.Refresh(registry.FOR_WRITE) // undo the change, otherwise next Set() will fail

	// Important test
	// We update v1(ver) after we created v2(ver2). At one point in time
	// this could cause both versions to be tagged as "default". Make sure
	// we don't have that situation. See comment above too
	err = ver.SetSave(labels.P("vv2").UI(), "v11")
	XNoErr(t, err)
	ver2.SetSave(labels.P("2nd").UI(), "3rd")

	XCheckGet(t, reg, "?inline", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestSetLabels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "labels": {
    "r2": "123.234"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 5,
      "labels": {
        "dd": "dd.foo"
      },
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 3,
          "isdefault": true,
          "labels": {
            "2nd": "3rd",
            "dd-ff": "dash",
            "dd-ff-ff": "dashes",
            "dd.ff": "dot",
            "dd_ff": "under",
            "ff": "ff.bar"
          },
          "createdat": "2024-01-01T12:00:03Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 1,
            "createdat": "2024-01-01T12:00:03Z",
            "modifiedat": "2024-01-01T12:00:03Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v2",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
            "defaultversionsticky": false
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 4,
              "isdefault": false,
              "labels": {
                "vv2": "v11"
              },
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 3,
              "isdefault": true,
              "labels": {
                "2nd": "3rd",
                "dd-ff": "dash",
                "dd-ff-ff": "dashes",
                "dd.ff": "dot",
                "dd_ff": "under",
                "ff": "ff.bar"
              },
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
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

	file.SetDefault(ver)
	XCheckGet(t, reg, "?inline", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestSetLabels",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "labels": {
    "r2": "123.234"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 5,
      "labels": {
        "dd": "dd.foo"
      },
      "createdat": "2024-01-01T12:00:03Z",
      "modifiedat": "2024-01-01T12:00:02Z",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v1",
          "self": "http://localhost:8181/dirs/d1/files/f1$details",
          "xid": "/dirs/d1/files/f1",
          "epoch": 4,
          "isdefault": true,
          "labels": {
            "vv2": "v11"
          },
          "createdat": "2024-01-01T12:00:03Z",
          "modifiedat": "2024-01-01T12:00:02Z",
          "ancestor": "v1",
          "filebase64": "",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "meta": {
            "fileid": "f1",
            "self": "http://localhost:8181/dirs/d1/files/f1/meta",
            "xid": "/dirs/d1/files/f1/meta",
            "epoch": 2,
            "createdat": "2024-01-01T12:00:03Z",
            "modifiedat": "2024-01-01T12:00:04Z",
            "readonly": false,
            "compatibility": "none",

            "defaultversionid": "v1",
            "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
            "defaultversionsticky": true
          },
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versions": {
            "v1": {
              "fileid": "f1",
              "versionid": "v1",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
              "xid": "/dirs/d1/files/f1/versions/v1",
              "epoch": 4,
              "isdefault": true,
              "labels": {
                "vv2": "v11"
              },
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
            },
            "v2": {
              "fileid": "f1",
              "versionid": "v2",
              "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
              "xid": "/dirs/d1/files/f1/versions/v2",
              "epoch": 3,
              "isdefault": false,
              "labels": {
                "2nd": "3rd",
                "dd-ff": "dash",
                "dd-ff-ff": "dashes",
                "dd.ff": "dot",
                "dd_ff": "under",
                "ff": "ff.bar"
              },
              "createdat": "2024-01-01T12:00:03Z",
              "modifiedat": "2024-01-01T12:00:02Z",
              "ancestor": "v1",
              "filebase64": ""
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

// Set bad attr names via HTTP since using internal APIs (e.g. SetSave)
// won't catch it.
func TestSetNameUser(t *testing.T) {
	reg := NewRegistry("TestSetNameUser")
	defer PassDeleteReg(t, reg)

	gm, rm, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	XNoErr(t, err)
	_, err = reg.Model.AddAttrMap("mymap",
		registry.NewItemType(STRING))
	XNoErr(t, err)
	_, err = reg.Model.AddAttr("*", ANY)
	XNoErr(t, err)

	_, err = gm.AddAttr("*", ANY)
	XNoErr(t, err)
	_, err = gm.AddAttrMap("mymap", registry.NewItemType(STRING))
	XNoErr(t, err)

	_, err = rm.AddAttr("*", ANY)
	XNoErr(t, err)
	_, err = rm.AddMetaAttr("*", ANY)
	XNoErr(t, err)
	_, err = rm.AddAttrMap("mymap", registry.NewItemType(STRING))
	XNoErr(t, err)

	XNoErr(t, reg.SaveModel())
	XNoErr(t, reg.Commit())

	base := "http://localhost:8181"
	for _, test := range []struct {
		name string
		msg  string
	}{
		{"a", ""},
		{"", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"\" is not valid: attribute name \"\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
		{"#a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"#a\" is not valid: attribute name \"#a\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
		{"$a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"$a\" is not valid: attribute name \"$a\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
		{"a$a", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"a$a\" is not valid: attribute name \"a$a\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
		{"a$", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"a$\" is not valid: attribute name \"a$\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
		{"a.", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "subject": "http://localhost:8181/",
  "title": "The attribute \"a.\" is not valid: attribute name \"a.\" must match: ^[a-z_][a-z_0-9]{0,62}$"
}
`},
	} {
		putFn := func(path string, name string, msg string) {
			body := bytes.NewBuffer([]byte(fmt.Sprintf(`{"%s":"hi"}`, name)))
			req, _ := http.NewRequest("PUT", base+path, body)
			t.Logf("  Path: %q", path)

			client := &http.Client{}

			resBody := []byte{}
			res, err := client.Do(req)
			XNoErr(t, err)
			if res != nil {
				resBody, _ = io.ReadAll(res.Body)
			}
			if msg == "" {
				if res.StatusCode/100 == 2 {
					return
				}
				t.Fatalf("%q should not have failed: %s", name, string(resBody))
			}
			if res.StatusCode == 200 {
				t.Logf("Body:\n%s", string(resBody))
				t.Fatalf("%q should have failed, but didn't", name)
			}
			XEqual(t, "", string(resBody), msg)
		}
		t.Logf("Name: %q", test.name)

		putFn("/", test.name, test.msg)
		putFn("/dirs/d1", test.name, test.msg)
		putFn("/dirs/d1/files/f1$details", test.name, test.msg)
		putFn("/dirs/d1/files/f1/versions/v1$details", test.name, test.msg)
		putFn("/dirs/d1/files/f1/meta", test.name, test.msg)
	}

	XHTTP(t, reg, "PUT", "/", `{
		"ext": {
		}
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestSetNameUser",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ext": {},

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/", `{
		"ext": {
		  "foo": "bar"
		}
	}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestSetNameUser",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ext": {
    "foo": "bar"
  },

  "dirsurl": "http://localhost:8181/dirs",
  "dirscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/", `{"mymap":{":bar":"bar"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "http://localhost:8181/",
  "title": "There was an error in the model definition provided: while processing \"mymap\", map key name \":bar\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
}
`)
	XHTTP(t, reg, "PUT", "/", `{"mymap":{"@bar":"bar"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "http://localhost:8181/",
  "title": "There was an error in the model definition provided: while processing \"mymap\", map key name \"@bar\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
}
`)
	// This is ok because "mymap" is under "ext" which is defined as "*"
	// and that allows ANYTHING as long as it's valid json
	XHTTP(t, reg, "PUT", "/", `{"ext":{"mymap":{"@bar":"bar"}}}`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1", `{"mymap":{"@bar":"bar"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "http://localhost:8181/dirs/d1",
  "title": "There was an error in the model definition provided: while processing \"mymap\", map key name \"@bar\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
}
`)
	// This is ok because "mymap" is under "ext" which is defined as "*"
	// and that allows ANYTHING as long as it's valid json
	XHTTP(t, reg, "PUT", "/dirs/d1",
		`{"ext":{"mymap":{"@bar":"bar"}}}`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"mymap":{"@bar":"bar"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "title": "There was an error in the model definition provided: while processing \"mymap\", map key name \"@bar\" must match: ^[a-z0-9][a-z0-9_.:\\-]{0,62}$"
}
`)
	// This is ok because "mymap" is under "ext" which is defined as "*"
	// and that allows ANYTHING as long as it's valid json
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"ext":{"mymap":{"@bar":"bar"}}}`, 200, `*`)

}
