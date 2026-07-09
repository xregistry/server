package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	// "github.com/xregistry/server/registry"
	// log "github.com/duglin/dlog"
)

func TestConstraintsMatchVersions(t *testing.T) {
	reg := NewRegistry("TestConstraintsMatchVersions")
	defer PassDeleteReg(t, reg)

	// First a basic good test
	modelSrc := `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "mystr": {
                  "type": "string",
                  "matchversions": true
                } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Now do some error checking on model definitions

	// attr must be a scalar
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "myobj": {
                  "type": "object",
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.myobj\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.myobj\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:3632"
}`)

	// map->obj bad
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "object" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.mymap\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.mymap\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:3632"
}`)

	// map->int is bad too
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "integer" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.mymap\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.mymap\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:3632"
}`)

	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "myarray": {
                  "type": "array",
                  "item": { "type": "integer" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.myarray\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.myarray\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:3632"
}`)

	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "myany": {
                  "type": "any",
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.myany\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.myany\" is not allowed to have \"matchversions\" set to \"true\" due to it not being a scalar attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:3632"
}`)

	// Must not be under arrays
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "myarray": {
                  "type": "array",
                  "item": { "type": "object",
                            "attributes": { "myint": {
                              "type": "integer", "matchversions": true }}}
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.myarray.item.myint\" is not allowed to have \"matchversions\" set to \"true\" due to it being in an array.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.myarray.item.myint\" is not allowed to have \"matchversions\" set to \"true\" due to it being in an array"
  },
  "source": "3ba414aa22c1:registry:shared_model:3640"
}`)

	// Must not be under maps
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "object",
                            "attributes": { "myint": {
                              "type": "integer", "matchversions": true }}}
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.mymap.item.myint\" is not allowed to have \"matchversions\" set to \"true\" due to it being in a map.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.mymap.item.myint\" is not allowed to have \"matchversions\" set to \"true\" due to it being in a map"
  },
  "source": "3ba414aa22c1:registry:shared_model:3648"
}`)

	// Must not be under ifvalues
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "myint": {
                  "type": "integer",
                  "ifvalues": { "5": {
                      "siblingattributes": {
                        "mystr": { "type": "string", "matchversions": true }
                      } } } }
                } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.myint.ifvalues.5.mystr\" is not allowed to have \"matchversions\" set to \"true\" due to it being in an \"ifvalues\" clause.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.myint.ifvalues.5.mystr\" is not allowed to have \"matchversions\" set to \"true\" due to it being in an \"ifvalues\" clause"
  },
  "source": "3ba414aa22c1:registry:shared_model:3613"
}`)

	// Must not be under "*"
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "attributes": {
                "*": { "type": "integer", "matchversions": true }
                } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.*\" is not allowed to have \"matchversions\" set to \"true\" due to it being in a \"*\" extension.",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.*\" is not allowed to have \"matchversions\" set to \"true\" due to it being in a \"*\" extension"
  },
  "source": "3ba414aa22c1:registry:shared_model:3621"
}`)

	// Now let's do some positive testing
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myint": { "type": "integer", "matchversions": true },
                "mystr": { "type": "string", "matchversions": true }
              } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{
      "myint": 5
    }`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-06-17T01:30:46.61369913Z",
  "modifiedat": "2026-06-17T01:30:46.61369913Z",
  "ancestor": "v1",
  "myint": 5
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{
      "myint": 5
    }`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "xid": "/dirs/d1/files/f1/versions/v2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-06-17T01:30:46.61369913Z",
  "modifiedat": "2026-06-17T01:30:46.61369913Z",
  "ancestor": "v1",
  "myint": 5
}
`)

	// Try integers
	// Unique: 5,0  Empty: 0
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{
      "myint": 0
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myint\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myint"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Unique: 1  Empty: 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{
      "myint": null
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myint\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myint"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Try strings
	// Unique: 1  Empty: 0
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v1": { "mystr": "hello" },
      "v2": { "mystr": "hello" }
    }`, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 2,
    "isdefault": false,
    "createdat": "2026-06-18T17:36:37.441821397Z",
    "modifiedat": "2026-06-18T17:36:37.574269417Z",
    "ancestor": "v1",
    "myint": 5,
    "mystr": "hello"
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 2,
    "isdefault": true,
    "createdat": "2026-06-18T17:36:37.493260018Z",
    "modifiedat": "2026-06-18T17:36:37.574269417Z",
    "ancestor": "v1",
    "myint": 5,
    "mystr": "hello"
  }
}
`)

	// Unique: hello,bye  Empty: 1
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v2": { "mystr": "bye" }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"mystr\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "mystr"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Unique: bye  Empty: 1
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v1": { "mystr": null }
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"mystr\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "mystr"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now let's test some complex types
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file",
              "hasdocument": false,
              "attributes": {
                "myint": { "type": "integer", "matchversions": true },
                "myobj": {
                  "type": "object",
                  "attributes": {
                    "mystr": { "type": "string", "matchversions": true },
                    "intobj": {
                      "type": "object",
                      "attributes": {
                        "int2": {
                          "type": "integer",
                          "matchversions": true
                        }
                      }
                    }
                  }
                }
              } } } } } }`
	// Should fail due to the model not matching the data
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_attribute",
  "title": "An unknown attribute (mystr) was specified for \"/dirs/d1/files/f1/versions/v1\".",
  "subject": "/dirs/d1/files/f1/versions/v1",
  "args": {
    "name": "mystr"
  },
  "source": "3ba414aa22c1:registry:entity:2508"
}`)

	// Delete data
	XHTTP(t, reg, "DELETE", "/dirs/d1", ``, 204, ``)

	// Should work this time
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Set all of them at once - just v1
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": { "myint": 3, "myobj": { "mystr": "hi", "intobj": { "int2": 2 }}}
    }`, 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-06-17T14:54:56.889629554Z",
    "modifiedat": "2026-06-17T14:54:56.889629554Z",
    "ancestor": "v1",
    "myint": 3,
    "myobj": {
      "intobj": {
        "int2": 2
      },
      "mystr": "hi"
    }
  }
}
`)

	// Do the same for v2 - should work
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 3, "myobj": { "mystr": "hi", "intobj": { "int2": 2 }}}
    }`, 200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-06-17T14:54:56.889629554Z",
    "modifiedat": "2026-06-17T14:54:56.889629554Z",
    "ancestor": "v1",
    "myint": 3,
    "myobj": {
      "intobj": {
        "int2": 2
      },
      "mystr": "hi"
    }
  }
}
`)

	// Now fail doing one level at a time - diff values
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 1, "myobj": { "mystr": "hi", "intobj": { "int2": 2 }}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myint\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myint"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 3, "myobj": { "mystr": "bye", "intobj": { "int2": 2 }}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myobj.mystr\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myobj.mystr"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
	  "v2": { "myint": 3, "myobj": { "mystr": "hi", "intobj": { "int2": 3 }}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myobj.intobj.int2\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 2. Versions w/o values: 0.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myobj.intobj.int2"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now fail doing one level at a time - missing values
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myobj": { "mystr": "hi", "intobj": { "int2": 2 }}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myint\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myint"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 3, "myobj": { "intobj": { "int2": 2 }}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myobj.mystr\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myobj.mystr"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
	  "v2": { "myint": 3, "myobj": { "mystr": "hi"}}
    }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#mismatched_version_attribute",
  "title": "The request would cause the \"myobj.intobj.int2\" attribute across the Versions of \"/dirs/d1/files/f1\" to be different.",
  "detail": "Unique values: 1. Versions w/o values: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "name": "myobj.intobj.int2"
  },
  "source": "3ba414aa22c1:registry:resource:2081"
}
`)

}

