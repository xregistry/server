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

	XHTTP(t, reg, "PUT", "/", `{"specversion":"x.y"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"specversion\" for \"/\" is not valid: invalid value: x.y.",
  "subject": "/",
  "args": {
    "error_detail": "invalid value: x.y",
    "name": "specversion"
  },
  "source": ":registry:entity:1200"
}
`)

	XHTTP(t, reg, "PUT", "/", `{
  "name": "nameIt",
  "description": "a very cool reg",
  "documentation": "https://docs.com",
  "labels": {
    "stage": "dev"
  }
}`, 200, `{
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

	// Start with a wildcard model (any extension allowed) so we can set
	// "req" on the registry BEFORE the model requires it - that way, by
	// the time we make "req" a required attribute, the existing data
	// already satisfies it and PUT /modelsource's forced revalidation
	// of existing data doesn't fail.
	model1 := `{
  "attributes": {
    "*": {
      "name": "*",
      "type": "any"
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model1, 200, model1+"\n")

	XHTTP(t, reg, "PUT", "/", `{
  "req": "testing"
}`, 200, `*`)

	model2 := `{
  "attributes": {
    "req": {
      "name": "req",
      "type": "string",
      "required": true
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model2, 200, model2+"\n")

	XHTTP(t, reg, "PUT", "/", `{
  "description": "testing"
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#required_attribute_missing",
  "title": "One or more mandatory attributes for \"/\" are missing: req.",
  "subject": "/",
  "args": {
    "list": "req"
  },
  "source": ":registry:entity:2761"
}
`)

	XHTTP(t, reg, "PUT", "/", `{
  "description": "testing",
  "req": "testing2"
}`, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryRequiredFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 5,
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

	// PUT /modelsource runs through the same model-definition validation
	// as the Go-API AddAttribute() calls this test used to make - so all
	// of these bad-default checks convert directly to HTTP model_error/
	// model_required_true/model_scalar_default checks.
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": 123
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"defstring\" \"default\" value must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"defstring\" \"default\" value must be of type \"string\""
  },
  "source": ":registry:shared_model:3541"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "default": "abc"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_required_true",
  "title": "Model attribute \"defstring\" needs to have a \"required\" value of \"true\" since a default value is provided.",
  "subject": "/model",
  "args": {
    "name": "defstring"
  },
  "source": ":registry:shared_model:3548"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "object",
      "required": true,
      "default": "hello"
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_scalar_default",
  "title": "Model attribute \"defstring\" is not allowed to have a default value since it is not a scalar.",
  "subject": "/model",
  "args": {
    "name": "defstring"
  },
  "source": ":registry:shared_model:3535"
}
`)

	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": {"key": "value"}
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"defstring\" \"default\" value must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"defstring\" \"default\" value must be of type \"string\""
  },
  "source": ":registry:shared_model:3541"
}
`)

	// Now the good "defstring" + an empty "myobj" - saved successfully.
	model := `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": "hello"
    },
    "myobj": {
      "name": "myobj",
      "type": "object"
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Bad nested default (myobj.defint) - same model_error as top-level.
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": "hello"
    },
    "myobj": {
      "name": "myobj",
      "type": "object",
      "attributes": {
        "defint": {
          "name": "defint",
          "type": "integer",
          "required": true,
          "default": "string"
        }
      }
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"myobj.defint\" \"default\" value must be of type \"integer\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"myobj.defint\" \"default\" value must be of type \"integer\""
  },
  "source": ":registry:shared_model:3541"
}
`)

	// Bad nested default (myobj.defint, non-scalar type) - model_scalar_default.
	XHTTP(t, reg, "PUT", "/modelsource", `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": "hello"
    },
    "myobj": {
      "name": "myobj",
      "type": "object",
      "attributes": {
        "defint": {
          "name": "defint",
          "type": "object",
          "required": true,
          "default": "string"
        }
      }
    }
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_scalar_default",
  "title": "Model attribute \"myobj.defint\" is not allowed to have a default value since it is not a scalar.",
  "subject": "/model",
  "args": {
    "name": "myobj.defint"
  },
  "source": ":registry:shared_model:3535"
}
`)

	// Finally, the fully-correct model: defstring + myobj.defint, both
	// with good defaults. This PUT /modelsource also forces a
	// revalidation of the existing registry data, which applies the new
	// defaults and saves - bumping the entity epoch from 1 to 2 with no
	// separate write needed (replaces the old Touch()/ValidateAndSave()
	// Go-API-only mechanism).
	model = `{
  "attributes": {
    "defstring": {
      "name": "defstring",
      "type": "string",
      "required": true,
      "default": "hello"
    },
    "myobj": {
      "name": "myobj",
      "type": "object",
      "attributes": {
        "defint": {
          "name": "defint",
          "type": "integer",
          "required": true,
          "default": 123
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "GET", "/", "", 200, `{
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

	XHTTP(t, reg, "PUT", "/", "{}", 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryDefaultFields",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 4,
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
  "epoch": 5,
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
  "epoch": 6,
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
  "epoch": 7,
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
  "epoch": 8,
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
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestRegistryRoot",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "2026-05-15T19:51:47.726218335Z",
  "modifiedat": "2026-05-15T19:51:47.726218335Z",

  "capabilities": {
    "available": {
      ".xregistry": {
        "mutable": false
      },
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
      "`+SPECVERSION+`"
    ],
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
    "`+SPECVERSION+`"
  ],
  "versionmodes": [
    "manual"
  ]
}
`)

	// Minimal + epoch=2
	XHTTP(t, reg, "GET", "/?inline=capabilities,modelsource", ``, 200,
		`{
  "specversion": "`+SPECVERSION+`",
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
      "`+SPECVERSION+`"
    ],
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
  "specversion": "`+SPECVERSION+`",
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
      "`+SPECVERSION+`"
    ],
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
  "specversion": "`+SPECVERSION+`",
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
      "`+SPECVERSION+`"
    ],
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

// Demonstrates a gap in Registry.VerifyData(): it never runs
// Group.Validate() (the "equals"/"enum" cross-attribute constraint
// checker) itself. VerifyData()'s resource loop marks each Resource's
// owning Group via AddGroupToValidate() (inside ValidateResource()), but
// VerifyData() never drains tx.GroupsToValidate - that only happens via
// Registry.Validate(), which VerifyData() never calls. So a model change
// that leaves EXISTING data non-compliant with a cross-attribute "equals"
// group constraint is only caught "by accident", whenever something else
// later happens to call Registry.Validate()/Tx.Validate() (e.g. HTTP's
// SerializeQuery(), HTTPPUTModelSource()'s explicit drain, or
// Tx.SaveAll()/SaveAllAndCommit()).
//
// "equals" is used (rather than "enum") because enum constraints get
// merged into the attribute's own static validation rules (see
// GetConstraints()), so ordinary per-Version attribute validation
// independently (and coincidentally) catches enum violations, masking
// this gap. "equals" is cross-entity/dynamic (compares to another live
// Group attribute's current value) and can ONLY be caught by the
// explicit Group.Validate() function.
//
// Exercises 3 variants of "how the model change reaches the server",
// reusing the same Registry, resetting the data (DELETE /dirs) and
// model (PUT /modelsource) back to the "v1" baseline between each one.
func TestRegistryVerifyDataMissesGroupConstraintOnModelChange(t *testing.T) {
	reg := NewRegistry("TestRegistryVerifyDataMissesGroupConstraintOnModelChange")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
      "groups": { "dirs": { "singular": "dir",
        "attributes": {
          "gstr": { "type": "string" },
          "gstr2": { "type": "string" }
        },
        "constraints": { "files.mystr": { "equals": "gstr" } },
        "resources": { "files": { "singular": "file",
          "attributes": { "mystr": { "type": "string" } } } } } } }`

	newModelSrc := `{
      "groups": { "dirs": { "singular": "dir",
        "attributes": {
          "gstr": { "type": "string" },
          "gstr2": { "type": "string" }
        },
        "constraints": { "files.mystr": { "equals": "gstr2" } },
        "resources": { "files": { "singular": "file",
          "attributes": { "mystr": { "type": "string" } } } } } } }`

	// v1 model+data (via real HTTP, so it's fully committed): valid -
	// f1.mystr ("foo") matches d1.gstr ("foo").
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource":`+
		modelSrc+`,
      "dirs": { "d1": { "gstr": "foo", "gstr2": "bar",
        "files": { "f1": { "mystr": "foo" } } } } }`,
		200, `*`)

	// Variant 1: pure Go API, no HTTP at all - Model.ApplyNewModelFromJSON()
	// directly, with no subsequent SaveAllAndCommit()/Registry.Validate()
	// call. This is the clearest, most direct reproduction: nothing else
	// in this call chain happens to drain tx.GroupsToValidate.

	// v2 model (pure Go API): retarget the constraint to gstr2.
	// f1.mystr ("foo") no longer equals d1.gstr2 ("bar") - a genuine
	// violation that ApplyNewModelFromJSON() should catch/reject.
	xErr := reg.Model.ApplyNewModelFromJSON([]byte(newModelSrc), true)
	XCheckErr(t, xErr, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"mystr\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "kind": "equals",
    "path": "mystr"
  },
  "source": ":registry:group:895"
}`)
	reg.Rollback()

	// Variant 2: single combined "PUT /" request containing BOTH the new
	// modelsource AND (unrelated) data changes in the same body - the
	// case the user specifically flagged: model needs to be applied
	// (and, eventually, verified) but not before any new data in the
	// same request has been loaded, since that new data may itself need
	// to be checked too. Registry.Update() currently defers VerifyData()
	// until after all incoming collections are upserted (see its
	// "modelchanged" check) - this variant confirms whether that
	// existing deferral correctly catches the violation once data
	// loading is done.

	// Single PUT / with the new modelsource AND an unrelated data
	// change (a second, harmless Resource) in the same request body.
	// f1 (untouched by this request) still violates the new
	// constraint once gstr2 becomes the target.
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource":`+
		newModelSrc+`,
      "dirs": { "d1": { "gstr": "foo", "gstr2": "bar",
        "files": {
          "f1": { "mystr": "foo" },
          "f2": { "mystr": "bar" }
        } } } }`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"mystr\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "kind": "equals",
    "path": "mystr"
  },
  "source": ":registry:group:895"
}
`)

	// Reset data+model back to the "v1" baseline for variant 3.
	XHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource":`+
		modelSrc+`,
      "dirs": { "d1": { "gstr": "foo", "gstr2": "bar",
        "files": { "f1": { "mystr": "foo" } } } } }`,
		200, `*`)

	// Variant 3: PUT /modelsource only (no data in the request body) -
	// the endpoint dedicated to model-only updates.
	XHTTP(t, reg, "PUT", "/modelsource", newModelSrc, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"mystr\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "kind": "equals",
    "path": "mystr"
  },
  "source": ":registry:group:895"
}
`)
}
