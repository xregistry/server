package registry

import (
	"os"
	"testing"

	. "github.com/xregistry/server/common"
)

// Temporary ad hoc test to run DiffFullTree across all registries in a
// given DB. Not part of the permanent suite - remove after validation.
// Run with: DIFFDB=testdb_fulltree go test ./registry/ -run TestZZDiffFullTree -v
func TestZZDiffFullTree(t *testing.T) {
	dbName := os.Getenv("DIFFDB")
	if dbName == "" {
		t.Skip("set DIFFDB env var to run this")
	}

	if err := OpenDB(dbName); err != nil {
		t.Fatalf("OpenDB: %s", err)
	}

	tx, err := NewTx()
	if err != nil {
		t.Fatalf("NewTx: %s", err)
	}
	defer tx.Rollback()

	results := Query(tx, `SELECT SID, UID FROM Registries`)
	defer results.Close()

	type reg struct{ sid, uid string }
	var regs []reg
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		regs = append(regs, reg{NotNilString(row[0]), NotNilString(row[1])})
	}

	for _, r := range regs {
		diff := DiffFullTree(tx, r.sid)
		if diff == "" {
			t.Logf("OK   %s (%s)", r.uid, r.sid)
		} else {
			t.Errorf("DIFF %s (%s):\n%s", r.uid, r.sid, diff)
		}
	}
}