func TestConstraintsGroupTypeErrors(t *testing.T) {
	reg := NewRegistry("TestConstraintsGroupTypeErrors")
	defer PassDeleteReg(t, reg)

	// First a basic good test
	modelSrc := `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": {
                  "default": "xyz",
                  "enum": [ "xyz", "abc" ],
                  "equals": "name"
                }
              },
              "resources": {"files": {"singular": "file", "hasdocument": false,
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": false
                  }
                } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	XHTTP(t, reg, "GET", "/model", ``, 200, `{
  "attributes": {
    "specversion": {
      "name": "specversion",
      "type": "string",
      "readonly": true,
      "required": true
    },
    "registryid": {
      "name": "registryid",
      "type": "string",
      "matchcase": true,
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "shortself": {
      "name": "shortself",
      "type": "url",
      "readonly": true,
      "immutable": true
    },
    "xid": {
      "name": "xid",
      "type": "xid",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "icon": {
      "name": "icon",
      "type": "url"
    },
    "labels": {
      "name": "labels",
      "type": "map",
      "item": {
        "type": "string"
      }
    },
    "createdat": {
      "name": "createdat",
      "type": "timestamp",
      "required": true
    },
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp",
      "required": true
    },
    "capabilities": {
      "name": "capabilities",
      "type": "object",
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "model": {
      "name": "model",
      "type": "object",
      "readonly": true,
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "modelsource": {
      "name": "modelsource",
      "type": "object",
      "attributes": {
        "*": {
          "name": "*",
          "type": "any"
        }
      }
    },
    "dirsurl": {
      "name": "dirsurl",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "dirscount": {
      "name": "dirscount",
      "type": "uinteger",
      "readonly": true,
      "required": true
    },
    "dirs": {
      "name": "dirs",
      "type": "map",
      "item": {
        "type": "object",
        "attributes": {
          "*": {
            "name": "*",
            "type": "any"
          }
        }
      }
    }
  },
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
      "attributes": {
        "dirid": {
          "name": "dirid",
          "type": "string",
          "matchcase": true,
          "immutable": true,
          "required": true
        },
        "self": {
          "name": "self",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "shortself": {
          "name": "shortself",
          "type": "url",
          "readonly": true,
          "immutable": true
        },
        "xid": {
          "name": "xid",
          "type": "xid",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "epoch": {
          "name": "epoch",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "name": {
          "name": "name",
          "type": "string"
        },
        "description": {
          "name": "description",
          "type": "string"
        },
        "documentation": {
          "name": "documentation",
          "type": "url"
        },
        "icon": {
          "name": "icon",
          "type": "url"
        },
        "labels": {
          "name": "labels",
          "type": "map",
          "item": {
            "type": "string"
          }
        },
        "createdat": {
          "name": "createdat",
          "type": "timestamp",
          "required": true
        },
        "modifiedat": {
          "name": "modifiedat",
          "type": "timestamp",
          "required": true
        },
        "deprecated": {
          "name": "deprecated",
          "type": "object",
          "attributes": {
            "alternative": {
              "name": "alternative",
              "type": "url"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "effective": {
              "name": "effective",
              "type": "timestamp"
            },
            "removal": {
              "name": "removal",
              "type": "timestamp"
            },
            "*": {
              "name": "*",
              "type": "any"
            }
          }
        },
        "constraints": {
          "name": "constraints",
          "type": "map",
          "item": {
            "type": "object",
            "attributes": {
              "default": {
                "name": "default",
                "type": "any"
              },
              "enum": {
                "name": "enum",
                "type": "array",
                "item": {
                  "type": "any"
                }
              },
              "equals": {
                "name": "equals",
                "type": "string"
              }
            }
          }
        },
        "filesurl": {
          "name": "filesurl",
          "type": "url",
          "readonly": true,
          "immutable": true,
          "required": true
        },
        "filescount": {
          "name": "filescount",
          "type": "uinteger",
          "readonly": true,
          "required": true
        },
        "files": {
          "name": "files",
          "type": "map",
          "item": {
            "type": "object",
            "attributes": {
              "*": {
                "name": "*",
                "type": "any"
              }
            }
          }
        }
      },
      "constraints": {
        "files.mystr": {
          "default": "xyz",
          "enum": [
            "xyz",
            "abc"
          ],
          "equals": "name"
        }
      },
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": false,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": false,
          "validatecompatibility": false,
          "strictvalidation": false,
          "attributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
              "matchcase": true,
              "immutable": true,
              "required": true
            },
            "versionid": {
              "name": "versionid",
              "type": "string",
              "matchcase": true,
              "immutable": true,
              "required": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
            },
            "xid": {
              "name": "xid",
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "name": {
              "name": "name",
              "type": "string"
            },
            "isdefault": {
              "name": "isdefault",
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
            },
            "description": {
              "name": "description",
              "type": "string"
            },
            "documentation": {
              "name": "documentation",
              "type": "url"
            },
            "icon": {
              "name": "icon",
              "type": "url"
            },
            "labels": {
              "name": "labels",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "createdat": {
              "name": "createdat",
              "type": "timestamp",
              "required": true
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp",
              "required": true
            },
            "ancestor": {
              "name": "ancestor",
              "type": "string",
              "matchcase": true,
              "required": true
            },
            "contenttype": {
              "name": "contenttype",
              "type": "string"
            },
            "format": {
              "name": "format",
              "type": "string"
            },
            "formatvalidated": {
              "name": "formatvalidated",
              "type": "boolean",
              "readonly": true
            },
            "formatvalidatedreason": {
              "name": "formatvalidatedreason",
              "type": "string",
              "readonly": true
            },
            "compatibilityvalidated": {
              "name": "compatibilityvalidated",
              "type": "boolean",
              "readonly": true
            },
            "compatibilityvalidatedreason": {
              "name": "compatibilityvalidatedreason",
              "type": "string",
              "readonly": true
            },
            "mystr": {
              "name": "mystr",
              "type": "string",
              "enum": [
                "abc",
                "def",
                "ghi"
              ],
              "strict": false
            }
          },
          "resourceattributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
              "matchcase": true,
              "immutable": true,
              "required": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
            },
            "xid": {
              "name": "xid",
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "metaurl": {
              "name": "metaurl",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "meta": {
              "name": "meta",
              "type": "object",
              "attributes": {
                "*": {
                  "name": "*",
                  "type": "any"
                }
              }
            },
            "versionsurl": {
              "name": "versionsurl",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "versionscount": {
              "name": "versionscount",
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "versions": {
              "name": "versions",
              "type": "map",
              "item": {
                "type": "object",
                "attributes": {
                  "*": {
                    "name": "*",
                    "type": "any"
                  }
                }
              }
            }
          },
          "metaattributes": {
            "fileid": {
              "name": "fileid",
              "type": "string",
              "matchcase": true,
              "immutable": true,
              "required": true
            },
            "self": {
              "name": "self",
              "type": "url",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "shortself": {
              "name": "shortself",
              "type": "url",
              "readonly": true,
              "immutable": true
            },
            "xid": {
              "name": "xid",
              "type": "xid",
              "readonly": true,
              "immutable": true,
              "required": true
            },
            "xref": {
              "name": "xref",
              "type": "url"
            },
            "epoch": {
              "name": "epoch",
              "type": "uinteger",
              "readonly": true,
              "required": true
            },
            "labels": {
              "name": "labels",
              "type": "map",
              "item": {
                "type": "string"
              }
            },
            "createdat": {
              "name": "createdat",
              "type": "timestamp",
              "required": true
            },
            "modifiedat": {
              "name": "modifiedat",
              "type": "timestamp",
              "required": true
            },
            "readonly": {
              "name": "readonly",
              "type": "boolean",
              "readonly": true,
              "required": true,
              "default": false
            },
            "compatibility": {
              "name": "compatibility",
              "type": "string",
              "enum": [
                "backward",
                "backward_transitive",
                "forward",
                "forward_transitive",
                "full",
                "full_transitive"
              ],
              "strict": true
            },
            "deprecated": {
              "name": "deprecated",
              "type": "object",
              "attributes": {
                "alternative": {
                  "name": "alternative",
                  "type": "url"
                },
                "documentation": {
                  "name": "documentation",
                  "type": "url"
                },
                "effective": {
                  "name": "effective",
                  "type": "timestamp"
                },
                "removal": {
                  "name": "removal",
                  "type": "timestamp"
                },
                "*": {
                  "name": "*",
                  "type": "any"
                }
              }
            },
            "defaultversionid": {
              "name": "defaultversionid",
              "type": "string",
              "matchcase": true,
              "required": true
            },
            "defaultversionurl": {
              "name": "defaultversionurl",
              "type": "url",
              "readonly": true,
              "required": true
            },
            "defaultversionsticky": {
              "name": "defaultversionsticky",
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}
`)

	// Let's test some error cases first

	// Missing period in key
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "filesmystr": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"filesmystr\" has an invalid key. It must be of the form \"<RESOURCES>.<PATH>\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"filesmystr\" has an invalid key. It must be of the form \"<RESOURCES>.<PATH>\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4095"
}`)

	// Unknown RESOURCES
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "datas.files": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"datas.files\" has an unknown Resource type \"datas\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"datas.files\" has an unknown Resource type \"datas\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4103"
}`)

	// Bad path - no attr
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.\" has an empty reference to an attribute.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.\" has an empty reference to an attribute"
  },
  "source": "3ba414aa22c1:registry:shared_model:4117"
}`)

	// Bad path - syntax error
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.fh'213'": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.fh'213'\" has an invalid path (fh'213'): Unexpected \"'\" in \"fh'213'\" at pos 3.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.fh'213'\" has an invalid path (fh'213'): Unexpected \"'\" in \"fh'213'\" at pos 3"
  },
  "source": "3ba414aa22c1:registry:shared_model:4111"
}`)

	// Unknown attribute - root
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.foo\" has an invalid path (foo): Attribute \"foo\" can not be found.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.foo\" has an invalid path (foo): Attribute \"foo\" can not be found"
  },
  "source": "3ba414aa22c1:registry:shared_model:4125"
}`)

	// Unknown attribute - in obj
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {
                    "type": "string",
                    "enum": [ "abc", "def", "ghi" ],
                    "strict": true
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr.foo\" has an invalid path (mystr.foo): Attribute \"mystr\" is scalar, so \"foo\" is invalid.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr.foo\" has an invalid path (mystr.foo): Attribute \"mystr\" is scalar, so \"foo\" is invalid"
  },
  "source": "3ba414aa22c1:registry:shared_model:4125"
}`)

	// Unknown attr, step into map - even though it's not valid for constraints
	// just test our infra
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mymap.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mymap": {
                    "type": "map",
                    "item": {
                      "type": "string"
                    }
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mymap.foo\" has an invalid path (mymap.foo): Attribute \"foo\" can not be found in \"mymap\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mymap.foo\" has an invalid path (mymap.foo): Attribute \"foo\" can not be found in \"mymap\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4125"
}`)

	// Valid attr, but Step into map - which is not allowed
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mymap.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mymap": {
                    "type": "map",
                    "item": {
                      "type": "object",
                      "attributes": {
                        "foo": {
                          "type": "integer"
                        }
                      }
                    }
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mymap.foo\" has a path (mymap.foo) that includes a map (mymap), which is not allowed.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mymap.foo\" has a path (mymap.foo) that includes a map (mymap), which is not allowed"
  },
  "source": "3ba414aa22c1:registry:shared_model:4139"
}`)

	// Unknown attr, step into array-even though it's not valid for constraints
	// just test our infra
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.myarray.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "myarray": {
                    "type": "array",
                    "item": {
                      "type": "object"
                    }
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.myarray.foo\" has an invalid path (myarray.foo): Attribute \"foo\" can not be found in \"myarray\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.myarray.foo\" has an invalid path (myarray.foo): Attribute \"foo\" can not be found in \"myarray\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4125"
}`)

	// Step into array - not allowed
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.myarray.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "myarray": {
                    "type": "array",
                    "item": {
                      "type": "object",
                      "attributes": {
                        "foo": { "type": "integer" }
                      }
                    }
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.myarray.foo\" has a path (myarray.foo) that includes an array (myarray), which is not allowed.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.myarray.foo\" has a path (myarray.foo) that includes an array (myarray), which is not allowed"
  },
  "source": "3ba414aa22c1:registry:shared_model:4134"
}`)

	// Unknown step into empty object
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.myobj.foo": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "myobj": {
                    "type": "object"
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.myobj.foo\" has an invalid path (myobj.foo): Attribute \"foo\" can not be found in \"myobj\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.myobj.foo\" has an invalid path (myobj.foo): Attribute \"foo\" can not be found in \"myobj\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4125"
}`)

	// Stop on object
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.myobj": {}
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "myobj": {
                    "type": "object"
                  }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.myobj\" has an invalid path (myobj): \"myobj\" must be a scalar.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.myobj\" has an invalid path (myobj): \"myobj\" must be a scalar"
  },
  "source": "3ba414aa22c1:registry:shared_model:4133"
}`)

	// Validate Enum list - strict, can't extend
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc", "bye" ] }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": { "type": "string", "enum": [ "abc", "def" ] }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an enum value (bye) that isn't part of the inherited attribute's enum list (abc, def).",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an enum value (bye) that isn't part of the inherited attribute's enum list (abc, def)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4030"
}`)

	// Validate Enum list - not strict, can extend
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc", "bye" ] }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": { "type":"string", "enum":["abc"],"strict":false }
                } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Validate Enum list - empty enum is same as missing
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": []  }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": { "type":"string", "enum":["abc"]}
                } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Validate Enum list - can add even when no enum on base attr
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc" ]  }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": { "type":"string" }
                } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Validate Enum list - constraint.enum must include attr.default if no
	// new default is defined
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc", "bye" ] }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {"type":"string", "default":"def","required": true }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an enum set (abc, bye) that doesn't include the attribute's default value (def).",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an enum set (abc, bye) that doesn't include the attribute's default value (def)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4224"
}`)

	// Validate Enum list - constraint.enum must include attr.default if no
	// new default is defined
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc" ] }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {"type":"string", "default":"def",
                    "enum": [ "abc", "def" ], "required": true }
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an enum set (abc) that doesn't include the attribute's default value (def).",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an enum set (abc) that doesn't include the attribute's default value (def)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4224"
}`)

	// Validate Enum list - extend attr w/enum + default
	// new default is defined
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc", "def" ], "default": "def" }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {"type":"string"}
                } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Validate Enum list - all enum values of the right/same type
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "enum": [ "abc", 123 ] }
              },
              "resources": {"files": {"singular": "file",
                "attributes": {
                  "mystr": {"type":"string"}
                } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an enum value (123) that must be of type \"string\".",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an enum value (123) that must be of type \"string\""
  },
  "source": "3ba414aa22c1:registry:shared_model:4020"
}`)

	// Validate Equals - "" == ignore
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "equals": "" }
              },
              "resources": {"files": {"singular": "file",
                "attributes": { "mystr": {"type":"string"} } } } } } }`

	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Validate Equals - missing attr
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "equals": "foo" }
              },
              "resources": {"files": {"singular": "file",
                "attributes": { "mystr": {"type":"string"} } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an \"equals\" reference (foo) that isn't valid: Attribute \"foo\" can not be found.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an \"equals\" reference (foo) that isn't valid: Attribute \"foo\" can not be found"
  },
  "source": "3ba414aa22c1:registry:shared_model:4256"
}`)

	// Validate Equals - non-scalar attr
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "equals": "labels" }
              },
              "resources": {"files": {"singular": "file",
                "attributes": { "mystr": {"type":"string"} } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" has an \"equals\" reference (labels) that must be a scalar.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" has an \"equals\" reference (labels) that must be a scalar"
  },
  "source": "3ba414aa22c1:registry:shared_model:4264"
}`)

	// Validate Equals - must be same type
	modelSrc = `{
      "groups": { "dirs": {
              "singular": "dir",
              "constraints": {
                "files.mystr": { "equals": "epoch" }
              },
              "resources": {"files": {"singular": "file",
                "attributes": { "mystr": {"type":"string"} } } } } } }`

	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true), `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Group Type \"dirs\" constraint \"files.mystr\" references an attribute of type \"string\" but its \"equals\" (epoch) references an attribute of type \"uinteger\". They need to match.",
  "subject": "/model",
  "args": {
    "error_detail": "Group Type \"dirs\" constraint \"files.mystr\" references an attribute of type \"string\" but its \"equals\" (epoch) references an attribute of type \"uinteger\". They need to match"
  },
  "source": "3ba414aa22c1:registry:shared_model:4271"
}`)

	reg.Model.SetChanged(false)
}

