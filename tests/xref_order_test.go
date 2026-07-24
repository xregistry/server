package tests

// These tests focus on a gap not covered by xref_test.go/ancestor_test.go/
// constraints_test.go: what happens when the xref TARGET resource is
// mutated (batch version create, default-version change) AFTER one or
// more xref SOURCES already point at it. This exercises the interaction
// between the version/default-version cascade and the xref fan-out logic
// (see plan.md "Backend / SQL re-architecture" item (b)) - specifically
// whether the fan-out correctly reflects the target's final state,
// including when multiple xref sources point at the same target, and
// when the target's default-version cascade and its xref-fan-out happen
// in the same request/transaction.
//
// IMPORTANT: These tests intentionally only use the public HTTP API
// (XHTTP) and don't reach into FullTree*-specific internals, so they can
// be ported to the pre-FullTree codebase (if needed) to distinguish
// pre-existing bugs from new regressions introduced by any future
// cascade-deferral work.

import (
	"testing"
)

// Adding multiple versions (in one batch request) to an xref TARGET,
// after one or more xref SOURCES already exist, must be reflected
// correctly in all of the sources' mirrored data (versions, versioncount,
// default version).
func TestXrefOrderMultiVersionAfterXref(t *testing.T) {
	reg := NewRegistry("TestXrefOrderMultiVersionAfterXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Target starts with just one version
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	// Two different xref sources pointing at the same target
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Now batch-add two more versions to the target in a single request
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v2": {},
      "v3": {}
    }`, 200, `*`)

	// Target itself should show all 3 versions, v3 as default (newest)
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta", "", 200, `{
  "fileid": "f1",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/f1",
  "xid": "/dirs/d1/files/f1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/f1/meta",
  "meta": {
    "fileid": "f1",
    "self": "http://localhost:8181/dirs/d1/files/f1/meta",
    "xid": "/dirs/d1/files/f1/meta",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v3",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 3
}
`)

	// Both xref sources must mirror the target's final state - 3 versions,
	// v3 as the default - not some stale intermediate state from the
	// batch's individual per-version saves.
	for _, rid := range []string{"fx", "fy"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"?inline=meta", "", 200, `{
  "fileid": "`+rid+`",
  "versionid": "v3",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`",
  "xid": "/dirs/d1/files/`+rid+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v2",

  "metaurl": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "meta": {
    "fileid": "`+rid+`",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
    "xid": "/dirs/d1/files/`+rid+`/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v3",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v3",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions",
  "versionscount": 3
}
`)

		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"/versions", "", 200, `{
  "v1": {
    "fileid": "`+rid+`",
    "versionid": "v1",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v1",
    "xid": "/dirs/d1/files/`+rid+`/versions/v1",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "ancestorid": "v1"
  },
  "v2": {
    "fileid": "`+rid+`",
    "versionid": "v2",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "xid": "/dirs/d1/files/`+rid+`/versions/v2",
    "epoch": 1,
    "isdefault": false,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "v1"
  },
  "v3": {
    "fileid": "`+rid+`",
    "versionid": "v3",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v3",
    "xid": "/dirs/d1/files/`+rid+`/versions/v3",
    "epoch": 1,
    "isdefault": true,
    "createdat": "YYYY-MM-DDTHH:MM:02Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "ancestorid": "v2"
  }
}
`)
	}
}

// Several oddities that can only show up when MULTIPLE resources xref
// the same target: batch-creating several sources pointing at the same
// target in a single request (order-independence within one Tx),
// deleting one source must not affect the target or the remaining
// sources, and re-pointing one source's xref away from the target must
// stop it from being fanned-out to while the other sources keep mirroring
// the original target correctly.
func TestXrefOrderMultipleSourcesSameTarget(t *testing.T) {
	reg := NewRegistry("TestXrefOrderMultipleSourcesSameTarget")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Target with 2 versions, and a second (unrelated) target to re-point
	// to later.
	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}
    }`, 200, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/w1", `{}`, 201, `*`)

	// Batch-create 3 xref sources in ONE request, all pointing at f1.
	XHTTP(t, reg, "POST", "/dirs/d1/files/?inline=meta", `{
      "fa": {"meta": {"xref": "/dirs/d1/files/f1"}},
      "fb": {"meta": {"xref": "/dirs/d1/files/f1"}},
      "fc": {"meta": {"xref": "/dirs/d1/files/f1"}}
    }`, 200, `*`)

	// All three must mirror f1's current default version (v2) - order of
	// creation within the batch must not matter.
	for _, rid := range []string{"fa", "fb", "fc"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"?inline=meta", "", 200, `{
  "fileid": "`+rid+`",
  "versionid": "v2",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`",
  "xid": "/dirs/d1/files/`+rid+`",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "meta": {
    "fileid": "`+rid+`",
    "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
    "xid": "/dirs/d1/files/`+rid+`/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 1,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
    "readonly": false,

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions",
  "versionscount": 2
}
`)
	}

	// Deleting one source (fb) must not affect the target or the other
	// two sources.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fb", ``, 204, ``)

	XHTTP(t, reg, "GET", "/dirs/d1/files/f1?inline=meta", "", 200, `{
  "fileid": "f1",
  "versionid": "v2",
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

    "defaultversionid": "v2",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/f1/versions/v2",
    "defaultversionsticky": false
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/f1/versions",
  "versionscount": 2
}
`)

	for _, rid := range []string{"fa", "fc"} {
		XHTTP(t, reg, "GET", "/dirs/d1/files/"+rid+"/meta", "", 200, `{
  "fileid": "`+rid+`",
  "self": "http://localhost:8181/dirs/d1/files/`+rid+`/meta",
  "xid": "/dirs/d1/files/`+rid+`/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v2",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/`+rid+`/versions/v2",
  "defaultversionsticky": false
}
`)
	}

	// Re-point fa's xref away from f1 to f2. fa must now mirror f2, while
	// fc keeps mirroring f1 correctly (no cross-contamination).
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fa/meta",
		`{"xref":"/dirs/d1/files/f2"}`, 200, `*`)

	// Now add a new version to f1 - fa (re-pointed to f2) must NOT pick
	// this up, but fc (still pointing at f1) must.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v3", `{}`, 201, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fa/meta", "", 200, `{
  "fileid": "fa",
  "self": "http://localhost:8181/dirs/d1/files/fa/meta",
  "xid": "/dirs/d1/files/fa/meta",
  "xref": "/dirs/d1/files/f2",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "w1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fa/versions/w1",
  "defaultversionsticky": false
}
`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fc/meta", "", 200, `{
  "fileid": "fc",
  "self": "http://localhost:8181/dirs/d1/files/fc/meta",
  "xid": "/dirs/d1/files/fc/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 2,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fc/versions/v3",
  "defaultversionsticky": false
}
`)
}

// Explicitly changing the xref TARGET's default version (via
// setdefaultversionid, making it sticky) after xref SOURCES already
// exist must fan-out correctly to all sources.
func TestXrefOrderTargetDefaultChangeAfterXref(t *testing.T) {
	reg := NewRegistry("TestXrefOrderTargetDefaultChangeAfterXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "POST", "/dirs/d1/files/f1/versions", `{
      "v1": {}, "v2": {}, "v3": {}
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Sanity: fx currently mirrors v3 (newest) as default, non-sticky
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx/meta", "", 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,

  "defaultversionid": "v3",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v3",
  "defaultversionsticky": false
}
`)

	// Now stick the TARGET's default version to v1 (older, non-newest)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1?setdefaultversionid=v1", `{}`,
		200, `*`)

	// Target itself should now be sticky on v1
	XHTTP(t, reg, "GET", "/dirs/d1/files/f1/meta", "", 200, `{
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

	// The xref SOURCE must mirror the target's new sticky default (v1),
	// not the old newest-wins default (v3).
	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 2,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 3
}
`)

	// Adding a brand new version (v4) to the target should NOT move the
	// sticky default off of v1, and the source must keep mirroring v1 too.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v4", `{}`, 201, `*`)

	XHTTP(t, reg, "GET", "/dirs/d1/files/fx?inline=meta", "", 200, `{
  "fileid": "fx",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/fx",
  "xid": "/dirs/d1/files/fx",
  "epoch": 1,
  "isdefault": true,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "ancestorid": "v1",

  "metaurl": "http://localhost:8181/dirs/d1/files/fx/meta",
  "meta": {
    "fileid": "fx",
    "self": "http://localhost:8181/dirs/d1/files/fx/meta",
    "xid": "/dirs/d1/files/fx/meta",
    "xref": "/dirs/d1/files/f1",
    "epoch": 3,
    "createdat": "YYYY-MM-DDTHH:MM:01Z",
    "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
    "readonly": false,

    "defaultversionid": "v1",
    "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
    "defaultversionsticky": true
  },
  "versionsurl": "http://localhost:8181/dirs/d1/files/fx/versions",
  "versionscount": 4
}
`)
}

// Combines the "defaultversionsticky=true-by-default" edge case (see
// TestHTTPSticky in tests/http1_test.go ~line 10142, where setting
// defaultversionid on a Meta whose model default for
// defaultversionsticky is true causes it to stick even though a plain
// PUT would normally just be ignored) with xref: reverting an xref (so
// the resource goes back to owning its own versions) and then setting
// defaultversionid must still honor a true default-sticky model setting.
func TestXrefOrderRevertWithStickyDefault(t *testing.T) {
	reg := NewRegistry("TestXrefOrderRevertWithStickyDefault")
	defer PassDeleteReg(t, reg)

	XHTTP(t, reg, "PUT", "/modelsource", `{
      "groups": {
        "dirs": {
          "singular": "dir",
          "resources": {
            "files": {
              "singular": "file",
              "hasdocument": false,
              "metaattributes": {
                "defaultversionsticky": {
                  "type": "boolean",
                  "required": true,
                  "enum": [ true ],
                  "default": true
                }
              }
            }
          }
        }
      }
    }`, 200, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)

	// Revert the xref and add a couple of versions of its own
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx?inline=meta", `{
      "meta": {"xref": null},
      "versions": {"a1": {}, "a2": {}}
    }`, 200, `*`)

	// Normally setting defaultversionid=a1 (not the newest) would be
	// ignored, but since defaultversionsticky defaults to true for this
	// resource type, it should stick.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"defaultversionid":"a1"}`, 200, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "epoch": 3,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:02Z",
  "readonly": false,

  "defaultversionid": "a1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/a1",
  "defaultversionsticky": true
}
`)
}
