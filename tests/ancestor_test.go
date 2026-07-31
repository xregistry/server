package tests

// err missing ancestor

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestAncestorBasic(t *testing.T) {
	reg := NewRegistry("TestAncestorBasic")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f2", `{}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2/versions/1",
  "xid": "/dirs/d1/files/f2/versions/1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-04-11T20:40:37.146317496Z",
  "modifiedat": "2025-04-11T20:40:37.146317496Z",
  "ancestorid": "1"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "v1": {}
}`, 200, `{
  "v1": {
    "fileid": "f3",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/v1",
    "xid": "/dirs/d1/files/f3/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  }
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f4/versions/v1", `{}`, 201, `{
  "fileid": "f4",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f4/versions/v1",
  "xid": "/dirs/d1/files/f4/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files", ``, 204, ``)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{"ancestorid": null}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestorid": ""}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"ancestorid\" for \"/dirs/d1/files/f2\" is not valid: value \"\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "error_detail": "value \"\" must match: ^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~:@]{0,127}$",
    "name": "ancestorid"
  },
  "source": "e4e59b8a76c4:registry:shared_model:71"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestorid": "vx"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f2\", the \"version\" with a \"versionid\" value of \"vx\" cannot be found.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "id": "vx",
    "singular": "version"
  },
  "source": ":registry:resource:1534"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestorid": "1"}`, 201, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestorid": "1"}`, 200, `{
  "fileid": "f2",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f2",
  "xid": "/dirs/d1/files/f2",
  "epoch": 2,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f2/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f2/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2", `{"ancestorid": "2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f2\", the \"version\" with a \"versionid\" value of \"2\" cannot be found.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "id": "2",
    "singular": "version"
  },
  "source": ":registry:resource:1534"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f2", `{"ancestorid": "2"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f2\", the \"version\" with a \"versionid\" value of \"2\" cannot be found.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "id": "2",
    "singular": "version"
  },
  "source": ":registry:resource:1534"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f3", `{"ancestorid": "1"}`, 201, `{
  "fileid": "f3",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f3",
  "xid": "/dirs/d1/files/f3",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f3/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f3/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f3", `{"ancestorid": "2"}`, 201, `{
  "fileid": "f3",
  "versionid": "2",
  "self": "http://localhost:8181/dirs/d1/files/f3/versions/2",
  "xid": "/dirs/d1/files/f3/versions/2",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "2"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5": {}, "3":{}, "4":{}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "2"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "3"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "4"
  }
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5":{"createdat": "2023-01-01T12:00:00Z","ancestorid":null},
  "3":{"ancestorid":null},
  "4":{"ancestorid":null}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 2,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "5"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 2,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "3"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 2,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "2"
  }
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f3/versions", `{
  "5": {"createdat":null, "ancestorid":null},
  "4":{ "ancestorid": "1"},
  "3":{"ancestorid": null}
}`, 200, `{
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "4"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "1"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 3,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "3"
  }
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1": {"ancestorid":"3"}, "2":{}, "3":{}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f4\", the request would create a circular list of ancestors: 1, 2, 3.",
  "subject": "/dirs/d1/files/f4",
  "args": {
    "list": "1, 2, 3"
  },
  "source": ":registry:resource:1598"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1": {"ancestorid":"2"}, "2":{}, "3":{}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f4\", the request would create a circular list of ancestors: 1, 2.",
  "subject": "/dirs/d1/files/f4",
  "args": {
    "list": "1, 2"
  },
  "source": ":registry:resource:1598"
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f4/versions", `{
  "1":{"ancestorid":"2"}, "2":{"ancestorid":"1"},
  "3":{"ancestorid":"4"}, "4":{"ancestorid":"3"}
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f4\", the request would create a circular list of ancestors: 1, 2, 3, 4.",
  "subject": "/dirs/d1/files/f4",
  "args": {
    "list": "1, 2, 3, 4"
  },
  "source": ":registry:resource:1598"
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f3/versions/1", `{}`, 204, ``)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f3/versions", ``, 200, `{
  "2": {
    "fileid": "f3",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/2",
    "xid": "/dirs/d1/files/f3/versions/2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "2"
  },
  "3": {
    "fileid": "f3",
    "versionid": "3",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/3",
    "xid": "/dirs/d1/files/f3/versions/3",
    "epoch": 3,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
    "ancestorid": "4"
  },
  "4": {
    "fileid": "f3",
    "versionid": "4",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/4",
    "xid": "/dirs/d1/files/f3/versions/4",
    "epoch": 4,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
    "ancestorid": "4"
  },
  "5": {
    "fileid": "f3",
    "versionid": "5",
    "self": "http://localhost:8181/dirs/d1/files/f3/versions/5",
    "xid": "/dirs/d1/files/f3/versions/5",
    "epoch": 3,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:03Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
    "ancestorid": "3"
  }
}
`)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f5/versions", `{"1":{}, "2":{}}`,
		200, `{
  "1": {
    "fileid": "f5",
    "versionid": "1",
    "self": "http://localhost:8181/dirs/d1/files/f5/versions/1",
    "xid": "/dirs/d1/files/f5/versions/1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1"
  },
  "2": {
    "fileid": "f5",
    "versionid": "2",
    "self": "http://localhost:8181/dirs/d1/files/f5/versions/2",
    "xid": "/dirs/d1/files/f5/versions/2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1"
  }
}
`)

	// Make sure ancestor doesn't get erased
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f5/versions/1", `{}`, 200, `{
  "fileid": "f5",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f5/versions/1",
  "xid": "/dirs/d1/files/f5/versions/1",
  "epoch": 2,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "1"
}
`)

	// all epochs should be 1 after this
	XHTTP(t, reg, "POST", "/dirs/d1/files/f6/versions",
		`{"v1":{}, "v2":{}, "v3":{}}`, 200, `{
  "v1": {
    "fileid": "f6",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v1",
    "xid": "/dirs/d1/files/f6/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  },
  "v2": {
    "fileid": "f6",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v2",
    "xid": "/dirs/d1/files/f6/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  },
  "v3": {
    "fileid": "f6",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f6/versions/v3",
    "xid": "/dirs/d1/files/f6/versions/v3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v2"
  }
}
`)

	// now delete v1 which causes v2's ancestor, epoch and modifiedat to
	// change. Note if we ever optimize things such that when we delete all
	// versions we just delete the resource instead, we'll need to find a
	// different way to make this side-effect aspect is tested some other way:w
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f6/versions",
		`{"v1":{"epoch":1}, "v2":{"epoch": 1}, "v3":{"epoch":1}}`, 204, ``)

}

func TestAncestorWithSicky(t *testing.T) {
	reg := NewRegistry("TestAncestorWithSticky")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?inline=meta", `{
      "meta":{"defaultversionsticky": true,"defaultversionid": "v1"},
      "versions":{"v1":{},"v2":{},"v3":{}}
    }`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,1)(v3->v2,2)")

	model2 := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 2
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model2, 200, model2+"\n")

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v3->v3,0)")
}

func TestAncestorOrdering(t *testing.T) {
	reg := NewRegistry("TestAncestorOrdering")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Timestamps should be the determining factor.
	// "versionsid" make sure we don't create the implied Version "1"
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00" },
    "v2": { "createdat": "2024-01-01T12:00:00" },
    "V3": { "createdat": "2023-01-01T12:00:00" },
    "v4": { "createdat": "2022-01-01T12:00:00" }
  }
}`, 201, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v4->v4,0)(V3->v4,1)(v2->V3,1)(v1->v2,2)")

	// Reverse the order of the timestamps, and clear ancestor
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2022-01-01T12:00:00", "ancestorid": null },
    "v2": { "createdat": "2023-01-01T12:00:00", "ancestorid": null },
    "V3": { "createdat": "2024-01-01T12:00:00", "ancestorid": null },
    "v4": { "createdat": "2025-01-01T12:00:00", "ancestorid": null }
  }
}`, 200, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,1)(V3->v2,1)(v4->V3,2)")

	// Make it into a tree 1<-2,3,4 diff timestamps
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00", "ancestorid": "v1" },
    "v2": { "createdat": "2023-01-01T12:00:00", "ancestorid": "v1" },
    "V3": { "createdat": "2024-01-01T12:00:00", "ancestorid": "v1" },
    "v4": { "createdat": "2022-01-01T12:00:00", "ancestorid": "v1" }
  }
}`, 200, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v4->v1,2)(v2->v1,2)(V3->v1,2)")

	// Same, but use same TS, so it'll alphabetize things (case insense)
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00"},
    "v2": { "createdat": "2023-01-01T12:00:00"},
    "V3": { "createdat": "2023-01-01T12:00:00"},
    "v4": { "createdat": "2023-01-01T12:00:00"}
  }
}`, 200, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,2)(V3->v1,2)(v4->v1,2)")

	// Deep tree and add a new more
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{
  "versions": {
    "v1": { "ancestorid": "v1" },
    "v2": { "ancestorid": "v1" },
    "V3": { "ancestorid": "v2" },
    "v4": { "ancestorid": "V3" },

    "v1.1.0": { "ancestorid": "v1" },
    "v1.1.1": { "ancestorid": "v1.1.0" },

    "v2.1.0": { "ancestorid": "v2" }
  }
}`, 200, `*`)

	// v4 is older than v1.1.. and v2, and then v1.1 < v2 alphabetically
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,1)(V3->v2,1)(v1.1.0->v1,1)(v4->V3,2)(v1.1.1->v1.1.0,2)(v2.1.0->v2,2)")

}