func TestConstraintsGroupTypeRuntime(t *testing.T) {
	reg := NewRegistry("TestConstraintsGroupTypeRuntime")
	defer PassDeleteReg(t, reg)

	// Just a happy-path one first
	data := `{
      "modelsource": {
	    "groups": { "dirs": {
	      "singular": "dir",
          "attributes": { "gstr": { "type": "string" } },
	      "constraints": {
	        "files.myobj.objstr": {
	          "default": "ooo"
	        },
	        "files.mystr": {
	          "default": "ghi",
	          "enum": [ "abc", "ghi", "foo" ],
	          "equals": "gstr"
	        },
	        "files.format": {}
	      },
	      "resources": {"files": {"singular": "file", "hasdocument": false,
	        "attributes": {
              "myobj": {
                "type": "object",
                "attributes": { "objstr": { "type": "string" } }
              },
	          "mystr": { "type": "string" }}}}}}
      },
      "dirs": {
        "d1": {
          "gstr": "foo",
          "files": {
            "f1": {
              "versions": {
                "v1": {
                  "name": "d1",
                  "mystr": "foo",
                  "myobj": { "objstr": "f1.myobj.objstr" }
                },
                "v2": {
                  "name": "d1",
                  "mystr": "foo",
                  "myobj": { "objstr": "f1.myobj.objstr" }
                }
              }
            }
          }
        }
      }
    }`

	XHTTP(t, reg, "PUT", "/?inline=dirs.files", data, 200, `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestConstraintsGroupTypeRuntime",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 2,
  "createdat": "2026-06-22T20:37:39.962543564Z",
  "modifiedat": "2026-06-22T20:37:39.977265265Z",

  "dirsurl": "http://localhost:8181/dirs",
  "dirs": {
    "d1": {
      "dirid": "d1",
      "self": "http://localhost:8181/dirs/d1",
      "xid": "/dirs/d1",
      "epoch": 1,
      "createdat": "2026-06-22T20:37:39.977265265Z",
      "modifiedat": "2026-06-22T20:37:39.977265265Z",
      "gstr": "foo",

      "filesurl": "http://localhost:8181/dirs/d1/files",
      "files": {
        "f1": {
          "fileid": "f1",
          "versionid": "v2",
          "self": "http://localhost:8181/dirs/d1/files/f1",
          "xid": "/dirs/d1/files/f1",
          "epoch": 1,
          "name": "d1",
          "isdefault": true,
          "createdat": "2026-06-22T20:37:39.977265265Z",
          "modifiedat": "2026-06-22T20:37:39.977265265Z",
          "ancestor": "v1",
          "myobj": {
            "objstr": "f1.myobj.objstr"
          },
          "mystr": "foo",

          "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
          "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
          "versionscount": 2
        }
      },
      "filescount": 1
    }
  },
  "dirscount": 1
}
`)

	// Clear entities
	XHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	// make sure default from gm is used (attr w/ & w/o default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups": { "dirs": { "singular": "dir", "constraints": {
	    "files.name": { "default": "gmName" },
	    "files.description": { "default": "gmDescription" }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": { "type":"string","default":"joe","required":true }
      }}}}}},
      "dirs": { "d1": { "files": { "f1": {}}}} }`, 200,
		`^(?s)^.*name": "gmName.*description": "gmDescription`)

	// make sure enum from gm is used (attr w/o enum) - good val
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups": { "dirs": { "singular": "dir", "constraints": {
	    "files.name": { "enum": [ "a", "b" ] }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs": { "d1": { "files": { "f1": {"name":"a"}}}} }`, 200,
		`^(?s)^.*name": "a"`)

	// make sure enum from gm is used (attr w/o enum) - bad val
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups": { "dirs": { "singular": "dir", "constraints": {
	    "files.name": { "enum": [ "a", "b" ] }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs": { "d1": { "files": { "f1": {"name":"c"}}}} }`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"name\" for \"/dirs/d1/files/f1/versions/1\" is not valid: value (c) must be one of the enum values: a, b.",
  "subject": "/dirs/d1/files/f1/versions/1",
  "args": {
    "error_detail": "value (c) must be one of the enum values: a, b",
    "name": "name"
  },
  "source": "3ba414aa22c1:registry:entity:2945"
}
`)

	// test "equals", equals=empty string, no check, they differ
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups": { "dirs": { "singular": "dir", "constraints": {
	    "files.name":{"equals":""}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"n", "files":{"f1":{"name":"c"}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// test "equals", missing from group, no check
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"files":{"f1":{"name":"c"}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// test "equals", present in group but "", no check
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"","files":{"f1":{"name":"c"}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"name\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "path": "name"
  },
  "source": "3ba414aa22c1:registry:group:850"
}
`)

	// test "equals", present in group, same in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{"name":"c"}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// test "equals", present in group, diff in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{"name":"c2"}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"name\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "path": "name"
  },
  "source": "3ba414aa22c1:registry:group:850"
}
`)

	// test "equals", present in group, bad default in gm, missing in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"default":"d", "equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"name\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "path": "name"
  },
  "source": "3ba414aa22c1:registry:group:850"
}
`)

	// test "equals", present in group, no default in gm, missing in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d1/files/f1\" not being compliant with its owning Group's \"equals\" constraint for attribute \"name\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "path": "name"
  },
  "source": "3ba414aa22c1:registry:group:850"
}
`)

	// test "equals", present in group, same default in gm, missing in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"default":"c", "equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// test "equals", present in group, def in attr, missing in f1
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"default":"c", "equals":"name"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true}
      }}}}}},
      "dirs":{"d1":{"name":"c","files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// Prep updates
	XHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
        "files.name": { "default":"n","enum":["n","z"],"equals":"name"}
      },
	  "resources": {"files": {"singular":"file", "hasdocument":false,
        "attributes": {
      }}}}}}}`, 200, `*`)

	// Create group+file, g.name present
	XHTTP(t, reg, "PUT", "/?inline=dirs.files",
		`{"dirs":{"d1":{"name":"n","files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n",\n *"createdat.*"name": "n",\n *"isdefault"`)

	// Update f1, bad name
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"name": "y"}`, 400,
		`^(?s)^.*invalid_attribute.*value \(y\).* n, z`)

	// Update f1, good alt name, but fails "equals" constraint
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"name": "z"}`, 400,
		`^(?s)^.*constraint_failure.*\\"equals\\".*\\"name\\"`)

	// Update f1, no name
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 200,
		`^(?s)^.*"name": "n"`)

	// Update model in a valid way
	XHTTP(t, reg, "PUT", "/modelsource", `{
	  "groups":{"dirs":{"singular":"dir","constraints":{
        "files.name": { "default":"z"}
      },
	  "resources": {"files": {"singular":"file", "hasdocument":false,
        "attributes": {
      }}}}}}`, 200, `*`)

	// Update f1, with a val that used to be invalid
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"name":"y"}`, 200,
		`^(?s)^.*"name": "y"`)

	// Update model in a bad way
	XHTTP(t, reg, "PUT", "/modelsource", `{
	  "groups":{"dirs":{"singular":"dir","constraints":{
        "files.name": { "enum":["a","b"]}
      },
	  "resources": {"files": {"singular":"file", "hasdocument":false,
        "attributes": {
      }}}}}}`, 400,
		`^(?s)^.*invalid_attribute.*\\"name\\".*value \(y\).*a, b`)
}

func TestConstraintsGroupErrors(t *testing.T) {
	reg := NewRegistry("TestConstraintsGroupErrors")
	defer PassDeleteReg(t, reg)

	// g.default set, gm.enum set, not in enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["a"] }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"b"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has a default value (b) that isn't part of the base attribute's enum list (a).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has a default value (b) that isn't part of the base attribute's enum list (a)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4055"
}
`)

	// g.default set, gm.enum set, in enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["b"] }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"b"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "b"`)

	// g.default, enum set, not in enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"b","enum":["a"]}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has a default value (b) that isn't part of the base attribute's enum list (a).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has a default value (b) that isn't part of the base attribute's enum list (a)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4054"
}
`)

	// g.default, enum set, gm.enum set - not proper subset
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"enum":["b"]}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"b","enum":["a"]}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has an enum value (a) that isn't part of the inherited attribute's enum list (b).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has an enum value (a) that isn't part of the inherited attribute's enum list (b)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4030"
}
`)

	// g.enum set, gm.default set not in g.enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["b"] }
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["a"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has an enum value (a) that isn't part of the inherited attribute's enum list (b).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has an enum value (a) that isn't part of the inherited attribute's enum list (b)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4030"
}
`)

	// g.default set, no gm, attr.enum set - not in attr.enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "c"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string"}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["a"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has an enum set (a) that doesn't include the Group Model's default value (c) for the same constraint.",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has an enum set (a) that doesn't include the Group Model's default value (c) for the same constraint"
  },
  "source": "3ba414aa22c1:registry:shared_model:4071"
}
`)

	// g.enum set, no gm, attr.enum set - not proper subset
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true,"enum":["d"]}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["a"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.name\" default value \"c\" must be one of the specified enum values (d) since \"strict\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.name\" default value \"c\" must be one of the specified enum values (d) since \"strict\" is \"true\""
  },
  "source": "3ba414aa22c1:registry:shared_model:3468"
}
`)

	// g.enum set, no gm, attr.default set - not in g.enum, not strict - ok
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true,"enum":["d"], "strict": false}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["c"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// g.enum set, no gm, attr.default set - not in g.enum, strict
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true,"enum":["d"]}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["a"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: \"groups.dirs.resources.files.name\" default value \"c\" must be one of the specified enum values (d) since \"strict\" is \"true\".",
  "subject": "/model",
  "args": {
    "error_detail": "\"groups.dirs.resources.files.name\" default value \"c\" must be one of the specified enum values (d) since \"strict\" is \"true\""
  },
  "source": "3ba414aa22c1:registry:shared_model:3465"
}
`)

	// g.equals set, no gm - ok
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"equals":"name"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "c"`)

	// g.equals set, gm.equals="" - ok
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["a"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has an enum set (a) that doesn't include the attribute's default value (c).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has an enum set (a) that doesn't include the attribute's default value (c)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4080"
}
`)

	// g.equals set, gm.equals=".." diff - not the same
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": {"equals":"description"}
	  },
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": {"type":"string","default":"c","required":true}
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"equals":"name"}},
        "files":{"f1":{}}}} }`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#bad_request",
  "title": "Group \"/dirs/d1\" constraint \"files.name\" has an \"equals\" value (name) that differs from the one defined in the Group Type (description).",
  "subject": "/dirs/d1",
  "args": {
    "error_detail": "Group \"/dirs/d1\" constraint \"files.name\" has an \"equals\" value (name) that differs from the one defined in the Group Type (description)"
  },
  "source": "3ba414aa22c1:registry:shared_model:4099"
}
`)

}

