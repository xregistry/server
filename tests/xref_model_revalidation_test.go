package tests

// TestXrefModelRevalidationFormatCascade checks that when a model
// change forces re-validation of an xref TARGET's format/compat
// system attributes (formatvalidated/compatibilityvalidated) with no
// other attribute change, that new value is correctly cascaded/
// mirrored into any xref SOURCE pointing at it.
//
// This exercises Registry.VerifyData() (registry/registry.go), the
// path used by Model.ApplyNewModel()/PUT-/modelsource to re-verify
// all existing data against a changed model - as opposed to the
// normal per-Resource HTTP PUT path (ValidateResource() called
// directly from Tx.Validate()).

import (
	"testing"
)

func TestXrefModelRevalidationFormatCascade(t *testing.T) {
	reg := NewRegistry("TestXrefModelRevalidationFormatCascade")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)

	// Create target f1 with format set, but validateformat=false so
	// formatvalidated is NOT computed/set yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1\n2\n3"
}`, 201, `*`)

	// Confirm formatvalidated is absent (validateformat is off).
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `*`)

	// Create xref fx -> f1.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Confirm the xref mirror also has no formatvalidated yet.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `*`)

	// Now flip validateformat=true at the MODEL level (no change to f1's
	// own data at all) - this forces Registry.VerifyData() to
	// re-validate every existing Resource, including f1 (setting
	// formatvalidated=true, since "1\n2\n3" is valid "numbers" format)
	// and fx (an xref source, whose runCascade() re-run should pick up
	// the target's fresh formatvalidated=true value).
	rm.SetValidateFormat(true)
	modelSrc := reg.Model.MustUserMarshal("", "  ")
	XHTTP(t, reg, "PUT", "/modelsource", modelSrc, 200, `*`)

	// The TARGET f1 must now show formatvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)

	// The xref SOURCE fx must mirror that same formatvalidated=true -
	// this is the part that was missing coverage.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`^(?s)^.*"epoch": 1,.*"formatvalidated": true`)
}

// TestXrefModelRevalidationCompatCascade is the compatibilityvalidated
// analog of TestXrefModelRevalidationFormatCascade above (gaps item 1's
// remaining "compatibilityvalidated variant" sub-case) - it checks that
// when a model change forces Registry.VerifyData() to re-validate an
// xref TARGET's compatibilityvalidated system attribute (with no other
// attribute change), that fresh value is correctly cascaded into any
// xref SOURCE pointing at it, exactly as formatvalidated is above.
func TestXrefModelRevalidationCompatCascade(t *testing.T) {
	reg := NewRegistry("TestXrefModelRevalidationCompatCascade")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)

	// Create target f1 with two versions (equal sums, so compatible
	// under any mode), but validatecompatibility=false so
	// compatibilityvalidated is NOT computed/set yet.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "versions": {
    "v1": { "format": "numbers", "file": "2" },
    "v2": { "format": "numbers", "file": "2" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:13.175487333Z",
  "modifiedat": "2026-07-27T00:28:13.175487333Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Confirm compatibilityvalidated is absent (validatecompatibility is
	// off).
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:15.548850155Z",
  "modifiedat": "2026-07-27T00:28:15.548850155Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// Create xref fx -> f1.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:28:17.538190459Z",
  "modifiedat": "2026-07-27T00:28:17.538190459Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)

	// Confirm the xref mirror also has no compatibilityvalidated yet.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:19.619210932Z",
  "modifiedat": "2026-07-27T00:28:19.619210932Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)

	// Now flip validatecompatibility=true at the MODEL level (no change
	// to f1's own data at all) - this forces Registry.VerifyData() to
	// re-validate every existing Resource, including f1 (setting
	// compatibilityvalidated=true, since v1/v2 are compatible under
	// "backward") and fx (an xref source, whose runCascade() re-run
	// should pick up the target's fresh compatibilityvalidated=true
	// value).
	rm.SetValidateCompatibility(true)
	modelSrc := reg.Model.MustUserMarshal("", "  ")
	XHTTP(t, reg, "PUT", "/modelsource", modelSrc, 200, `{
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
      "resources": {
        "files": {
          "plural": "files",
          "singular": "file",
          "maxversions": 0,
          "setversionid": true,
          "hasdocument": true,
          "versionmode": "manual",
          "singleversionroot": false,
          "validateformat": true,
          "validatecompatibility": true,
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
            "ancestorid": {
              "name": "ancestorid",
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
            "fileurl": {
              "name": "fileurl",
              "type": "url"
            },
            "fileproxyurl": {
              "name": "fileproxyurl",
              "type": "url"
            },
            "file": {
              "name": "file",
              "type": "any"
            },
            "filebase64": {
              "name": "filebase64",
              "type": "string"
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

	// The TARGET f1 must now show compatibilityvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:23.801192485Z",
  "modifiedat": "2026-07-27T00:28:23.801192485Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	// The xref SOURCE fx must mirror that same
	// compatibilityvalidated=true - this is the part that was missing
	// coverage.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:28:25.925416292Z",
  "modifiedat": "2026-07-27T00:28:25.925416292Z",
  "ancestorid": "v1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
}
