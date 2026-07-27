package tests

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

// This file covers the "MISSING TEST SCENARIOS" identified in the repo-root
// `gaps` write-up (xref test coverage gap analysis). Each test below is
// numbered/named to match the corresponding item in that document.

// TestXrefFormatCascadeOnDirectSave (gaps item 1, remaining half) checks
// that formatvalidated correctly cascades into an xref source through the
// NORMAL per-Resource save path (runCascade() called directly from
// ValidateResource(), not via Registry.VerifyData()'s model-driven
// revalidation pass, which is what tests/xref_model_revalidation_test.go
// covers instead). format_test.go has zero xref mentions, so this was
// completely unverified before.
func TestXrefFormatCascadeOnDirectSave(t *testing.T) {
	reg := NewRegistry("TestXrefFormatCascadeOnDirectSave")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
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
          "validatecompatibility": false,
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Create the target with a valid "numbers" document straight away.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "format": "numbers",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:29.606428854Z",
  "modifiedat": "2026-07-27T00:24:29.606428854Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Create the xref source pointing at it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:24:31.661658328Z",
  "modifiedat": "2026-07-27T00:24:31.661658328Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// The mirror must already show formatvalidated=true.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:33.708753687Z",
  "modifiedat": "2026-07-27T00:24:33.708753687Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now directly PUT a new (still valid) document onto the TARGET - a
	// perfectly normal save, no model change involved at all.
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
  "v2": {
    "format": "numbers",
    "file": "2"
  }
}`, 200, `{
  "v2": {
    "fileid": "f1",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/f1/versions/v2$details",
    "xid": "/dirs/d1/files/f1/versions/v2",
    "epoch": 1,
    "isdefault": true,
    "createdat": "2026-07-27T00:24:35.911759653Z",
    "modifiedat": "2026-07-27T00:24:35.911759653Z",
    "ancestorid": "1",
    "contenttype": "application/json",
    "format": "numbers",
    "formatvalidated": true
  }
}
`)

	// The xref mirror must reflect both the new default Version and the
	// (still true) formatvalidated flag for it.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:37.975662554Z",
  "modifiedat": "2026-07-27T00:24:37.975662554Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 2
}
`)
}