func TestConstraintsGroupRuntime(t *testing.T) {
	reg := NewRegistry("TestConstraintsGroupRuntime")
	defer PassDeleteReg(t, reg)

	// g.default,enum,equals all set ; f1 present (not default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{
          "files.name":{"default":"n","enum":["n","z"],"equals":"name"}},
        "files":{"f1":{"name":"z"}}}} }`, 200,
		`^(?s)^.*"name": "z"`)

	// g.default,enum,equals all set ; f1 absent
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{
          "files.name":{"default":"n","enum":["n","z"],"equals":"name"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.default set ; f1 absent
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"n"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.default set ; f1 present (not default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"n"}},
        "files":{"f1":{"name":"z"}}}} }`, 200,
		`^(?s)^.*"name": "z"`)

	// g.default absent, gm.default set, f1 absent
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{}},
        "files":{"f1":{"name":"z"}}}} }`, 200,
		`^(?s)^.*"name": "z"`)

	// g.default present, gm.default set, f1 present (not default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": { "type": "string", "default": "n", "required": true }
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"z"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "z"`)

	// g.enum present, gm.default set, f1 absent
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
        "name": { "type": "string", "default": "n", "required": true }
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["n","z"]}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.default,enum present, f1 absent
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"n","enum":["n","z"]}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.default,enum present, f1 present (not default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"default":"n","enum":["n","z"]}},
        "files":{"f1":{"name":"z"}}}} }`, 200,
		`^(?s)^.*"name": "z"`)

	// g.enum present, f1 present (not in enum)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["n","z"]}},
        "files":{"f1":{"name":"c"}}}} }`, 400,
		`^(?s)^.*invalid_attribute.*value \(c\).* n, z".*"name": "name"`)

	// g.enum present, f1 present (in enum)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["n","z"]}},
        "files":{"f1":{"name":"n"}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.enum present, no defaults, f1 absent - create ok, update f1 w/bad
	// val then good val
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"enum":["n","z"]}},
        "files":{"f2":{"name":null}}}} }`, 200,
		`^(?s)^.*"epoch": 1,\n *"isdefault`) // no "name"

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{"name":"y"}`, 400,
		`^(?s)^.*invalid_attribute.*value \(y\) must.* n, z.*"name": "name"`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{"name":"n"}`, 200,
		`^(?s)^.*"name": "n"`)

	// g.equals present, no g.attr, f1 present
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "constraints":{"files.name":{"equals":"name"}},
        "files":{"f1":{"name":"n"}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// g.equals present, g.attr, f1 present/diff - err
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "name": "n",
        "constraints":{"files.name":{"equals":"name"}},
        "files":{"f1":{"name":"d"}}}} }`, 400,
		`^(?s)^.*constraint_failure.*"name"`)

	// g.equals present, g.attr, f1 present/same - ok
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "name": "n",
        "constraints":{"files.name":{"equals":"name"}},
        "files":{"f1":{"name":"n"}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// Prep for update file tests
	XHTTP(t, reg, "DELETE", "/dirs", ``, 204, ``)

	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{},
	  "resources": {"files": {"singular":"file", "hasdocument":false,
        "attributes": {
      }}}}}},
      "dirs":{"d1":{
        "name": "n",
        "constraints":{"files.name":{
          "default":"n", "enum": ["n", "z"], "equals":"name"}},
        "files":{"f1":{}}}} }`, 200,
		`^(?s)^.*"name": "n"`)

	// Update file - missing attr
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 200,
		`^(?s)^.*"name": "n"`)

	// Update file - bad attr
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"name":"y"}`, 400,
		`^(?s)^.*invalid_attribute.*value \(y\).* n, z`)

	// Update group - missing attr
	XHTTP(t, reg, "PATCH", "/dirs/d1", `{"name":null}`, 200,
		`^(?s)^.*"epoch": 2,\n  "createdat`) // No "name"

	// Update file - good non-default attr
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"name":"z"}`, 200,
		`^(?s)^.*"name": "z"`)

	// Update group - diff attr - err
	XHTTP(t, reg, "PATCH", "/dirs/d1?inline=files", `{"name":"n"}`, 400,
		`^(?s)^.*constraint_failure.*\\"equals\\".*\\"name\\"`)

	// Update group - matching attr
	XHTTP(t, reg, "PATCH", "/dirs/d1", `{"name":"z"}`, 200,
		`^(?s)^.*"name": "z"`)

}

// TestConstraintsLayering tests merging of group model (gm) and group instance
// (g) constraints when each contributes a different subset of
// {default, enum, equals}.
func TestConstraintsLayering(t *testing.T) {
	reg := NewRegistry("TestConstraintsLayering")
	defer PassDeleteReg(t, reg)

	// --- Case 1: gm.{default:"b"} + g.{enum:["a","b"]} ---
	// Success: gm.default is within g.enum, so resource gets "b"
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "b" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "constraints":{"files.name":{"enum":["a","b"]}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "b"`)

	// --- Case 2: gm.{default:"n"} + g.{equals:"name"} ---
	// Success: group.name matches gm.default so equals constraint is satisfied
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "n" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "n",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "n"`)

	// Error: group.name differs from gm.default (which is applied as default)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "n" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "x",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{}}}}}`, 400,
		`^(?s)^.*constraint_failure`)

	// No group.name: equals check skipped; resource still gets gm.default
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "n" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "n"`)

	// --- Case 3: gm.{enum:["a","b"]} + g.{equals:"name"} ---
	// Success: resource value in gm.enum and equals group.name
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["a","b"] }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "a",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{"name":"a"}}}}}`, 200,
		`^(?s)^.*"name": "a"`)

	// Error: resource value not in gm.enum (even if it matches group.name)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["a","b"] }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "c",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{"name":"c"}}}}}`, 400,
		`^(?s)^.*invalid_attribute`)

	// --- Case 4: gm.{equals:"name"} + g.{default:"n"} ---
	// Success: g.default matches group.name so equals constraint is satisfied
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "n",
	    "constraints":{"files.name":{"default":"n"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "n"`)

	// Error: g.default is applied but group.name differs
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "x",
	    "constraints":{"files.name":{"default":"n"}},
	    "files":{"f1":{}}}}}`, 400,
		`^(?s)^.*constraint_failure`)

	// No group.name: equals check skipped; resource gets g.default
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "constraints":{"files.name":{"default":"n"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "n"`)

	// --- Case 5: gm.{equals:"name"} + g.{enum:["a","b"]} ---
	// Success: resource in g.enum and equals group.name
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "a",
	    "constraints":{"files.name":{"enum":["a","b"]}},
	    "files":{"f1":{"name":"a"}}}}}`, 200,
		`^(?s)^.*"name": "a"`)

	// Error: resource value not in g.enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "c",
	    "constraints":{"files.name":{"enum":["a","b"]}},
	    "files":{"f1":{"name":"c"}}}}}`, 400,
		`^(?s)^.*invalid_attribute`)

	// --- Case 6: gm.{default:"a",enum:["a","b"]} + g.{equals:"name"} ---
	// Success: resource gets gm.default, in gm.enum, equals group.name
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "a", "enum": ["a","b"] }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "a",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "a"`)

	// Error: gm.default("a") applied but group.name("b") differs via equals
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "a", "enum": ["a","b"] }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "b",
	    "constraints":{"files.name":{"equals":"name"}},
	    "files":{"f1":{}}}}}`, 400,
		`^(?s)^.*constraint_failure`)

	// --- Case 7: gm.{default:"a",equals:"name"} + g.{enum:["a","b"]} ---
	// Success: g.enum includes gm.default; gm.equals check satisfied
	// (The error case where gm.default not in g.enum is tested in
	// TestConstraintsGroupErrors)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "default": "a", "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "a",
	    "constraints":{"files.name":{"enum":["a","b"]}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "a"`)

	// --- Case 8: gm.{enum:["a","b"],equals:"name"} + g.{default:"a"} ---
	// Success: g.default in gm.enum; equals check satisfied
	// (The error case where g.default not in gm.enum is tested in
	// TestConstraintsGroupErrors)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "enum": ["a","b"], "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "a",
	    "constraints":{"files.name":{"default":"a"}},
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"name": "a"`)

	// --- Empty equals in g with gm.equals set ---
	// g.equals="" is treated as "not set"; gm.equals still applies
	// Error: group.name exists and differs from resource value
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "x",
	    "constraints":{"files.name":{"equals":""}},
	    "files":{"f1":{"name":"y"}}}}}`, 400,
		`^(?s)^.*constraint_failure`)

	// Success: group.name matches resource value (gm.equals still applies via "")
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.name": { "equals": "name" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes":{}}}}}},
	  "dirs":{"d1":{
	    "name": "z",
	    "constraints":{"files.name":{"equals":""}},
	    "files":{"f1":{"name":"z"}}}}}`, 200,
		`^(?s)^.*"name": "z"`)
}

// TestConstraintsEqualsNestedGroupPath tests that 'equals' can reference a
// nested group attribute path (e.g., "gobj.gfoo") and is validated at model
// definition time and enforced at runtime.
func TestConstraintsEqualsNestedGroupPath(t *testing.T) {
	reg := NewRegistry("TestConstraintsEqualsNestedGroupPath")
	defer PassDeleteReg(t, reg)

	// Model validation: nested group attribute path for equals is accepted
	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": {
	      "gobj": {
	        "type": "object",
	        "attributes": { "gfoo": { "type": "string" } }
	      }
	    },
	    "constraints": {
	      "files.mystr": { "equals": "gobj.gfoo" }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "mystr": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Runtime: gobj.gfoo="abc", mystr="abc" => equals satisfied
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{
	    "gobj": {"gfoo": "abc"},
	    "files":{"f1":{"mystr":"abc"}}}}}`, 200,
		`^(?s)^.*"mystr": "abc"`)

	// Runtime: gobj.gfoo="abc", mystr="xyz" => equals fails
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{
	    "gobj": {"gfoo": "abc"},
	    "files":{"f1":{"mystr":"xyz"}}}}}`, 400,
		`^(?s)^.*constraint_failure`)

	// Runtime: gobj absent => equals check skipped
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{
	    "files":{"f1":{"mystr":"anything"}}}}}`, 200,
		`^(?s)^.*"mystr": "anything"`)

	// Model error: equals path traverses through ifvalues-only attribute
	// (gfoo is defined only via ifvalues siblingattributes, not static attrs)
	modelSrcBad := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": {
	      "gobj": {
	        "type": "object",
	        "ifvalues": {
	          "special": {
	            "siblingattributes": { "gfoo": { "type": "string" } }
	          }
	        }
	      }
	    },
	    "constraints": {
	      "files.mystr": { "equals": "gfoo" }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "mystr": { "type": "string" } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrcBad, true),
		`^(?s)^.*model_error.*equals.*gfoo.*can not be found`)

	reg.Model.SetChanged(false)
}

