package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type SQLBackend struct {
	XRSConfig *Config
}

var _ Backend = &SQLBackend{}

func NewSQLBackend(c *Config) Backend {
	return &SQLBackend{
		XRSConfig: c,
	}
}

func (sqlBE *SQLBackend) NewTx(tx *Tx) *XRError {
	return tx.NewTx()
}

func (sqlBE *SQLBackend) FindRegistry(tx *Tx, config *Config, id string,
	accessMode int) (*Registry, *XRError) {

	return FindRegistry(tx, config, id, accessMode)
}

func (sqlBE *SQLBackend) RegisterEntity(e *Entity) *XRError {
	switch e.Type {
	case ENTITY_REGISTRY:
		// Add Registry and Model to the DB's tables

		e.EntityInsert() // add to Entities table

		DoOne(e.Tx, `INSERT INTO Registries(SID, UID) VALUES(?,?)`,
			e.DbSID, e.UID)

		DoOne(e.Tx, `INSERT INTO Models(RegistrySID) VALUES(?)`, e.DbSID)

	case ENTITY_GROUP:
		panic("Not implemented yet: Group")

	case ENTITY_RESOURCE:
		panic("Not implemented yet: Resource")

	case ENTITY_META:
		panic("Not implemented yet: Meta")

	case ENTITY_VERSION:
		panic("Not implemented yet: Version")

	default:
		panic(fmt.Sprintf("Uknown type: %d", e.Type))
	}

	return nil
}

func (sqlBE *SQLBackend) RefreshEntity(e *Entity, accessMode int) *XRError {
	mode := ""
	if accessMode == FOR_WRITE {
		mode = " FOR UPDATE"

		// Need to lock the entity so we grab the latest stuff
		results := Query(e.Tx,
			`SELECT UID FROM Entities WHERE eSID=? FOR UPDATE`, e.DbSID)
		PanicIf(len(results.AllRows) != 1, "Rows: %d", len(results.AllRows))
		results.Close()
	}

	log.FuncPrintf("tx: %s Refreshing %q, mode: %v", e.Tx.uuid, e.XID, mode)

	results := Query(e.Tx, `
        SELECT PropName, PropValue, PropType, IsSystemProp
        FROM Props
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false
              AND IsCalcDynamic=false`+mode, e.DbSID)
	defer results.Close()

	// Erase all old props first
	e.Object = map[string]any{}
	e.System = map[string]any{}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])
		isSystemProp := NotNilBoolDef(row[3], false)

		var xErr *XRError
		if isSystemProp {
			xErr = e.SetSystemFromDBName(name, &val, propType)
		} else {
			xErr = e.SetFromDBName(name, &val, propType)
		}
		if xErr != nil {
			return xErr
		}
	}

	return nil
}

func (sqlBE *SQLBackend) GetResourceContents(e *Entity) []byte {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		contentID := e.Get("#contentid")

		results := Query(e.Tx, `
            SELECT Content FROM ResourceContents WHERE VersionSID=? `,
			contentID)
		defer results.Close()

		row := results.NextRow()
		if row == nil {
			// No data so just return
			return nil
		}

		if results.NextRow() != nil {
			panic("too many results")
		}

		return (*(row[0])).([]byte)
	}
	panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
}
