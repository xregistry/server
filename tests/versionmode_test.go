package tests

// Tests for the "modifiedat" and "semver" versionmode algorithms (see
// registry/versionmodes.go). Modeled after the "createdat" coverage in
// ancestor_test.go (TestAncestorOrdering in particular).

import (
	"testing"
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
}`, 201, `*`)

	res := XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `*`)
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
}`, 201, `*`)

	res := XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `*`)
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

	res = XHTTP(t, reg, "GET", "/dirs/d1/files/f1/versions", "", 200, `*`)
	versions = res.ToMap()
	v15, ok := versions["1.5.0"].(map[string]any)
	if !ok {
		t.Fatalf("missing version 1.5.0 in response")
	}
	if got, _ := v15["ancestorid"].(string); got != "1.0.0-beta" {
		t.Errorf("version 1.5.0: got ancestorid=%q, want %q", got, "1.0.0-beta")
	}
}