// TestConstraintsEqualsWildcardModelError tests that 'equals' referencing a
// wildcard ('*') group attribute is rejected at model definition time.
func TestConstraintsEqualsWildcardModelError(t *testing.T) {
	reg := NewRegistry("TestConstraintsEqualsWildcardModelError")
	defer PassDeleteReg(t, reg)

	// Model error: equals references "*" which is a wildcard, not a named attr
	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": {
	      "*": { "type": "string" }
	    },
	    "constraints": {
	      "files.mystr": { "equals": "someattr" }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "mystr": { "type": "string" } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s)^.*model_error.*equals.*someattr.*can not be found`)

	reg.Model.SetChanged(false)
}

// TestConstraintsMultipleResourceTypes verifies that constraints for different
// resource types within the same group are independent and don't bleed into
// each other.
func TestConstraintsMultipleResourceTypes(t *testing.T) {
	reg := NewRegistry("TestConstraintsMultipleResourceTypes")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "constraints": {
	      "files.mystr": {
	        "default": "file-default",
	        "enum": ["file-default","file-other"] },
	      "docs.mystr": {
	        "default": "doc-default",
	        "enum": ["doc-default","doc-other"] }
	    },
	    "resources": {
	      "files": {"singular": "file", "hasdocument": false,
	        "attributes": { "mystr": { "type": "string" } }},
	      "docs":  {"singular": "doc",  "hasdocument": false,
	        "attributes": { "mystr": { "type": "string" } }}
	    } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Each resource type gets its own default from constraints
	// (JSON output orders docs before files alphabetically)
	XHTTP(t, reg, "PUT", "/?inline=dirs.files,dirs.docs",
		`{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{
	    "files":{"f1":{}},
	    "docs":{"d1":{}}}}}`, 200,
		`^(?s)^.*"mystr": "doc-default".*"mystr": "file-default"`)

	// Constraints don't bleed: file with "doc-other" is rejected
	// (not in files enum)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details",
		`{"mystr":"doc-other"}`, 400,
		`^(?s)^.*invalid_attribute`)

	// Constraints don't bleed: doc with "file-other" is rejected
	// (not in docs enum)
	XHTTP(t, reg, "PUT", "/dirs/d1/docs/d2$details",
		`{"mystr":"file-other"}`, 400,
		`^(?s)^.*invalid_attribute`)

	// Valid values within each type's enum work fine
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f3$details",
		`{"mystr":"file-other"}`, 201,
		`^(?s)^.*"mystr": "file-other"`)
	XHTTP(t, reg, "PUT", "/dirs/d1/docs/d3$details",
		`{"mystr":"doc-other"}`, 201,
		`^(?s)^.*"mystr": "doc-other"`)

	// Group instance constraints also stay per-type
	// Use subsets of gm enums that include the gm defaults
	XHTTP(t, reg, "PUT", "/?inline=dirs.files,dirs.docs",
		`{"modelsource": `+modelSrc+`,
	  "dirs":{"d2":{
	    "constraints":{
	      "files.mystr":{"enum":["file-default","file-other"]},
	      "docs.mystr": {"enum":["doc-default","doc-other"]}
	    },
	    "files":{"f1":{"mystr":"file-other"}},
	    "docs":{"d1":{"mystr":"doc-other"}}}}}`, 200,
		`^(?s)^.*"mystr": "doc-other".*"mystr": "file-other"`)

	// Cross-type enum contamination would reject valid values
	// Restrict files in d3 to just "file-default"; "file-other" is rejected
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d3":{
	    "constraints":{
	      "files.mystr":{"enum":["file-default"]}
	    },
	    "files":{"f2":{"mystr":"file-other"}}}}}`, 400,
		`^(?s)^.*invalid_attribute`)

	reg.Model.SetChanged(false)
}

