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
	xNoErr(t, err)
	xCheck(t, d1 != nil, "D1 is nil")

	dt, err := reg.AddGroup("dirs", "d1")
	xCheck(t, dt == nil && err != nil, "Dup should fail")

	d2, isNew, err := reg.UpsertGroup("dirs", "d1")
	xCheck(t, d2 != nil && err == nil, "Update should have worked")
	xCheck(t, isNew == false, "Should not be new")
	xCheck(t, d2 == d1, "Should be the same")

	f1, err := d1.AddResource("files", "f1", "v1")
	xNoErr(t, err)
	ft, err := d1.AddResource("files", "f1", "v1")
	xCheck(t, ft == nil && err != nil, "Dup files should have failed - 1")
	ft, err = d1.AddResource("files", "f1", "v2")
	xCheck(t, ft == nil && err != nil, "Dup files should have failed - 2")

	f1.AddVersion("v2")
	d2, _ = reg.AddGroup("dirs", "d2")
	f2, _ := d2.AddResource("files", "f2", "v1")
	f2.AddVersion("v1.1")

	// /dirs/d1/f1/v1
	//            /v2
	//      /d2/f2/v1
	//             v1.1

	// Check basic GET first
	xCheckGet(t, reg, "/dirs/d1",
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
	xCheckGet(t, reg, "/dirs/xxx", "Not found\n")
	xCheckGet(t, reg, "dirs/xxx", "Not found\n")
	xCheckGet(t, reg, "/dirs/xxx/yyy", "Unknown Resource type: yyy\n")
	xCheckGet(t, reg, "dirs/xxx/yyy", "Unknown Resource type: yyy\n")

	g, err := reg.FindGroup("dirs", "d1", false, registry.FOR_WRITE)
	g.AccessMode = d1.AccessMode // cheat a little
	xNoErr(t, err)
	xJSONCheck(t, g, d1)

	g, err = reg.FindGroup("xxx", "d1", false, registry.FOR_WRITE)
	xCheck(t, err == nil && g == nil, "Finding xxx/d1 should have failed")

	g, err = reg.FindGroup("dirs", "xx", false, registry.FOR_WRITE)
	xCheck(t, err == nil && g == nil, "Finding dirs/xxx should have failed")

	r, err := d1.FindResource("files", "f1", false, registry.FOR_WRITE)
	xCheck(t, err == nil && r != nil, "Finding resource failed")
	r.AccessMode = f1.AccessMode // minor cheat
	xJSONCheck(t, r, f1)

	r2, err := d1.FindResource("files", "xxx", false, registry.FOR_WRITE)
	xCheck(t, err == nil && r2 == nil, "Finding files/xxx didn't work")

	r2, err = d1.FindResource("xxx", "f1", false, registry.FOR_WRITE)
	xCheck(t, err == nil && r2 == nil, "Finding xxx/f1 didn't work")

	err = d1.Delete()
	xNoErr(t, err)

	g, err = reg.FindGroup("dirs", "d1", false, registry.FOR_WRITE)
	xCheck(t, err == nil && g == nil, "Finding delete group failed")
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
	xNoErr(t, err)
	reg.SaveAllAndCommit()

	_, err = reg.AddGroup("dirs", "d1")
	xCheckErr(t, err, "Required property \"req\" is missing")
	reg.Rollback()
	reg.Refresh(registry.FOR_WRITE)

	g1, err := reg.AddGroupWithObject("dirs", "d1",
		Object{"req": "test"})
	xNoErr(t, err)
	reg.SaveAllAndCommit()

	err = g1.SetSave("req", nil)
	xCheckErr(t, err, "Required property \"req\" is missing")

	err = g1.SetSave("req", "again")
	xNoErr(t, err)
}
