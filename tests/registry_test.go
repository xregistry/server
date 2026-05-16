package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestRegistryCreate(t *testing.T) {
	reg := NewRegistry("TestRegistryCreate")
	defer PassDeleteReg(t, reg)

	// Check basic GET first
	XCheckGet(t, reg, "/",
		`{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryCreate",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
	XCheckGet(t, reg, "/xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/xxx) cannot be found.",
  "detail": "Unknown Group type: xxx.",
  "subject": "/xxx",
  "source": "e4e59b8a76c4:registry:info:558"
}
`)
	XCheckGet(t, reg, "xxx", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/xxx) cannot be found.",
  "detail": "Unknown Group type: xxx.",
  "subject": "/xxx",
  "source": "e4e59b8a76c4:registry:info:558"
}
`)
	XCheckGet(t, reg, "/xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/xxx) cannot be found.",
  "detail": "Unknown Group type: xxx.",
  "subject": "/xxx",
  "source": "e4e59b8a76c4:registry:info:558"
}
`)
	XCheckGet(t, reg, "xxx/yyy", `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#not_found",
  "title": "The targeted entity (/xxx) cannot be found.",
  "detail": "Unknown Group type: xxx.",
  "subject": "/xxx",
  "source": "e4e59b8a76c4:registry:info:558"
}
`)

	// make sure dups generate an error
	reg2, err := registry.NewRegistry(nil, "TestRegistryCreate")
	defer reg2.Rollback()
	if err == nil || reg2 != nil {
		t.Errorf("Creating same named registry worked!")
	}

	// make sure it was really created
	reg3, err := registry.FindRegistry(nil, "TestRegistryCreate",
		registry.FOR_WRITE)
	defer reg3.Rollback()
	XCheck(t, err == nil && reg3 != nil,
		"Finding TestRegistryCreate should have worked")

	reg3, err = registry.NewRegistry(nil, "")
	defer PassDeleteReg(t, reg3)
	XNoErr(t, err)
	XCheck(t, reg3 != nil, "reg3 shouldn't be nil")
	XCheck(t, reg3 != reg, "reg3 should be different from reg")

	XCheckGet(t, reg, "", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryCreate",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2024-01-01T12:00:01Z",
  "modifiedat": "2024-01-01T12:00:01Z"
}
`)
}

func TestRegistryDelete(t *testing.T) {
	reg, err := registry.NewRegistry(nil, "TestRegistryDelete")
	defer reg.Rollback()
	XNoErr(t, err)

	err = reg.Delete()
	XNoErr(t, err)
	reg.SaveAllAndCommit()

	reg, err = registry.FindRegistry(nil, "TestRegistryDelete",
		registry.FOR_WRITE)
	defer reg.Rollback()
	XCheck(t, reg == nil && err == nil,
		"Finding TestRegistryCreate found one but shouldn't")
}

func TestRegistryRefresh(t *testing.T) {
	reg := NewRegistry("TestRegistryRefresh")
	defer PassDeleteReg(t, reg)

	reg.Entity.Object["xxx"] = "yyy"
	XCheck(t, reg.Get("xxx") == "yyy", "xxx should be yyy")

	err := reg.Refresh(registry.FOR_WRITE)
	XNoErr(t, err)

	XCheck(t, reg.Get("xxx") == nil, "xxx should not be there")
}

func TestRegistryFind(t *testing.T) {
	reg, err := registry.FindRegistry(nil, "TestRegistryFind",
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
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/\" are missing: req.",
  "subject": "/",
  "args": {
    "list": "req"
  },
  "source": "e4e59b8a76c4:registry:entity:2149"
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
  "title": "There was an error in the model definition provided: \"defstring\" \"default\" value must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"defstring\" \"default\" value must be of type \"string\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2962"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:    "defstring",
		Type:    STRING,
		Default: "abc",
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_required_true",
  "title": "Model attribute \"defstring\" needs to have a \"required\" value of \"true\" since a default value is provided.",
  "subject": "/model",
  "args": {
    "name": "defstring"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2969"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     OBJECT,
		Required: true,
		Default:  "hello",
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_scalar_default",
  "title": "Model attribute \"defstring\" is not allowed to have a default value since it is not a scalar.",
  "subject": "/model",
  "args": {
    "name": "defstring"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2954"
}`)

	_, err = reg.Model.AddAttribute(&registry.Attribute{
		Name:     "defstring",
		Type:     STRING,
		Required: true,
		Default:  map[string]any{"key": "value"},
	})
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"defstring\" \"default\" value must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"defstring\" \"default\" value must be of type \"string\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2960"
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
	err = reg.SaveModel(true)
	XNoErr(t, err)

	_, err = obj.AddAttribute(&registry.Attribute{
		Name:     "defint",
		Type:     INTEGER,
		Required: true,
		Default:  "string",
	})
	XNoErr(t, err)
	err = reg.SaveModel(true)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"myobj.defint\" \"default\" value must be of type \"integer\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"myobj.defint\" \"default\" value must be of type \"integer\""
  },
  "source": "e4e59b8a76c4:registry:shared_model:2960"
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
	err = reg.SaveModel(true)
	XCheckErr(t, err, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_scalar_default",
  "title": "Model attribute \"myobj.defint\" is not allowed to have a default value since it is not a scalar.",
  "subject": "/model",
  "args": {
    "name": "myobj.defint"
  },
  "source": "e4e59b8a76c4:registry:shared_model:2954"
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
	err = reg.SaveModel(true)
	XNoErr(t, err)

	// Commit before we call Set below otherwise the Tx will be rolled back
	reg.Refresh(registry.FOR_WRITE)
	reg.Touch() // Force a validation which will set all defaults
	reg.ValidateAndSave(false)

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

func TestRegistryRoot(t *testing.T) {
	reg := NewRegistry("TestRegistryRoot")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "GET", "/?inline=capabilities,modelsource", ``, 200,
		`{
  "specversion": "1.0-rc2",
  "registryid": "TestRegistryRoot",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2026-05-15T19:51:47.726218335Z",
  "modifiedat": "2026-05-15T19:51:47.726218335Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "capabilitiesoffered": {
        "mutable": false
      },
      "entities": {
        "mutable": true
      },
      "export": {
        "mutable": false
      },
      "model": {
        "mutable": false
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {
      "avro*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "jsonschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "numbers": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "protobuf*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ],
      "xmlschema*": [
        "backward",
        "backward_transitive",
        "forward",
        "forward_transitive",
        "full",
        "full_transitive"
      ]
    },
    "flags": [
      "binary",
      "collections",
      "doc",
      "epoch",
      "filter",
      "ignore",
      "inline",
      "setdefaultversionid",
      "sort",
      "specversion"
    ],
    "formats": [
      "avro*",
      "jsonschema*",
      "numbers",
      "protobuf*",
      "xmlschema*"
    ],
    "ignores": [
      "capabilities",
      "defaultversionid",
      "defaultversionsticky",
      "epoch",
      "id",
      "modelsource",
      "readonly"
    ],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": true,
    "versionmodes": [
      "createdat",
      "manual"
    ]
  },
  "modelsource": {}
}
`)
	// epoch=1

	// First, make caps minimal
	XHTTP(t, reg, "PUT", "/capabilities", `{
    "available": {
      "capabilities": { "mutable": true },
      "modelsource": { "mutable": true }
    },
    "flags": [ "inline" ]
    }`, 200, `{
  "available": {
    "capabilities": {
      "mutable": true
    },
    "entities": {
      "mutable": true
    },
    "modelsource": {
      "mutable": true
    }
  },
  "compatibilities": {},
  "flags": [
    "inline"
  ],
  "formats": [],
  "ignores": [],
  "pagination": false,
  "shortself": false,
  "specversions": [
    "1.0-rc2"
  ],
  "stickyversions": true,
  "versionmodes": [
    "manual"
  ]
}
`)

	// Minimal + epoch=2
	XHTTP(t, reg, "GET", "/?inline=capabilities,modelsource", ``, 200,
		`{
  "specversion": "1.0-rc2",
  "registryid": "TestRegistryRoot",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-05-15T19:53:28.842649795Z",
  "modifiedat": "2026-05-15T19:53:28.864538646Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "entities": {
        "mutable": true
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {},
    "flags": [
      "inline"
    ],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  },
  "modelsource": {}
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{"description":"testing"}`, 200,
		`{
  "description": "testing"
}
`)

	// epoch=3
	XHTTP(t, reg, "GET", "/?inline=capabilities,modelsource", ``, 200,
		`{
  "specversion": "1.0-rc2",
  "registryid": "TestRegistryRoot",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 3,
  "createdat": "2026-05-15T19:53:28.842649795Z",
  "modifiedat": "2026-05-15T19:53:28.864538646Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "entities": {
        "mutable": true
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {},
    "flags": [
      "inline"
    ],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  },
  "modelsource": {
    "description": "testing"
  }
}
`)

	// Now, make sure we don't lose anything on a PUT
	XHTTP(t, reg, "PUT", "/?inline=capabilities,modelsource", `{}`, 200,
		`{
  "specversion": "1.0-rc2",
  "registryid": "TestRegistryRoot",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
  "createdat": "2026-05-15T19:47:09.150460505Z",
  "modifiedat": "2026-05-15T19:47:09.178548005Z",

  "capabilities": {
    "available": {
      "capabilities": {
        "mutable": true
      },
      "entities": {
        "mutable": true
      },
      "modelsource": {
        "mutable": true
      }
    },
    "compatibilities": {},
    "flags": [
      "inline"
    ],
    "formats": [],
    "ignores": [],
    "pagination": false,
    "shortself": false,
    "specversions": [
      "1.0-rc2"
    ],
    "stickyversions": true,
    "versionmodes": [
      "manual"
    ]
  },
  "modelsource": {
    "description": "testing"
  }
}
`)

}