// TestConstraintsDefaultNotRequiresRequired verifies that a constraint default
// can be applied to a resource attribute that is not marked required at the
// model level.
func TestConstraintsDefaultNotRequiresRequired(t *testing.T) {
	reg := NewRegistry("TestConstraintsDefaultNotRequiresRequired")
	defer PassDeleteReg(t, reg)

	// Attribute is optional (no required, no model-level default)
	// Constraint default should still be applied when resource is created
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.optattr": { "default": "constrained-default" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes": {
	      "optattr": { "type": "string" }
	    }}}}}},
	  "dirs":{"d1":{
	    "files":{"f1":{}}}}}`, 200,
		`^(?s)^.*"optattr": "constrained-default"`)

	// Explicitly providing a value overrides the constraint default
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.optattr": { "default": "constrained-default" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes": {
	      "optattr": { "type": "string" }
	    }}}}}},
	  "dirs":{"d1":{
	    "files":{"f1":{"optattr":"my-value"}}}}}`, 200,
		`^(?s)^.*"optattr": "my-value"`)

	// Explicitly nulling the attribute is treated as absent, so
	// the constraint default is still applied
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.optattr": { "default": "constrained-default" }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes": {
	      "optattr": { "type": "string" }
	    }}}}}},
	  "dirs":{"d1":{
	    "files":{"f1":{"optattr":null}}}}}`, 200,
		`^(?s)^.*"optattr": "constrained-default"`)
}

// TestConstraintsDeepNestedPath tests constraint paths at 3 or more levels
// deep (e.g., files.a.b.c).
func TestConstraintsDeepNestedPath(t *testing.T) {
	reg := NewRegistry("TestConstraintsDeepNestedPath")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "constraints": {
	      "files.a.b.c": { "enum": ["x","y"] }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {
	        "a": {
	          "type": "object",
	          "attributes": {
	            "b": {
	              "type": "object",
	              "attributes": {
	                "c": { "type": "string" }
	              }
	            }
	          }
	        }
	      } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Valid: a.b.c value is in enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{"files":{"f1":{"a":{"b":{"c":"x"}}}}}}}`, 200,
		`^(?s)^.*"c": "x"`)

	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{"files":{"f1":{"a":{"b":{"c":"y"}}}}}}}`, 200,
		`^(?s)^.*"c": "y"`)

	// Error: a.b.c value not in enum
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{"files":{"f1":{"a":{"b":{"c":"z"}}}}}}}`, 400,
		`^(?s)^.*invalid_attribute`)

	// Model error: path stops at non-object
	modelSrcBadPath := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "constraints": {
	      "files.a.b.c.d": { "enum": ["x","y"] }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {
	        "a": {
	          "type": "object",
	          "attributes": {
	            "b": {
	              "type": "object",
	              "attributes": {
	                "c": { "type": "string" }
	              }
	            }
	          }
	        }
	      } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrcBadPath, true),
		`^(?s)^.*model_error.*a.b.c.d.*c.*scalar`)

	reg.Model.SetChanged(false)
}

