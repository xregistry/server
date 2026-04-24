package tests

// ├ │ └

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

var RepoBase = "https://raw.githubusercontent.com/xregistry/spec/main"

func TestXRBasic(t *testing.T) {
	reg := NewRegistry("TestXRBasic")
	defer PassDeleteReg(t, reg)

	os.Setenv("XR_SERVER", "")
	XCLI(t, "get", "", "", "*", false)

	os.Setenv("XR_SERVER", "http://example.com")
	XCLI(t, "get", "", "", "*", false)

	cmd := exec.Command("../xr")
	out, err := cmd.CombinedOutput()
	XNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), ":")

	// Just look for the first 3 lines of 'xr' to look right
	XEqual(t, "", lines, "xRegistry CLI\n\nUsage")

	// Make sure we can validate the various spec owned model files
	files := []string{
		"sample-model.json",
		"endpoint/model.json",
		"message/model.json",
		"schema/model.json",
	}
	paths := os.Getenv("XR_MODEL_PATH") + ":files:" + RepoBase
	os.Setenv("XR_MODEL_PATH", paths)

	for _, file := range files {
		fn := file
		if !strings.HasPrefix(fn, "http") {
			fn, _ = FindModelFile(file)
		}
		if fn == "" {
			t.Errorf("Can't find %q in %q", file, paths)
			t.FailNow()
		}

		XCLI(t, "model verify "+fn, "", "", "", true)
	}

	// Test for no server specified
	XCLIServer("localhost:8181")

	XCLI(t, "get", "", `{
  "specversion": "`+SPECVERSION+`",
  "registryid": "TestXRBasic",
  "self": "http://localhost:8181/",
  "xid": "/",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z"
}
`, "", true)

	XCLI(t, "model group create -v dirs:dir", "",
		"", "Created Group type: dirs:dir\n", true)

	XCLI(t, "model resource create -v -g dirs files:file", "",
		"", "Created Resource type: files\n", true)

	XCLI(t, "model get", "",
		`xRegistry Model:

ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
├ dirs        map/object   -     -    y
│ └ *         any          -     -    y
├ dirscount   uinteger     y     y    y
└ dirsurl     url          y     y    -

GROUP: dirs / dir

  ATTRIBUTES:    TYPE         REQ   RO   MUT   DEFAULT
  ├ files        map/object   -     -    y
  │ └ *          any          -     -    y
  ├ filescount   uinteger     y     y    y
  └ filesurl     url          y     y    -

  RESOURCE: files / file
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false
`, "", true)

	XCLI(t, "model get -a", "",
		`xRegistry Model:

ATTRIBUTES:       TYPE         REQ   RO   MUT   DEFAULT
├ capabilities    object       -     -    y
│ └ *             any          -     -    y
├ createdat       timestamp    y     -    y
├ description     string       -     -    y
├ dirs            map/object   -     -    y
│ └ *             any          -     -    y
├ dirscount       uinteger     y     y    y
├ dirsurl         url          y     y    -
├ documentation   url          -     -    y
├ epoch           uinteger     y     y    y
├ icon            url          -     -    y
├ labels          map/string   -     -    y
├ model           object       -     y    y
│ └ *             any          -     -    y
├ modelsource     object       -     -    y
│ └ *             any          -     -    y
├ modifiedat      timestamp    y     -    y
├ name            string       -     -    y
├ registryid      string       y     y    -
├ self            url          y     y    -
├ specversion     string       y     y    y
└ xid             xid          y     y    -

GROUP: dirs / dir

  ATTRIBUTES:         TYPE         REQ   RO   MUT   DEFAULT
  ├ createdat         timestamp    y     -    y
  ├ deprecated        object       -     -    y
  │ ├ alternative     url          -     -    y
  │ ├ documentation   url          -     -    y
  │ ├ effective       timestamp    -     -    y
  │ ├ removal         timestamp    -     -    y
  │ └ *               any          -     -    y
  ├ description       string       -     -    y
  ├ dirid             string       y     -    -
  ├ documentation     url          -     -    y
  ├ epoch             uinteger     y     y    y
  ├ files             map/object   -     -    y
  │ └ *               any          -     -    y
  ├ filescount        uinteger     y     y    y
  ├ filesurl          url          y     y    -
  ├ icon              url          -     -    y
  ├ labels            map/string   -     -    y
  ├ modifiedat        timestamp    y     -    y
  ├ name              string       -     -    y
  ├ self              url          y     y    -
  └ xid               xid          y     y    -

  RESOURCE: files / file
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false

    ATTRIBUTES:                      TYPE         REQ   RO   MUT   DEFAULT
    ├ ancestor                       string       y     -    y
    ├ compatibilityvalidated         boolean      -     y    y
    ├ compatibilityvalidatedreason   string       -     y    y
    ├ contenttype                    string       -     -    y
    ├ createdat                      timestamp    y     -    y
    ├ description                    string       -     -    y
    ├ documentation                  url          -     -    y
    ├ epoch                          uinteger     y     y    y
    ├ file                           any          -     -    y
    ├ filebase64                     string       -     -    y
    ├ fileid                         string       y     -    -
    ├ fileproxyurl                   url          -     -    y
    ├ fileurl                        url          -     -    y
    ├ format                         string       -     -    y
    ├ formatvalidated                boolean      -     y    y
    ├ formatvalidatedreason          string       -     y    y
    ├ icon                           url          -     -    y
    ├ isdefault                      boolean      y     y    y     false
    ├ labels                         map/string   -     -    y
    ├ modifiedat                     timestamp    y     -    y
    ├ name                           string       -     -    y
    ├ self                           url          y     y    -
    ├ versionid                      string       y     -    -
    └ xid                            xid          y     y    -

    RESOURCE ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
    ├ fileid               string       y     -    -
    ├ meta                 object       -     -    y
    │ └ *                  any          -     -    y
    ├ metaurl              url          y     y    -
    ├ self                 url          y     y    -
    ├ versions             map/object   -     -    y
    │ └ *                  any          -     -    y
    ├ versionscount        uinteger     y     y    y
    ├ versionsurl          url          y     y    -
    └ xid                  xid          y     y    -

    META ATTRIBUTES:         TYPE         REQ   RO   MUT   DEFAULT
    ├ compatibility          string       -     -    y
    ├ createdat              timestamp    y     -    y
    ├ defaultversionid       string       y     -    y
    ├ defaultversionsticky   boolean      y     -    y     false
    ├ defaultversionurl      url          y     y    y
    ├ deprecated             object       -     -    y
    │ ├ alternative          url          -     -    y
    │ ├ documentation        url          -     -    y
    │ ├ effective            timestamp    -     -    y
    │ ├ removal              timestamp    -     -    y
    │ └ *                    any          -     -    y
    ├ epoch                  uinteger     y     y    y
    ├ fileid                 string       y     -    -
    ├ labels                 map/string   -     -    y
    ├ modifiedat             timestamp    y     -    y
    ├ readonly               boolean      y     y    y     false
    ├ self                   url          y     y    -
    ├ xid                    xid          y     y    -
    └ xref                   url          -     -    y
`, "", true)

	XCLI(t, "create /dirs/d1/files/f1/versions/v1 -vd hello_world", "",
		"", "Created: /dirs/d1/files/f1/versions/v1\n", true)

	XCLI(t, "get /dirs/d1/files/f1", "",
		"hello_world", "", true)

	XCLI(t, "get /dirs/d1/files/f1$details", "",
		`{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestor": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`, "", true)
}

