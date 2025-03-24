package tests

import (
	"testing"

	"github.com/xregistry/server/registry"
)

func TestCreateRegistry(t *testing.T) {
	reg := NewRegistry("TestCreateRegistry")
	defer PassDeleteReg(t, reg)

	// Check basic GET first
	xCheckGet(t, reg, "/",
		`{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCreateRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
	xCheckGet(t, reg, "/xxx", "Unknown Group type: xxx\n")
	xCheckGet(t, reg, "xxx", "Unknown Group type: xxx\n")
	xCheckGet(t, reg, "/xxx/yyy", "Unknown Group type: xxx\n")
	xCheckGet(t, reg, "xxx/yyy", "Unknown Group type: xxx\n")

	// make sure dups generate an error
	reg2, err := registry.NewRegistry(nil, "TestCreateRegistry")
	defer reg2.Rollback()
	if err == nil || reg2 != nil {
		t.Errorf("Creating same named registry worked!")
	}

	// make sure it was really created
	reg3, err := registry.FindRegistry(nil, "TestCreateRegistry")
	defer reg3.Rollback()
	xCheck(t, err == nil && reg3 != nil,
		"Finding TestCreateRegistry should have worked")

	reg3, err = registry.NewRegistry(nil, "")
	defer PassDeleteReg(t, reg3)
	xNoErr(t, err)
	xCheck(t, reg3 != nil, "reg3 shouldn't be nil")
	xCheck(t, reg3 != reg, "reg3 should be different from reg")

	xCheckGet(t, reg, "", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestCreateRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
}

func TestDeleteRegistry(t *testing.T) {
	reg, err := registry.NewRegistry(nil, "TestDeleteRegistry")
	defer reg.Rollback()
	xNoErr(t, err)

	err = reg.Delete()
	xNoErr(t, err)
	reg.SaveAllAndCommit()

	reg, err = registry.FindRegistry(nil, "TestDeleteRegistry")
	defer reg.Rollback()
	xCheck(t, reg == nil && err == nil,
		"Finding TestCreateRegistry found one but shouldn't")
}

func TestRefreshRegistry(t *testing.T) {
	reg := NewRegistry("TestRefreshRegistry")
	defer PassDeleteReg(t, reg)

	reg.Entity.Object["xxx"] = "yyy"
	xCheck(t, reg.Get("xxx") == "yyy", "xxx should be yyy")

	err := reg.Refresh()
	xNoErr(t, err)

	xCheck(t, reg.Get("xxx") == nil, "xxx should not be there")
}

func TestFindRegistry(t *testing.T) {
	reg, err := registry.FindRegistry(nil, "TestFindRegistry")
	defer reg.Rollback()
	xCheck(t, reg == nil && err == nil,
		"Shouldn't have found TestFindRegistry")

	reg, err = registry.NewRegistry(nil, "TestFindRegistry")
	defer reg.SaveAllAndCommit()
	defer reg.Delete() // PassDeleteReg(t, reg)
	xNoErr(t, err)

	reg2, err := registry.FindRegistry(nil, reg.UID)
	defer reg2.Rollback()
	xNoErr(t, err)
	xJSONCheck(t, reg2, reg)
}

func TestRegistryProps(t *testing.T) {
	reg := NewRegistry("TestRegistryProps")
	defer PassDeleteReg(t, reg)

	err := reg.SetSave("specversion", "x.y")
	if err == nil {
		t.Errorf("Setting specversion to x.y should have failed")
		t.FailNow()
	}
	reg.SetSave("name", "nameIt")
	reg.SetSave("description", "a very cool reg")
	reg.SetSave("documentation", "https://docs.com")
	reg.SetSave("labels.stage", "dev")

	xCheckGet(t, reg, "", `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryProps",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "name": "nameIt",
  "description": "a very cool reg",
  "documentation": "https://docs.com",
  "labels": {
    "stage": "dev"
  },
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z"
}
`)
}

func TestRegistryRequiredFields(t *testing.T) {
	reg := NewRegistry("TestRegistryRequiredFields")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name:     "req",
		Type:     registry.STRING,
		Required: true,
	})
	xNoErr(t, err)

	// Commit before we call Set below otherwise the Tx will be rolled back
	reg.SaveAllAndCommit()

	err = reg.SetSave("description", "testing")
	xCheckErr(t, err, "Required property \"req\" is missing")

	xNoErr(t, reg.JustSet("req", "testing2"))
	xNoErr(t, reg.SetSave("description", "testing"))

	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryRequiredFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "description": "testing",
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "req": "testing2"
}
`)

}

func TestRegistryDefaultFields(t *testing.T) {
	reg := NewRegistry("TestRegistryDefaultFields")
	defer PassDeleteReg(t, reg)

	_, err := reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     registry.STRING,
		Required: true,
		Default:  123,
	})
	xCheckErr(t, err, `"model.defstring" "default" value must be of type "string"`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     registry.OBJECT,
		Required: true,
		Default:  "hello",
	})
	xCheckErr(t, err, `"model.defstring" is not a scalar, so "default" is not allowed`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     registry.STRING,
		Required: true,
		Default:  map[string]any{"key": "value"},
	})
	xCheckErr(t, err, `"model.defstring" "default" value must be of type "string"`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     registry.STRING,
		Required: true,
		Default:  "hello",
	})
	xNoErr(t, err)

	obj, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj",
		Type: registry.OBJECT,
	})
	xNoErr(t, err)

	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     registry.INTEGER,
		Required: true,
		Default:  "string",
	})
	xCheckErr(t, err, `"model.myobj.defint" "default" value must be of type "integer"`)

	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     registry.OBJECT,
		Required: true,
		Default:  "string",
	})
	xCheckErr(t, err, `"model.myobj.defint" is not a scalar, so "default" is not allowed`)

	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     registry.INTEGER,
		Required: true,
		Default:  123,
	})
	xNoErr(t, err)

	// Commit before we call Set below otherwise the Tx will be rolled back
	reg.Refresh()
	reg.Touch() // Force a validation which will set all defaults

	xHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	xHTTP(t, reg, "PUT", "/", "{}", 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	xHTTP(t, reg, "PUT", "/", `{
  "defstring": "updated hello",
  "myobj": {}
}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "updated hello",
  "myobj": {
    "defint": 123
  }
}
`)

	xHTTP(t, reg, "PUT", "/", `{
  "myobj": {
    "defint": 666
  }
}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello",
  "myobj": {
    "defint": 666
  }
}
`)

	xHTTP(t, reg, "PUT", "/", `{
}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	xHTTP(t, reg, "PUT", "/", `{
  "myobj": null
}`, 200, `{
  "specversion": "`+registry.SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 7,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)
}