// TestConstraintsGroupInstanceNewKey tests that a group instance can add a
// constraint key not present in the group model's constraints, and that both
// gm and g constraints apply independently.
func TestConstraintsGroupInstanceNewKey(t *testing.T) {
	reg := NewRegistry("TestConstraintsGroupInstanceNewKey")
	defer PassDeleteReg(t, reg)

	// gm has a constraint for files.mystr; g adds a NEW key files.myint
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": {
	  "groups":{"dirs":{"singular":"dir","constraints":{
	    "files.mystr": { "enum": ["a","b"] }
	  },
	  "resources": {"files": {"singular":"file","hasdocument":false,
	    "attributes": {
	      "mystr": { "type": "string" },
	      "myint": { "type": "integer" }
	    }}}}}},
	  "dirs":{"d1":{
	    "constraints":{
	      "files.myint": {"default": 5}
	    },
	    "files":{"f1":{"mystr":"a"}}}}}`, 200,
		`^(?s)^.*"myint": 5.*"mystr": "a"`)

	// Both constraints apply: gm.enum for mystr
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{"mystr":"c"}`, 400,
		`^(?s)^.*invalid_attribute`)

	// Both constraints apply: g.default for myint even with explicit mystr
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2$details", `{"mystr":"b"}`, 201,
		`^(?s)^.*"myint": 5.*"mystr": "b"`)
}