func TestXRModel(t *testing.T) {
	reg := NewRegistry("TestXRModel")
	defer PassDeleteReg(t, reg)

	os.Setenv("XR_SERVER", "localhost:8181")

	XCLI(t, "model update -vd @files/dir/model-dirs-inc-docs.json", "",
		"", "Model updated\n", true)

	XCLI(t, "model group create -v gts:gt", "",
		"", "Created Group type: gts:gt\n", true)

	XCLI(t, "model resource create -vg gts rts:rt", "",
		"", "Created Resource type: rts\n", true)

	XCLI(t, "model resource create -vg gt2s:gt2 rt2s:rt2", "",
		"", "Created Group type: gt2s:gt2\nCreated Resource type: rt2s\n", true)

	XCLI(t, "model get", "",

		`xRegistry Model:

ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
├ dirs        map/object   -     -    y
│ └ *         any          -     -    y
├ dirscount   uinteger     y     y    y
├ dirsurl     url          y     y    -
├ docs        map/object   -     -    y
│ └ *         any          -     -    y
├ docscount   uinteger     y     y    y
├ docsurl     url          y     y    -
├ gt2s        map/object   -     -    y
│ └ *         any          -     -    y
├ gt2scount   uinteger     y     y    y
├ gt2surl     url          y     y    -
├ gts         map/object   -     -    y
│ └ *         any          -     -    y
├ gtscount    uinteger     y     y    y
└ gtsurl      url          y     y    -

GROUP: dirs / dir

  ATTRIBUTES:    TYPE         REQ   RO   MUT   DEFAULT
  ├ files        map/object   -     -    y
  │ └ *          any          -     -    y
  ├ filescount   uinteger     y     y    y
  └ filesurl     url          y     y    -

  RESOURCE: files / file
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false

GROUP: docs / doc

  ATTRIBUTES:    TYPE         REQ   RO   MUT   DEFAULT
  ├ types        map/object   -     -    y
  │ └ *          any          -     -    y
  ├ typescount   uinteger     y     y    y
  └ typesurl     url          y     y    -

  RESOURCE: types / type
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false

GROUP: gt2s / gt2

  ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
  ├ rt2s        map/object   -     -    y
  │ └ *         any          -     -    y
  ├ rt2scount   uinteger     y     y    y
  └ rt2surl     url          y     y    -

  RESOURCE: rt2s / rt2
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false

GROUP: gts / gt

  ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
  ├ rts         map/object   -     -    y
  │ └ *         any          -     -    y
  ├ rtscount    uinteger     y     y    y
  └ rtsurl      url          y     y    -

  RESOURCE: rts / rt
    Max versions        : 0
    Set version id      : true
    Set version sticky  : true
    Has document        : true
    Version mode        : manual
    Single version root : false
    Validate format     : false
    Validate compat     : false
    Strict valiation    : false
    Consistent format   : false
`, "", true)

	// Test some ifvalues
	XCLI(t, "model update -vd @-", `{
  "attributes": {
    "mystr": {
      "type": "string",
      "ifvalues": {
        "abc": {
          "siblingattributes": {
            "myint": {
              "type": "integer"
            }
          }
        },
        "def": {
          "siblingattributes": {
            "myobj": {
              "type": "object",
              "attributes": {
                "anint": {
                  "type": "integer",
                  "ifvalues": {
                    "1": {
                      "siblingattributes": {
                        "int2": {
                          "type": "integer"
                        },
                        "int3": {
                          "type": "integer"
                        }
                      }
                    }
                  }
                }
              }
            },
            "foo": {
              "type": "string"
            },
            "zzz": {
              "type": "string"
            }
          }
        }
      }
    },
    "obj": {
      "type": "object",
      "attributes": {
        "astr": {
          "type": "string",
          "ifvalues": {
            "foo": {
              "siblingattributes": {
                "amap": {
                  "type": "map",
                  "item": {
                    "type": "string"
                  }
                }
              }
            }
          }
        }
      }
    },
    "obj2": {
      "type": "object",
      "attributes": {
        "astr": {
          "type": "string",
          "ifvalues": {
            "foo": {},
            "foo2": {}
          }
        }
      }
    }
  }
}`, "", "Model updated\n", true)

	XCLI(t, "model get", "",
		`xRegistry Model:

ATTRIBUTES:   TYPE         REQ   RO   MUT   DEFAULT
├ mystr       string       -     -    y
│ >> if mystr="abc"
├ myint       integer      -     -    y
│ << endif
│ >> if mystr="def"
├ foo         string       -     -    y
├ myobj       object       -     -    y
│ ├ anint     integer      -     -    y
│ │ >> if anint="1"
│ ├ int2      integer      -     -    y
│ └ int3      integer      -     -    y
│   << endif
├ zzz         string       -     -    y
│ << endif
├ obj         object       -     -    y
│ ├ astr      string       -     -    y
│ │ >> if astr="foo"
│ └ amap      map/string   -     -    y
│   << endif
└ obj2        object       -     -    y
  └ astr      string       -     -    y
    >> if astr="foo"
    << endif
    >> if astr="foo2"
    << endif
`, "", true)

}

