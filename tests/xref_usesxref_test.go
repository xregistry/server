package tests

// These tests specifically exercise the Registries.UsesXref internal
// fast-path flag itself (registry/init.sql, Resource.runCascade() in
// resource.go) - not general xref functional behavior (already covered
// by xref_test.go/xref_order_test.go). Since UsesXref is a raw,
// internal-only DB column (never exposed via the HTTP API), these
// tests reach into the registry package directly to check its value,
// unlike the HTTP-only convention used elsewhere in this package.

import (
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func getUsesXref(t *testing.T, reg *registry.Registry) bool {
	t.Helper()
	tx := reg.GetTx()
	results := registry.Query(tx, `SELECT UsesXref FROM Registries WHERE SID=?`,
		reg.DbSID)
	defer results.Close()
	row := results.NextRow()
	if row == nil {
		t.Fatalf("no Registries row found for %q", reg.UID)
	}
	return NotNilBoolDef(row[0], false)
}

// A fresh Registry that has never used xref must have UsesXref=false.
// Creating the first xref must flip it to true (and stay true across
// unrelated saves). Deleting that xref (going back to zero xrefs in
// the Registry) must eventually flip it back to false. Creating a new
// xref again afterward must flip it back to true - the flag's
// lifecycle must be fully reversible, not "sticky" once cleared.
func TestUsesXrefLifecycle(t *testing.T) {
	reg := NewRegistry("TestUsesXrefLifecycle")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	// Brand-new Registry, never touched xref.
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false on a fresh Registry")
	}

	// An unrelated save (no xref involved) must not flip it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after an unrelated save")
	}

	// Create the first (and only) xref in this Registry.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true right after creating an xref")
	}

	// Another unrelated save must not clear it.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/f2/versions/v1", `{}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref to stay true after an unrelated save")
	}

	// Clear the (only) xref in the Registry - flag should go back to
	// false since a rescan finds nothing left.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta", `{}`, 200, `*`)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after clearing the last xref")
	}

	// Create a new xref again - must correctly flip back to true, not
	// stay stuck false.
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fy/meta",
		`{"xref":"/dirs/d1/files/f2"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after re-creating an xref")
	}

	// Deleting the xref SOURCE Resource entirely (rather than just
	// clearing its xref attribute) must also correctly rescan/clear.
	XHTTP(t, reg, "DELETE", "/dirs/d1/files/fy", ``, 204, ``)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after deleting the xref source")
	}
}

// Deleting an entire Group containing an xref source (bulk-delete path
// that bypasses Go-level Resource.Delete()) must still correctly
// rescan and clear UsesXref - this is the whole reason the clearing
// side of the flag is handled by DB triggers rather than only in Go.
func TestUsesXrefClearedByGroupBulkDelete(t *testing.T) {
	reg := NewRegistry("TestUsesXrefClearedByGroupBulkDelete")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d2/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after creating an xref")
	}

	// Bulk-delete the SOURCE's entire group ("d2").
	XHTTP(t, reg, "DELETE", "/dirs/d2", ``, 204, ``)
	if getUsesXref(t, reg) != false {
		t.Fatalf("expected UsesXref=false after bulk-deleting the only xref source's group")
	}
}

// Deleting the entire Registry while it still has an active xref must
// not error out (regression test for MySQL error 1442 - "Can't update
// table 'Registries' in stored function/trigger because it is already
// used by statement which invoked this stored function/trigger" - hit
// during development because ResourcesTrigger/FullTreeXref's own
// UsesXref-clearing UPDATE against Registries collides with the
// outermost DELETE FROM Registries statement that's cascading through
// them). PassDeleteReg() (deferred above) already deletes the
// Registry, so this test's real assertion is simply that it doesn't
// panic.
func TestUsesXrefRegistryDeleteWithActiveXref(t *testing.T) {
	reg := NewRegistry("TestUsesXrefRegistryDeleteWithActiveXref")
	defer PassDeleteReg(t, reg)

	gm, _ := reg.Model.AddGroupModel("dirs", "dir")
	gm.AddResourceModel("files", "file", 0, true, false)

	XHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1", `{}`, 201, `*`)
	XHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref":"/dirs/d1/files/f1"}`, 201, `*`)
	if getUsesXref(t, reg) != true {
		t.Fatalf("expected UsesXref=true after creating an xref")
	}
}