// TestConstraintsMatchVersionsWithEquals tests that matchversions and an equals
// constraint can coexist on the same attribute and are both independently
// enforced.
func TestConstraintsMatchVersionsWithEquals(t *testing.T) {
	reg := NewRegistry("TestConstraintsMatchVersionsWithEquals")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": { "gattr": { "type": "string" } },
	    "constraints": {
	      "files.myattr": { "equals": "gattr" }
	    },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {
	        "myattr": { "type": "string", "matchversions": true }
	      } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Setup: group gattr="x", resource f1 with myattr="x" - both checks pass
	XHTTP(t, reg, "PUT", "/?inline=dirs.files", `{"modelsource": `+modelSrc+`,
	  "dirs":{"d1":{
	    "gattr": "x",
	    "files":{"f1":{"versions":{"v1":{"myattr":"x"}}}}}}}`, 200,
		`^(?s)^.*"myattr": "x"`)

	// matchversions: trying to create v2 with a different myattr value fails
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details",
		`{"myattr":"y"}`, 400,
		`^(?s)^.*mismatched_version_attribute`)

	// matchversions: creating v2 with the same value is ok
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2$details",
		`{"myattr":"x"}`, 201,
		`^(?s)^.*"myattr": "x"`)

	// equals: updating group gattr to a different value triggers
	// constraint_failure
	XHTTP(t, reg, "PATCH", "/dirs/d1?inline=files", `{"gattr":"y"}`, 400,
		`^(?s)^.*constraint_failure`)

	// equals: updating group gattr to match resource value is ok
	XHTTP(t, reg, "PATCH", "/dirs/d1", `{"gattr":"x"}`, 200,
		`^(?s)^.*"gattr": "x"`)

	reg.Model.SetChanged(false)
}

// TestConstraintsXref tests that constraint defaults are NOT applied to xref'd
// resources (per spec). Note: per-spec, enum and equals SHOULD be enforced for
// xref'd resources, but that enforcement is not yet implemented.
//
// The source resource lives in a group instance with NO constraint so its
// "name" attribute is genuinely absent. The xref resource lives in a group
// instance that HAS a constraint default. If the default were incorrectly
// applied to the xref, "name" would appear in the response.
func TestConstraintsXref(t *testing.T) {
	reg := NewRegistry("TestConstraintsXref")
	defer PassDeleteReg(t, reg)

	// No group-TYPE constraint; constraint lives only on group instance d2.
	// This ensures d1/s1 has no "name" when created with {}.
	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {} } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Create source resource s1 in d1 (no constraint on d1) - name stays absent
	XHTTP(t, reg, "PUT", "/dirs/d1/files/s1$details", `{}`, 201,
		`^(?s)^.*"epoch": 1,\n *"isdefault`) // no "name"

	// Create d2 with a group-instance constraint default for files.name
	XHTTP(t, reg, "PATCH", "/dirs/d2",
		`{"constraints":{"files.name":{"default":"constrained-default"}}}`, 201, `*`)

	// Confirm the constraint default IS applied to a normal resource in d2
	XHTTP(t, reg, "PUT", "/dirs/d2/files/f1$details", `{}`, 201,
		`^(?s)^.*"name": "constrained-default"`)

	// Create xref resource fx in d2 pointing to s1 (which has no name)
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/s1"}`, 201, `*`)

	// GET fx: constraint default MUST NOT be applied to the xref'd resource.
	// "name" appears between "epoch" and "isdefault" in the JSON output, so
	// the pattern below verifies "name" is absent.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`^(?s)^.*"epoch": 1,\n *"isdefault`) // no "name"

	// Make sure our model validation is ok too - we blew up at one point
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))
}
