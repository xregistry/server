package registry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

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

// DB Actions

func (sqlBE *SQLBackend) DBExists(xrsConfig *Config, name string) bool {
	defer log.Trace(name)()

	db, err := sql.Open("mysql",
		xrsConfig.GetAsString("db.user")+":"+
			xrsConfig.GetAsString("db.password")+"@tcp("+
			xrsConfig.GetAsString("db.host")+":"+
			xrsConfig.GetAsString("db.port")+")/")
	PanicIf(err != nil, "Error opening DB: %s", err)
	defer db.Close()

	rows, err := db.Query(`
        SELECT SCHEMA_NAME
        FROM INFORMATION_SCHEMA.SCHEMATA
        WHERE SCHEMA_NAME=?`, name)
	PanicIf(err != nil, "Error querying DB: %s", err)
	defer rows.Close()

	found := rows.Next()
	log.FuncPrintf("found: %v", found)
	return found
}

func (sqlBE *SQLBackend) DBCreate(xrsConfig *Config, name string) *XRError {
	defer log.Trace(name)()

	db, err := sql.Open("mysql",
		xrsConfig.GetAsString("db.user")+":"+
			xrsConfig.GetAsString("db.password")+"@tcp("+
			xrsConfig.GetAsString("db.host")+":"+
			xrsConfig.GetAsString("db.port")+")/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	if _, err = db.Exec("CREATE DATABASE " + name); err != nil {
		panic(err)
	}

	if _, err = db.Exec("USE " + name); err != nil {
		panic(err)
	}

	log.FuncPrintf("Creating DB")

	for _, cmd := range strings.Split(initDB, ";") {
		cmd = strings.TrimSpace(cmd)
		cmd = replaceVariables(cmd)
		if cmd == "" {
			continue
		}

		log.FuncPrintf("CMD: %s", cmd)
		if _, err := db.Exec(cmd); err != nil {
			panic(fmt.Sprintf("Error on: %s\n%s", cmd, err))
		}
	}

	return nil
}

func replaceVariables(str string) string {
	if str == "" {
		return str
	}

	vars := [][2]string{
		{"$$", ";"}, // can't use ; in file
		{"$ENTITY_REGISTRY", StrTypes(ENTITY_REGISTRY)},
		{"$ENTITY_GROUP", StrTypes(ENTITY_GROUP)},
		{"$ENTITY_RESOURCE", StrTypes(ENTITY_RESOURCE)},
		{"$ENTITY_META", StrTypes(ENTITY_META)},
		{"$ENTITY_VERSION", StrTypes(ENTITY_VERSION)},
		{"$DB_IN", string(DB_IN)},
		{"$MAX_VARCHAR", fmt.Sprintf("%d", MAX_VARCHAR)},
		{"$MAX_PROPNAME", fmt.Sprintf("%d", MAX_PROPNAME)},
	}

	for _, vs := range vars {
		str = strings.Replace(str, vs[0], vs[1], -1)
	}
	return str
}

func (sqlBE *SQLBackend) DBDelete(xrsConfig *Config, name string) *XRError {
	defer log.Trace(name)()

	db, err := sql.Open("mysql",
		xrsConfig.GetAsString("db.user")+":"+
			xrsConfig.GetAsString("db.password")+"@tcp("+
			xrsConfig.GetAsString("db.host")+":"+
			xrsConfig.GetAsString("db.port")+")/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("DROP DATABASE IF EXISTS " + name)
	if err != nil {
		panic(err)
	}
	return nil
}

func (sqlBE *SQLBackend) DBList(xrsConfig *Config) ([]string, *XRError) {
	defer log.Trace()()

	db, err := sql.Open("mysql",
		xrsConfig.GetAsString("db.user")+":"+
			xrsConfig.GetAsString("db.password")+"@tcp("+
			xrsConfig.GetAsString("db.host")+":"+
			xrsConfig.GetAsString("db.port")+")/")
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}
	defer db.Close()

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
	}
	defer rows.Close()

	sysNames := []string{"information_schema", "mysql",
		"performance_schema", "sys"}

	names := []string{}
	for rows.Next() {
		name := ""
		if err := rows.Scan(&name); err != nil {
			return nil, NewXRError("server_error", "/").SetDetail(err.Error() + ".")
		}
		if !ArrayContains(sysNames, name) {
			names = append(names, name)
		}
	}

	return names, nil
}

// TX Actions

// It's ok for this to be called multiple times for the same Tx just to
// make sure we have an active transaction - it's a no-op at that point
func (sqlBE *SQLBackend) NewTx(tx *Tx) *XRError {
	defer log.Trace("tx: %s tx.NewTx", tx.uuid)()

	if tx.tx != nil {
		return nil
	}

	DBName := tx.Config.GetAsString("db.name")
	if DBName == "" {
		return NewXRError("server_error", "/").SetDetail("No DBName set.")
	}

	DB, xErr := OpenDB(tx.Config, DBName)
	if xErr != nil {
		return xErr
	}

	// REPEATABLE READ (InnoDB's default) rather than READ COMMITTED: it
	// gives every plain (non-locking) SELECT in this Tx one consistent
	// snapshot/read-view established at the transaction's first read -
	// effectively "snapshot at tx start" for our purposes - while
	// SELECT ... FOR UPDATE reads (see entity.go's FOR_WRITE fetches)
	// still always see latest-committed data and take real row locks.
	// This combination is what makes the FOR_READ/FOR_WRITE distinction
	// in Entity.AccessMode actually mean something at the DB level.
	t, err := DB.BeginTx(context.Background(),
		&sql.TxOptions{sql.LevelRepeatableRead, false})
	if err != nil {
		CloseDB(tx.Config)
		return NewXRError("server_error", "/").SetDetail(err.Error() + ".")
		// panic("Error talking to the DB: %s", err)
	}

	tx.tx = t
	tx.CreateTime = time.Now().UTC().Format(time.RFC3339Nano)
	tx.Cache = map[string]*Entity{}
	// tx.stack = GetStack()

	log.FuncPrintf("tx: %s Begin transaction", tx.uuid)

	if log.HasVerbose("DEBUG_NewTX") {
		var connID int64
		t.QueryRow("SELECT CONNECTION_ID()").Scan(&connID)
		tx.connID = connID

		var autocommit int
		var isoLevel string
		err := t.QueryRow("SELECT @@autocommit, "+
			"@@session.transaction_isolation").Scan(&autocommit, &isoLevel)
		if err != nil {
			log.Printf("tx: %s connID=%d error checking "+
				"autocommit/isolation: %s", tx.uuid, connID, err)
		} else {
			log.Printf("tx: %s connID=%d autocommit=%d isolation=%s",
				tx.uuid, connID, autocommit, isoLevel)
		}

		log.Printf("tx: %s bound to MySQL CONNECTION_ID=%d", tx.uuid, connID)
	}

	return nil
}

func (sqlBE *SQLBackend) Commit(tx *Tx) *XRError {
	Must(tx.tx.(*sql.Tx).Commit())
	return nil
}

func (sqlBE *SQLBackend) Rollback(tx *Tx) *XRError {
	Must(tx.tx.(*sql.Tx).Rollback())
	return nil
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