func TestXRUpdateRegistry(t *testing.T) {
	reg := NewRegistry("TestXRUpdateRegistry")
	defer PassDeleteReg(t, reg)

	XCLIServer("localhost:8181")

	XCLI(t, "create", "",
		"", "Must specify the XID of an entity.\n",
		false)

	XCLI(t, "create /", "",
		"", "To create a registry use the 'xrserver registry create' command.\n",
		false)

	XCLI(t, "update", "", "", "Must specify the XID of an entity.\n", false)
	XCLI(t, "update /", "", "", "", true)
	XCLI(t, "update -v /", "", "", "Updated: /\n", true)

	XCLI(t, "update -vo json /", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 4,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "Updated: /\n", true)

	XCLI(t, "update -o=json / --set name=myreg", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 5,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "name": "myreg",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	XCLI(t, "update -o=json / --set name= --set=description=cool", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "description": "cool",
  "epoch": 6,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "name": "",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	XCLI(t, "update -o=json / --set name --set=description=5", "",
		"", `The attribute "description" for "/" is not valid: must be a string.
`, false)

	XCLI(t, "update -o=json / --set name --set=description=\"5\"", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "description": "5",
  "epoch": 7,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	XCLI(t, "update / -o=json --set=labels.foo=5 --del description "+
		"--del=labels", "",
		"", `The attribute "labels.foo" for "/" is not valid: must be a string.
`, false)

	XCLI(t, "update / -o=json --set=labels.foo=\"5\" --del description "+
		"--del=labels", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 8,
  "labels": {
    "foo": "5"
  },
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	XCLI(t, "update / -o=json --add=labels.bar=zzz", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 9,
  "labels": {
    "bar": "zzz",
    "foo": "5"
  },
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	XCLI(t, "update / -o=json --add=labels.yyy=yay --del=labels.foo", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 10,
  "labels": {
    "bar": "zzz",
    "yyy": "yay"
  },
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

}

func TestXRGroupType(t *testing.T) {
	reg := NewRegistry("TestXRGroupType")
	defer PassDeleteReg(t, reg)

	XCLIServer("localhost:8181")

	XCLI(t, "model group create", "",
		"", "At least one Group type name must be specified.\n", false)

	XCLI(t, "model group create dirs", "",
		"", "Group type name must be of the form: PLURAL:SINGULAR.\n", false)

	XCLI(t, "model group create dirs dir", "",
		"", "Group type name must be of the form: PLURAL:SINGULAR.\n", false)

	XCLI(t, "model group create dirs:dir", "", "", "", true)

	XCLI(t, "model group create dirs:dir", "", "",
		"PLURAL value (dirs) conflicts with an existing Group PLURAL name.\n",
		false)

	XCLI(t, "model group create -o table -v dirs2:dir2", "",
		"GROUP: dirs2 / dir2\n", "Created Group type: dirs2:dir2\n", true)

	XCLI(t, "model group create dirs3:dir3 -v -o json", "",
		`{
  "plural": "dirs3",
  "singular": "dir3",
  "attributes": {
    "createdat": {
      "name": "createdat",
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
    "description": {
      "name": "description",
      "type": "string"
    },
    "dir3id": {
      "name": "dir3id",
      "type": "string",
      "matchcase": true,
      "immutable": true,
      "required": true
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "readonly": true,
      "required": true
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
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp",
      "required": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "xid": {
      "name": "xid",
      "type": "xid",
      "readonly": true,
      "immutable": true,
      "required": true
    }
  }
}
`, "Created Group type: dirs3:dir3\n", true)

	XCLI(t, "model group create -ao table -v dirs4:dir4", "",
		`GROUP: dirs4 / dir4

  ATTRIBUTES:         TYPE         REQ   RO   MUT   DEFAULT
  ├ createdat         timestamp    y     -    y
  ├ deprecated        object       -     -    y
  │ ├ alternative     url          -     -    y
  │ ├ documentation   url          -     -    y
  │ ├ effective       timestamp    -     -    y
  │ ├ removal         timestamp    -     -    y
  │ └ *               any          -     -    y
  ├ description       string       -     -    y
  ├ dir4id            string       y     -    -
  ├ documentation     url          -     -    y
  ├ epoch             uinteger     y     y    y
  ├ icon              url          -     -    y
  ├ labels            map/string   -     -    y
  ├ modifiedat        timestamp    y     -    y
  ├ name              string       -     -    y
  ├ self              url          y     y    -
  └ xid               xid          y     y    -
`, "Created Group type: dirs4:dir4\n", true)

	XCLI(t, "model group create -aro table dirs5:dir5", "",
		`GROUP: dirs5 / dir5

  ATTRIBUTES:         TYPE         REQ   RO   MUT   DEFAULT
  ├ createdat         timestamp    y     -    y
  ├ deprecated        object       -     -    y
  │ ├ alternative     url          -     -    y
  │ ├ documentation   url          -     -    y
  │ ├ effective       timestamp    -     -    y
  │ ├ removal         timestamp    -     -    y
  │ └ *               any          -     -    y
  ├ description       string       -     -    y
  ├ dir5id            string       y     -    -
  ├ documentation     url          -     -    y
  ├ epoch             uinteger     y     y    y
  ├ icon              url          -     -    y
  ├ labels            map/string   -     -    y
  ├ modifiedat        timestamp    y     -    y
  ├ name              string       -     -    y
  ├ self              url          y     y    -
  └ xid               xid          y     y    -
`, "", true)

	XCLI(t, "model group create -aro xxx dirs6:dir6", "",
		"", "--output must be one of 'json', 'none', 'table'.\n", false)

	XCLI(t, "model group get dirs2", "", "GROUP: dirs2 / dir2\n", "", true)

	XCLI(t, "model group get -o json dirs2", "",
		`{
  "plural": "dirs2",
  "singular": "dir2",
  "attributes": {
    "createdat": {
      "name": "createdat",
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
    "description": {
      "name": "description",
      "type": "string"
    },
    "dir2id": {
      "name": "dir2id",
      "type": "string",
      "matchcase": true,
      "immutable": true,
      "required": true
    },
    "documentation": {
      "name": "documentation",
      "type": "url"
    },
    "epoch": {
      "name": "epoch",
      "type": "uinteger",
      "readonly": true,
      "required": true
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
    "modifiedat": {
      "name": "modifiedat",
      "type": "timestamp",
      "required": true
    },
    "name": {
      "name": "name",
      "type": "string"
    },
    "self": {
      "name": "self",
      "type": "url",
      "readonly": true,
      "immutable": true,
      "required": true
    },
    "xid": {
      "name": "xid",
      "type": "xid",
      "readonly": true,
      "immutable": true,
      "required": true
    }
  }
}
`, "", true)

	XCLI(t, "model group get -a -o json dirs2 dirs", "",
		`[
  {
    "plural": "dirs2",
    "singular": "dir2",
    "attributes": {
      "createdat": {
        "name": "createdat",
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
      "description": {
        "name": "description",
        "type": "string"
      },
      "dir2id": {
        "name": "dir2id",
        "type": "string",
        "matchcase": true,
        "immutable": true,
        "required": true
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "readonly": true,
        "required": true
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
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp",
        "required": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "immutable": true,
        "required": true
      }
    }
  },
  {
    "plural": "dirs",
    "singular": "dir",
    "attributes": {
      "createdat": {
        "name": "createdat",
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
      "description": {
        "name": "description",
        "type": "string"
      },
      "dirid": {
        "name": "dirid",
        "type": "string",
        "matchcase": true,
        "immutable": true,
        "required": true
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "readonly": true,
        "required": true
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
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp",
        "required": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "immutable": true,
        "required": true
      }
    }
  }
]
`, "", true)

	XCLI(t, "model group get -o table dirs2 dirs", "",
		`GROUP: dirs2 / dir2

GROUP: dirs / dir
`, "", true)

	XCLI(t, "model resource create -v -g dirs2 files:file", "",
		``, "Created Resource type: files\n", true)

	XCLI(t, "model resource create -v -g dirs2 files2:file2 -o table", "",
		`RESOURCE: files2 / file2
  Max versions        : 0
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, "Created Resource type: files2\n", true)

	XCLI(t, "model resource create -v -g dirs7 files2:file2 -o table", "",
		``, "Group type \"dirs7\" does not exist.\n", false)

	XCLI(t, "model resource create -v -g dirs7:dir7 files2:file2 -o table", "",
		`RESOURCE: files2 / file2
  Max versions        : 0
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, `Created Group type: dirs7:dir7
Created Resource type: files2
`, true)

	XCLI(t, "model group list", "", `GROUP          RESOURCES   DESCRIPTION
dirs / dir     0
dirs2 / dir2   2
dirs3 / dir3   0
dirs4 / dir4   0
dirs5 / dir5   0
dirs7 / dir7   1
`, ``, true)

	XCLI(t, "model group list -o json", "", `*`, ``, true) // cheat

	XCLI(t, "model resource list", "", "",
		"A Group type name must be provided via the --group flag.\n", false)

	XCLI(t, "model resource list files -g dirs", "", "",
		"No arguments allowed.\n", false)

	XCLI(t, "model resource list -g dirs7", "", `RESOURCE         HAS DOC   EXT ATTRS   DESCRIPTION
files2 / file2   true      0
`,
		"", true)

	XCLI(t, "model resource list -g dirs7 -o json", "", `{
  "files2": {
    "plural": "files2",
    "singular": "file2",
    "maxversions": 0,
    "setversionid": true,
    "setdefaultversionsticky": true,
    "hasdocument": true,
    "versionmode": "manual",
    "singleversionroot": false,
    "validateformat": false,
    "validatecompatibility": false,
    "strictvalidation": false,
    "consistentformat": false,
    "attributes": {
      "ancestor": {
        "name": "ancestor",
        "type": "string",
        "matchcase": true,
        "required": true
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
      "contenttype": {
        "name": "contenttype",
        "type": "string"
      },
      "createdat": {
        "name": "createdat",
        "type": "timestamp",
        "required": true
      },
      "description": {
        "name": "description",
        "type": "string"
      },
      "documentation": {
        "name": "documentation",
        "type": "url"
      },
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "readonly": true,
        "required": true
      },
      "file2": {
        "name": "file2",
        "type": "any"
      },
      "file2base64": {
        "name": "file2base64",
        "type": "string"
      },
      "file2id": {
        "name": "file2id",
        "type": "string",
        "matchcase": true,
        "immutable": true,
        "required": true
      },
      "file2proxyurl": {
        "name": "file2proxyurl",
        "type": "url"
      },
      "file2url": {
        "name": "file2url",
        "type": "url"
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
      "icon": {
        "name": "icon",
        "type": "url"
      },
      "isdefault": {
        "name": "isdefault",
        "type": "boolean",
        "readonly": true,
        "required": true,
        "default": false
      },
      "labels": {
        "name": "labels",
        "type": "map",
        "item": {
          "type": "string"
        }
      },
      "modifiedat": {
        "name": "modifiedat",
        "type": "timestamp",
        "required": true
      },
      "name": {
        "name": "name",
        "type": "string"
      },
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
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
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "immutable": true,
        "required": true
      }
    },
    "resourceattributes": {
      "file2id": {
        "name": "file2id",
        "type": "string",
        "matchcase": true,
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
      "metaurl": {
        "name": "metaurl",
        "type": "url",
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
      },
      "versionscount": {
        "name": "versionscount",
        "type": "uinteger",
        "readonly": true,
        "required": true
      },
      "versionsurl": {
        "name": "versionsurl",
        "type": "url",
        "readonly": true,
        "immutable": true,
        "required": true
      },
      "xid": {
        "name": "xid",
        "type": "xid",
        "readonly": true,
        "immutable": true,
        "required": true
      }
    },
    "metaattributes": {
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
      "createdat": {
        "name": "createdat",
        "type": "timestamp",
        "required": true
      },
      "defaultversionid": {
        "name": "defaultversionid",
        "type": "string",
        "matchcase": true,
        "required": true
      },
      "defaultversionsticky": {
        "name": "defaultversionsticky",
        "type": "boolean",
        "required": true,
        "default": false
      },
      "defaultversionurl": {
        "name": "defaultversionurl",
        "type": "url",
        "readonly": true,
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
      "epoch": {
        "name": "epoch",
        "type": "uinteger",
        "readonly": true,
        "required": true
      },
      "file2id": {
        "name": "file2id",
        "type": "string",
        "matchcase": true,
        "immutable": true,
        "required": true
      },
      "labels": {
        "name": "labels",
        "type": "map",
        "item": {
          "type": "string"
        }
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
      "self": {
        "name": "self",
        "type": "url",
        "readonly": true,
        "immutable": true,
        "required": true
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
      }
    }
  }
}
`,
		"", true)

}

func TestXRIgnore(t *testing.T) {
	reg := NewRegistry("TestXRIgnore")
	defer PassDeleteReg(t, reg)

	os.Setenv("XR_SERVER", "localhost:8181")

	// cmd, stdin, stdout, stderr, pass?

	XCLI(t, "model resource create files:file -g dirs:dir --has-doc=false", "",
		"", "", true)

	XCLI(t, "create /dirs/d1/files/f1 --set fileid=f2", ``,
		"", `The specified "fileid" value (f2) for "/dirs/d1/files/f1" needs to be "f1".`+"\n", false)

	XCLI(t, "create -v /dirs/d1/files/f1 -d @- --ignore=id", `{ "fileid": "f2" }`,
		"", `Created: /dirs/d1/files/f1`+"\n", true)

	XCLI(t, "update -v /dirs/d1/files/f1 --set epoch=5", ``,
		"", `The specified epoch value (5) for "/dirs/d1/files/f1/versions/1" does not match its current value (1).`+"\n", false)

	XCLI(t, "update -v /dirs/d1/files/f1 --set epoch=5 --set fileid=foo --ignore=id --ignore=epoch", ``,
		"", `Updated: /dirs/d1/files/f1`+"\n", true)
}

func TestXRResourceType(t *testing.T) {
	reg := NewRegistry("TestXRResourceType")
	defer PassDeleteReg(t, reg)

	XCLIServer("localhost:8181")

	// XCLI(t, "cmd", "stdin", "stdout", "stderr", true/false)
	XCLI(t, "model resource create files", "", "",
		"A Group type name must be provided via the --group flag.\n", false)
	XCLI(t, "model resource create files -g dirs", "", "",
		`Group type "dirs" does not exist.`+"\n", false)
	XCLI(t, "model resource create files -g dirs:dir", "", "",
		"Resource type name must be of the form: PLURAL:SINGULAR.\n", false)

	XCLI(t, "model resource create files:file -g dirs:dir", "", "", "", true)
	XCLI(t, "model resource delete files -g dirs", "", "", "", true)

	XCLI(t, "model resource create -v files:file -g dirs", "",
		"", "Created Resource type: files\n", true)

	XCLI(t, "model resource create f2s:f2 -g dirs -o table", "",
		`RESOURCE: f2s / f2
  Max versions        : 0
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, "", true)

	XCLI(t, "model resource create f3s:f3 -g dirs -o table "+
		"--consistent-format --description desc --docs docURL --has-doc "+
		"--icon iconURL --label foo=bar --label=abc=def --max-versions 1 "+
		"--model-compat-with mcw --model-version 1.0 --set-default-sticky "+
		"--set-version-id --single-version-root --strict-validation "+
		"--type-map tm1=json --type-map tm2=string --validate-compat "+
		"--validate-format --version-mode=createdat", "",
		`RESOURCE: f3s / f3
  Description         : desc
  Documentation       : docURL
  Max versions        : 1
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : createdat
  Single version root : true
  Validate format     : true
  Validate compat     : true
  Strict valiation    : true
  Consistent format   : true
  Icon URL            : iconURL
  Model version       : 1.0
  Model Compat with   : mcw
  Labels              : abc=def
                        foo=bar
  Type map            : tm1=json
                        tm2=string
`, ``, true)

	XCLI(t, "model resource create f4s:f4 -g dirs -o table "+
		"--consistent-format=true --has-doc=true "+
		"--set-default-sticky=true --set-version-id=true "+
		"--single-version-root=true --strict-validation=true "+
		"--validate-compat=true --validate-format=true", "",
		`RESOURCE: f4s / f4
  Max versions        : 0
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : manual
  Single version root : true
  Validate format     : true
  Validate compat     : true
  Strict valiation    : true
  Consistent format   : true
`, ``, true)

	XCLI(t, "model resource create f5s:f5 -g dirs -o table "+
		"--consistent-format=false --has-doc=false "+
		"--set-default-sticky=false --set-version-id=false "+
		"--single-version-root=false --strict-validation=false "+
		"--validate-compat=false --validate-format=false", "",
		`RESOURCE: f5s / f5
  Max versions        : 0
  Set version id      : false
  Set version sticky  : false
  Has document        : false
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, ``, true)

	XCLI(t, "model resource create f6s:f6 -g dirs -o table "+
		"--no-consistent-format --no-has-doc "+
		"--no-set-default-sticky --no-set-version-id "+
		"--no-single-version-root --no-strict-validation "+
		"--no-validate-compat --no-validate-format", "",
		`RESOURCE: f6s / f6
  Max versions        : 0
  Set version id      : false
  Set version sticky  : false
  Has document        : false
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, ``, true)

	XCLI(t, "model resource create f7s:f7 -g dirs -o table "+
		"--no-consistent-format=true --no-has-doc=true "+
		"--no-set-default-sticky=true --no-set-version-id=true "+
		"--no-single-version-root=true --no-strict-validation=true "+
		"--no-validate-compat=true --no-validate-format=true", "",
		`RESOURCE: f7s / f7
  Max versions        : 0
  Set version id      : false
  Set version sticky  : false
  Has document        : false
  Version mode        : manual
  Single version root : false
  Validate format     : false
  Validate compat     : false
  Strict valiation    : false
  Consistent format   : false
`, ``, true)

	XCLI(t, "model resource create f8s:f8 -g dirs -o table "+
		"--no-consistent-format=false --no-has-doc=false "+
		"--no-set-default-sticky=false --no-set-version-id=false "+
		"--no-single-version-root=false --no-strict-validation=false "+
		"--no-validate-compat=false --no-validate-format=false", "",
		`RESOURCE: f8s / f8
  Max versions        : 0
  Set version id      : true
  Set version sticky  : true
  Has document        : true
  Version mode        : manual
  Single version root : true
  Validate format     : true
  Validate compat     : true
  Strict valiation    : true
  Consistent format   : true
`, ``, true)

	// Some errors and test create vs update vs upsert
	XCLI(t, "model resource create fsc:fc -g dirs -f -v", "",
		"", "Created Resource type: fsc\n", true)

	// ---

	XCLI(t, "model resource update fs:f -g dirs -v", "",
		"", `Resource type "fs" doesn't exists.`+"\n", false)

	XCLI(t, "model resource update fs:f -g dirs -f -v", "",
		"", "Created Resource type: fs\n", true)

	XCLI(t, "model resource update fs:f -g dirs -v", "",
		"", "Updated Resource type: fs\n", true)

	XCLI(t, "model resource update fs -g dirs -v", "",
		"", "Updated Resource type: fs\n", true)

	// ---

	XCLI(t, "model resource upsert fs:f -g dirs -f -v", "",
		"", `Error: unknown shorthand flag: 'f' in -f
Usage:
  xr model resource upsert PLURAL:SINGULAR... [flags]

Flags:
  -a, --all                        Include default attributes in output
      --consistent-format          Enforce same format values
      --description string         Description text
      --docs string                Documenations URL
  -g, --group string               Group plural name (create with ":SINGULAR")
      --has-doc                    Supports domain doc (true*)
      --icon string                Icon URL
      --label stringArray          NAME[=VALUE)]
      --max-versions int           Max versions allowed (0=unlimited*)
      --model-compat-with string   URI of model
      --model-version string       Model version string
      --no-consistent-format       Allow varying format values (true*)
      --no-has-doc                 Doesn't support domain doc
      --no-set-default-sticky      Can't set sticky version
      --no-set-version-id          VersionID is not settable
      --no-single-version-root     Allow multiple verson roots (true*)
      --no-strict-validation       Disable strict validation (true*)
      --no-validate-compat         Disable compatibility validation (true*)
      --no-validate-format         Disable format validation (true*)
  -o, --output string              Output format: none*, table, json
      --set-default-sticky         Can set sticky version (true*)
      --set-version-id             Version ID is settable (true*)
      --single-version-root        Restrict to single root
      --strict-validation          Enforce strict validation
      --type-map stringArray       NAME[=VALUE)]
      --validate-compat            Enable compatibility validation
      --validate-format            Enable format validation
      --version-mode string        Versioning algorithm

Global Flags:
      --errjson         Print errors as json
  -?, --help            Help for xr
  -s, --server string   xRegistry server URL
  -v, --verbose         Be chatty

Version: edd264c38c5e

unknown shorthand flag: 'f' in -f
`, false)

	XCLI(t, "model resource upsert fs -g dirs -v", "",
		"", "Updated Resource type: fs\n", true)

	XCLI(t, "model resource upsert fs:f -g dirs -v", "",
		"", "Updated Resource type: fs\n", true)

	XCLI(t, "model resource upsert fs2 -g dirs -v", "",
		"", "Resource type name must be of the form: PLURAL:SINGULAR.\n", false)

	XCLI(t, "model resource upsert fs22:f22 -g dirs -v", "",
		"", "Created Resource type: fs22\n", true)
}
