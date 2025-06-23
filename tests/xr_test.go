package tests

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
	xCLI(t, "get", "", "", "*", false)

	os.Setenv("XR_SERVER", "http://example.com")
	xCLI(t, "get", "", "", "*", false)

	cmd := exec.Command("../xr")
	out, err := cmd.CombinedOutput()
	xNoErr(t, err)
	lines, _, _ := strings.Cut(string(out), ":")

	// Just look for the first 3 lines of 'xr' to look right
	xCheckEqual(t, "", lines, "xRegistry CLI\n\nUsage")

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

		xCLI(t, "model verify "+fn, "", "", "", true)
	}

	// Test for no server specified
	xCLIServer("localhost:8181")

	xCLI(t, "get", "", `{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 1,
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "registryid": "TestXRBasic",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	xCLI(t, "model group create -v dirs:dir", "",
		"", "Created Group type: dirs:dir\n", true)

	xCLI(t, "model resource create -v -g dirs files:file", "",
		"", "Created Resource type: files:file\n", true)

	xCLI(t, "model get", "",
		`xRegistry Model:

ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
dirs        map(object) -   -  y   
dirscount   uinteger    y   y  y   
dirsurl     url         y   y  -   

GROUP: dirs / dir

  ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
  files       map(object) -   -  y   
  filescount  uinteger    y   y  y   
  filesurl    url         y   y  -   

  RESOURCE: files/ file
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:  TYPE   REQ RO MUT DEFAULT
    file         any    -   -  y   
    filebase64   string -   -  y   
    fileproxyurl url    -   -  y   
    fileurl      url    -   -  y   
`, "", true)

	xCLI(t, "model get -a", "",
		`xRegistry Model:

ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
capabilities  object      -   -  y   
createdat     timestamp   y   -  y   
description   string      -   -  y   
dirs          map(object) -   -  y   
dirscount     uinteger    y   y  y   
dirsurl       url         y   y  -   
documentation url         -   -  y   
epoch         uinteger    y   y  y   
icon          url         -   -  y   
labels        map(string) -   -  y   
model         object      -   y  y   
modelsource   object      -   -  y   
modifiedat    timestamp   y   -  y   
name          string      -   -  y   
registryid    string      y   y  -   
self          url         y   y  -   
specversion   string      y   y  y   
xid           xid         y   y  -   

GROUP: dirs / dir

  ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
  createdat     timestamp   y   -  y   
  description   string      -   -  y   
  dirid         string      y   -  -   
  documentation url         -   -  y   
  epoch         uinteger    y   y  y   
  files         map(object) -   -  y   
  filescount    uinteger    y   y  y   
  filesurl      url         y   y  -   
  icon          url         -   -  y   
  labels        map(string) -   -  y   
  modifiedat    timestamp   y   -  y   
  name          string      -   -  y   
  self          url         y   y  -   
  xid           xid         y   y  -   

  RESOURCE: files/ file
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
    ancestor      string      y   -  y   
    contenttype   string      -   -  y   
    createdat     timestamp   y   -  y   
    description   string      -   -  y   
    documentation url         -   -  y   
    epoch         uinteger    y   y  y   
    file          any         -   -  y   
    filebase64    string      -   -  y   
    fileid        string      y   -  -   
    fileproxyurl  url         -   -  y   
    fileurl       url         -   -  y   
    icon          url         -   -  y   
    isdefault     boolean     y   y  y   false
    labels        map(string) -   -  y   
    modifiedat    timestamp   y   -  y   
    name          string      -   -  y   
    self          url         y   y  -   
    versionid     string      y   -  -   
    xid           xid         y   y  -   

    RESOURCE ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
    fileid               string      y   -  -   
    meta                 object      -   -  y   
    metaurl              url         y   y  -   
    self                 url         y   y  -   
    versions             map(object) -   -  y   
    versionscount        uinteger    y   y  y   
    versionsurl          url         y   y  -   
    xid                  xid         y   y  -   

    META ATTRIBUTES:       TYPE      REQ RO MUT DEFAULT
    compatibility          string    y   -  y   "none"
    compatibilityauthority string    -   -  y   
    createdat              timestamp y   -  y   
    defaultversionid       string    y   -  y   
    defaultversionsticky   boolean   y   -  y   false
    defaultversionurl      url       y   y  y   
    deprecated             object    -   -  y   
    epoch                  uinteger  y   y  y   
    fileid                 string    y   -  -   
    modifiedat             timestamp y   -  y   
    readonly               boolean   y   y  y   false
    self                   url       y   y  -   
    xid                    xid       y   y  -   
    xref                   url       -   -  y   
`, "", true)

	xCLI(t, "create /dirs/d1/files/f1/versions/v1 -vd hello_world", "",
		"", "Created: /dirs/d1/files/f1/versions/v1\n", true)

	xCLI(t, "get /dirs/d1/files/f1", "",
		"hello_world\n", "", true)

	xCLI(t, "get /dirs/d1/files/f1$details", "",
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

	xCLI(t, "model update -vd @files/dir/model-dirs-inc-docs.json", "",
		"", "Model updated\n", true)

	xCLI(t, "model group create -v gts:gt", "",
		"", "Created Group type: gts:gt\n", true)

	xCLI(t, "model resource create -vg gts rts:rt", "",
		"", "Created Resource type: rts:rt\n", true)

	xCLI(t, "model resource create -vg gt2s:gt2 rt2s:rt2", "",
		"", "Created Group type: gt2s:gt2\nCreated Resource type: rt2s:rt2\n", true)

	xCLI(t, "model get", "",

		`xRegistry Model:

ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
dirs        map(object) -   -  y   
dirscount   uinteger    y   y  y   
dirsurl     url         y   y  -   
docs        map(object) -   -  y   
docscount   uinteger    y   y  y   
docsurl     url         y   y  -   
gt2s        map(object) -   -  y   
gt2scount   uinteger    y   y  y   
gt2surl     url         y   y  -   
gts         map(object) -   -  y   
gtscount    uinteger    y   y  y   
gtsurl      url         y   y  -   

GROUP: dirs / dir

  ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
  files       map(object) -   -  y   
  filescount  uinteger    y   y  y   
  filesurl    url         y   y  -   

  RESOURCE: files/ file
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:  TYPE   REQ RO MUT DEFAULT
    file         any    -   -  y   
    filebase64   string -   -  y   
    fileproxyurl url    -   -  y   
    fileurl      url    -   -  y   

GROUP: docs / doc

  ATTRIBUTES:  TYPE        REQ RO MUT DEFAULT
  formats      map(object) -   -  y   
  formatscount uinteger    y   y  y   
  formatsurl   url         y   y  -   

  RESOURCE: formats/ format
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES:    TYPE   REQ RO MUT DEFAULT
    format         any    -   -  y   
    formatbase64   string -   -  y   
    formatproxyurl url    -   -  y   
    formaturl      url    -   -  y   

GROUP: gt2s / gt2

  ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
  rt2s        map(object) -   -  y   
  rt2scount   uinteger    y   y  y   
  rt2surl     url         y   y  -   

  RESOURCE: rt2s/ rt2
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES: TYPE   REQ RO MUT DEFAULT
    rt2         any    -   -  y   
    rt2base64   string -   -  y   
    rt2proxyurl url    -   -  y   
    rt2url      url    -   -  y   

GROUP: gts / gt

  ATTRIBUTES: TYPE        REQ RO MUT DEFAULT
  rts         map(object) -   -  y   
  rtscount    uinteger    y   y  y   
  rtsurl      url         y   y  -   

  RESOURCE: rts/ rt
    Max versions      : 0
    Set version id    : true
    Set version sticky: true
    Has document      : true

    ATTRIBUTES: TYPE   REQ RO MUT DEFAULT
    rt          any    -   -  y   
    rtbase64    string -   -  y   
    rtproxyurl  url    -   -  y   
    rturl       url    -   -  y   
`, "", true)

}

func TestXRUpdateRegistry(t *testing.T) {
	reg := NewRegistry("TestXRUpdateRegistry")
	defer PassDeleteReg(t, reg)

	xCLIServer("localhost:8181")

	xCLI(t, "create", "",
		"", "To create a registry use the 'xrserver registry create' command\n",
		false)

	xCLI(t, "create /", "",
		"", "To create a registry use the 'xrserver registry create' command\n",
		false)

	xCLI(t, "update", "", "", "", true)
	xCLI(t, "update -v", "", "", "Updated: /\n", true)
	xCLI(t, "update /", "", "", "", true)
	xCLI(t, "update -v /", "", "", "Updated: /\n", true)

	xCLI(t, "update -vo json /", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 6,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "Updated: /\n", true)

	xCLI(t, "update -o=json / --set name=myreg", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 7,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "name": "myreg",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	xCLI(t, "update -o=json / --set name= --set=description=cool", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "description": "cool",
  "epoch": 8,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "name": "",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	xCLI(t, "update -o=json / --set name --set=description=5", "",
		"", "Attribute \"description\" must be a string\n", false)

	xCLI(t, "update -o=json / --set name --set=description=\"5\"", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "description": "5",
  "epoch": 9,
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "registryid": "TestXRUpdateRegistry",
  "self": "http://localhost:8181/",
  "specversion": "`+SPECVERSION+`",
  "xid": "/"
}
`, "", true)

	xCLI(t, "update -o=json --set=labels.foo=5 --del description "+
		"--del=labels", "",
		"", "Attribute \"labels.foo\" must be a string\n", false)

	xCLI(t, "update -o=json --set=labels.foo=\"5\" --del description "+
		"--del=labels", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 10,
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

	xCLI(t, "update -o=json --add=labels.bar=zzz", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 11,
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

	xCLI(t, "update -o=json --add=labels.yyy=yay --del=labels.foo", "",
		`{
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "epoch": 12,
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

func TestXRGroup(t *testing.T) {
	reg := NewRegistry("TestXRGroup")
	defer PassDeleteReg(t, reg)

	xCLIServer("localhost:8181")

	xCLI(t, "model group create", "",
		"", "At least one Group type name must be specified\n", false)

	xCLI(t, "model group create dirs", "",
		"", "Group type name must be of the form: PLURAL:SINGULAR\n", false)

	xCLI(t, "model group create dirs dir", "",
		"", "Group type name must be of the form: PLURAL:SINGULAR\n", false)

	xCLI(t, "model group create dirs:dir", "", "", "", true)

	xCLI(t, "model group create dirs:dir", "", "",
		"PLURAL value (dirs) conflicts with an existing Group PLURAL name\n",
		false)

	xCLI(t, "model group create -o table -v dirs2:dir2", "",
		"GROUP: dirs2 / dir2\n", "Created Group type: dirs2:dir2\n", true)

	xCLI(t, "model group create dirs3:dir3 -v -o json", "",
		`{
  "plural": "dirs3",
  "singular": "dir3",
  "attributes": {
    "createdat": {
      "name": "createdat",
      "type": "timestamp",
      "required": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "dir3id": {
      "name": "dir3id",
      "type": "string",
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

	xCLI(t, "model group create -ao table -v dirs4:dir4", "",
		`GROUP: dirs4 / dir4

  ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
  createdat     timestamp   y   -  y   
  description   string      -   -  y   
  dir4id        string      y   -  -   
  documentation url         -   -  y   
  epoch         uinteger    y   y  y   
  icon          url         -   -  y   
  labels        map(string) -   -  y   
  modifiedat    timestamp   y   -  y   
  name          string      -   -  y   
  self          url         y   y  -   
  xid           xid         y   y  -   
`, "Created Group type: dirs4:dir4\n", true)

	xCLI(t, "model group create -aro table dirs5:dir5", "",
		`GROUP: dirs5 / dir5

  ATTRIBUTES:   TYPE        REQ RO MUT DEFAULT
  createdat     timestamp   y   -  y   
  description   string      -   -  y   
  dir5id        string      y   -  -   
  documentation url         -   -  y   
  epoch         uinteger    y   y  y   
  icon          url         -   -  y   
  labels        map(string) -   -  y   
  modifiedat    timestamp   y   -  y   
  name          string      -   -  y   
  self          url         y   y  -   
  xid           xid         y   y  -   
`, "", true)

	xCLI(t, "model group create -aro xxx dirs6:dir6", "",
		"", "--output must be one of 'json', 'none', 'table'\n", false)

	xCLI(t, "model group get dirs2", "", "GROUP: dirs2 / dir2\n", "", true)

	xCLI(t, "model group get -o json dirs2", "",
		`{
  "plural": "dirs2",
  "singular": "dir2",
  "attributes": {
    "createdat": {
      "name": "createdat",
      "type": "timestamp",
      "required": true
    },
    "description": {
      "name": "description",
      "type": "string"
    },
    "dir2id": {
      "name": "dir2id",
      "type": "string",
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

	xCLI(t, "model group get -a -o json dirs2 dirs", "",
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
      "description": {
        "name": "description",
        "type": "string"
      },
      "dir2id": {
        "name": "dir2id",
        "type": "string",
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
      "description": {
        "name": "description",
        "type": "string"
      },
      "dirid": {
        "name": "dirid",
        "type": "string",
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

	/*
			xCLI(t, "model group get -r -o table dirs2 dirs", "",
				`
		`, "", true)
	*/

}
