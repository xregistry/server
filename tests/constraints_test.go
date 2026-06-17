package tests

import (
	// "fmt"
	. "github.com/xregistry/server/common"
	"testing"
	// "github.com/xregistry/server/registry"
)

func TestConstraintsMatchVersions(t *testing.T) {
	reg := NewRegistry("TestConstraintsMatchVersions")
	defer PassDeleteReg(t, reg)

	// First a basic good test
	modelSrc := `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "mystr": {
                  "type": "string",
                  "matchversions": true
                } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true)) // verifyData=false

	// Now do some error checking on model definitions

	// attr must be a scalar
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myobj": {
                  "type": "object",
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.myobj.*scalar.*`)

	// map->obj bad
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "object" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.mymap.*scalar.*`)

	// map->int is bad too
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "integer" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.mymap.*scalar.*`)

	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myarray": {
                  "type": "array",
                  "item": { "type": "integer" },
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.myarray.*scalar.*`)

	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myany": {
                  "type": "any",
                  "matchversions": true
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.myany.*scalar.*`)

	// Must not be under arrays
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myarray": {
                  "type": "array",
                  "item": { "type": "object",
                            "attributes": { "myint": {
                              "type": "integer", "matchversions": true }}}
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.myarray.item.myint.*in an array.*`)

	// Must not be under maps
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "mymap": {
                  "type": "map",
                  "item": { "type": "object",
                            "attributes": { "myint": {
                              "type": "integer", "matchversions": true }}}
                } } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.mymap.item.myint.*in a map.*`)

	// Must not be under ifvalues
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "myint": {
                  "type": "integer",
                  "ifvalues": { "5": {
                      "siblingattributes": {
                        "mystr": { "type": "string", "matchversions": true }
                      } } } }
                } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.myint.ifvalues.5.mystr.*in an \\"ifvalues\\".*`)

	// Must not be under "*"
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
              "attributes": {
                "*": { "type": "integer", "matchversions": true }
                } } } } } }`
	XCheckErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true),
		`^(?s).*model_error.*groups.dirs.resources.files.\*\\" is not.*matchversion.*extension.*`)

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
    }`, 400, `^(?s)^.*mismatched_version.*f1.*Unique.* 2\..*w/o.*: 0\..*$`)

	// Unique: 1  Empty: 1
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v2", `{
      "myint": null
    }`, 400, `^(?s)^.*mismatched_version.*f1.*Unique.* 1\..*w/o.*: 1\..*$`)

	// Try strings
	// Unique: 1  Empty: 0
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v1": { "mystr": "hello" },
      "v2": { "mystr": "hello" }
    }`, 200, `^(?s)^.*v1.*hello.*v2.*hello.*$`)

	// Unique: hello,bye  Empty: 1
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v2": { "mystr": "bye" }
    }`, 400, `^(?s)^.*mismatched_version.*f1.*Unique.* 2\..*w/o.*: 0\..*$`)

	// Unique: bye  Empty: 1
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions", `{
      "v1": { "mystr": null }
    }`, 400, `^(?s)^.*mismatched_version.*f1.*Unique.* 1\..*w/o.*: 1\..*$`)

	// Now let's test some complex types
	modelSrc = `{
      "groups": { "dirs": { "singular": "dir", "resources": { "files": {
              "singular": "file", "hasdocument": false,
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
		`^(?s)^.*unknown attribute.*mystr.*$`)

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
    }`, 400, `^(?s)^.*mismatched_version_attribute.*myint.*f1.*: 2\..*: 0.*$`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 3, "myobj": { "mystr": "bye", "intobj": { "int2": 2 }}}
    }`, 400, `^(?s)^.*mismatched_version_attribute.*mystr.*f1.*: 2\..*: 0.*$`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
	  "v2": { "myint": 3, "myobj": { "mystr": "hi", "intobj": { "int2": 3 }}}
    }`, 400, `^(?s)^.*mismatched_version_attribute.*int2.*f1.*: 2\..*: 0.*$`)

	// Now fail doing one level at a time - missing values
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myobj": { "mystr": "hi", "intobj": { "int2": 2 }}}
    }`, 400, `^(?s)^.*mismatched_version_attribute.*myint.*f1.*: 1\..*: 1.*$`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": { "myint": 3, "myobj": { "intobj": { "int2": 2 }}}
    }`, 400, `^(?s)^.*mismatched_version_attribute.*mystr.*f1.*: 1\..*: 1.*$`)

	// Now fail doing one level at a time
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
	  "v2": { "myint": 3, "myobj": { "mystr": "hi"}}
    }`, 400, `^(?s)^.*mismatched_version_attribute.*int2.*f1.*: 1\..*: 1.*$`)

}
