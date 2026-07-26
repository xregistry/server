package tests

// These tests target the deferred, once-per-Tx validation design added
// for plan.md "Backend / SQL re-architecture" item (b) (Double
// Version.FullSave() on resource creation): registry/db.go's
// Tx.ResourcesToValidate/AddResourceToValidate, drained by
// Registry.Validate() (registry/registry.go), wired into
// Resource.ValidateResource() (registry/resource.go) and
// registry/fulltree.go's fullSaveDefaultVerCascade().
//
// Specifically, they cover Resource.SetDefault() (registry/resource.go)
// - used by Version.DeleteSetNextVersion() when the current default
// Version is deleted - as a THIRD path (besides Resource.EnsureLatest())
// that can set "defaultversionid" and therefore mark a Resource for
// (re-)validation. Live tracing (added temporarily during development)
// confirmed that in all of these scenarios the cascade runs exactly once
// per request and never needs to fall back to the defensive "ver == nil"
// re-entrant EnsureLatest() call inside fullSaveDefaultVerCascade(),
// because SetDefault()/EnsureLatest() always resolve "defaultversionid"
// to a real, existing Version before Resource.ValidateResource() reaches
// its final cascade step. These tests lock in that black-box behavior
// (correct final default-version state) so a future change to the
// validation-deferral or delete-time default-fixup logic can't silently
// regress it.
//
// IMPORTANT: These intentionally only use the public HTTP API (XHTTP),
// same convention as xref_order_test.go, so they're portable and don't
// depend on FullTree*-specific internals.

import (
	"testing"
)

// Deleting the current (non-sticky) default Version must recompute the
// default to the next-newest remaining Version.
func TestCascadeDeferDeleteNonStickyDefault(t *testing.T) {
	reg := NewRegistry("TestCascadeDeferDeleteNonStickyDefault")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	// v3 is the newest so it's the (non-sticky) default
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "defaultversionsticky": false
}
`)

	// Delete the current default with no ?setdefaultversionid - must
	// fall back to the next-newest remaining Version (v2).
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)
}

// Deleting the current STICKY default Version, with no explicit
// ?setdefaultversionid, must un-stick and recompute the default to the
// newest remaining Version (Resource.SetDefault(nil) path).
func TestCascadeDeferDeleteStickyDefaultUnsticks(t *testing.T) {
	reg := NewRegistry("TestCascadeDeferDeleteStickyDefaultUnsticks")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	// Explicitly stick the default to the oldest Version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"defaultversionid":"v1","defaultversionsticky":true}`, 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v1",
  "defaultversionsticky": true
}
`)

	// Delete the sticky default with no ?setdefaultversionid - must
	// un-stick and pick the newest remaining Version (v3).
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v1", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
  "defaultversionsticky": false
}
`)
}

// Deleting the current sticky default Version WITH an explicit
// ?setdefaultversionid must keep the result sticky and pointed at the
// requested Version (Resource.SetDefault(nextVersion) path).
func TestCascadeDeferDeleteStickyDefaultExplicitNext(t *testing.T) {
	reg := NewRegistry("TestCascadeDeferDeleteStickyDefaultExplicitNext")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/meta",
		`{"defaultversionid":"v1","defaultversionsticky":true}`, 200, `*`)

	// Delete the sticky default, explicitly choosing v2 as the new
	// (still sticky) default.
	XHTTP(t, reg, "DELETE",
		"/dirs/d1/files/f1/versions/v1?setdefaultversionid=v2", "",
		204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": true
}
`)
}

// Deleting a xref TARGET's current default Version must fan out
// correctly to any xref SOURCE(s) pointing at it - this ties
// Resource.SetDefault()'s cascade mark together with
// fullSaveXrefFanOutForTarget in the same deferred drain.
func TestCascadeDeferDeleteDefaultWithXrefFanOut(t *testing.T) {
	reg := NewRegistry("TestCascadeDeferDeleteDefaultWithXrefFanOut")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Delete the target's current (non-sticky) default - v3 - with no
	// explicit next. Target's new default becomes v2.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/f1/versions/v3", "", 204, "")

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
  "fileid": "f1",
  "self": "http://localhost:8181/dirs/d1/files/f1/meta",
  "xid": "/dirs/d1/files/f1/meta",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
  "defaultversionsticky": false
}
`)

	// The xref source must mirror the SAME final state - v2 as default -
	// not a stale v3 reference nor an intermediate/partial cascade
	// result.
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v2",
  "defaultversionsticky": false
}
`)
}
