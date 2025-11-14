package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestCreateRegistry(t *testing.T) {
	reg := NewRegistry("TestCreateRegistry")
	defer PassDeleteReg(t, reg)

	// Check basic GET first
	XCheckGet(t, reg, "/",
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestCreateRegistry",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
	XCheckGet(t, reg, "/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "subject": "http://localhost:8181/xxx",
  "title": "The specified entity cannot be found: /xxx",
  "detail": "Unknown Group type: xxx"
}
`)
	XCheckGet(t, reg, "xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "subject": "http://localhost:8181/xxx",
  "title": "The specified entity cannot be found: /xxx",
  "detail": "Unknown Group type: xxx"
}
`)
	XCheckGet(t, reg, "/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "subject": "http://localhost:8181/xxx",
  "title": "The specified entity cannot be found: /xxx",
  "detail": "Unknown Group type: xxx"
}
`)
	XCheckGet(t, reg, "xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "subject": "http://localhost:8181/xxx",
  "title": "The specified entity cannot be found: /xxx",
  "detail": "Unknown Group type: xxx"
}
`)

	// make sure dups generate an error
	reg2, err := registry.NewRegistry(nil, "TestCreateRegistry")
	defer reg2.Rollback()
	if err == nil || reg2 != nil {
		t.Errorf("Creating same named registry worked!")
	}

	// make sure it was really created
	reg3, err := registry.FindRegistry(nil, "TestCreateRegistry",
		registry.FOR_WRITE)
	defer reg3.Rollback()
	XCheck(t, err == nil && reg3 != nil,
		"Finding TestCreateRegistry should have worked")

	reg3, err = registry.NewRegistry(nil, "")
	defer PassDeleteReg(t, reg3)
	XNoErr(t, err)
	XCheck(t, reg3 != nil, "reg3 shouldn't be nil")
	XCheck(t, reg3 != reg, "reg3 should be different from reg")

	XCheckGet(t, reg, "", `{
  "specversion": "`+SPECVERSION+`",
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
	XNoErr(t, err)

	err = reg.Delete()
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	reg, err = registry.FindRegistry(nil, "TestDeleteRegistry",
		registry.FOR_WRITE)
	defer reg.Rollback()
	XCheck(t, reg == nil && err == nil,
		"Finding TestCreateRegistry found one but shouldn't")
}

func TestRefreshRegistry(t *testing.T) {
	reg := NewRegistry("TestRefreshRegistry")
	defer PassDeleteReg(t, reg)

	reg.Entity.Object["xxx"] = "yyy"
	XCheck(t, reg.Get("xxx") == "yyy", "xxx should be yyy")

	err := reg.Refresh(registry.FOR_WRITE)
	XNoErr(t, err)

	XCheck(t, reg.Get("xxx") == nil, "xxx should not be there")
}

func TestFindRegistry(t *testing.T) {
	reg, err := registry.FindRegistry(nil, "TestFindRegistry",
		registry.FOR_WRITE)
	defer reg.Rollback()
	XCheck(t, reg == nil && err == nil,
		"Shouldn't have found TestFindRegistry")

	reg, err = registry.NewRegistry(nil, "TestFindRegistry")
	defer reg.SaveAllAndCommit()
	defer reg.Delete() // PassDeleteReg(t, reg)
	XNoErr(t, err)

	reg2, err := registry.FindRegistry(nil, reg.UID, registry.FOR_WRITE)
	defer reg2.Rollback()
	XNoErr(t, err)
	reg2.AccessMode = reg.AccessMode
	XJSONCheck(t, reg2, reg)
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

	XCheckGet(t, reg, "", `{
  "specversion": "`+SPECVERSION+`",
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
		Type:     STRING,
		Required: true,
	})
	XNoErr(t, err)

	// Commit before we call Set below otherwise the Tx will be rolled back
	reg.SaveAllAndCommit()

	err = reg.SetSave("description", "testing")
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "subject": "/",
  "title": "The request cannot be processed as provided: required property \"req\" is missing"
}`)

	XNoErr(t, reg.JustSet("req", "testing2"))
	XNoErr(t, reg.SetSave("description", "testing"))

	XHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+SPECVERSION+`",
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
		Type:     STRING,
		Required: true,
		Default:  123,
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.defstring\" \"default\" value must be of type \"string\""
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:    "defstring",
		Type:    STRING,
		Default: "abc",
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.defstring\" must have \"require\" set to \"true\" since a default value is defined"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     OBJECT,
		Required: true,
		Default:  "hello",
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.defstring\" is not a scalar, so \"default\" is not allowed"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     STRING,
		Required: true,
		Default:  map[string]any{"key": "value"},
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.defstring\" \"default\" value must be of type \"string\""
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     STRING,
		Required: true,
		Default:  "hello",
	})
	XNoErr(t, err)

	obj, err := reg.Model.AddAttribute(&registry.Attribute{
		Name: "myobj",
		Type: OBJECT,
	})
	XNoErr(t, err)
	err = reg.SaveModel()
	XNoErr(t, err)

	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     INTEGER,
		Required: true,
		Default:  "string",
	})
	XNoErr(t, err)
	err = reg.SaveModel()
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.myobj.defint\" \"default\" value must be of type \"integer\""
}`)
	reg.LoadModel()

	obj = reg.Model.Attributes["myobj"]
	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     OBJECT,
		Required: true,
		Default:  "string",
	})
	XNoErr(t, err)
	err = reg.SaveModel()
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "subject": "/",
  "title": "There was an error in the model definition provided: \"model.myobj.defint\" is not a scalar, so \"default\" is not allowed"
}`)
	reg.LoadModel()

	obj = reg.Model.Attributes["myobj"]
	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     INTEGER,
		Required: true,
		Default:  123,
	})
	XNoErr(t, err)
	err = reg.SaveModel()
	XNoErr(t, err)

	// Commit before we call Set below otherwise the Tx will be rolled back
	reg.Refresh(registry.FOR_WRITE)
	reg.Touch() // Force a validation which will set all defaults

	XHTTP(t, reg, "GET", "/", "", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	XHTTP(t, reg, "PUT", "/", "{}", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	XHTTP(t, reg, "PUT", "/", `{
  "defstring": "updated hello",
  "myobj": {}
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "PUT", "/", `{
  "myobj": {
    "defint": 666
  }
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
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

	XHTTP(t, reg, "PUT", "/", `{
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 6,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:02Z",
  "defstring": "hello"
}
`)

	XHTTP(t, reg, "PUT", "/", `{
  "myobj": null
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
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