// TestConstraintsModelLevelXrefRealValue covers gaps items 2 and 3
// together: a constraint defined at the MODEL level (group-model-wide,
// not just on one group instance - see TestConstraintsXref, which only
// exercised a group-INSTANCE constraint), combined with a xref target that
// has a REAL, explicitly-set value for the constrained attribute (not the
// constraint's default). The xref mirror must show the target's real
// value, not independently re-derive the model-level default for itself.
func TestConstraintsModelLevelXrefRealValue(t *testing.T) {
	reg := NewRegistry("TestConstraintsModelLevelXrefRealValue")
	defer PassDeleteReg(t, reg)

	// Model-level (group-model-wide) constraint default - applies to
	// every group instance of type "dirs", not just one.
	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "constraints": { "files.name": { "default": "model-default" } },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {} } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// Target t1, in d1, with an EXPLICIT (non-default) value.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details",
		`{"name":"explicit-name"}`, 201,
		`{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "name": "explicit-name",
  "isdefault": true,
  "createdat": "2026-07-27T00:24:40.026517496Z",
  "modifiedat": "2026-07-27T00:24:40.026517496Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// Sanity: a sibling resource in the SAME (also constrained) group,
	// created with no explicit value, DOES get the model-level default.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/other$details", `{}`, 201,
		`{
  "fileid": "other",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/other",
  "xid": "/dirs/d1/files/other",
  "epoch": 1,
  "name": "model-default",
  "isdefault": true,
  "createdat": "2026-07-27T00:24:42.10898385Z",
  "modifiedat": "2026-07-27T00:24:42.10898385Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/other/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/other/versions",
  "versionscount": 1
}
`)

	// Xref fx, in a DIFFERENT group instance (d2 - which is subject to
	// the exact same model-level constraint, since it's model-wide, not
	// per-instance) pointing at t1.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:24:44.057664921Z",
  "modifiedat": "2026-07-27T00:24:44.057664921Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	// The mirror MUST show t1's real value ("explicit-name"), not
	// "model-default" - proving the mirror copies the target's actual
	// current value rather than the xref resource re-computing its own
	// (model-wide) constraint default.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "name": "explicit-name",
  "isdefault": true,
  "createdat": "2026-07-27T00:24:46.14321773Z",
  "modifiedat": "2026-07-27T00:24:46.14321773Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefEqualsEnforcedOnXref (gaps item 4) verifies that "equals" group
// constraints ARE enforced against a xref's mirrored value, exactly like
// any other Version's - since Group.Validate() (registry/group.go) scans
// FullEntities/FullTreeTable broadly, without distinguishing real Version
// rows from xref-mirrored (IsXrefVerCopy) ones. The target resource lives
// in an unconstrained group (so its own save succeeds) with a value that
// would violate the xref SOURCE's group's "equals" constraint; creating
// the xref itself must fail.
func TestXrefEqualsEnforcedOnXref(t *testing.T) {
	reg := NewRegistry("TestXrefEqualsEnforcedOnXref")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": { "gattr": { "type": "string" } },
	    "constraints": { "files.myattr": { "equals": "gattr" } },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "myattr": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1: gattr="" (unset) - group-level "equals" constraint is skipped
	// entirely for d1 (Group.Validate() bails when the group attr is
	// nil), so t1 can have any myattr value.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details",
		`{"myattr":"not-z"}`, 201, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:48.279000117Z",
  "modifiedat": "2026-07-27T00:24:48.279000117Z",
  "ancestorid": "1",
  "myattr": "not-z",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// d2: gattr="z" - its "equals" constraint requires myattr=="z".
	XHTTP(t, reg, "PATCH", "/dirs/d2", `{"gattr":"z"}`, 201, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2026-07-27T00:24:50.385692582Z",
  "modifiedat": "2026-07-27T00:24:50.385692582Z",
  "gattr": "z",

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	// Creating a REAL resource in d2 with myattr="not-z" fails normally.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/real$details",
		`{"myattr":"not-z"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d2/files/real\" not being compliant with its owning Group's \"equals\" constraint for attribute \"myattr\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d2/files/real",
  "args": {
    "kind": "equals",
    "path": "myattr"
  },
  "source": ":registry:group:895"
}
`)

	// Xref-ing d1's t1 (myattr="not-z") from within d2 (whose "equals"
	// constraint requires myattr=="z") must ALSO be rejected - the
	// mirrored value violates d2's own constraint exactly like a real
	// resource's value would.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d2/files/fx\" not being compliant with its owning Group's \"equals\" constraint for attribute \"myattr\".",
  "detail": "Versions: 1.",
  "subject": "/dirs/d2/files/fx",
  "args": {
    "kind": "equals",
    "path": "myattr"
  },
  "source": ":registry:group:895"
}
`)

	// Fixing t1's value so it DOES satisfy d2's constraint allows the
	// xref to be created.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/t1$details",
		`{"myattr":"z"}`, 200, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:24:56.4441368Z",
  "modifiedat": "2026-07-27T00:24:56.614718Z",
  "ancestorid": "1",
  "myattr": "z",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:24:58.626595317Z",
  "modifiedat": "2026-07-27T00:24:58.626595317Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:00.804455349Z",
  "modifiedat": "2026-07-27T00:25:00.971747273Z",
  "ancestorid": "1",
  "myattr": "z",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefEnumEnforcedOnXref (gaps item 4) is the "enum" analog of
