package tests

import (
	"fmt"
	"testing"

	"github.com/xregistry/server/registry"
)

func TestDBRows(t *testing.T) {
	// Make sure we don't create extra extra stuff in the DB.
	reg := NewRegistry("TestDBRows")
	defer PassDeleteReg(t, reg)

	_, _, err := reg.Model.CreateModels("dirs", "dir", "files", "file")
	xNoErr(t, err)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/f1/versions/v1$details", `{}`, 201, `{
  "fileid": "f1",
  "versionid": "v1",
  "self": "http://localhost:8181/dirs/d1/files/f1/versions/v1$details",
  "xid": "/dirs/d1/files/f1/versions/v1",
  "epoch": 1,
  "isdefault": true,
  "createdat": "2025-01-01T12:00:01Z",
  "modifiedat": "2025-01-01T12:00:01Z"
}
`)

	xHTTP(t, reg, "PUT", "/dirs/d1/files/fx/meta",
		`{"xref": "/dirs/d1/files/f1"}`, 201, `{
  "fileid": "fx",
  "self": "http://localhost:8181/dirs/d1/files/fx/meta",
  "xid": "/dirs/d1/files/fx/meta",
  "xref": "/dirs/d1/files/f1",
  "epoch": 1,
  "createdat": "YYYY-MM-DDTHH:MM:01Z",
  "modifiedat": "YYYY-MM-DDTHH:MM:01Z",
  "readonly": false,
  "compatibility": "none",

  "defaultversionid": "v1",
  "defaultversionurl": "http://localhost:8181/dirs/d1/files/fx/versions/v1",
  "defaultversionsticky": false
}
`)

	strFn := func(v any) string {
		vp := v.(*any)
		return registry.NotNilString(vp)
	}

	rows, err := reg.Query("SELECT e.Path,p.PropName,p.PropValue "+
		"FROM Props AS p "+
		"JOIN Entities AS e ON (p.EntitySID=e.eSID) WHERE p.RegistrySID=? "+
		"ORDER BY Path, PropName ",
		reg.DbSID)
	xNoErr(t, err)

	result := ""
	for _, row := range rows {
		result += fmt.Sprintf("%s: %s -> %s\n",
			strFn(row[0]), strFn(row[1]), strFn(row[2]))
	}
	result = MaskTimestamps(result)

	// Some thing to note about this output, for those new to this stuff
	// - each name ends with , (DB_IN) for each parsing/searching
	// - d1's modifiedat timestamp was changed due to fx being created
	// - props that start with "#" are private and for system use/tracking
	// - fx's #createdat is when it was created, if needed when xref is del'd
	// - fx's #epoch is saved so we can calc the new epoch if xref is del'd
	// - #nextversionid is what vID we should use on next system set vID
	// - All entities need at least one Prop, so fx needs 'fileid'
	xCheckEqual(t, "", result,
		`: createdat, -> YYYY-MM-DDTHH:MM:01Z
: epoch, -> 2
: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
: registryid, -> TestDBRows
: specversion, -> 1.0-rc1
dirs/d1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1: dirid, -> d1
dirs/d1: epoch, -> 2
dirs/d1: modifiedat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/f1: fileid, -> f1
dirs/d1/files/f1/meta: #nextversionid, -> 1
dirs/d1/files/f1/meta: compatibility, -> none
dirs/d1/files/f1/meta: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: defaultversionid, -> v1
dirs/d1/files/f1/meta: defaultversionsticky, -> false
dirs/d1/files/f1/meta: epoch, -> 1
dirs/d1/files/f1/meta: fileid, -> f1
dirs/d1/files/f1/meta: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/meta: readonly, -> false
dirs/d1/files/f1/versions/v1: ancestor, -> v1
dirs/d1/files/f1/versions/v1: createdat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: epoch, -> 1
dirs/d1/files/f1/versions/v1: modifiedat, -> YYYY-MM-DDTHH:MM:02Z
dirs/d1/files/f1/versions/v1: versionid, -> v1
dirs/d1/files/fx: fileid, -> fx
dirs/d1/files/fx/meta: #createdat, -> YYYY-MM-DDTHH:MM:03Z
dirs/d1/files/fx/meta: #epoch, -> 1
dirs/d1/files/fx/meta: #nextversionid, -> 2
dirs/d1/files/fx/meta: fileid, -> fx
dirs/d1/files/fx/meta: xref, -> /dirs/d1/files/f1
`)
}
