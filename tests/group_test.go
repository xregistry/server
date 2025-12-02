package tests

import (
	"testing"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestCreateGroup(t *testing.T) {
	reg := NewRegistry("TestCreateGroup")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, true, true)

	d1, err := reg.AddGroup("dirs", "d1")
	XNoErr(t, err)
	XCheck(t, d1 != nil, "D1 is nil")

	dt, err := reg.AddGroup("dirs", "d1")
	XCheck(t, dt == nil && err != nil, "Dup should fail")

	d2, isNew, err := reg.UpsertGroup("dirs", "d1")
	XCheck(t, d2 != nil && err == nil, "Update should have worked")
	XCheck(t, isNew == false, "Should not be new")
	XCheck(t, d2 == d1, "Should be the same")

	f1, err := d1.AddResource("files", "f1", "v1")
	XNoErr(t, err)
	ft, err := d1.AddResource("files", "f1", "v1")
	XCheck(t, ft == nil && err != nil, "Dup files should have failed - 1")
	ft, err = d1.AddResource("files", "f1", "v2")
	XCheck(t, ft == nil && err != nil, "Dup files should have failed - 2")

	f1.AddVersion("v2")
	d2, _ = reg.AddGroup("dirs", "d2")
	f2, _ := d2.AddResource("files", "f2", "v1")
	f2.AddVersion("v1.1")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1
	//             v1.1

	// Check basic GET first
	XCheckGet(t, reg, "/dirs/d1",
		`{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 1
}
`)
	XCheckGet(t, reg, "/dirs/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/xxx) cannot be found.",
  "subject": "/dirs/xxx",
  "source": ":registry:httpStuff:1730"
}
`)
	XCheckGet(t, reg, "dirs/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/xxx) cannot be found.",
  "subject": "/dirs/xxx",
  "source": ":registry:httpStuff:1730"
}
`)
	XCheckGet(t, reg, "/dirs/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/xxx/yyy) cannot be found.",
  "detail": "Unknown Resource type: yyy.",
  "subject": "/dirs/xxx/yyy",
  "source": ":registry:info:595"
}
`)
	XCheckGet(t, reg, "dirs/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/dirs/xxx/yyy) cannot be found.",
  "detail": "Unknown Resource type: yyy.",
  "subject": "/dirs/xxx/yyy",
  "source": ":registry:info:595"
}
`)

	g, err := reg.FindGroup("dirs", "d1", false, registry.FOR_WRITE)
	g.AccessMode = d1.AccessMode // cheat a little
	XNoErr(t, err)
	XJSONCheck(t, g, d1)

	g, err = reg.FindGroup("xxx", "d1", false, registry.FOR_WRITE)
	XCheck(t, err == nil && g == nil, "Finding xxx/d1 should have failed")

	g, err = reg.FindGroup("dirs", "xx", false, registry.FOR_WRITE)
	XCheck(t, err == nil && g == nil, "Finding dirs/xxx should have failed")

	r, err := d1.FindResource("files", "f1", false, registry.FOR_WRITE)
	XCheck(t, err == nil && r != nil, "Finding resource failed")
	r.AccessMode = f1.AccessMode // minor cheat
	XJSONCheck(t, r, f1)

	r2, err := d1.FindResource("files", "xxx", false, registry.FOR_WRITE)
	XCheck(t, err == nil && r2 == nil, "Finding files/xxx didn't work")

	r2, err = d1.FindResource("xxx", "f1", false, registry.FOR_WRITE)
	XCheck(t, err == nil && r2 == nil, "Finding xxx/f1 didn't work")

	err = d1.Delete()
	XNoErr(t, err)

	g, err = reg.FindGroup("dirs", "d1", false, registry.FOR_WRITE)
	XCheck(t, err == nil && g == nil, "Finding delete group failed")
}

func TestGroupRequiredFields(t *testing.T) {
	reg := NewRegistry("TestGroupRequiredFields")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	_, err := gm.AddAttribute(&registry.Attribute{
		Name:     "req",
		Type:     STRING,
		Required: true,
	})
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	_, err = reg.AddGroup("dirs", "d1")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1\" are missing: req.",
  "subject": "/dirs/d1",
  "args": {
    "list": "req"
  },
  "source": ":registry:entity:2146"
}`)
	reg.Rollback()
	reg.Refresh(registry.FOR_WRITE)

	g1, err := reg.AddGroupWithObject("dirs", "d1",
		Object{"req": "test"})
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	err = g1.SetSave("req", nil)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/dirs/d1\" are missing: req.",
  "subject": "/dirs/d1",
  "args": {
    "list": "req"
  },
  "source": ":registry:entity:2146"
}`)

	err = g1.SetSave("req", "again")
	XNoErr(t, err)
}