func TestAncestorRoots(t *testing.T) {
	reg := NewRegistry("TestAncestorOrdering")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Start with singlversionroot=default (which should be 'false')

	// Timestamps should be the determining factor.
	// "versions" makes sure we don't create the implied Version "1"
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00", "ancestorid":"v1" },
    "v2": { "createdat": "2024-01-01T12:00:00", "ancestorid":"v2" }
  }
}`, 201, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"), "(v2->v2,0)(v1->v1,0)")

	model2 := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "singleversionroot": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model2, 200, model2+"\n")

	// Trying to turn singleversionroot=true should generate an error
	model3 := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "singleversionroot": true
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model3, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1778"
}
`)

	// convert a root into a leaf and try again
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v2",
		`{"ancestorid":"v1"}`, 200, `*`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"), "(v1->v1,0)(v2->v1,2)")

	XHTTP(t, reg, "PUT", "/modelsource", model3, 200, model3+"\n")

	// make sure an add of a root fails
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3",
		`{"ancestorid":"v3"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions/v3",
		`{"ancestorid":"v3"}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1/versions",
		`{"v3":{"ancestorid":"v3"}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1",
		`{"versions":{"v3":{"ancestorid":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v3":{"ancestorid":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f1\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestorid":"v1"},"v3":{"ancestorid":"v3"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#multiple_roots",
  "title": "The operation would result in multiple root Versions for \"/dirs/d1/files/f2\", which is not allowed for \"files\".",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "plural": "files"
  },
  "source": ":registry:resource:1408"
}
`)

}

func TestAncestorCircles(t *testing.T) {
	reg := NewRegistry("TestAncestorCircles")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestorid":"v1"},"v2":{"ancestorid":"v1"}}}`,
		201, `*`)

	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestorid":"v2"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f1\", the request would create a circular list of ancestors: v1, v2.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "list": "v1, v2"
  },
  "source": ":registry:resource:1598"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestorid":"v2"},"v2":{"ancestorid":"v1"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f2\", the request would create a circular list of ancestors: v1, v2.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "list": "v1, v2"
  },
  "source": ":registry:resource:1598"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2",
		`{"versions":{"v1":{"ancestorid":"v2"},"v2":{"ancestorid":"v1"},
		              "v3":{"ancestorid":"v4"},"v4":{"ancestorid":"v3"}}}`,
		400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#ancestor_circular_reference",
  "title": "For \"/dirs/d1/files/f2\", the request would create a circular list of ancestors: v1, v2, v3, v4.",
  "subject": "/dirs/d1/files/f2",
  "args": {
    "list": "v1, v2, v3, v4"
  },
  "source": ":registry:resource:1598"
}
`)

}

func TestAncestorMaxVersions(t *testing.T) {
	reg := NewRegistry("TestAncestorCircles")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 1,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false,
              "enum": [
                false
              ]
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// the circular ref shouldn't be an issue because we'll delete the
	// oldest one due to maxversions
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions":{
    "v1":{"ancestorid":"v2"},
    "v2":{"ancestorid":"v1"}
  }
}`,
		201, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	//  v2->v1->v3->v3
	// Should delete v3
	model2 := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "maxversions": 2,
          "metaattributes": {
            "defaultversionsticky": {
              "type": "boolean",
              "required": true,
              "default": false
            }
          }
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model2, 200, model2+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestorid":"v1"},"v2":{"ancestorid":"v1"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions":{"v1":{"ancestorid":"v3"},"v3":{"ancestorid":"v3"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,2)")

	// v3->v2->v1 + default=v1/sticky
	// should delete v2, v3 becomes root
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"meta":{"defaultversionid":"v1","defaultversionsticky":true},
          "versions":{"v1":{"ancestorid":"v1"},"v3":{"ancestorid":"v2"}}}`,
		200, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 3,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v3->v3,0)")
}

func TestAncestorErrors(t *testing.T) {
	reg := NewRegistry("TestAncestorErrors")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestorid":"v2"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"v2\" cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "v2",
    "singular": "version"
  },
  "source": ":registry:resource:1534"
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestorid":"v1"}}}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1",
		`{"versions": {"v1":{"ancestorid":"v2"}}}`, 400,
		`{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"v2\" cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "v2",
    "singular": "version"
  },
  "source": ":registry:resource:1534"
}
`)

}

// VAS2String fetches all Versions of the Resource at "xid" (via HTTP) and
// builds a "(vid->ancestorid,pos)" string for each one - just like the old
// Go-API-only GetOrderedVersionIDs()/VersionAncestor mechanism did. Pos is
// derived the same way the server's "VersionAncestors" SQL view computes it
// (registry/init.sql): 0=root (vid==ancestorid), 1=middle (some other
// Version's ancestorid points at it), 2=leaf (otherwise). The ORDER of the
// entries is just sorted by (pos,vid) for determinism - it's not guaranteed
// to match the server's own internal ordering, but each individual
// "vid->ancestorid,pos" tuple is what matters for these tests.
func VAS2String(t *testing.T, reg *registry.Registry, xid string) string {
	res := XHTTP(t, reg, "GET", xid+"/versions", "", 200, `*`)
	versions := res.ToMap()

	ancestors := map[string]string{}
	createdAt := map[string]string{}
	for vid, v := range versions {
		vm := v.(map[string]any)
		ancestors[vid] = vm["ancestorid"].(string)
		createdAt[vid] = vm["createdat"].(string)
	}

	hasChild := map[string]bool{}
	for vid, aid := range ancestors {
		if vid != aid {
			hasChild[aid] = true
		}
	}

	pos := func(vid string) string {
		if vid == ancestors[vid] {
			return "0"
		}
		if hasChild[vid] {
			return "1"
		}
		return "2"
	}

	vids := []string{}
	for vid := range ancestors {
		vids = append(vids, vid)
	}
	sort.Slice(vids, func(i, j int) bool {
		if posI, posJ := pos(vids[i]), pos(vids[j]); posI != posJ {
			return posI < posJ
		}
		if createdAt[vids[i]] != createdAt[vids[j]] {
			return createdAt[vids[i]] < createdAt[vids[j]]
		}
		// The VersionUID DB column uses a case-insensitive collation, so
		// match that here for the final tiebreaker.
		return strings.ToLower(vids[i]) < strings.ToLower(vids[j])
	})

	str := ""
	for _, vid := range vids {
		str += fmt.Sprintf("(%s->%s,%s)", vid, ancestors[vid], pos(vid))
	}
	return str
}
