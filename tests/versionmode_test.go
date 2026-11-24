package tests

// Tests for the "modifiedat" and "semver" versionmode algorithms (see
// registry/versionmodes.go). Modeled after the "createdat" coverage in
// ancestor_test.go (TestAncestorOrdering in particular).

import (
	"testing"

	. "github.com/xregistry/server/common"
)

func TestVersionModeModifiedat(t *testing.T) {
	reg := NewRegistry("TestVersionModeModifiedat")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "versionmode": "modifiedat"
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// "modifiedat" (not "createdat") should be the determining factor.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions": {
    "v1": { "modifiedat": "2025-01-01T12:00:00" },
    "v2": { "modifiedat": "2024-01-01T12:00:00" },
    "v3": { "modifiedat": "2023-01-01T12:00:00" },
    "v4": { "modifiedat": "2022-01-01T12:00:00" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 4
}
`)

	res := XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `{
  "v1": {
    "fileid": "f1",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
    "xid": "/dirs/d1/files/f1/versions/v1",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "v2"
  },
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:03Z",
    "ancestorid": "v3"
  },
  "v3": {
    "fileid": "f1",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
    "xid": "/dirs/d1/files/f1/versions/v3",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:04Z",
    "ancestorid": "v4"
  },
  "v4": {
    "fileid": "f1",
    "versionid": "v4",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v4",
    "xid": "/dirs/d1/files/f1/versions/v4",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:05Z",
    "ancestorid": "v4"
  }
}
`)
	versions := res.ToMap()

	expected := map[string]string{
		"v1": "v2",
		"v2": "v3",
		"v3": "v4",
		"v4": "v4",
	}
	for vid, exp := range expected {
		v, ok := versions[vid].(map[string]any)
		if !ok {
			t.Fatalf("missing version %q in response", vid)
		}
		got, _ := v["ancestorid"].(string)
		if got != exp {
			t.Errorf("version %q: got ancestorid=%q, want %q", vid, got, exp)
		}
	}
}

func TestVersionModeSemver(t *testing.T) {
	reg := NewRegistry("TestVersionModeSemver")
	defer PassDeleteReg(t, reg)

	model := `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "setversionid": true,
          "versionmode": "semver"
        }
      }
    }
  }
}`
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Ordering (oldest->newest) per semver precedence rules should be:
	// 1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-beta < 1.0.0 < 1.5.0 < 2.0.0
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "2.0.0",
  "versions": {
    "2.0.0": {},
    "1.5.0": {},
    "1.0.0": {},
    "1.0.0-beta": {},
    "1.0.0-alpha.1": {},
    "1.0.0-alpha": {}
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "2.0.0",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "1.5.0",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 6
}
`)

	res := XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `{
  "1.0.0": {
    "fileid": "f1",
    "versionid": "1.0.0",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0",
    "xid": "/dirs/d1/files/f1/versions/1.0.0",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-beta"
  },
  "1.0.0-alpha": {
    "fileid": "f1",
    "versionid": "1.0.0-alpha",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-alpha",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-alpha",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha"
  },
  "1.0.0-alpha.1": {
    "fileid": "f1",
    "versionid": "1.0.0-alpha.1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-alpha.1",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-alpha.1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha"
  },
  "1.0.0-beta": {
    "fileid": "f1",
    "versionid": "1.0.0-beta",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-beta",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-beta",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha.1"
  },
  "1.5.0": {
    "fileid": "f1",
    "versionid": "1.5.0",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.5.0",
    "xid": "/dirs/d1/files/f1/versions/1.5.0",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0"
  },
  "2.0.0": {
    "fileid": "f1",
    "versionid": "2.0.0",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/2.0.0",
    "xid": "/dirs/d1/files/f1/versions/2.0.0",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.5.0"
  }
}
`)
	versions := res.ToMap()

	expected := map[string]string{
		"1.0.0-alpha":   "1.0.0-alpha",
		"1.0.0-alpha.1": "1.0.0-alpha",
		"1.0.0-beta":    "1.0.0-alpha.1",
		"1.0.0":         "1.0.0-beta",
		"1.5.0":         "1.0.0",
		"2.0.0":         "1.5.0",
	}
	for vid, exp := range expected {
		v, ok := versions[vid].(map[string]any)
		if !ok {
			t.Fatalf("missing version %q in response", vid)
		}
		got, _ := v["ancestorid"].(string)
		if got != exp {
			t.Errorf("version %q: got ancestorid=%q, want %q", vid, got, exp)
		}
	}

	// Deleting the middle of the chain should splice it, not orphan the
	// remaining Versions (see chainWillDelete() in versionmodes.go).
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/1.0.0", "", 204, ``)

	res = XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `{
  "1.0.0-alpha": {
    "fileid": "f1",
    "versionid": "1.0.0-alpha",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-alpha",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-alpha",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha"
  },
  "1.0.0-alpha.1": {
    "fileid": "f1",
    "versionid": "1.0.0-alpha.1",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-alpha.1",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-alpha.1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha"
  },
  "1.0.0-beta": {
    "fileid": "f1",
    "versionid": "1.0.0-beta",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.0.0-beta",
    "xid": "/dirs/d1/files/f1/versions/1.0.0-beta",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.0.0-alpha.1"
  },
  "1.5.0": {
    "fileid": "f1",
    "versionid": "1.5.0",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/1.5.0",
    "xid": "/dirs/d1/files/f1/versions/1.5.0",
    "epoch": 2,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "1.0.0-beta"
  },
  "2.0.0": {
    "fileid": "f1",
    "versionid": "2.0.0",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/2.0.0",
    "xid": "/dirs/d1/files/f1/versions/2.0.0",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "1.5.0"
  }
}
`)
	versions = res.ToMap()
	v15, ok := versions["1.5.0"].(map[string]any)
	if !ok {
		t.Fatalf("missing version 1.5.0 in response")
	}
	if got, _ := v15["ancestorid"].(string); got != "1.0.0-beta" {
		t.Errorf("version 1.5.0: got ancestorid=%q, want %q", got, "1.0.0-beta")
	}
}