// TestXrefEqualsEnforcedOnXref above: the target resource lives in an
// unconstrained group with a value that would violate the xref SOURCE's
// group's "enum" constraint; creating the xref itself must fail.
func TestXrefEnumEnforcedOnXref(t *testing.T) {
	reg := NewRegistry("TestXrefEnumEnforcedOnXref")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "myattr": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1: no constraint at all - t1 can have any myattr value.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details",
		`{"myattr":"c"}`, 201, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:02.944921365Z",
  "modifiedat": "2026-07-27T00:25:02.944921365Z",
  "ancestorid": "1",
  "myattr": "c",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// d2: a group-instance "enum" constraint restricting files.myattr.
	XHTTP(t, reg, "PATCH", "/dirs/d2",
		`{"constraints":{"files.myattr":{"enum":["a","b"]}}}`, 201, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:04.962217162Z",
  "modifiedat": "2026-07-27T00:25:04.962217162Z",
  "constraints": {
    "files.myattr": {
      "enum": [
        "a",
        "b"
      ]
    }
  },

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	// Confirm the enum constraint IS enforced normally: creating a REAL
	// resource in d2 with myattr="c" (not in the enum) fails.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/real$details",
		`{"myattr":"c"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#invalid_attribute",
  "title": "The attribute \"myattr\" for \"/dirs/d2/files/real/versions/1\" is not valid: value (c) must be one of the enum values: a, b.",
  "subject": "/dirs/d2/files/real/versions/1",
  "args": {
    "error_detail": "value (c) must be one of the enum values: a, b",
    "name": "myattr"
  },
  "source": ":registry:entity:3203"
}
`)

	// Xref-ing d1's t1 (myattr="c") from within d2 (whose "enum"
	// constraint requires "a" or "b") must ALSO be rejected - the
	// mirrored value violates d2's own enum constraint exactly like a
	// real resource's value would.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#constraint_failure",
  "title": "The request would result in one or more Versions of \"/dirs/d2/files/fx\" not being compliant with its owning Group's \"enum\" constraint for attribute \"myattr\".",
  "detail": "Versions: 1. Must be one of: a, b.",
  "subject": "/dirs/d2/files/fx",
  "args": {
    "kind": "enum",
    "path": "myattr"
  },
  "source": ":registry:group:969"
}
`)

	// Fixing t1's value so it DOES satisfy d2's enum allows the xref to
	// be created.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/t1$details",
		`{"myattr":"a"}`, 200, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:10.95808122Z",
  "modifiedat": "2026-07-27T00:25:11.091261947Z",
  "ancestorid": "1",
  "myattr": "a",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:13.068354283Z",
  "modifiedat": "2026-07-27T00:25:13.068354283Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:15.223114541Z",
  "modifiedat": "2026-07-27T00:25:15.391481487Z",
  "ancestorid": "1",
  "myattr": "a",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefConstraintViolationOnTargetUpdateAfterXref checks the reverse
// direction/timing from the two tests above: a xref is created first
// while fully compliant with its own group's "equals"/"enum" constraints,
// then the TARGET is later updated to a value that would violate those
// constraints (via its mirrored value). Since the constraint truly
// belongs to the xref SOURCE's group (not the target's), the target's
// own update must succeed (it isn't itself constrained) - but the
// resulting xref mirror update, if/when re-validated, is what would need
// to be caught. This documents current behavior for that scenario.
func TestXrefConstraintViolationOnTargetUpdateAfterXref(t *testing.T) {
	reg := NewRegistry("TestXrefConstraintViolationOnTargetUpdateAfterXref")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "myattr": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1 (unconstrained): target t1 with myattr="a" - compliant with
	// d2's enum below.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details",
		`{"myattr":"a"}`, 201, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:17.381184216Z",
  "modifiedat": "2026-07-27T00:25:17.381184216Z",
  "ancestorid": "1",
  "myattr": "a",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// d2: enum constraint restricting files.myattr to "a"/"b".
	XHTTP(t, reg, "PATCH", "/dirs/d2",
		`{"constraints":{"files.myattr":{"enum":["a","b"]}}}`, 201, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:19.496446651Z",
  "modifiedat": "2026-07-27T00:25:19.496446651Z",
  "constraints": {
    "files.myattr": {
      "enum": [
        "a",
        "b"
      ]
    }
  },

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	// Xref is created while compliant.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:21.423305562Z",
  "modifiedat": "2026-07-27T00:25:21.423305562Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:23.515923235Z",
  "modifiedat": "2026-07-27T00:25:23.515923235Z",
  "ancestorid": "1",
  "myattr": "a",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)

	// Now update t1 (in the UNCONSTRAINED d1) to a value that would
	// violate d2's enum. t1's own save succeeds (d1 has no constraint
	// on myattr), and its cascade updates fx's mirrored value too -
	// current behavior does NOT re-run d2's Group.Validate() as part of
	// t1's save (only d1's own group is re-validated on a group-level
	// save), so this succeeds even though fx now mirrors a
	// constraint-violating value for d2.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/t1$details",
		`{"myattr":"c"}`, 200, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:25.573062855Z",
  "modifiedat": "2026-07-27T00:25:25.693362417Z",
  "ancestorid": "1",
  "myattr": "c",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:27.731699639Z",
  "modifiedat": "2026-07-27T00:25:27.857600485Z",
  "ancestorid": "1",
  "myattr": "c",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefCompatModeChangeCascade (gaps item 5) combines a Version-content-
// unrelated compatibility MODE change (meta.compatibility, an
// instance-level attribute) with an active xref, exercising the normal
// runCascade() fan-out path (as opposed to the model-driven
// Registry.VerifyData() path already covered by
// tests/xref_model_revalidation_test.go).
func TestXrefCompatModeChangeCascade(t *testing.T) {
	reg := NewRegistry("TestXrefCompatModeChangeCascade")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
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
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// v1 sum=2, v2 sum=2 - equal, so compatible under any mode, including
	// bidirectionally (needed since "full" checks both directions).
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
  "createdat": "2026-07-27T00:25:31.864526553Z",
  "modifiedat": "2026-07-27T00:25:31.864526553Z",
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

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:33.901048058Z",
  "modifiedat": "2026-07-27T00:25:33.901048058Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", ``, 200,
		`{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:35.886918015Z",
  "modifiedat": "2026-07-27T00:25:35.886918015Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:37.948014926Z",
  "modifiedat": "2026-07-27T00:25:37.948014926Z",
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

	// Change ONLY the compatibility MODE on the target - no version
	// content change at all - equal sums remain compatible under "full"
	// (bidirectional) too.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details",
		`{"meta":{"compatibility":"full"}}`, 200,
		`{
  "fileid": "f1",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:39.976131708Z",
  "modifiedat": "2026-07-27T00:25:40.114064757Z",
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

	// The xref mirror must reflect BOTH the new mode and the refreshed
	// compatibilityvalidated result.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", ``, 200,
		`{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "2026-07-27T00:25:42.126789781Z",
  "modifiedat": "2026-07-27T00:25:42.272503534Z",
  "readonly": false,
  "compatibility": "full",

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2$details",
  "defaultversionsticky": false
}
`)
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:44.2221894Z",
  "modifiedat": "2026-07-27T00:25:44.356102502Z",
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

// TestXrefSurvivesModelAttributeRemoval (gaps item 6) verifies that
// removing an attribute definition from the model (that an existing xref
// TARGET's real data still uses) doesn't break or lose the xref'd mirror -
// the value should simply become an untyped extension attribute, still
// present and still correctly mirrored.
func TestXrefSurvivesModelAttributeRemoval(t *testing.T) {
	reg := NewRegistry("TestXrefSurvivesModelAttributeRemoval")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "extra": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details",
		`{"extra":"value1"}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:46.350994676Z",
  "modifiedat": "2026-07-27T00:25:46.350994676Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:48.418456014Z",
  "modifiedat": "2026-07-27T00:25:48.418456014Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:50.468764805Z",
  "modifiedat": "2026-07-27T00:25:50.468764805Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now apply a new model that no longer defines "extra" explicitly,
	// but keeps a "*" wildcard so it survives as an untyped extension
	// attribute (a model without ANY wildcard would instead reject
	// existing "extra" data as an unknown_attribute during revalidation -
	// that's a model-authoring choice, not what this test is about).
	newModelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "*": { "type": "any" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, newModelSrc, true))

	// The target's data survives (now as an untyped extension attr) ...
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200,
		`{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:52.495299534Z",
  "modifiedat": "2026-07-27T00:25:52.495299534Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// ... and so does the xref mirror.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:25:54.526697612Z",
  "modifiedat": "2026-07-27T00:25:54.526697612Z",
  "ancestorid": "1",
  "extra": "value1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefConstraintDefaultTransitionCascade (gaps item 7) checks that
// when an xref TARGET's explicitly-set, constrained attribute is cleared
// (so the owning group's constraint default newly kicks in for it), that
// transition (explicit value -> constraint-derived default) is correctly
// cascaded into any xref SOURCE mirroring that target.
func TestXrefConstraintDefaultTransitionCascade(t *testing.T) {
	reg := NewRegistry("TestXrefConstraintDefaultTransitionCascade")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": { "singular": "dir",
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {} } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1 has a group-instance constraint default for files.name.
	XHTTP(t, reg, "PATCH", "/dirs/d1",
		`{"constraints":{"files.name":{"default":"constrained-default"}}}`,
		201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2026-07-27T00:25:56.630217839Z",
  "modifiedat": "2026-07-27T00:25:56.630217839Z",
  "constraints": {
    "files.name": {
      "default": "constrained-default"
    }
  },

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	// Target t1, in d1, with an EXPLICIT value overriding the default.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details",
		`{"name":"explicit-name"}`, 201,
		`{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "name": "explicit-name",
  "isdefault": true,
  "createdat": "2026-07-27T00:25:58.546653572Z",
  "modifiedat": "2026-07-27T00:25:58.546653572Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// Xref fx, in an unconstrained group d2, pointing at t1.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:00.678975282Z",
  "modifiedat": "2026-07-27T00:26:00.678975282Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "name": "explicit-name",
  "isdefault": true,
  "createdat": "2026-07-27T00:26:02.695894651Z",
  "modifiedat": "2026-07-27T00:26:02.695894651Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)

	// Clear t1's explicit value - d1's constraint default should now
	// kick in for t1 itself.
	XHTTP(t, reg, "PATCH", "/dirs/d1/files/t1$details",
		`{"name":null}`, 200, `{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 2,
  "name": "constrained-default",
  "isdefault": true,
  "createdat": "2026-07-27T00:26:04.746646219Z",
  "modifiedat": "2026-07-27T00:26:04.852643874Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// The xref mirror must pick up the TRANSITION to the newly-applied
	// default value.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 2,
  "name": "constrained-default",
  "isdefault": true,
  "createdat": "2026-07-27T00:26:06.79689196Z",
  "modifiedat": "2026-07-27T00:26:06.904081147Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefGroupConstraintDrivenResaveCascade (gaps item 8) checks that a
// save which is driven by a group-level constraint interaction (an
// "equals" constraint tying a Resource attribute to a Group attribute,
// forcing Tx.AddGroupToValidate()) still correctly re-triggers xref
// fan-out for any xref pointing at the affected Resource - i.e. the
// UsesXref/constraint-triggered-resave interaction isn't accidentally
// skipping the mirror refresh.
func TestXrefGroupConstraintDrivenResaveCascade(t *testing.T) {
	reg := NewRegistry("TestXrefGroupConstraintDrivenResaveCascade")
	defer PassDeleteReg(t, reg)

	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "attributes": { "gattr": { "type": "string" } },
	    "constraints": { "files.myattr": { "equals": "gattr" } },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": { "myattr": { "type": "string" } } } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1: gattr="x", target t1 with myattr="x" (satisfies "equals").
	XHTTP(t, reg, "PUT", "/dirs/d1?inline=files", `{
  "gattr": "x",
  "files": { "t1": { "myattr": "x" } }
}`, 201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:08.943866833Z",
  "modifiedat": "2026-07-27T00:26:08.943866833Z",
  "gattr": "x",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "t1": {
      "fileid": "t1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/t1",
      "xid": "/dirs/d1/files/t1",
      "epoch": 1,
      "isdefault": true,
      "createdat": "2026-07-27T00:26:08.943866833Z",
      "modifiedat": "2026-07-27T00:26:08.943866833Z",
      "ancestorid": "1",
      "myattr": "x",

      "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)

	// Xref fx, in an unconstrained group d2, pointing at t1.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:10.934615061Z",
  "modifiedat": "2026-07-27T00:26:10.934615061Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T00:26:12.920278284Z",
  "modifiedat": "2026-07-27T00:26:12.920278284Z",
  "ancestorid": "1",
  "myattr": "x",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)

	// Change BOTH the group attribute and the Resource's constrained
	// attribute together (so the "equals" constraint still holds) - this
	// forces Tx.AddGroupToValidate() (group-level constraint re-check)
	// AND a real Resource/Version save for t1 in the same request.
	XHTTP(t, reg, "PUT", "/dirs/d1?inline=files", `{
  "gattr": "z",
  "files": { "t1": { "myattr": "z" } }
}`, 200, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 2,
  "createdat": "2026-07-27T00:26:14.952753349Z",
  "modifiedat": "2026-07-27T00:26:15.060297738Z",
  "gattr": "z",

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "files": {
    "t1": {
      "fileid": "t1",
      "versionid": "1",
      "self": "http://localhost:8181/dirs/d1/files/t1",
      "xid": "/dirs/d1/files/t1",
      "epoch": 2,
      "isdefault": true,
      "createdat": "2026-07-27T00:26:14.952753349Z",
      "modifiedat": "2026-07-27T00:26:15.060297738Z",
      "ancestorid": "1",
      "myattr": "z",

      "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
      "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
      "versionscount": 1
    }
  },
  "filescount": 1
}
`)

	// The xref mirror must reflect the new value - proving fan-out still
	// runs correctly for a save driven by this constraint interaction.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T00:26:16.965851585Z",
  "modifiedat": "2026-07-27T00:26:17.074970693Z",
  "ancestorid": "1",
  "myattr": "z",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefMultipleConstraintSourcesCascade (gaps item 9) combines a
// group-INSTANCE constraint default (on the target's own group) with a
// model-level (group-model-wide) constraint default for a DIFFERENT
// attribute, with the xref target and source living in different group
// instances that each have their own instance-level constraint overrides -
// a combinatorial scenario not exercised by any existing xref test.
func TestXrefMultipleConstraintSourcesCascade(t *testing.T) {
	reg := NewRegistry("TestXrefMultipleConstraintSourcesCascade")
	defer PassDeleteReg(t, reg)

	// Model-level default for "name" (applies everywhere unless a group
	// instance overrides it), plus "description" left free for each group
	// instance to default independently via its own constraints.
	modelSrc := `{
	  "groups": { "dirs": {
	    "singular": "dir",
	    "constraints": { "files.name": { "default": "model-name-default" } },
	    "resources": {"files": {"singular": "file", "hasdocument": false,
	      "attributes": {} } } } } }`
	XNoErr(t, reg.Model.ApplyNewModel(nil, modelSrc, true))

	// d1 (target's group) ALSO sets its own instance-level default for
	// "description" (a different attribute than the model-level one).
	XHTTP(t, reg, "PATCH", "/dirs/d1",
		`{"constraints":{"files.description":{"default":"d1-desc-default"}}}`,
		201, `{
  "dirid": "d1",
  "self": "http://localhost:8181/dirs/d1",
  "xid": "/dirs/d1",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:19.169651848Z",
  "modifiedat": "2026-07-27T00:26:19.169651848Z",
  "constraints": {
    "files.description": {
      "default": "d1-desc-default"
    }
  },

  "filesurl": "http://localhost:8181/dirs/d1/files",
  "filescount": 0
}
`)

	// d2 (xref source's group) has its OWN, DIFFERENT instance-level
	// default for "description" - if constraint defaults were (wrongly)
	// applied to the xref itself rather than mirroring the target, this
	// would leak through as "d2-desc-default" instead of the target's
	// real "d1-desc-default".
	XHTTP(t, reg, "PATCH", "/dirs/d2",
		`{"constraints":{"files.description":{"default":"d2-desc-default"}}}`,
		201, `{
  "dirid": "d2",
  "self": "http://localhost:8181/dirs/d2",
  "xid": "/dirs/d2",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:21.098406778Z",
  "modifiedat": "2026-07-27T00:26:21.098406778Z",
  "constraints": {
    "files.description": {
      "default": "d2-desc-default"
    }
  },

  "filesurl": "http://localhost:8181/dirs/d2/files",
  "filescount": 0
}
`)

	// Target t1, in d1: gets the model-level "name" default AND its own
	// group's "description" default.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/t1$details", `{}`, 201,
		`{
  "fileid": "t1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/t1",
  "xid": "/dirs/d1/files/t1",
  "epoch": 1,
  "name": "model-name-default",
  "isdefault": true,
  "description": "d1-desc-default",
  "createdat": "2026-07-27T00:26:23.122033494Z",
  "modifiedat": "2026-07-27T00:26:23.122033494Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d1/files/t1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/t1/versions",
  "versionscount": 1
}
`)

	// Xref fx, in d2, pointing at t1.
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/t1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d2/files/fx/meta",
  "xid": "/dirs/d2/files/fx/meta",
  "xref": "/dirs/d1/files/t1",
  "epoch": 1,
  "createdat": "2026-07-27T00:26:25.057135187Z",
  "modifiedat": "2026-07-27T00:26:25.057135187Z",
  "readonly": false,

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d2/files/fx/versions/1",
  "defaultversionsticky": false
}
`)

	// The mirror must show t1's REAL values from ITS OWN constraint
	// context (d1's), not d2's (the xref's own group) constraint
	// defaults, and not the model-level default recomputed independently.
	XHTTP(t, reg, "GET", "/dirs/d2/files/fx$details", ``, 200,
		`{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d2/files/fx",
  "xid": "/dirs/d2/files/fx",
  "epoch": 1,
  "name": "model-name-default",
  "isdefault": true,
  "description": "d1-desc-default",
  "createdat": "2026-07-27T00:26:27.123365035Z",
  "modifiedat": "2026-07-27T00:26:27.123365035Z",
  "ancestorid": "1",

  "metaurl": "http://localhost:8181/dirs/d2/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d2/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefFormatValidationFailureCascade verifies that a FAILED format
// validation state (formatvalidated=false/compatibilityvalidated=false,
// via an "unknown format" document - the only kind of format failure
// that doesn't outright reject the save, see EnsureCompat()) on the xref
// TARGET is correctly mirrored into the xref SOURCE too, not just the
// "everything is valid" happy path already covered by
// TestXrefFormatCascadeOnDirectSave. This is created BEFORE the xref
// exists, so the xref creation itself must pick up the already-failed
// state.
func TestXrefFormatValidationFailureCascade(t *testing.T) {
	reg := NewRegistry("TestXrefFormatValidationFailureCascade")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
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
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Create the target with an "unknown" format - EnsureCompat() doesn't
	// reject this outright (only "strict" mode would), it just flags
	// formatvalidated=false/compatibilityvalidated=false with a
	// "Unknown format" reason.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "unknown",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:18:59.386435643Z",
  "modifiedat": "2026-07-27T01:18:59.386435643Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// Create the xref source pointing at the already-failed target.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:19:01.83746876Z",
  "modifiedat": "2026-07-27T01:19:01.83746876Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// The mirror must show the SAME failed state, not silently omit it
	// or default it back to "true"/absent.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:04.344048762Z",
  "modifiedat": "2026-07-27T01:19:04.344048762Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Now fix the target's format - the failure state must clear on
	// both the target AND (more importantly) the mirror.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "1"
}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:06.908004939Z",
  "modifiedat": "2026-07-27T01:19:07.055228045Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:09.446274109Z",
  "modifiedat": "2026-07-27T01:19:09.587498469Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefFormatValidationFailureCascadeAfterUpdate is the "after the
// fact" variant of TestXrefFormatValidationFailureCascade: the xref is
// created while the target is VALID (so the mirror initially shows
// formatvalidated=true), and only THEN does the target transition into
// the failed ("unknown format") state via a normal direct update - the
// mirror must pick up that failure transition too, not just the initial
// state at xref-creation time.
func TestXrefFormatValidationFailureCascadeAfterUpdate(t *testing.T) {
	reg := NewRegistry("TestXrefFormatValidationFailureCascadeAfterUpdate")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
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
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "1"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:54.525985019Z",
  "modifiedat": "2026-07-27T01:19:54.525985019Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:19:57.177137209Z",
  "modifiedat": "2026-07-27T01:19:57.177137209Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// Confirm the mirror starts out fully valid.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:19:59.93952847Z",
  "modifiedat": "2026-07-27T01:19:59.93952847Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Flip the TARGET (already xref'd) to an unknown format via a
	// normal direct update - no model change involved.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "unknown",
  "file": "1"
}`, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:02.787321133Z",
  "modifiedat": "2026-07-27T01:20:02.984856352Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// The mirror must now reflect the failure too.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 2,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:05.655873226Z",
  "modifiedat": "2026-07-27T01:20:05.845462478Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "unknown",
  "formatvalidated": false,
  "formatvalidatedreason": "Unknown format",
  "compatibilityvalidated": false,
  "compatibilityvalidatedreason": "Unknown format",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}

// TestXrefTargetUpdateRejectionDoesNotCorruptMirror verifies that when a
// direct update to the xref TARGET is REJECTED outright (a real
// "compatibility_violation" 400, not just a formatvalidated/
// compatibilityvalidated=false flag - see FormatNumbers.IsCompatible())
// the target's data is left untouched AND the xref SOURCE's mirror
// remains correctly in-sync with the target's last-good state - i.e. a
// failed write to the target must not leave the mirror stale, corrupted,
// or partially updated.
func TestXrefTargetUpdateRejectionDoesNotCorruptMirror(t *testing.T) {
	reg := NewRegistry("TestXrefTargetUpdateRejectionDoesNotCorruptMirror")
	defer PassDeleteReg(t, reg)

	model := registry.Model{}
	gm, _ := model.AddGroupModel("dirs", "dir")
	rm, _ := gm.AddResourceModel("files", "file", 0, true, true)
	rm.SetValidateFormat(true)
	rm.SetValidateCompatibility(true)

	XHTTP(t, reg, "PUT", "/modelsource", model.MustUserMarshal("", "  "),
		200, `{
  "groups": {
    "dirs": {
      "plural": "dirs",
      "singular": "dir",
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
          "strictvalidation": false
        }
      }
    }
  }
}
`)

	// Target starts with a single Version, sum=5.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1$details", `{
  "meta": { "compatibility": "backward" },
  "format": "numbers",
  "file": "5"
}`, 201, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:20.723551092Z",
  "modifiedat": "2026-07-27T01:20:20.723551092Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "2026-07-27T01:20:22.72379931Z",
  "modifiedat": "2026-07-27T01:20:22.72379931Z",
  "readonly": false,
  "compatibility": "backward",

  "defaultversionid": "1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/1$details",
  "defaultversionsticky": false
}
`)

	// Confirm the mirror's initial good state.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:24.854766301Z",
  "modifiedat": "2026-07-27T01:20:24.854766301Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)

	// Attempt to add a new default Version with sum=2 (< 5) - "backward"
	// compatibility requires the new Version's sum to be >= the old
	// one, so this must be REJECTED outright (not just flagged false).
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
  "v2": {
    "format": "numbers",
    "file": "2"
  }
}`, 400, `{
  "type": "https://github.com/xregistry/spec/blob/main/core/spec.md#compatibility_violation",
  "title": "The request would cause one or more Versions of \"/dirs/d1/files/f1\" to violate its compatibility rule (backward).",
  "detail": "Version \"/dirs/d1/files/f1/versions/v2\" (sum: 2) isn't \"backward\" compatible with \"/dirs/d1/files/f1/versions/1\" (sum: 5).",
  "subject": "/dirs/d1/files/f1",
  "args": {
    "compat": "backward"
  },
  "source": ":registry:format_numbers:109"
}
`)

	// The target itself must be completely unaffected by the rejected
	// request - still just the original Version.
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1$details", ``, 200, `{
  "fileid": "f1",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/f1$details",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:29.063641818Z",
  "modifiedat": "2026-07-27T01:20:29.063641818Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 1
}
`)

	// And the mirror must ALSO still be showing that same untouched,
	// last-good state - not corrupted, not partially applied, and not
	// pointing at a Version that was never actually created.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx$details", ``, 200, `{
  "fileid": "fx",
  "versionid": "1",
  "self": "http://localhost:8181/dirs/d1/files/fx$details",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2026-07-27T01:20:31.1941946Z",
  "modifiedat": "2026-07-27T01:20:31.1941946Z",
  "ancestorid": "1",
  "contenttype": "application/json",
  "format": "numbers",
  "formatvalidated": true,
  "compatibilityvalidated": true,

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 1
}
`)
}