func vmModel(mode string) string {
	return `{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {
          "singular": "file",
          "hasdocument": false,
          "setversionid": true,
          "versionmode": "` + mode + `"
        }
      }
    }
  }
}`
}

func TestVersionModeCreatedat(t *testing.T) {
	reg := NewRegistry("TestVersionModeCreatedat")
	defer PassDeleteReg(t, reg)

	model := vmModel("createdat")
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions": {
    "v1": { "createdat": "2025-01-01T12:00:00Z" },
    "v2": { "createdat": "2024-01-01T12:00:00Z" },
    "v3": { "createdat": "2023-01-01T12:00:00Z" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v3->v3,0)(v2->v3,1)(v1->v2,2)")

	// Delete middle: chain is spliced
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2", "", 204, ``)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v3->v3,0)(v1->v3,2)")

	// Delete the oldest: next one becomes the root
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", "", 204, ``)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)")

	// A new Version with a middle timestamp slots in properly
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v0", `{
  "createdat": "2020-01-01T12:00:00Z"
}`, 201, `{
  "fileid": "f1",
  "versionid": "v0",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v0",
  "xid": "/dirs/d1/files/f1/versions/v0",
  "epoch": 1,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v0"
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/vm", `{
  "createdat": "2022-01-01T12:00:00Z"
}`, 201, `{
  "fileid": "f1",
  "versionid": "vm",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/vm",
  "xid": "/dirs/d1/files/f1/versions/vm",
  "epoch": 1,
  "isdefault": false,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v0"
}
`)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v0->v0,0)(vm->v0,1)(v1->vm,2)")
}

func TestVersionModeModifiedatDelete(t *testing.T) {
	reg := NewRegistry("TestVersionModeModifiedatDelete")
	defer PassDeleteReg(t, reg)

	model := vmModel("modifiedat")
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1", `{
  "versionid": "v1",
  "versions": {
    "v1": { "modifiedat": "2025-01-01T12:00:00Z" },
    "v2": { "modifiedat": "2024-01-01T12:00:00Z" },
    "v3": { "modifiedat": "2023-01-01T12:00:00Z" }
  }
}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",

  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2", "", 204, ``)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v3->v3,0)(v1->v3,2)")
}

func TestVersionModeSemverOrdering(t *testing.T) {
	reg := NewRegistry("TestVersionModeSemverOrdering")
	defer PassDeleteReg(t, reg)

	model := vmModel("semver")
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Numeric (not lexical) comparison, and inserting one at a time
	for _, vid := range []string{"1.10.0", "1.9.0", "1.2.0", "0.9.9"} {
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/"+vid, `{}`,
			201, `*`)
	}
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(0.9.9->0.9.9,0)(1.9.0->1.2.0,1)(1.2.0->0.9.9,1)(1.10.0->1.9.0,2)")

	// Delete the newest, then the oldest
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/1.10.0", "", 204, ``)
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/0.9.9", "", 204, ``)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(1.2.0->1.2.0,0)(1.9.0->1.2.0,2)")

	// Non-semver IDs must not break anything (lenient compare)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/foo", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "foo",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/foo",
  "xid": "/dirs/d1/files/f1/versions/foo",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "1.9.0"
}
`)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"), "(1.2.0->1.2.0,0)(1.9.0->1.2.0,1)(foo->1.9.0,2)")
}

func TestVersionModeManual(t *testing.T) {
	reg := NewRegistry("TestVersionModeManual")
	defer PassDeleteReg(t, reg)

	model := vmModel("manual")
	XHTTP(t, reg, "PUT", "/modelsource", model, 200, model+"\n")

	// Unspecified ancestors chain onto the newest Version
	for _, vid := range []string{"v1", "v2", "v3"} {
		XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/"+vid, `{}`,
			201, `*`)
	}
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,1)(v3->v2,2)")

	// Ancestor pointing to a non-existing Version is an error
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v4",
		`{"ancestorid":"nope"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#unknown_id",
  "title": "While processing \"/dirs/d1/files/f1\", the \"version\" with a \"versionid\" value of \"nope\" cannot be found.",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "id": "nope",
    "singular": "version"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)

	// Explicit ancestor is honored
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v5",
		`{"ancestorid":"v1"}`, 201, `{
  "fileid": "f1",
  "versionid": "v5",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v5",
  "xid": "/dirs/d1/files/f1/versions/v5",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1"
}
`)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v2->v1,1)(v3->v2,2)(v5->v1,2)")

	// Deleting a Version turns its children into roots
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v2", "", 204, ``)
	XEqual(t, "", VAS2String(t, reg, "/dirs/d1/files/f1"),
		"(v1->v1,0)(v3->v3,0)(v5->v1,2)")
}

func TestVersionModeInvalid(t *testing.T) {
	reg := NewRegistry("TestVersionModeInvalid")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", vmModel("bogus"), 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#model_error",
  "title": "There was an error in the model definition provided: Resource \"files\" has an invalid 'versionmode' value (bogus). Must be one of: createdat,manual,modifiedat,semver.",
  "subject": "/model",
  "args": {
    "error_detail": "Resource \"files\" has an invalid 'versionmode' value (bogus). Must be one of: createdat,manual,modifiedat,semver"
  },
  "instance": "xxx",
  "source": "xxx"
}
`)
}
