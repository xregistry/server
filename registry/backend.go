package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

func init() {
	xErr := RegisterBackend("mysql", NewSQLBackend)
	if xErr != nil {
		panic(xErr.String() + xErr.Detail)
	}
}

type SQLBackend struct {
	XRSConfig *Config
}

var _ Backend = &SQLBackend{}

func NewSQLBackend(c *Config) (Backend, *XRError) {
	return &SQLBackend{
		XRSConfig: c,
	}, nil
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
	err := tx.tx.(*sql.Tx).Commit()
	if !IsNil(err) {
		return NewXRError("server_error", "/").
			SetDetailf("Error: %s", err.Error())
	}

	return nil
}

func (sqlBE *SQLBackend) Rollback(tx *Tx) *XRError {
	err := tx.tx.(*sql.Tx).Rollback()
	if !IsNil(err) {
		return NewXRError("server_error", "/").
			SetDetailf("Error: %s", err.Error())
	}

	return nil
}

// clearValidationSystemProps bulk-clears the given system prop(s) (e.g.
// "formatvalidated"/"formatvalidatedreason" or "compatibilityvalidated"/
// "compatibilityvalidatedreason") from every Version of every Resource
// instance of the ResourceModel identified by modelSID, in one indexed
// sweep. Called by Model.Save() right after a validateformat/
// validatecompatibility true->false transition is detected, so
// EnsureCompat() (registry/resource.go) no longer needs to defensively
// re-clear these on every single save while validation stays off - this
// one-time, model-change-triggered sweep is the sole owner of clearing
// stale values.
func (sqlBE *SQLBackend) ClearResourceModelSystemProps(rm *ResourceModel, props []string) *XRError {
	if len(props) == 0 {
		return nil
	}

	reg := rm.GroupModel.Model.Registry

	placeholders := make([]string, len(props))
	args := make([]any, 0, len(props)+4)
	for i, name := range props {
		placeholders[i] = "?"
		args = append(args, name+string(DB_IN))
	}
	args = append(args, reg.DbSID, rm.SID, reg.DbSID, rm.SID)

	// Clear both the Version's own row AND the Resource-level
	// IsDefaultVerCopy mirror of it (same mirroring mechanism as
	// isdefault/createdat/modifiedat - the Resource-level copy is
	// what HTTP GET on the Resource actually serves).
	_, xErr := Do(reg.Tx, `
        DELETE FROM Props
        WHERE PropName IN (`+strings.Join(placeholders, ",")+`)
              AND (
                eSID IN (
                    SELECT SID FROM Versions WHERE ResourceSID IN (
                        SELECT SID FROM Resources
                        WHERE RegistrySID=? AND ModelSID=?))
                OR
                eSID IN (
                    SELECT SID FROM Resources
                    WHERE RegistrySID=? AND ModelSID=?)
              )`, args...)
	return xErr
}

func (sqlBE *SQLBackend) ListRegistries(tx *Tx) ([]string, *XRError) {
	// Order by UID, ASC - case insensitive
	results, xErr := Query(tx, `SELECT UID FROM Registries ORDER BY UID ASC`)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	res := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		res = append(res, NotNilString(row[0]))
	}

	return res, nil
}

func (sqlBE *SQLBackend) DeleteRegistry(r *Registry) *XRError {
	xErr := DoOne(r.Tx, `DELETE FROM Registries WHERE SID=?`, r.DbSID)

	return xErr
}

func (sqlBE *SQLBackend) MapRegistryUID2SID(tx *Tx, uid string) (string, *XRError) {
	results, xErr := Query(tx, `SELECT SID FROM Registries WHERE UID=?`, uid)
	if xErr != nil {
		return "", xErr
	}
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		log.FuncPrintf("tx: %s None found", tx.uuid)
		return "", nil
	}

	return NotNilString(row[0]), nil
}

func (sqlBE *SQLBackend) RegistryGetUsesXref(r *Registry) (bool, *XRError) {
	// UsesXref lives on the raw Registries table (not Entities/
	// Props, since it's a plain internal flag, not a real
	// attribute), so it needs its own tiny supplemental lookup here -
	// a single indexed PK read, once per Tx.
	results, xErr := Query(r.Tx, `SELECT UsesXref FROM Registries WHERE SID=?`,
		r.DbSID)
	if xErr != nil {
		return false, xErr
	}
	defer results.Close()

	if row := results.NextRow(); row != nil {
		return NotNilBoolDef(row[0], false), nil
	}

	panic("can't find registry: " + r.DbSID)
	return false, nil
}

func (sqlBE *SQLBackend) RegistrySetUsesXref(r *Registry, b bool) *XRError {
	xErr := DoZeroOne(r.Tx,
		`UPDATE Registries SET UsesXref=? WHERE SID=? AND UsesXref<>?`,
		b, r.Tx.Registry.DbSID, b)

	return xErr
}

func (sqlBE *SQLBackend) RegistryRecalcUsesXref(r *Registry) *XRError {
	// We do this via triggers in init.sql so no need to do it here.
	// However, here's the SQL I think we'd use:
	/*
			   UPDATE Registries
			       SET UsesXref = EXISTS(
			           SELECT 1 FROM Metas
			               WHERE RegistrySID=OLD.RegSID
			               AND xRefXID IS NOT NULL AND xRefXID != "" )
			       WHERE SID=OLD.RegSID AND UsesXref=true
		       may not need the != ""  part, not sure
	*/

	return nil
}

func (sqlBE *SQLBackend) GetGroups(r *Registry, accessMode int) ([]*Group, *XRError) {
	entities, xErr := sqlBE.getEntitiesFromQuery(r.Tx, r.DbSID, accessMode,
		fmt.Sprintf(`e.Type=%d`, ENTITY_GROUP))
	if xErr != nil {
		return nil, xErr
	}

	groups := make([]*Group, len(entities))
	for i, e := range entities {
		gType, _, _ := strings.Cut(e.XID, string(DB_IN))
		g := r.Tx.GetGroup(r, gType, e.UID)
		if g == nil {
			g = &Group{
				Entity:   *e,
				Registry: r,
			}
			g.Self = g
			g.Tx.AddGroup(g)
		}
		// TODO/WATCH - if it's already in the cache as R/O should we update
		// e.Object?
		groups[i] = g
	}

	return groups, nil
}
func (sqlBE *SQLBackend) GetGroupIDsByType(r *Registry, gType string,
	accessMode int) ([]string, *XRError) {

	results, xErr := Query(r.Tx, `
            SELECT UID FROM Entities
            WHERE ParentSID=? AND Plural=?`,
		r.DbSID, gType)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	list := make([]string, 0, len(results.AllRows))
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		list = append(list, NotNilString(row[0]))
	}

	return list, nil
}

// regUID, entityXID, err
func (sqlBE *SQLBackend) MapShortSelf(tx *Tx, ss string) (string, string, *XRError) {

	query := fmt.Sprintf(`
        SELECT r.UID, e.XID
        FROM Entities AS e
        JOIN Registries AS r ON (r.SID = e.RegSID)
        WHERE e.eSID = ?`)

	results, xErr := Query(tx, query, ss)
	if xErr != nil {
		return "", "", xErr
	}
	defer results.Close()

	row := results.NextRow()
	if row != nil {
		return NotNilString(row[0]), NotNilString(row[1]), nil
	}

	return "", "", nil
}
func (sqlBE *SQLBackend) GenerateResultEntities(reg *Registry, what string,
	XIDs []string) (EntityIterator, *XRError) {

	query, args, err := GenerateQuery(reg, what, XIDs)
	if err != nil {
		return nil, err
	}
	results, xErr := Query(reg.Tx, query, args...)
	if xErr != nil {
		return nil, xErr
	}
	// defer results.Close()

	iter := &sqlEntityIterator{
		Results:    results,
		Tx:         reg.Tx,
		AccessMode: FOR_READ,
		offset:     0,
		size:       uint64(len(results.AllRows)),
		total:      math.MaxUint64,
	}

	// Get "total", it's 14th column
	// TotalCount (see GenerateQuery) rides along on every row of the
	// (already fully in-memory - see Query()) paginated result, computed
	// as a whole-partition aggregate *before* the page-range filter was
	// applied - so it already reflects the entire result set, and we
	// don't need a second query just to get it.
	if results != nil && len(results.AllRows) > 0 {
		row := results.AllRows[0]
		if len(row) > 14 {
			totalStr := NotNilString(row[14])
			if totalStr != "" {
				if v, err := strconv.ParseUint(totalStr, 10, 64); err == nil {
					iter.total = v
				}
			}
		}
	}

	return iter, nil
}

func (sqlBE *SQLBackend) DeleteGroup(g *Group) *XRError {
	xErr := DoOne(g.Tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)

	return xErr
}

func (sqlBE *SQLBackend) HasReadOnlyResource(g *Group) (bool, *XRError) {
	// Make sure we don't have any readonly Resources. Callers (HTTPDelete/
	// HTTPDeleteGroups) already lock g itself FOR_WRITE, but that doesn't
	// lock the descendant Resources/Metas whose readonly Props this scans -
	// without FOR UPDATE here, a concurrent Tx setting readonly=true on a
	// descendant after our RR snapshot was established could be missed,
	// letting this Group be improperly deleted.
	results, xErr := Query(g.Tx, `
        SELECT EXISTS(SELECT 1 FROM Props
        WHERE RegSID=? AND Type=`+StrTypes(ENTITY_META)+` AND
          XID LIKE '`+g.XID+`/%' AND
          PropName='readonly`+string(DB_IN)+`' AND
          PropValue='true') FOR UPDATE`,
		g.Registry.DbSID)
	if xErr != nil {
		return false, xErr
	}
	defer results.Close()

	row := results.NextRow()

	return NotNilInt(row[0]) != 0, nil
}

func (sqlBE *SQLBackend) FindBadEqualsVersions(g *Group, gPP *PropPath, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError) {
	// validateEquals checks the "equals" half of a constraint: every
	// Version (real or xref-mirrored, since this scans Entities/
	// Props broadly) of resPlural, under this Group, must have
	// pp's value equal to this Group's own value of constraint.Equals.
	//
	// This scans ALL Resources/Versions under the Group, not just
	// whatever triggered this validation run, so it must use a locking
	// (FOR UPDATE) read rather than a plain SELECT. Under RR isolation a
	// plain SELECT would reuse this Tx's original snapshot (established
	// at its first read), which could predate a concurrent Tx's commit of
	// a new/changed sibling Resource or Version in this same Group -
	// letting two Txs each pass a constraint that's violated once
	// combined. FOR UPDATE forces this read to see latest-committed data
	// and to block on any in-flight conflicting Tx, closing that gap.

	query := fmt.Sprintf(`
            SELECT
                r.XID, v.UID, vp.PropValue
            FROM Resources r
            JOIN Entities AS v ON (
                v.RegSID=r.RegistrySID AND
                v.ParentSID=r.SID AND
                v.Type=?
            )
            JOIN Props AS gp ON (
                gp.RegSID=r.RegistrySID AND
                gp.eSID=r.GroupSID AND
                gp.PropName=?
            )
            LEFT JOIN Props AS vp ON (
                vp.RegSID=v.RegSID AND
                vp.eSID=v.eSID AND
                vp.PropName=?
            )
            WHERE
                # r.RegistrySID=? AND
                r.GroupSID=? AND
                r.Plural=? AND
                # r.GroupSID=? AND
                # r.ModelSID=? AND
                # #r.Plural=? AND
                (vp.PropValue IS NULL OR vp.PropValue<>gp.PropValue)
            FOR UPDATE`)

	results, xErr := Query(g.Tx, query,
		ENTITY_VERSION, gPP.DB(), rPP.DB(),
		// g.Registry.DbSID, g.DbSID, rm.Plural)
		g.DbSID, rm.Plural)
	// TODO figure out why we can query by rm.SID instead. For some reason
	// rm.SID doesn't match r.ModelSID. Must have something to do with either
	// that we're updating the model during this tx, or because the test is
	// using ximportresources.
	// See:  // bad: import with xref + group constraints / enum bad
	// in test_constraints.go
	// g.DbSID, rm.SID)
	if xErr != nil {
		return "", nil, xErr
	}
	defer results.Close()

	rXID := ""
	vIDs := []string{}

	for {
		row := results.NextRow()
		if row == nil {
			break
		}

		// r.XID, v.UID, vp.PropValue

		// log.Printf("tx: %s %q %q %q", g.Tx.uuid,
		// NotNilString(row[0]), NotNilString(row[1]),
		// NotNilString(row[2]))

		// Stop on 2nd Resource
		if rXID != "" && rXID != NotNilString(row[0]) {
			break
		}
		rXID = NotNilString(row[0])
		vIDs = append(vIDs, NotNilString(row[1]))

	}

	return rXID, vIDs, nil
}

func (sqlBE *SQLBackend) FindBadEnumVersions(g *Group, constraint *Constraint, rm *ResourceModel, rPP *PropPath) (string, []string, *XRError) {

	// validateEnum checks the "enum" half of a constraint: every Version
	// (real or xref-mirrored, since this scans Entities/Props
	// broadly - so a xref whose mirrored value violates the hosting
	// group's "enum" constraint is caught here too) of resPlural, under
	// this Group, must have pp's value (when set) be one of
	// constraint.Enum's values.
	//
	// See validateEquals()'s comment above for why this must be a FOR
	// UPDATE (locking) read rather than a plain SELECT.

	// Encode each enum value the same way prepDBProperty() encodes a
	// real attribute value before it's written to PropValue (booleans
	// as "true"/"false", everything else via fmt.Sprintf("%v", v)) so
	// the SQL string comparison lines up with what's actually stored.
	placeholders := make([]string, 0, len(constraint.Enum))

	args := []any{ENTITY_VERSION, rPP.DB(), g.Registry.DbSID, g.DbSID,
		rm.Plural}
	enumArgs := make([]any, 0, len(constraint.Enum))

	for _, v := range constraint.Enum {
		/*
			// encode v for proper mysql value (bool is special)
			if b, ok := v.(bool); ok {
				if b {
					v = "true"
				} else {
					v = "false"
				}
			} else {
				v = fmt.Sprintf("%v", v)
			}
		*/
		v = fmt.Sprintf("%v", v)
		enumArgs = append(enumArgs, v)
		placeholders = append(placeholders, "?")
	}
	args = append(args, enumArgs...)

	query := fmt.Sprintf(`
            SELECT
                r.XID, v.UID, vp.PropValue
            FROM Resources r
            JOIN Entities AS v ON (
                v.RegSID=r.RegistrySID AND
                v.ParentSID=r.SID AND
                v.Type=?
            )
            LEFT JOIN Props AS vp ON (
                vp.RegSID=v.RegSID AND
                vp.eSID=v.eSID AND
                vp.PropName=?
            )
            WHERE
                r.RegistrySID=? AND
                r.GroupSID=? AND
                r.Plural=? AND
                vp.PropValue IS NOT NULL AND
                vp.PropValue NOT IN (%s)
            FOR UPDATE
            `, strings.Join(placeholders, ","))

	results, xErr := Query(g.Tx, query, args...)
	if xErr != nil {
		return "", nil, xErr
	}
	defer results.Close()

	rXID := ""
	vIDs := []string{}

	for {
		row := results.NextRow()
		if row == nil {
			break
		}

		// Stop on 2nd Resource
		if rXID != "" && rXID != NotNilString(row[0]) {
			break
		}
		rXID = NotNilString(row[0])
		vIDs = append(vIDs, NotNilString(row[1]))
	}

	return rXID, vIDs, nil
}

func (sqlBE *SQLBackend) GetResources(g *Group, accessMode int) ([]*Resource, *XRError) {
	entities, xErr := sqlBE.getEntitiesFromQuery(g.Tx, g.Registry.DbSID, FOR_WRITE,
		`e.ParentSID=?`, g.DbSID)
	if xErr != nil {
		return nil, xErr
	}

	resources := make([]*Resource, len(entities))
	for i, e := range entities {
		parts := strings.Split(e.XID, "/") // /gType/gID/rType/rID
		r := g.Tx.GetResource(g, parts[3], e.UID)
		if r == nil {
			r = &Resource{
				Entity: *e,
				Group:  g,
			}
			r.Self = r
			r.Tx.AddResource(r)
		}
		// TODO/WATCH - if it's already in the cache as R/O should we update
		// e.Object?
		resources[i] = r
	}

	return resources, nil
}

func (sqlBE *SQLBackend) GetResourceIDsByType(g *Group, rType string,
	accessMode int) ([]string, *XRError) {

	results, xErr := Query(g.Tx, `
            SELECT UID FROM Entities
            WHERE ParentSID=? AND Plural=?`,
		g.DbSID, rType)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	list := make([]string, 0, len(results.AllRows))
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		list = append(list, NotNilString(row[0]))
	}

	return list, nil
}

func (sqlBE *SQLBackend) RegisterEntity(e *Entity) *XRError {
	switch e.Type {
	case ENTITY_REGISTRY:
		// Add Registry and Model to the DB's tables

		xErr := DoOne(e.Tx, `INSERT INTO Registries(SID, UID) VALUES(?,?)`,
			e.DbSID, e.UID)
		if xErr != nil {
			return xErr
		}

		xErr = DoOne(e.Tx, `INSERT INTO Models(RegistrySID) VALUES(?)`,
			e.DbSID)
		if xErr != nil {
			return xErr
		}

	case ENTITY_GROUP:
		xErr := DoOne(e.Tx, `
            INSERT INTO "Groups"(
                SID, RegistrySID, UID,
                ModelSID, XID, Abstract,
                Plural, Singular)
            VALUES(?,?,?,?,?,?,?,?)`,

			e.DbSID, e.Registry.DbSID, e.UID,
			e.GroupModel.SID, e.XID, e.Abstract,
			e.Plural, e.Singular)
		if xErr != nil {
			return xErr
		}

	case ENTITY_RESOURCE:
		xErr := DoOne(e.Tx, `
            INSERT INTO Resources(
                SID, UID, RegistrySID,
                GroupSID, ModelSID,
                XID, Abstract,
                Plural, Singular)
            VALUES(?,?,?,?,?,?,?,?,?)`,

			e.DbSID, e.UID, e.Registry.DbSID,
			e.ParentSID, e.ResourceModel.SID,
			e.XID, e.Abstract,
			e.Plural, e.Singular)
		if xErr != nil {
			return xErr
		}

	case ENTITY_META:
		xErr := DoOne(e.Tx, `
            INSERT INTO Metas(SID, RegistrySID, ResourceSID,
                        XID, Abstract,
                        Plural, Singular)
            VALUES(?,?,?,?,?,?,?)`,

			e.DbSID, e.Registry.DbSID, e.ParentSID,
			e.XID, e.Abstract,
			e.ResourceModel.Plural, e.ResourceModel.Singular)
		if xErr != nil {
			return xErr
		}

	case ENTITY_VERSION:
		xErr := DoOne(e.Tx, `
            INSERT INTO Versions(SID, UID, RegistrySID,
                ResourceSID, XID, Abstract)
            VALUES(?,?,?,?,?,?)`,

			e.DbSID, e.UID, e.Registry.DbSID, e.ParentSID,
			e.XID, e.Abstract)
		if xErr != nil {
			return xErr
		}

	default:
		panic(fmt.Sprintf("Uknown type: %d", e.Type))
	}

	// Old e.EntityInsert() code

	// EntityInsert adds a row to Entities for a newly-created
	// Registry/Group/Resource/Meta/Version - called from the same places
	// that insert into the corresponding "real" entity table, right after
	// e's fields (DbSID, ParentSID, etc.) have been populated. It also
	// writes e's write-once calculated ("IsCalcStatic") attributes here,
	// since they're provably immutable for the rest of this entity's
	// lifetime (see SaveCalcStaticInsert()'s doc comment), this only ever runs
	// once, at creation.
	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	// e.DbSID is always freshly generated for a brand-new entity, so
	// this REPLACE always inserts (never replaces) exactly 1 row.
	xErr := DoOne(e.Tx, `
        REPLACE INTO Entities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, XID, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?,?,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Abstract, e.XID)
	if xErr != nil {
		return xErr
	}

	// Old e.SaveCalcStaticInsert() code

	// SaveCalcStaticInsert writes e's write-once calculated attributes:
	// xid (every entity type) and Version.RESOURCEid (e.g. "fileid",
	// pointing at the owning Resource's UID). Neither can ever change
	// after creation: an entity's UID/XID is immutable (no rename API -
	// reusing an existing ID just errors instead of renaming), and a
	// Version's owning Resource never changes. So these only need to be
	// computed once - here, called from EntityInsert() right after
	// creation.
	// Marked IsCalcStatic=true so later reads/cascades can identify them
	// and, e.g., exclude them when copying an entity's "real" props
	// elsewhere.
	//
	// Also gives a brand-new Version its very first isdefault row (via
	// SaveVersionCalc() - see its doc comment): unlike xid/RESOURCEid,
	// isdefault's VALUE can change post-creation, but nothing else ever
	// INSERTs its row (the end-of-tx SaveDefaultVersionCascade() bulk
	// fix-up is an UPDATE...JOIN, so it can only correct an existing row,
	// never create one) - so without this, a new Version would have no
	// isdefault attribute at all until something else happened to trigger
	// a recompute.

	// xid - every entity type. Plain single-row INSERT, always exactly
	// 1 (this is called once, at creation, on a brand-new eSID).
	xErr = DoOne(e.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, true,
               false, false, false, true, false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.XID, "xid"+string(DB_IN), e.XID, e.Abstract)
	if xErr != nil {
		return xErr
	}

	if e.Type == ENTITY_VERSION {
		// e is always a real, in-memory Version, which always has a
		// parent Resource, so e.ParentSID is never empty here. The
		// owning Resource is guaranteed to already exist (e was just
		// created as one of its Versions), so this always inserts
		// exactly 1 row.
		xErr = DoOne(e.Tx, `
            INSERT INTO Props(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
                IsCalcStatic, IsCalcDynamic)
            SELECT ?,?,?,?,?,?,?,?, CONCAT(r.Singular,?), r.UID, 'string', ?,
                   true, false, false, false, true, false
            FROM Resources AS r WHERE r.SID=?`,
			e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
			e.DbSID, e.UID, e.XID, "id"+string(DB_IN), e.Abstract,
			e.ParentSID)
		if xErr != nil {
			return xErr
		}

		// Give this brand-new Version its first isdefault row. Its
		// DELETE is a no-op here (nothing exists yet for this eSID),
		// so this just inserts the initial value based on the owning
		// Resource's Meta.defaultVID at this moment - which is fine
		// even if it's not yet the final answer, since
		// SaveDefaultVersionCascade() will fix it up (for every
		// Version of this Resource) once EnsureLatest() has settled
		// on the true default, at end-of-tx.
		// e.SaveVersionCalc()

		// Old SaveVersionCalc() code
		// SaveVersionCalc (re)computes the calculated 'isdefault' attribute
		// for a (real, non-xref-synthetic) Version - the only Version-level
		// calculated value that can actually change post-creation (xid and
		// RESOURCEid are write-once - see SaveCalcStaticInsert(), called
		// once from EntityInsert() instead). Unlike those, isdefault isn't
		// recomputed on every subsequent Save() either: its row is inserted
		// exactly once, from SaveCalcStaticInsert() at creation, and its
		// VALUE is thereafter only ever corrected in bulk, for every Version
		// of a Resource at once, by the end-of-tx SaveDefaultVersionCascade()
		// UPDATE. This func itself is only ever called from
		// SaveCalcStaticInsert(), for that one initial insert.

		// isdefault - true only if this Version is the owning Resource's
		// current default (via its Meta.defaultVID), or - for a Resource
		// with no defaultVID set but which is itself an xref source - if
		// it matches the xref target's default. In the common non-xref
		// case this just checks m.defaultVID. The owning Resource's Meta
		// is guaranteed to exist, so this always inserts exactly 1 row.
		xErr = DoOne(e.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?,?,?,?,?,?,?,?, ?,
               IF(m.defaultVID IS NOT NULL AND ?=m.defaultVID, 'true', 'false'),
               'boolean', ?, true, false, false, false, false, true
        FROM Metas AS m WHERE m.ResourceSID=?`,
			e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
			e.UID, e.XID, "isdefault"+string(DB_IN), e.UID, e.Abstract,
			e.ParentSID)
		if xErr != nil {
			return xErr
		}
	}

	return nil
}

func (sqlBE *SQLBackend) LoadModelJSON(r *Registry) (string, string, *XRError) {
	// Load Registry model
	results, xErr := Query(r.Tx,
		`SELECT Model,Changed FROM Models WHERE RegistrySID=?`,
		r.DbSID)
	if xErr != nil {
		return "", "", xErr
	}
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		return "", "", nil
		/*
			NewXRError("server_error", "/model").
				SetDetailf("tx: %s Can't find registry: %s", r.Tx.uuid, r.UID)
		*/
	}

	model := NotNilString(row[0])
	changedUUID := NotNilString(row[1])

	return model, changedUUID, nil
}

func (sqlBE *SQLBackend) SaveModel(m *Model, changeUUID string) *XRError {
	buf, _ := json.Marshal(m)
	modelStr := string(buf)

	// log.FuncPrintf("tx: %s Saving model itself", x.Registry.Tx.uuid)
	xErr := DoZeroTwo(m.Registry.Tx, `
        INSERT INTO Models(RegistrySID, Model, Changed)
            VALUES(?,?,?)
            ON DUPLICATE KEY UPDATE Model=?, Changed=?`,

		m.Registry.DbSID, modelStr, changeUUID,
		modelStr, changeUUID)

	return xErr
}

func (sqlBE *SQLBackend) RegisterModelEntity(me any) *XRError {
	if gm, ok := me.(*GroupModel); ok {
		xErr := DoOne(gm.Model.Registry.Tx,
			`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
			gm.SID, gm.Model.Registry.DbSID, nil,
			"/"+gm.Plural, gm.Plural, gm.Singular)
		if xErr != nil {
			return xErr
		}
	} else if rm, ok := me.(*ResourceModel); ok {
		gm := rm.GroupModel
		gmAbs := "/" + gm.Plural

		xErr := DoOne(rm.GroupModel.Model.Registry.Tx,
			`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
			rm.SID, gm.Model.Registry.DbSID, gm.SID,
			gmAbs+"/"+rm.Plural, rm.Plural, rm.Singular)
		if xErr != nil {
			return xErr
		}
	} else {
		panic("I don't know who I am")
	}

	return nil
}

func (sqlBE *SQLBackend) DeleteModelEnityByAbstract(reg *Registry, abstract string) *XRError {
	xErr := DoOne(reg.Tx, `DELETE FROM ModelEntities
        WHERE RegistrySID=? AND Abstract=?`,
		reg.DbSID, abstract)

	return xErr
}

func (sqlBE *SQLBackend) GetModelEntitiesByAbstract(r *Registry) (map[string]string, *XRError) {
	// Load old model's ModelEntities so we can diff them
	existingModelEntities := map[string]string{} // Abstract->SID
	results, xErr := Query(r.Tx,
		`SELECT SID,Abstract FROM ModelEntities WHERE RegistrySID=?`,
		r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		sid := NotNilString(row[0])
		abs := NotNilString(row[1])
		existingModelEntities[abs] = sid
	}

	return existingModelEntities, nil
}

func (sqlBE *SQLBackend) CountGroupModelUsed(r *Registry, gmSID string) (int, *XRError) {
	results, xErr := Query(r.Tx,
		`SELECT COUNT(*) FROM "Groups" WHERE ModelSID=?`, gmSID)
	if xErr != nil {
		return 0, xErr
	}
	count := NotNilInt(results.NextRow()[0])
	results.Close()

	return count, nil
}

func (sqlBE *SQLBackend) CountResourceModelUsed(r *Registry, rmSID string) (int, *XRError) {
	results, xErr := Query(r.Tx,
		`SELECT COUNT(*) FROM Resources WHERE ModelSID=?`, rmSID)
	if xErr != nil {
		return 0, xErr
	}
	count := NotNilInt(results.NextRow()[0])
	results.Close()

	return count, nil
}

// versionXID, attr, err
func (sqlBE *SQLBackend) IsHasDocUsed(r *Registry, rm *ResourceModel) (string, string, *XRError) {
	names := []string{
		rm.Singular + string(DB_IN),
		rm.Singular + "url" + string(DB_IN),
		rm.Singular + "base64" + string(DB_IN),
		rm.Singular + "proxyurl" + string(DB_IN),
	}

	query := `
        SELECT v.XID, p.PropName FROM Versions v
        JOIN Resources r ON v.ResourceSID = r.SID
        JOIN Props p ON p.eSID = v.SID
        WHERE r.ModelSID = ?
        AND p.PropName IN (?, ?, ?, ?)
        AND p.PropValue IS NOT NULL
        LIMIT 1`

	results, xErr := Query(r.Tx, query, rm.SID,
		names[0], names[1], names[2], names[3])
	if xErr != nil {
		return "", "", xErr
	}
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		return "", "", nil
	}

	vXID := NotNilString(row[0])
	attr := strings.TrimRight(NotNilString(row[1]), string(DB_IN))

	return vXID, attr, nil
}

// versionXID, attr, err
func (sqlBE *SQLBackend) HasDocContent(r *Resource) (string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from ValidateResource()
	// on the write path.
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// The nested EXISTS subquery below reads ResourceContents, a
	// different table than the outer query's Versions - MySQL does NOT
	// propagate the outer query's FOR UPDATE into a correlated
	// subquery's own table reads (confirmed empirically elsewhere in
	// this investigation, e.g. the VersionAncestors view bug), so
	// lockExpr must be applied to the subquery explicitly too.
	query := `
        SELECT v.XID FROM Versions v
        WHERE v.ResourceSID = ?
        AND EXISTS (
            SELECT 1 FROM ResourceContents rc
            WHERE rc.VersionSID = v.SID` + lockExpr + `
        )
        LIMIT 1` + lockExpr

	results, xErr := Query(r.Tx, query, r.DbSID)
	if xErr != nil {
		return "", xErr
	}
	defer results.Close()

	row := results.NextRow()
	if row != nil {
		// Found a version with document content
		return NotNilString(row[0]), nil
	}
	return "", nil
}

func (sqlBE *SQLBackend) RefreshEntity(e *Entity, accessMode int) *XRError {
	mode := ""
	if accessMode == FOR_WRITE {
		mode = " FOR UPDATE"

		// Need to lock the entity so we grab the latest stuff
		results, xErr := Query(e.Tx,
			`SELECT UID FROM Entities WHERE eSID=? FOR UPDATE`, e.DbSID)
		if xErr != nil {
			return xErr
		}
		PanicIf(len(results.AllRows) != 1, "Rows: %d", len(results.AllRows))
		results.Close()
	}

	log.FuncPrintf("tx: %s Refreshing %q, mode: %v", e.Tx.uuid, e.XID, mode)

	results, xErr := Query(e.Tx, `
        SELECT PropName, PropValue, PropType, IsSystemProp
        FROM Props
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false
              AND IsCalcDynamic=false`+mode, e.DbSID)
	if xErr != nil {
		return xErr
	}
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

func (sqlBE *SQLBackend) ClearUserProps(e *Entity) *XRError {
	// Delete all user props for this entity, we assume that NewObject
	// contains everything we want going forward
	_, xErr := Do(e.Tx, `DELETE FROM Props
              WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                    AND IsXrefVerCopy=false AND IsSystemProp=false
                    AND IsCalcStatic=false AND IsCalcDynamic=false`,
		e.DbSID)

	return xErr
}

func batchUpdateProps(e *Entity, isSystem bool, args []any) *XRError {
	size := len(args) / 13

	isSystemStr := "false"
	if isSystem {
		isSystemStr = "true"
	}

	// e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
	// e.DbSID, e.UID, e.XID,
	// row.Name, *row.Value, row.Type, e.Abstract, row.DocView
	// IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
	// IsSystemProp, IsCalcStatic, IsCalcDynamic
	rowPlaceholder := "(?,?,?,?,?, ?,?,?, ?,?,?,?,?," +
		"false,false,false," +
		isSystemStr + ",false,false)"

	holders := strings.Repeat(rowPlaceholder+",", size-1)
	holders += rowPlaceholder

	_, xErr := Do(e.Tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID,
            eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsSystemProp, IsCalcStatic, IsCalcDynamic)
        VALUES `+holders, args...)

	return xErr
}

// BatchDeleteProps deletes multiple own/system Props
// rows for e (identified by DB PropName) in as few
// "DELETE ... WHERE PropName IN (...)" statements as possible, chunked
// the same way as DBWritePropsBatch. Matches DBWriteProp's
// single-row delete filter (own rows only - never cascaded/copied
// ones), regardless of isSystem, since own vs. system PropNames never
// collide.
func batchDeleteProps(e *Entity, names []string) *XRError {
	if len(names) == 0 {
		return nil
	}

	size := len(names)
	batchSize := min(size, dbPropBatchChunkSize)
	args := make([]any, 1, batchSize+1) // +1 for e.DbSID
	holders := strings.Repeat("?,", batchSize)

	pos := 0
	argsSize := 0
	args[0] = e.DbSID
	for {
		if pos == size || argsSize == batchSize {
			if argsSize == 0 {
				break
			}

			_, xErr := Do(e.Tx, `
                    DELETE FROM Props
                    WHERE eSID=? AND PropName IN (`+holders[:argsSize*2-1]+`)
                        AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                        AND IsXrefVerCopy=false`, args...)
			if xErr != nil {
				return xErr
			}
			args = args[:1] // save e.DbSID
			argsSize = 0
		} else {
			args = append(args, names[pos])
			argsSize++
			pos++
		}
	}

	return nil
}

func (sqlBE *SQLBackend) LockResourceBySID(tx *Tx, sid string) *XRError {
	results, xErr := Query(tx,
		"SELECT SID FROM Resources WHERE SID=? FOR UPDATE",
		sid)
	if xErr != nil {
		return xErr
	}
	results.Close()

	return nil
}

func (sqlBE *SQLBackend) DeleteResource(r *Resource) *XRError {
	// Any xref source's stale mirror is cleared by ResourcesTrigger
	// (init.sql), which fires for every deletion path (this, whole-
	// Group delete, whole-Registry delete) uniformly.
	xErr := DoOne(r.Tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)

	return xErr
}

func (sqlBE *SQLBackend) GetResourceContents(e *Entity) ([]byte, *XRError) {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		contentID := e.Get("#contentid")

		results, xErr := Query(e.Tx, `
            SELECT Content FROM ResourceContents WHERE VersionSID=? `,
			contentID)
		if xErr != nil {
			return nil, xErr
		}
		defer results.Close()

		row := results.NextRow()
		if row == nil {
			// No data so just return
			return nil, nil
		}

		if results.NextRow() != nil {
			panic("too many results")
		}

		return (*(row[0])).([]byte), nil
	}

	panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
}

func (sqlBE *SQLBackend) SetResourceContents(e *Entity, val []byte) *XRError {
	if IsNil(val) {
		// Remove the content
		_, xErr := Do(e.Tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
			e.DbSID)
		if xErr != nil {
			return xErr
		}
	} else {
		// Update the content
		xErr := DoOneTwo(e.Tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
                VALUES(?,?)`, e.DbSID, val)
		if xErr != nil {
			return xErr
		}
	}
	return nil
}

func (sqlBE *SQLBackend) RecalcVersionsIsDefault(r *Resource) *XRError {
	// Fix up isdefault on every one of this Resource's OWN Versions -
	// they were each set based on their own state at their own last
	// Save(), so if the default just moved to a different Version, the
	// old default's row (and the new one's, if it wasn't the one that
	// triggered this call) would otherwise go stale. This must run
	// BEFORE the copy below, so ver's own "isdefault" row is already
	// correct by the time it gets mirrored into the Resource.
	_, xErr := Do(r.Tx, `
        UPDATE Props AS p
        JOIN Versions AS v ON (v.SID=p.eSID)
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        SET p.PropValue = IF(v.UID=m.defaultVID, 'true', 'false')
        WHERE v.ResourceSID=? AND p.PropName=?`,
		r.DbSID, "isdefault"+string(DB_IN))

	return xErr
}

func (sqlBE *SQLBackend) DeleteResourceDefaultVersionProps(r *Resource) *XRError {
	_, xErr := Do(r.Tx,
		`DELETE FROM Props WHERE eSID=? AND IsDefaultVerCopy=true`,
		r.DbSID)
	return xErr
}

func (sqlBE *SQLBackend) CopyResourceDefaultVersionProps(r *Resource) *XRError {
	// IsCalcDynamic isn't excluded here (unlike IsCalcStatic) so ver's
	// own "isdefault" row (just fixed up above) is mirrored into the
	// Resource too - like createdat/modifiedat, it's just copied
	// content, no special-casing needed. It's simply absent whenever
	// there's no default Version to copy from at all (see the ver ==
	// nil branch above).
	ver, xErr := r.GetDefault()
	if xErr != nil {
		return xErr
	}

	_, xErr = Do(r.Tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               true, false, false
        FROM Props
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false`,
		r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID, r.DbSID,
		r.UID, r.XID, r.Abstract, ver.DbSID)

	return xErr
}

func (sqlBE *SQLBackend) FindBadAncestorsCreatedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	// Search the DB for all Versions of this Resource, sorted by 'createdat'
	// and return the ones that do not have the proper 'ancestorid' value.
	// Meaning, they don't point to the next oldest one (based on createdat)
	lockExpr := ""
	if lock {
		lockExpr = " FOR UPDATE"
	}

	results, xErr := Query(r.Tx, `
                SELECT UID, ExpectedAncestorID FROM (
                  SELECT CreatedAt,
                         UID,
                         AncestorID,
                         IFNULL(lag(UID) OVER (ORDER BY CreatedAt, UID),
                                UID) AS ExpectedAncestorID
                  FROM Versions
                  WHERE RegistrySID=? AND ResourceSID=?`+lockExpr+`) AS list
                WHERE list.AncestorID != list.ExpectedAncestorID
                ORDER BY CreatedAt ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	AVs := []*AncestorVersion{}

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		AVs = append(AVs, &AncestorVersion{
			VersionUID: NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
		})
	}

	return AVs, nil
}

func (sqlBE *SQLBackend) FindBadAncestorsModifiedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	// Search the DB for all Versions of this Resource, sorted by
	// 'modifiedat' and return the ones that do not have the proper
	// 'ancestorid' value. Meaning, they don't point to the next oldest
	// one (based on modifiedat)
	lockExpr := ""
	if lock {
		lockExpr = " FOR UPDATE"
	}

	results, xErr := Query(r.Tx, `
                SELECT UID, ExpectedAncestorID FROM (
                  SELECT ModifiedAt,
                         UID,
                         AncestorID,
                         IFNULL(lag(UID) OVER (ORDER BY ModifiedAt, UID),
                                UID) AS ExpectedAncestorID
                  FROM Versions
                  WHERE RegistrySID=? AND ResourceSID=?`+lockExpr+`) AS list
                WHERE list.AncestorID != list.ExpectedAncestorID
                ORDER BY ModifiedAt ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	AVs := []*AncestorVersion{}

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		AVs = append(AVs, &AncestorVersion{
			VersionUID: NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
		})
	}

	return AVs, nil
}

func (sqlBE *SQLBackend) FindBadAncestorsSemVer(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	// Search the DB for all Versions of this Resource, sorted by
	// 'semver' and return the ones that do not have the proper
	// 'ancestorid' value. Meaning, they don't point to the next oldest
	// one (based on semver)

	lockExpr := ""
	if lock {
		lockExpr = " FOR UPDATE"
	}

	results, xErr := Query(r.Tx, `
                SELECT UID, AncestorID FROM Versions
                WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	AVs := []*AncestorVersion{}

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		AVs = append(AVs, &AncestorVersion{
			VersionUID: NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
		})
	}

	// Sort oldest->newest by semver precedence, falling back to a
	// case-insensitive 'versionid' comparison to break ties (same
	// tie-break convention used elsewhere for "newest"/"oldest" - see
	// the versionmode doc in model.md), so results are deterministic.
	sort.Slice(AVs, func(i, j int) bool {
		if c := CompareSemver(AVs[i].VersionUID, AVs[j].VersionUID); c != 0 {
			return c < 0
		}
		iVUID := strings.ToLower(AVs[i].VersionUID)
		jVUID := strings.ToLower(AVs[j].VersionUID)
		return iVUID < jVUID
	})

	AVs2 := []*AncestorVersion{}

	// Pick just AVs that are bad
	for i, av := range AVs {
		expectedAncestorID := av.VersionUID // oldest/root points to itself
		if i > 0 {
			expectedAncestorID = AVs[i-1].VersionUID
		}

		if av.AncestorID == expectedAncestorID {
			continue
		}

		av.AncestorID = expectedAncestorID // set to proper value
		AVs2 = append(AVs2, av)
	}

	return AVs2, nil
}

func (sqlBE *SQLBackend) FindNewestVersionID(r *Resource, excludeTBD bool) (string, *XRError) {

	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	exclude := ""
	if excludeTBD {
		exclude = ` AND v.AncestorID <> '` + ANCESTORID_TBD + `'`
	}

	results, xErr := Query(r.Tx, `
                SELECT v.UID FROM Versions AS v
                WHERE
                  v.ResourceSID=? `+exclude+`
                  AND NOT EXISTS (
                    SELECT 1 FROM Versions AS v2
                    WHERE v2.ResourceSID=v.ResourceSID AND
                      v2.AncestorID=v.UID AND
                      v2.UID <> v.UID`+lockExpr+`)
                ORDER BY v.CreatedAt DESC, v.UID `+FILTER_CI_COLLATE+` DESC
                LIMIT 1`+lockExpr,
		r.DbSID)
	if xErr != nil {
		return "", xErr
	}

	defer results.Close()

	for {
		row := results.NextRow()
		if row == nil { // nothing
			break
		}

		return NotNilString(row[0]), nil
	}

	// Must have a cycle - let the caller (non-backend specific code) deal
	// with it

	return "", nil
}

func (sqlBE *SQLBackend) GetVersionIDs(r *Resource) ([]*VersionAncestor, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID().
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Find all version IDs for this Resource
	results, xErr := Query(r.Tx, `
            SELECT UID, AncestorID, CreatedAt FROM Versions
            WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	VAs := ([]*VersionAncestor)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		VAs = append(VAs, &VersionAncestor{
			VID:        NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
			CreatedAt:  NotNilString(row[2]),
		})
	}

	return VAs, nil
}

func (sqlBE *SQLBackend) GetRootVersionIDs(r *Resource) ([]string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors()
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	results, xErr := Query(r.Tx, `
            SELECT UID FROM Versions
            WHERE RegistrySID=? AND ResourceSID=? AND UID=AncestorID`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	vIDs := ([]string)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vIDs = append(vIDs, NotNilString(row[0]))
	}

	return vIDs, nil
}

func (sqlBE *SQLBackend) GetProblematicVersions(r *Resource) ([]*VersionAncestor, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). This is called from
	// CheckAncestors() on the write path, and its correlated subquery
	// needs its own lock hint too (see HasCircularAncestors()).
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// Find all versions that point to non-existing versions
	results, xErr := Query(r.Tx, `
            SELECT v1.UID, v1.AncestorID, v1.CreatedAt FROM Versions AS v1
            WHERE v1.RegistrySID=? AND
                  v1.ResourceSID=? AND
                  (v1.AncestorID='`+ANCESTORID_TBD+`' OR (
                      v1.UID<>v1.AncestorID AND
                      NOT EXISTS(SELECT 1 FROM Versions AS v2
                                WHERE v2.RegistrySID=v1.RegistrySID AND
                                      v2.ResourceSID=v1.ResourceSID AND
                                      v2.UID=v1.AncestorID`+lockExpr+`)))
            ORDER BY CreatedAt ASC, UID ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	vers := ([]*VersionAncestor)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:        NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
			CreatedAt:  NotNilString(row[2]),
		})
	}

	return vers, nil
}

func (sqlBE *SQLBackend) GetChildVersionIDs(r *Resource, parentVID string) ([]string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from WillDelete() on
	// the write path.
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Find all versions that point 'parentVID'.
	// Note that roots will include themselves - not sure if this is ok or not
	results, xErr := Query(r.Tx, `
            SELECT UID FROM Versions
            WHERE RegistrySID=? AND ResourceSID=? AND AncestorID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID, parentVID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	vIDs := ([]string)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vIDs = append(vIDs, NotNilString(row[0]))
	}

	return vIDs, nil
}

func (sqlBE *SQLBackend) GetNumberOfVersions(r *Resource) (int, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from write paths
	// (e.g. Version.Delete(), UpsertVersion()).
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Get the list of Version IDs for this Resource
	results, xErr := Query(r.Tx, `
            SELECT COUNT(*) FROM Versions
            WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	if xErr != nil {
		return 0, xErr
	}
	defer results.Close()

	row := results.NextRow()
	return NotNilInt(row[0]), nil
}

func (sqlBE *SQLBackend) GetVersions(r *Resource, accessMode int) ([]*Version, *XRError) {
	entities, xErr := sqlBE.getEntitiesFromQuery(r.Tx, r.Registry.DbSID,
		accessMode, `e.ParentSID=? AND e.Type=?`, r.DbSID, ENTITY_VERSION)
	if xErr != nil {
		return nil, xErr
	}

	versions := make([]*Version, len(entities))

	for i, e := range entities {
		v := r.Tx.GetVersion(r, e.UID)
		if v == nil {
			v = &Version{
				Entity:   *e,
				Resource: r,
			}
			v.Self = v
			v.Tx.AddVersion(v)
		}
		// TODO/WATCH - if it's already in the cache as R/O should we update
		// e.Object?
		versions[i] = v
	}

	return versions, nil
}

func (sqlBE *SQLBackend) GetXrefSources(r *Resource) ([]string, *XRError) {
	XIDs := []string{}

	// FOR UPDATE: this is r (the target) reading the "who xrefs me" set
	// from Metas.xRefXID - a plain SELECT here would still be pinned
	// to this Tx's original RR snapshot, and could miss a source Meta
	// that set its xref to r AND committed after this Tx began (e.g. a
	// brand new source Resource created and xref'd to r by a concurrent
	// Tx that's already committed by the time we run this). Missing it
	// here means that source's mirrored Props never get refreshed to
	// reflect r's just-saved change, even though r is committing last.
	results, xErr := Query(r.Tx, `
        SELECT res.XID
        FROM Metas AS m
        JOIN Resources AS res ON (res.SID=m.ResourceSID)
        WHERE m.RegistrySID=? AND m.xRefXID=?  FOR UPDATE`,
		r.Registry.DbSID, r.XID)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		XIDs = append(XIDs, NotNilString(row[0]))
	}

	return XIDs, nil
}

// numValues, numNonEmptyUnique, numEmpty, error
func (sqlBE *SQLBackend) CountNonMatchingVersions(r *Resource, vAttrPath *PropPath) (int, int, int, *XRError) {

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors(),
	// otherwise this can miss sibling Version
	// rows committed by other Txs after this tx's snapshot was taken.
	lockExpr := ""
	// if meta := r.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
	if r.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	query := `
        SELECT count(*),p.PropName,p.PropValue FROM Entities e
        LEFT JOIN Props AS p ON ( p.eSID=e.eSID AND p.PropName=?)
        WHERE e.RegSID = ?  AND e.ParentSID = ?  AND e.Type = ?
        GROUP BY PropValue` + lockExpr

	results, xErr := Query(r.Tx, query, vAttrPath.DB(),
		r.Registry.DbSID, r.DbSID, ENTITY_VERSION)
	if xErr != nil {
		return 0, 0, 0, xErr
	}
	defer results.Close()

	numEmpty := 0
	numNonEmpty := 0
	numValues := 0
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		if IsNil(*row[2]) {
			// if NotNilString(row[2]) == ""
			numEmpty = NotNilInt(row[0])
		} else {
			numNonEmpty++
		}
		numValues++
	}

	return numValues, numNonEmpty, numEmpty, nil
}

func (sqlBE *SQLBackend) DeleteMeta(meta *Meta) *XRError {
	// Props/Entities rows for this Meta are cleaned up by
	// ResourcesTrigger (ParentSID=OLD.SID) when the owning Resource is
	// deleted right after this.
	xErr := DoOne(meta.Tx, `DELETE FROM Metas WHERE SID=?`, meta.DbSID)

	return xErr
}

func (sqlBE *SQLBackend) DeleteVersion(v *Version) *XRError {
	xErr := DoZeroOne(v.Tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)

	return xErr
}

// Delete xref'd meta props, all Versions xref'd props (all?), xref'd Versions
func (sqlBE *SQLBackend) ClearXrefState(meta *Meta) *XRError {
	_, xErr := Do(meta.Tx, `DELETE FROM Props
        WHERE eSID=? AND IsXrefPropCopy=true`,
		meta.DbSID)
	if xErr != nil {
		return xErr
	}
	_, xErr = Do(meta.Tx, `DELETE FROM Props
        WHERE ParentSID=? AND IsXrefVerCopy=true`,
		meta.ParentSID)
	if xErr != nil {
		return xErr
	}
	_, xErr = Do(meta.Tx, `DELETE FROM Entities
        WHERE ParentSID=? AND IsXrefVerCopy=true`,
		meta.ParentSID)

	return xErr
}

func (sqlBE *SQLBackend) CopyXrefState(meta *Meta) *XRError {
	/*  Shouldn't need to look at the DB since it's locked/cached
		results := Query(meta.Tx, `
	        SELECT xRefXID FROM Metas WHERE SID=?`, meta.DbSID)
		row := results.NextRow()
		results.Close()
		if row == nil || NotNilString(row[0]) == "" {
			return nil
		}
		xRefXID := NotNilString(row[0])
	*/
	// Delete panic once we know we can delete the above query code
	PanicIf(meta.AccessMode != FOR_WRITE, "%q should be locked", meta.XID)
	xRefXID := meta.GetAsString("xref")

	// Resolve the target live, by RegistrySID+XID (XID alone isn't
	// unique across the whole DB, only within one Registry) - never by
	// a cached SID, so this always reflects reality even if the
	// target didn't exist (or existed under a different SID) the last
	// time this ran. This is a source reading its xref TARGET's row,
	// the mirror image of SaveXrefFanOutForTarget's target-reads-
	// sources direction (which correctly uses FindResourceByXID(...,
	// FOR_WRITE)) - so it must be FOR UPDATE too: a plain SELECT here
	// would still be pinned to this Tx's RR snapshot and could copy
	// stale target data into this source's mirror even after a
	// concurrent Tx already committed a newer version of the target,
	// which would then feed this source's Group constraint validation
	// with stale mirrored data.
	tResults, xErr := Query(meta.Tx, `
        SELECT m.SID, m.ResourceSID, r.Singular FROM Resources AS r
        JOIN Metas AS m ON (m.ResourceSID=r.SID)
        WHERE r.RegistrySID=? AND r.XID=?
        FOR UPDATE`, meta.Registry.DbSID, xRefXID)
	if xErr != nil {
		return xErr
	}

	tRow := tResults.NextRow()
	tResults.Close()
	if tRow == nil {
		return nil
	}
	targetMetaSID := NotNilString(tRow[0])
	targetResourceSID := NotNilString(tRow[1])
	targetSingular := NotNilString(tRow[2])

	// meta is always the real Meta entity, so its owning Resource is
	// directly accessible via meta.Self.(*Meta).Resource - no need to
	// look it up as a "parent" entity at all. meta always has a parent
	// Resource, so meta.ParentSID is never empty here.
	resource := meta.Self.(*Meta).Resource

	// Copy the target's meta.* props into this (source) Meta, excluding
	// its own xref and "<singular>id" attrs, and any '#' internal props.
	_, xErr = Do(meta.Tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               false, true, false
        FROM Props
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false
              AND IsCalcDynamic=false
              AND PropName NOT IN (?, ?) AND LEFT(PropName,1)<>'#'`,
		meta.Registry.DbSID, meta.Type, meta.Plural, meta.Singular,
		meta.ParentSID, meta.DbSID, meta.UID, meta.XID,
		meta.Abstract,
		targetMetaSID,
		"xref"+string(DB_IN), targetSingular+"id"+string(DB_IN))
	if xErr != nil {
		return xErr
	}

	// Create the xref'd Versions
	if resource != nil {
		xErr := resource.SaveXrefVersionCopies(targetResourceSID)
		if xErr != nil {
			return xErr
		}
	}

	return nil
}

func (sqlBE *SQLBackend) CopyXrefDefaultVersionProps(r *Resource) *XRError {
	// No real default Version - this Resource may be an xref
	// source with no Versions of its own, in which case its
	// "current default" is really the xref target's current
	// default, copied in as a synthetic Version by
	// SaveXrefVersionCopies(). Copy from THAT synthetic
	// Props eSID instead of Props.
	//
	// This is a source reading its xref TARGET's row (same
	// direction as SaveXrefCascadeInsert()'s tResults query), so it
	// needs its own FOR UPDATE too: a plain SELECT here would still
	// be pinned to this Tx's original RR snapshot, and could miss
	// the target entirely (or see a stale defaultVID) even if the
	// target Resource/Meta/Version was created AND committed by a
	// concurrent Tx after this Tx began - silently leaving this
	// source's mirrored default-version Props missing/stale.
	tResults, xErr := Query(r.Tx, `
            SELECT v.SID FROM Metas AS srcM
            JOIN Resources AS tr ON (tr.RegistrySID=srcM.RegistrySID AND
                                      tr.XID=srcM.xRefXID)
            JOIN Metas AS m ON (m.ResourceSID=tr.SID)
            JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                    v.UID=m.defaultVID)
            WHERE srcM.ResourceSID=? FOR UPDATE`,
		r.DbSID)
	if xErr != nil {
		return xErr
	}
	tRow := tResults.NextRow()
	tResults.Close()
	if tRow == nil {
		return nil
	}
	targetDefVerSID := NotNilString(tRow[0])
	synthESID := fmt.Sprintf("-%s-%s", r.DbSID, targetDefVerSID)

	// IsCalcDynamic isn't excluded here (unlike IsCalcStatic) so
	// the synthetic version's own "isdefault" row (already
	// correctly computed by SaveXrefVersionCopies(), since this
	// synthESID always corresponds to the target's CURRENT
	// default version) is copied in too - just like createdat/
	// modifiedat, it's simply mirrored content, not a special
	// case. If the xref is dangling (tRow == nil, above) nothing
	// gets copied at all, so "isdefault" - along with every other
	// mirrored attribute - is naturally absent, exactly like a
	// Resource with no default Version at all.
	_, xErr = Do(r.Tx, `
            REPLACE INTO Props(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   true, false, false
            FROM Props WHERE eSID=? AND IsXrefVerCopy=true
                  AND IsCalcStatic=false`,
		r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID,
		r.DbSID, r.UID, r.XID, r.Abstract, synthESID)

	return xErr
}

// copy xref vers
// SaveXrefVersionCopies (re)creates the synthetic Entities/
// Props Version rows for srcResource (a real, in-memory
// *Resource - either e.Self.(*Meta).Resource in the direct-Save() case,
// or one resolved via Registry.FindResourceByXID() in the xref fan-out
// case)
// that xrefs targetResourceSID, one set of rows per Version the target
// currently has - all done via set-based SQL (no per-Version Go loop/
// round-trip) by joining against Versions/Resources/Metas directly.
// targetResourceSID is passed as a bare SID rather than a loaded
// *Resource: it's only ever used inside SQL WHERE/JOIN clauses here,
// never as a Go-level field access, so there's no need to pay for
// loading/caching that entity just to extract its SID back out again.
// The synthetic eSID for each target Version is deterministically
// CONCAT('-', sourceResourceSID, '-', v.SID), matching the
// "-<srcRSID>-<verSID>" convention.
//
// Every INSERT...SELECT below that reads the target's Versions/Props
// uses FOR UPDATE: this is srcResource (a source) reading its xref
// TARGET's rows, the mirror image of SaveXrefFanOutForTarget's
// target-reads-sources direction (which already locks each source
// FOR_WRITE). Without FOR UPDATE here, a plain SELECT would still be
// pinned to this Tx's RR snapshot and could copy stale target Version/
// Prop data into the source's mirror even after a concurrent Tx
// already committed a newer Version, which would then feed this
// source's own Group constraint validation with stale mirrored data.
// (The DELETE...JOIN statements above don't need this: DELETE/UPDATE
// searches always read latest-committed data in InnoDB, unlike plain
// SELECTs - only these INSERT...SELECTs need the explicit FOR UPDATE.)
func (sqlBE *SQLBackend) CopyXrefVersions(r *Resource, targetSID string) *XRError {
	if r == nil {
		return nil
	}

	targetResourceSID := targetSID
	srcResource := r
	sourceResourceSID := r.DbSID
	synthAbstract := srcResource.Abstract + string(DB_IN) + "versions"

	// Idempotent: this is called both from SaveXrefCascadeInsert
	// (which already cleared out ALL of this source's xref-version
	// rows first) and directly from SaveXrefFanOutForTarget
	// (which does not) - so clear out just the synthetic versions that
	// correspond to the target's CURRENT version set before
	// recreating them, or a second Save() of the same target Version
	// would hit a duplicate-key error here.
	_, xErr := Do(srcResource.Tx, `
        DELETE ft FROM Props AS ft
        JOIN Versions AS v ON (ft.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	if xErr != nil {
		return xErr
	}

	_, xErr = Do(srcResource.Tx, `
        DELETE fe FROM Entities AS fe
        JOIN Versions AS v ON (fe.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	if xErr != nil {
		return xErr
	}

	// One Entities row per target Version, all at once.
	_, xErr = Do(srcResource.Tx, `
        REPLACE INTO Entities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, XID, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID, ?,
               CONCAT(?, '/versions/', v.UID), true
        FROM Versions AS v WHERE v.ResourceSID=? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, synthAbstract, srcResource.XID,
		targetResourceSID)
	if xErr != nil {
		return xErr
	}

	// Copy each target Version's own props onto its corresponding
	// synthetic eSID, for every current Version at once (excluding the
	// target's own "xref" - Versions never have one, but kept for
	// parity with the old per-row exclusion).
	_, xErr = Do(srcResource.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ft.PropName, ft.PropValue, ft.PropType, ?, false,
               false, false, true
        FROM Versions AS v
        JOIN Props AS ft ON (ft.eSID=v.SID)
        WHERE v.ResourceSID=? AND ft.IsDefaultVerCopy=false
              AND ft.IsXrefPropCopy=false AND ft.IsXrefVerCopy=false
              AND ft.IsCalcStatic=false AND ft.IsCalcDynamic=false
              AND ft.PropName<>? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.XID, synthAbstract,
		targetResourceSID, "xref"+string(DB_IN))
	if xErr != nil {
		return xErr
	}

	// Calculated attrs for every synthetic version at once: xid and
	// RESOURCEid (using the SOURCE resource's singular/UID, since
	// that's every synthetic version's effective parent) are static -
	// wholesale recreated here only because the whole synthetic-
	// version set itself is being recreated (the xref pointer moved),
	// not because they individually change; isdefault is genuinely
	// dynamic (mirrors the target's own per-Version isdefault).
	_, xErr = Do(srcResource.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, CONCAT(?, '/versions/', v.UID), 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v WHERE v.ResourceSID=? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.XID,
		"xid"+string(DB_IN), srcResource.XID, synthAbstract, targetResourceSID)
	if xErr != nil {
		return xErr
	}

	_, xErr = Do(srcResource.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               CONCAT(r.Singular, ?), r.UID, 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v
        JOIN Resources AS r ON (r.SID=?)
        WHERE v.ResourceSID=? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.XID,
		"id"+string(DB_IN), synthAbstract, sourceResourceSID,
		targetResourceSID)
	if xErr != nil {
		return xErr
	}

	_, xErr = Do(srcResource.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, IF(m.defaultVID=v.UID, 'true', 'false'), 'boolean', ?,
               false, false, false, true, false, true
        FROM Versions AS v
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        WHERE v.ResourceSID=? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.XID,
		"isdefault"+string(DB_IN), synthAbstract, targetResourceSID)

	return xErr
}

func (sqlBE *SQLBackend) GetEntityFromXID(tx *Tx, regSID string, xidStr string,
	allAttrs bool, anyCase bool, accessMode int) (*Entity, *XRError) {

	defer log.Trace("tx: %s xid:%q all:%v", tx.uuid, xidStr, allAttrs)()

	iter, xErr := sqlBE.GetEntitiesFromXID(tx, regSID, xidStr, allAttrs,
		false, anyCase, accessMode)
	if xErr != nil {
		return nil, xErr
	}

	e, xErr := iter.Next()
	if xErr != nil {
		return nil, xErr
	}

	// Make sure there was only one
	e2, xErr := iter.Next()
	if xErr != nil {
		return nil, xErr
	}
	PanicIf(e2 != nil, "too many entities: %v", e2)

	return e, nil
}

func (sqlBE *SQLBackend) GetEntitiesFromXID(tx *Tx, regSID string,
	xidStr string, allAttrs bool, children bool, anyCase bool,
	accessMode int) (EntityIterator, *XRError) {

	defer log.Trace("tx: %s xid:%q all:%v child:%v",
		tx.uuid, xidStr, allAttrs, children)()

	// RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,
	// PropName,PropValue,PropType,IsSystemProp
	//   0     1     2       3         4      5   6     7      8
	//     9        10         11         12

	args := []any{regSID}

	XID := "XID"
	if anyCase {
		XID = "LowerXID"
		xidStr = strings.ToLower(xidStr)
	}
	XIDExpr := `(` + XID + `=?`
	args = append(args, xidStr)

	if children {
		XIDExpr += ` OR ` + XID + ` LIKE ?`
		args = append(args, xidStr+"/%")
	}
	XIDExpr += `)`

	// If the caller already wants FOR_WRITE on this (not-yet-cached)
	// entity, lock its rows as of THIS initial fetch rather than waiting
	// for some later Entity.Lock()/Refresh(FOR_WRITE) call - without
	// this, Entity.Lock() would see AccessMode already == FOR_WRITE
	// (set below by readNextEntity()) and skip re-locking entirely,
	// leaving the entity believed-locked in Go but never actually
	// row-locked in the DB. Matches Entity.Refresh()'s own " FOR UPDATE"
	// handling for its Props-only reload query.
	lockExpr := ""
	if accessMode == FOR_WRITE {
		lockExpr = ` FOR UPDATE`
	}

	subsetProps := ""
	if !allAttrs {
		subsetProps = `
            AND p.IsDefaultVerCopy=false AND p.IsXrefPropCopy=false
            AND p.IsXrefVerCopy=false AND p.IsCalcStatic=false
            AND p.IsCalcDynamic=false`
	}

	queryString := `SELECT
            p.RegSID,
            p.Type,
            p.Plural,
            p.Singular,
            p.ParentSID,
            p.eSID,
            p.UID,
            p.Abstract,
            p.XID,
            p.PropName,
            p.PropValue,
            p.PropType,
            p.IsSystemProp
        FROM Props AS p WHERE RegSID=? AND ` + XIDExpr + subsetProps + `
        ORDER BY XID` + lockExpr

	// log.Printf("Query:\n%s\nArgs: %v", queryString, args)

	results, xErr := Query(tx, queryString, args...)
	if xErr != nil {
		return nil, xErr
	}
	// defer results.Close()

	iterator := &sqlEntityIterator{
		Results:    results,
		Tx:         tx,
		AccessMode: accessMode,
		offset:     0,
		size:       uint64(len(results.AllRows)),
		total:      math.MaxUint64, // unknown
	}

	return iterator, nil
}

func (sqlBE *SQLBackend) getEntitiesFromQuery(tx *Tx, regID string,
	accessMode int, query string, args ...any) ([]*Entity, *XRError) {

	defer log.Trace("tx: %s %s", tx.uuid, query)()

	// RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,
	// PropName,PropValue,PropType,IsSystemProp
	//   0     1     2       3         4      5   6     7      8
	//     9        10         11         12

	if query != "" {
		query = "AND (" + query + ") "
	}

	args = append(append([]any{}, regID), args...)

	// See GetEntitiesFromXID()'s matching comment - without this, a
	// FOR_WRITE caller's very first (not-yet-cached) fetch would never
	// actually lock these rows, yet Entity.Lock() would believe it's
	// already locked (AccessMode == FOR_WRITE, stamped by
	// readNextEntity() below) and silently skip re-locking.
	lockExpr := ""
	if accessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	results, xErr := Query(tx, `
        SELECT
            e.RegSID as RegSID,
            e.Type as Type,
            e.Plural as Plural,
            e.Singular as Singular,
            e.ParentSID as ParentSID,
            e.eSID as eSID,
            e.UID as UID,
            e.Abstract as Abstract,
            e.XID as XID,
            p.PropName as PropName,
            p.PropValue as PropValue,
            p.PropType as PropType,
            p.IsSystemProp as IsSystemProp
        FROM Entities AS e
        LEFT JOIN Props AS p ON (
            e.eSID=p.eSID AND p.IsDefaultVerCopy=false AND p.IsXrefPropCopy=false
            AND p.IsXrefVerCopy=false AND p.IsCalcStatic=false
            AND p.IsCalcDynamic=false)
        WHERE e.RegSID=? `+query+` ORDER BY XID`+lockExpr, args...)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	entities := []*Entity{}
	for {
		e, xErr := readNextEntity(tx, results, accessMode)
		if xErr != nil {
			return nil, xErr
		}
		if e == nil {
			break
		}
		entities = append(entities, e)
	}

	// See GetEntitiesFromXID()'s matching comment: lock each returned
	// entity's Resource+Meta+Versions family too, if applicable.
	if accessMode == FOR_WRITE {
		for _, e := range entities {
			if xErr := lockEntityFamily(tx, e); xErr != nil {
				return nil, xErr
			}
		}
	}

	return entities, nil
}

type sqlEntityIterator struct {
	Results    *Result
	Tx         *Tx
	AccessMode int

	offset uint64
	size   uint64
	total  uint64
}

func (sei *sqlEntityIterator) Next() (*Entity, *XRError) {
	e, xErr := readNextEntity(sei.Tx, sei.Results, sei.AccessMode)
	if xErr != nil {
		return nil, xErr
	}

	if e != nil && sei.AccessMode == FOR_WRITE {
		if xErr := lockEntityFamily(sei.Tx, e); xErr != nil {
			return nil, xErr
		}
	}

	if e == nil {
		sei.Close()
	} else {
		sei.offset++
	}

	return e, nil
}

func (sei *sqlEntityIterator) Close()         { sei.Results.Close() }
func (sei *sqlEntityIterator) Offset() uint64 { return sei.offset }
func (sei *sqlEntityIterator) Size() uint64   { return sei.size }
func (sei *sqlEntityIterator) Total() uint64  { return sei.total }

func (sqlBE *SQLBackend) Query(tx *Tx, query string, args ...any) ([][]any, *XRError) {
	results, xErr := Query(tx, query, args...)
	if xErr != nil {
		return nil, xErr
	}
	defer results.Close()

	data := ([][]any)(nil)
	/*
	   Ks := make([]string, len(results.colTypes))

	   for i, t := range results.colTypes {
	           Ks[i] = t.Kind().String()
	   }
	*/

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if data == nil {
			data = [][]any{}
		}
		// row == []*any
		r := make([]any, len(row))
		for i, d := range row {
			r[i] = d
			/*
			   k := Ks[i]
			   if k == "slice" {
			           r[i] = NotNilString(d)
			   } else if k == "int64" || k == "uint64" {
			           r[i] = NotNilInt(d)
			   } else {
			           log.Printf("%v", reflect.ValueOf(*d).Type().String())
			           log.Printf("%v", reflect.ValueOf(*d).Type().Kind().String())
			           log.Printf("Ks: %v", Ks)
			           log.Printf("i: %d", i)
			           panic("help")
			   }
			*/
		}
		data = append(data, r)
	}

	return data, nil
}

// GENERATE QUERY STUFF

// generateFilterCTE builds the "cte(...) AS (...), FilterMatches AS (...)"
// portion (everything *after* the "WITH RECURSIVE " keyword) of the query
// used to find entities matching the (OR'd, AND'd) filter expressions.
//
// Each top-level OR expression (arm) is tagged with a bit (1<<i). The seed
// of the recursive CTE aggregates those bits per matching eSID with
// BIT_OR, and the recursive (parent-walking) step carries the mask
// forward unchanged. A node reached via more than one child (each
// possibly carrying a different mask) is reconciled by the final
// "FilterMatches" step, which does a BIT_OR(mask) GROUP BY eSID over the
// whole (already small, since it's per-node not per-leaf) cte.
//
// This lets callers (see FiltersRelativeToAbstractMasked in info.go)
// later determine, for any given entity in the result, exactly which top-
// level OR expression(s) are responsible for it being there -- which is
// required to compute a correct (neither too broad nor too narrow) nested
// <COLLECTION>url filter for that entity.
//
// IMPORTANT (perf): the recursive member below must stay "UNION DISTINCT"
// (not "UNION ALL"). MySQL materializes each iteration of a recursive CTE
// as an (unindexed) temp table, and distinctness is what keeps the walk
// bounded to ~one pass per tree node. Since "mask" is now part of the
// tuple being compared, at most one row per (node, distinct-mask) survives
// each iteration -- still bounded (a node can only be reached via a
// handful of distinct arm combinations in practice) -- rather than one row
// per *matching leaf* underneath that node, which is what plain "UNION
// ALL" here would produce. The one-time, non-recursive arm-tagging seed
// (built via "UNION ALL" below) doesn't have this concern since it's a
// flat aggregation, not something that compounds across recursion depth.
func generateFilterCTE(reg *Registry, filters [][]*FilterExpr) (string, []interface{}) {
	query := ""
	args := []interface{}{}

	query += `cte(eSID,Type,ParentSID,XID,OrMask) AS (
    -- The seed: entities that directly match one (or more) of the OR
    -- expressions below, tagged with a bit-mask of which one(s).
    SELECT e.eSID,e.Type,e.ParentSID,e.XID,matched.OrMask
    FROM Entities AS e
    JOIN (
      SELECT tagged.eSID, BIT_OR(tagged.ArmMask) AS OrMask
      FROM ( -- start of the OR Filter groupings`
	firstOr := true
	for orIndex, OrFilters := range filters {
		if !firstOr {
			query += `
        UNION ALL -- Adding another OR`
		}
		firstOr = false
		armBit := fmt.Sprintf("%d", uint64(1)<<uint(orIndex))
		query += `
      -- start of one Filter AND grouping (expr1 AND expr2).
      -- Find all SIDs for the leaves for entities (SIDs) of interest,
      -- tagged with this OR arm's bit (` + armBit + `).
      SELECT list.eSID, ` + armBit + ` AS ArmMask FROM (
        SELECT count(*) as cnt,e2.eSID,e2.XID FROM Entities AS e1
        RIGHT JOIN (
          -- start of expr1 - below finds SearchNodes/SIDs of interest`
		firstAnd := true
		andCount := 0
		for _, filter := range OrFilters { // AndFilters
			propNameSearch := "PropName=?"
			filterPropName := filter.PropName

			if filter.PP.HasWild {
				// mysql format: column_name REGEXP 'pattern'
				propNameSearch = "PropName REGEXP ?"
				has := false

				// Convert wildcards into appropriate regexp
				if filter.Operator == FILTER_PRESENT ||
					filter.Operator == FILTER_ABSENT {
					// include .* at the end
					filterPropName, has = filter.PP.DBFilterExists()
				} else {
					filterPropName, has = filter.PP.DBFilter()
				}

				PanicIf(!has, "Must have *") // Sanity check
				// log.Printf("tx: %s fpn: %q", reg.Tx.uuid, filterPropName)
				// log.Printf("tx: %s fpn: %s", reg.Tx.uuid, filter.PP.Debug())
			} else {
				if filter.Operator == FILTER_PRESENT ||
					filter.Operator == FILTER_ABSENT {

					// convert from "=" to "LIKE"
					propNameSearch = "PropName LIKE ?"
					// include % at the end
					filterPropName = filter.PropName + "%"
				}
			}

			andCount++
			if !firstAnd {
				query += `
          UNION ALL`
			}
			firstAnd = false

			if filter.Operator == FILTER_PRESENT { // ?filter=xxx
				check := "(Abstract=? AND " + propNameSearch + " AND "

				args = append(args, reg.DbSID, filter.Abstract,
					filterPropName)
				check += "PropValue IS NOT NULL)"
				// We may match lots of attrs, but we only want
				// 1 to appear so the list.cnt check doesn't treat
				// 2+ rows at matching more than one filter expression
				check += " GROUP BY eSID" // " LIMIT 1"
				query += `
          (SELECT eSID,Type,XID,ParentSID FROM Props  -- FILTER_PRESENT
           WHERE RegSID=? AND ` + check + ")" // Need () for groupBy/limit

			} else if filter.Operator == FILTER_ABSENT { // ?filter=xxx=null
				// Look for non-existing prop
				args = append(args, reg.DbSID, filter.Abstract,
					filterPropName)

				query += `
          -- Entities that don't have the specified prop
          SELECT e.eSID,e.Type,e.XID,ParentSID FROM Entities AS e
          WHERE e.RegSID=? AND e.Abstract=? AND
            NOT EXISTS (SELECT 1 FROM Props WHERE
              RegSID=e.RegSID AND eSID=e.eSID AND ( ` +
					propNameSearch + `))`

			} else if filter.Operator == FILTER_EQUAL { // ?filter=xxx=zzz
				check := "(Abstract=? AND " + propNameSearch + " AND "

				args = append(args, reg.DbSID, filter.Abstract,
					filterPropName)
				value, wildcard := LikeWildcardIt(filter.Value)
				args = append(args, value)
				if !wildcard {
					// Strings:case-insensitive per spec; others:exact match
					args = append(args, value)
					check += "((PropType='string' AND PropValue " +
						FILTER_CI_COLLATE + "=?)" +
						" OR (PropType<>'string' AND PropValue=?))"
				} else {
					args = append(args, value)
					check += "((PropType<>'string' AND PropValue=?) " +
						" OR (PropType='string' AND PropValue " +
						FILTER_CI_COLLATE + " LIKE ?))"
				}
				check += ")"
				query += `
          SELECT eSID,Type,XID,ParentSID FROM Props
            WHERE RegSID=? AND ` + check

			} else if filter.Operator == FILTER_NOT_EQUAL { // ?filter=x!=z
				args = append(args, reg.DbSID, filter.Abstract,
					filterPropName)
				query += `
          -- Entities that don't have the specified prop
          SELECT e.eSID,e.Type,e.XID,e.ParentSID FROM Entities AS e
          WHERE e.RegSID=? AND e.Abstract=? AND
            NOT EXISTS (SELECT 1 FROM Props WHERE
              RegSID=e.RegSID AND eSID=e.eSID AND (` +
					propNameSearch + ` AND `

				value, wildcard := LikeWildcardIt(filter.Value)
				args = append(args, value)
				if !wildcard {
					// Strings:case-insensitive per spec;others:exact match
					args = append(args, value)
					query += "((PropType='string' AND PropValue " +
						FILTER_CI_COLLATE + "=?)" +
						" OR (PropType<>'string' AND PropValue=?))"
				} else {
					args = append(args, value)
					query += "((PropType<>'string' AND PropValue=?) " +
						" OR (PropType='string' AND PropValue " +
						FILTER_CI_COLLATE + " LIKE ?))"
				}
				query += "))"

			} else if filter.Operator == FILTER_LESS ||
				filter.Operator == FILTER_LESS_EQUAL ||
				filter.Operator == FILTER_GREATER ||
				filter.Operator == FILTER_GREATER_EQUAL { // ?filter=x<z etc

				var sqlOp string
				switch filter.Operator {
				case FILTER_LESS:
					sqlOp = "<"
				case FILTER_LESS_EQUAL:
					sqlOp = "<="
				case FILTER_GREATER:
					sqlOp = ">"
				case FILTER_GREATER_EQUAL:
					sqlOp = ">="
				}

				check := "(Abstract=? AND " + propNameSearch + " AND "
				args = append(args, reg.DbSID, filter.Abstract,
					filterPropName)

				// Numeric: numeric comparison
				// String and others: case-insensitive string comparison
				args = append(args, filter.Value, filter.Value)
				check += "(CASE WHEN PropType IN ('integer','decimal','uinteger')" +
					" THEN CAST(PropValue AS DECIMAL) " + sqlOp + " CAST(? AS DECIMAL)" +
					" ELSE PropValue " + FILTER_CI_COLLATE + " " + sqlOp + " ? END))"
				query += `
          SELECT eSID,Type,XID,ParentSID FROM Props
            WHERE RegSID=? AND ` + check

			} else {
				PanicIf(true, "Bad filter.op: %#v", filter)
			}
		} // end of AndFilter
		query += `
          -- end of expr1
        ) AS result ON ( result.eSID=e1.eSID )
        -- For each result found, find all Leaves under the matching entity.
        -- The Leaves that show up 'cnt' times, where cnt is the # of
        -- expressions in each filter (the ANDs), are branches to return.
        -- Note we return the XID of each Leaf, not the XID of the matching
        -- entity. The entity that matches isn't important.
        JOIN Entities AS e2 ON (
          (
            (
              -- Non-meta objects, just compare the XID
              result.Type<>` + StrTypes(ENTITY_META) + ` AND
              ( result.XID = '/' OR
                e2.XID=result.XID OR
                e2.XID LIKE CONCAT(result.XID,'/%')
              )
            )
            OR
            (
              -- For 'meta' objects, compare it's parent's XID
              result.Type=` + StrTypes(ENTITY_META) + ` AND
              (e2.eSID = result.ParentSID OR e2.ParentSID = result.ParentSID)
              -- ( e2.XID=TRIM(TRAILING '/meta' FROM result.XID) OR
                -- e2.XID LIKE CONCAT(TRIM(TRAILING 'meta' FROM result.XID),'%')
              -- )
            )
          )
          AND e2.eSID IN (SELECT * from Leaves)
        ) GROUP BY e2.eSID
        -- end of RIGHT JOIN
      ) as list
      WHERE list.cnt=?   -- cnt is the # of operands in the AND filter
      -- end of one Filter AND grouping (expr1 AND expr2 ...)`
		args = append(args, andCount)
	} // end of OrFilter

	query += `
      ) AS tagged -- end of all OR Filter groupings
      GROUP BY tagged.eSID
    ) AS matched ON e.eSID = matched.eSID

    -- This is the recursive part of the query.
    -- Find all of the parents (and 'meta' sub-objects) of the found
    -- entities, up to root of Reg, carrying the OrMask along the way.
    -- NOTE: this must stay "UNION DISTINCT" (see the perf comment on
    -- generateFilterCTE above) -- do not change to "UNION ALL".
    UNION DISTINCT SELECT
      e.eSID,e.Type,e.ParentSID,e.XID,cte.OrMask
    FROM Entities AS e
    INNER JOIN cte ON
      (
        -- Find its parent
        e.eSID=cte.ParentSID
        OR
        -- If this is a Resource, grab its 'meta' sub-object
        ( cte.Type=` + StrTypes(ENTITY_RESOURCE) + ` AND
          e.Type=` + StrTypes(ENTITY_META) + ` AND
          e.ParentSID=cte.eSID
        )
      )
  ),
  -- A node can be reached via multiple children, potentially carrying
  -- different OrMask values; reconcile those here with one final BIT_OR.
  FilterMatches AS (
    SELECT eSID, BIT_OR(OrMask) AS OrMask FROM cte GROUP BY eSID
  )`

	return query, args
}

// topLevelSlashCount returns, as a string, the SUBSTRING_INDEX() count
// needed to truncate any row's XID (top-level entity's own, or one of
// its descendants') down to just its top-level parent's XID, given the
// XIDs[0] prefix used for a "Coll" query. E.g. XIDs[0]=="/dirs" means
// direct children look like "/dirs/d1" (3 "/"-delimited pieces), so
// SUBSTRING_INDEX(anyDescendantXID, '/', 3) always yields "/dirs/d1" no
// matter how deeply nested the descendant is.
//
// Shared by both the "?sort=" substring_index join and the pagination
// DENSE_RANK() window function below - both need to identify, for any
// row, its top-level parent's own XID.
func topLevelSlashCount(xid string) string {
	count := strings.Count(xid, "/")
	if count == 1 { // e.g /dirs
		return "3" // want substring below as: /dirs/d1
	} else if count == 3 { // e.g. /dirs/d1/files
		return "5" // want substring below as: /dirs/d1/files/f1
	} else { // e.g. /dirs/d1/files/f1/versions
		return "7" // want substring below as: /dirs/d1/files/f1/versions/v1
	}
}

// sortKey = attribute name, -NAME means descending, no "-" means ascending
func GenerateQuery(reg *Registry, what string, XIDs []string) (string, []interface{}, *XRError) {
	query := ""
	args := []any{}

	info := reg.GetRequestInfo()
	filters := info.Filters
	docView := info.DoDocView()
	sortKey := info.SortKey
	limit := info.Limit
	offset := info.Offset

	// ?filter=excludeall returns nothing
	if len(filters) == 1 && filters[0][0].Path == "excludeall"+string(DB_IN) {
		if what != "Coll" {
			return "", nil, NewXRError("bad_filter",
				info.OriginalRequest.URL.RequestURI(),
				"value=excludeall",
				"error_detail=\"excludeall\" must only be used "+
					"on collections")
		}
		return "", nil, nil
	}

	if sortKey != "" && what != "Coll" {
		return "", nil, NewXRError("bad_sort", "",
			"value="+sortKey,
			"error_detail=can't sort on a non-collection results")
	}

	if limit != 0 && what != "Coll" {
		return "", nil, NewXRError("pagination_noncollection", "")
	}

	ascDesc := "ASC"
	sortJoin := ""
	sortOrder := ""

	if sortKey != "" {
		if sortKey[0] == '-' {
			ascDesc = "DESC"
			sortKey = sortKey[1:]
		}

		// We only need to look at the first XID because if there's more
		// than one then they should all be for the same level
		slashCount := topLevelSlashCount(XIDs[0])

		/*
					sortOrder = `
			    sj.PropValue IS NOT NULL,
			    CASE WHEN sj.PropType IN ('integer','decimal','uinteger')
			      THEN CAST(sj.PropValue AS DECIMAL) END ` + ascDesc + `,
			    CASE WHEN sj.PropType NOT IN ('integer','decimal','uinteger')
			      THEN sj.PropValue END ` + ascDesc + `,
			`
		*/

		sortOrder = `
    sj.PropValue IS NOT NULL ` + ascDesc + `,
    CASE
      WHEN sj.PropType IN ('integer','decimal','uinteger') THEN
        IFNULL(CAST(sj.PropValue AS DECIMAL), 0)
      WHEN sj.PropType NOT IN ('integer','decimal','uinteger') THEN
        sj.PropValue
      ELSE NULL
    END COLLATE utf8mb4_general_ci ` + ascDesc + `,
`

		sortJoin = `
  LEFT JOIN Props AS sj ON (
    sj.RegSID = ft.RegSID AND
    sj.XID = substring_index(ft.XID, '/', ` + slashCount + `) AND
    sj.PropName = '` + sortKey + `')
`
	}

	// A cap of 64 is imposed because each top-level OR expression is
	// tagged with a bit (1<<i) so we can BIT_OR them together while
	// walking the tree, and track (per entity) which OR expression(s)
	// actually caused it to be included in the result set. This is what
	// lets us compute an accurate (not overly-broad, not overly-narrow)
	// nested <COLLECTION>url filter for each entity in the result.
	if len(filters) > 64 {
		return "", nil, NewXRError("bad_filter",
			info.OriginalRequest.URL.RequestURI(),
			"value=filter",
			"error_detail=too many OR'd filter expressions "+
				fmt.Sprintf("(%d), the max supported is 64", len(filters)))
	}

	// filterCTE/filterArgs hold the "WITH RECURSIVE cte(...) AS (...),
	// FilterMatches AS (...)" definition (sans the leading "WITH
	// RECURSIVE" keyword) that's prepended to the whole query when
	// filters are in use. FilterMatches maps each matching eSID (and all
	// of its ancestors, and the eSID's the 'meta' sub-objects) to a
	// bit-mask (BIT_OR'd) of which top-level OR expression(s) are
	// responsible for it being in the result. See FiltersRelativeToAbstractMasked
	// in info.go for how this mask is later used to compute the correct
	// nested <COLLECTION>url filter for each entity.
	filterCTE, filterArgs := "", []interface{}(nil)
	if len(filters) != 0 {
		filterCTE, filterArgs = generateFilterCTE(reg, filters)
	}

	maskCol := "'0' AS FilterMask"
	filterJoin := ""
	if len(filters) != 0 {
		// CAST to CHAR so the Go driver always hands us a consistent,
		// full-range (up to 64-bit) string regardless of how it would
		// otherwise represent a BIGINT UNSIGNED.
		maskCol = "CAST(IFNULL(fm.OrMask,0) AS CHAR) AS FilterMask"
		filterJoin = " LEFT JOIN FilterMatches AS fm ON fm.eSID=ft.eSID"
	}

	// When pagination is active (limit != 0, only ever true when
	// what=="Coll") we need two extra columns computed directly
	// alongside the rest of this SELECT's other per-row columns (same
	// level as the "ft"/"sj" aliases, so they can reference them):
	//   - LowerXID: re-exposed so the wrapping queries below (which no
	//     longer have access to the "ft" alias) can re-apply the same
	//     tiebreak ordering used today.
	//   - TopIdx: a DENSE_RANK() window function that assigns every row
	//     (top-level entity or descendant) its top-level ancestor's
	//     page index, all in one pass with no extra join needed to
	//     "propagate" the index down to child rows. This works because
	//     rows are ordered (both here and via the existing non-paginated
	//     ORDER BY below) such that every entity's descendant rows are
	//     always physically contiguous with it - true even when ?sort=
	//     is in play, since every row in a subtree resolves to the same
	//     sort value (that of its top-level ancestor, via "sj" above).
	//     DENSE_RANK() ties rows sharing the same (sort value, ancestor
	//     XID) tuple to the same rank, and gives the next *distinct*
	//     tuple (i.e. entity) the next rank - exactly the page index we
	//     want, computed using the exact same ORDER BY terms (sortOrder)
	//     as the plain, non-paginated query below, just replacing the
	//     final "ft.LowerXID ASC" per-row tiebreak with the coarser
	//     "top-level ancestor's XID" tiebreak so ties are ancestor-wide.
	paginate := limit != 0
	rankCol := "" // used for ordering results
	totalCountCol := ""
	if paginate {
		slashCount := topLevelSlashCount(XIDs[0])
		rankOrder := sortOrder +
			`    SUBSTRING_INDEX(ft.XID, '/', ` + slashCount + `) ASC`
		rankCol = `,ft.LowerXID AS LowerXID,
  DENSE_RANK() OVER (ORDER BY ` + rankOrder + `) AS TopIdx`
	} else {
		// Not paginating - "TotalCount" is computed for real (via the
		// win1/win2 wrapping below) only when paginate==true. Project a
		// cheap literal here instead of just omitting the column, so
		// GenerateQuery() always returns the same column shape either
		// way - this avoids the column count ever silently drifting out
		// of sync between the "paginate" and "!paginate" code paths as
		// more columns get added over time. This is a constant literal,
		// not a computed aggregate, so it costs nothing extra to compute
		// (no extra join/derived table/window function). Use '-' rather
		// than '0' so any accidental future attempt to read/parse this
		// column as an int when !paginate fails loudly instead of
		// silently looking like a valid (zero) count.
		totalCountCol = `,'-' AS TotalCount`
	}

	args = []interface{}{}
	if len(filters) != 0 {
		query = "WITH RECURSIVE " + filterCTE + "\n"
		args = append(args, filterArgs...)
	}
	args = append(args, reg.DbSID)
	query += `
SELECT
  ft.RegSID,ft.Type,ft.Plural,ft.Singular,ft.ParentSID,ft.eSID,ft.UID,ft.Abstract,ft.XID,ft.PropName,ft.PropValue,ft.PropType,ft.IsSystemProp,` + maskCol + rankCol + totalCountCol + `
  FROM Props AS ft` + filterJoin + sortJoin + `
  WHERE ft.RegSID=?
`

	// Exclude generated attributes/entities if 'doc view' is turned on.
	// Meaning, only grab Props that have 'DocView' set to 'true'. These
	// should be (mainly) just the ones we set explicitly.
	if docView {
		query += `  AND ft.DocView=true
`
	}

	// Remove entities that are higher than the GET PATH specified
	if what != "Registry" && len(XIDs) > 0 {
		query += "  AND ("
		for i, p := range XIDs {
			if i > 0 {
				query += " OR "
			}
			query += "ft.XID=? OR ft.XID LIKE ?"
			args = append(args, p, p+"/%")
		}
		query += ")\n"

	}

	if len(filters) != 0 {
		// The actual eSID-matching/tree-walking logic lives in the
		// "FilterMatches" CTE built by generateFilterCTE() above (joined
		// in via "fm" in the FROM clause). All we need here is to
		// restrict the result to entities that showed up in it.
		query += `  AND fm.eSID IS NOT NULL
`
	}

	if !paginate {
		query += `  ORDER BY ` + sortOrder +
			`    ft.LowerXID ASC;`
	} else {
		// Wrap the query above (which already computed TopIdx for
		// every row) twice more:
		//   - win1 (inner wrap): adds TotalCount, a whole-partition
		//     (no ORDER BY, no WHERE yet applied) aggregate of the
		//     highest TopIdx seen - i.e. the total number of top-level
		//     entities in the *entire* (filtered, but not paginated)
		//     result set. This MUST happen in its own layer, before the
		//     page-range WHERE below is applied, otherwise it would
		//     only reflect the count within the current page.
		//   - outermost: restricts to the requested page
		//     (TopIdx BETWEEN offset+1 AND offset+limit) and re-applies
		//     the original per-row tiebreak order (now via the plain
		//     TopIdx/LowerXID columns, since the "ft"/"sj" aliases are
		//     no longer in scope this far out).
		innerQuery := query
		query = `SELECT RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,PropName,PropValue,PropType,IsSystemProp,FilterMask,TotalCount
  FROM (
    SELECT win1.*, CAST(MAX(win1.TopIdx) OVER () AS CHAR) AS TotalCount
    FROM (
` + innerQuery + `
    ) AS win1
  ) AS win2
  WHERE win2.TopIdx BETWEEN ? AND ?
  ORDER BY win2.TopIdx ASC, win2.LowerXID ASC;`
		args = append(args, offset+1, offset+limit)
	}

	if log.GetLevel() > 3 || log.HasVerbose("genq") {
		log.Printf("tx: %s Query:\n%s\n\n", reg.Tx.uuid, SubQuery(query, args))
	}
	return query, args, nil
}

// Convert each non-escaped * into SQL % for LIKE queries
func LikeWildcardIt(str string) (string, bool) {
	wild := false
	res := strings.Builder{}

	prevch := '\000'
	for _, ch := range str {
		if ch == '*' && prevch != '\\' {
			res.WriteRune('%')
			wild = true
		} else {
			res.WriteRune(ch)
		}
		prevch = ch
	}

	return res.String(), wild
}

// Create a new Entity based on what's in the DB. Similar to Refresh()
func readNextEntity(tx *Tx, results *Result, accessMode int) (*Entity, *XRError) {
	entity := (*Entity)(nil)

	// RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,
	// PropName,PropValue,PropType,IsSystemProp,FilterMask
	//   0     1     2       3         4      5   6     7      8
	//     9        10         11         12        13
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		// log.Printf("tx: %s Row(%d): %#v", tx.uuid, len(row), row)
		if log.HasVerbose("readNextEntity") {
			str := "("
			for _, c := range row {
				if IsNil(c) || IsNil(*c) {
					str += "nil,"
				} else {
					str += fmt.Sprintf("%s,", *c)
				}
			}
			log.Printf("tx: %s Row: %s)", tx.uuid, str)
		}
		eType := int((*row[1]).(int64))
		plural := NotNilString(row[2])
		uid := NotNilString(row[6])

		if entity == nil {
			entity = &Entity{
				EntityExtensions: EntityExtensions{},

				Tx:         tx,
				AccessMode: accessMode,
				Registry:   tx.Registry,
				DbSID:      NotNilString(row[5]),
				ParentSID:  NotNilString(row[4]),
				Plural:     plural,
				Singular:   NotNilString(row[3]),
				UID:        uid,

				Type:     eType,
				XID:      NotNilString(row[8]),
				Abstract: NotNilString(row[7]),
			}

			entity.GroupModel, entity.ResourceModel =
				AbstractToModels(tx.Registry, entity.Abstract)

			// FilterMask is only meaningful/present when the query
			// applied a ?filter=. It tells us which top-level OR
			// filter expression(s) actually caused this entity to be
			// in the result set, which is needed to compute a correct
			// nested <COLLECTION>url filter for it. Stored as generic
			// "stuff" rather than a dedicated struct field.
			if len(row) > 13 {
				if maskStr := NotNilString(row[13]); maskStr != "" {
					mask, err := strconv.ParseUint(maskStr, 10, 64)
					if err != nil {
						// FilterMask is always the result of
						// CAST(...AS CHAR) on a BIGINT UNSIGNED (or
						// the literal '0') on the SQL side, so this
						// should never fail - if it does, something
						// is seriously wrong
						panic(fmt.Sprintf(
							"error parsing FilterMask: %q: %s",
							maskStr, err))
					}
					if mask != 0 {
						entity.SetStuff("filterMask", mask)
					}
				}
			}
		} else {
			// If the next row isn't part of the current Entity then
			// push it back into the result set so we'll grab it the next time
			// we're called. And exit.
			if entity.Type != eType || entity.Plural != plural || entity.UID != uid {
				results.Push()
				break
			}
		}

		propName := NotNilString(row[9])
		propVal := NotNilString(row[10])
		propType := NotNilString(row[11])
		isSystemProp := NotNilBoolDef(row[12], false)

		// Edge case - no props but entity is there
		if propName == "" && propVal == "" && propType == "" {
			continue
		}

		var xErr *XRError
		if isSystemProp {
			xErr = entity.SetSystemFromDBName(propName, &propVal, propType)
		} else {
			xErr = entity.SetFromDBName(propName, &propVal, propType)
		}
		if xErr != nil {
			return nil, xErr
		}
	}

	return entity, nil
}

func (sqlBE *SQLBackend) SaveEntity(e *Entity) *XRError {
	// old - make a dup so we can delete some attributes
	newObj := e.NewObject // maps.Clone(e.NewObject)

	// Delete all user props for this entity, we assume that NewObject
	// contains everything we want going forward
	xErr := sqlBE.ClearUserProps(e)
	if xErr != nil {
		return xErr
	}

	resSingular := ""
	resHasDoc := false
	if rm := e.GetResourceModel(); rm != nil {
		resSingular = rm.Singular
		resHasDoc = rm.GetHasDocument()
	}

	var traverse func(pp *PropPath, val any) *XRError
	traverse = func(pp *PropPath, val any) *XRError {
		if IsNil(val) { // Skip empty attributes
			return nil
		}

		valValue := reflect.ValueOf(val)

		switch valValue.Kind() {
		case reflect.Map:
			keys := valValue.MapKeys()
			count := 0
			for _, keyValue := range keys {
				if keyValue.Kind() != reflect.String {
					return NewXRError("invalid_attribute", e.XID,
						"name="+pp.RemoveLast().UI(),
						"error_detail="+
							fmt.Sprintf("map key (%s) needs to be a string, "+
								"not %s", pp.Last().Text,
								keyValue.Kind().String()))
				}

				k := keyValue.Interface().(string)
				v := valValue.MapIndex(keyValue).Interface()
				// "RESOURCE" is special - call SetDBProp if it's present
				if resHasDoc && pp.Len() == 0 && k == resSingular {
					if xErr := setDBPropertyBatch(e, pp.P(k), v); xErr != nil {
						return xErr
					}
				} else if k[0] == '#' {
					if xErr := setDBPropertyBatch(e, pp.P(k), v); xErr != nil {
						return xErr
					}
				} else {
					if IsNil(v) {
						continue
					}
					if xErr := traverse(pp.P(k), v); xErr != nil {
						return xErr
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return setDBPropertyBatch(e, pp, map[string]any{})
			}

		case reflect.Slice:
			if valValue.Len() == 0 {
				valValue = reflect.MakeSlice(reflect.TypeOf(val), 0, 0)
				return setDBPropertyBatch(e, pp, valValue.Interface())
			}
			for i := 0; i < valValue.Len(); i++ {
				v := valValue.Index(i).Interface()
				if xErr := traverse(pp.I(i), v); xErr != nil {
					return xErr
				}
			}

		case reflect.Struct:
			panic("a struct")
			// If this is ever needed, use reflect to traverse into val
			// like we do for map & slice above. The stuff below is old/wrong
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				if IsNil(v) {
					continue
				}
				if xErr := traverse(pp.P(k), v); xErr != nil {
					return xErr
				}
				count++
			}
			if count == 0 {
				return setDBPropertyBatch(e, pp, struct{}{})
			}

		default:
			// must be scalar so add it
			return setDBPropertyBatch(e, pp, val)
		}
		return nil
	}

	xErr = traverse(NewPP(), newObj)
	if xErr != nil {
		return xErr
	}

	// Flush all buffered own-property rows from the traversal above as
	// one (or a few, if chunked) multi-row REPLACE INTO, instead of the
	// one-row-per-property writes SetDBProperty() would have done.
	xErr = doDBPropertyBatch(e)
	if xErr != nil {
		return xErr
	}

	return nil
}

// dbPropBatchChunkSize caps how many rows go into a single REPLACE
// INTO statement in DoDBPropertyBatch(), purely as a max_allowed_packet
// safety net - not expected to matter for realistic per-entity
// attribute counts.
const dbPropBatchChunkSize = 200

// dbPropRow holds the per-property info needed to write (or delete) an
// own-property Props row - see prepDBProperty()/
// SetDBProperty()/SetDBPropertyBatch()/DoDBPropertyBatch() below. Kept
// as a struct (rather than individual values) so it's easy to add more
// per-row info later without reshuffling every call site. Value==nil
// means "delete this row".
type dbPropRow struct {
	Name    string  // DB PropName (includes trailing DB_IN)
	Value   *string // nil = delete
	Type    string  // PropType (string, boolean, int, ...)
	DocView bool
}

// SetDBPropertyBatch is like SetDBProperty() but, instead of writing
// the property's Props row immediately, buffers it into
// e.dbPropBatch for a later single multi-row REPLACE INTO via
// DoDBPropertyBatch() - see Save()'s traversal loop, the only caller.
// The nil/delete case is a no-op here: Save() already issues one
// blanket DELETE for all of this entity's own-prop rows before
// traversal starts, so there's never a per-prop delete to buffer.
func setDBPropertyBatch(e *Entity, pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.uuid, pp, val)()

	row, skip, xErr := prepDBProperty(e, pp, val)
	if xErr != nil {
		return xErr
	}
	if skip || row.Value == nil {
		return nil
	}

	// dbPropBatch buffers own-property Props row info during
	// Save()'s traversal (see SetDBPropertyBatch()/DoDBPropertyBatch()
	// below) so they can be written as a single multi-row REPLACE INTO
	// instead of one round trip per property.
	dbPropBatch := []dbPropRow(nil)

	tmp, ok := e.GetStuff("dbPropBatch")
	if !ok || IsNil(tmp) {
		dbPropBatch = []dbPropRow{}
	} else {
		dbPropBatch = tmp.([]dbPropRow)
	}

	dbPropBatch = append(dbPropBatch, row)
	e.SetStuff("dbPropBatch", dbPropBatch)

	return nil
}

// This will save a single property/value in the DB. This assumes
// the caller is traversing the Object and splitting it into individual props
// prepDBProperty validates and converts val into the form needed to
// write (or delete) an own-property Props row for pp, shared by
// SetDBProperty() (immediate single-row write) and
// SetDBPropertyBatch() (buffered, for Save()'s traversal loop - see
// DoDBPropertyBatch()). Returns skip=true if nothing further needs to
// be written (e.g. dontStore props, or the RESOURCE-content special
// case, which is written directly to ResourceContents here and never
// touches Props).
func prepDBProperty(e *Entity, pp *PropPath, val any) (row dbPropRow,
	skip bool, xErr *XRError) {

	PanicIf(pp.UI() == "", "pp is empty")

	row.Name = pp.DB()
	row.DocView = true

	if len(row.Name) > MAX_PROPNAME {
		return dbPropRow{}, false, NewXRError("invalid_attribute",
			e.XID, "name="+row.Name,
			"error_detail="+
				fmt.Sprintf("attribute names must not exceed %d chars",
					MAX_PROPNAME))
	}

	_, propsMap := e.GetPropsOrdered()
	specProp, ok := propsMap[pp.Top()]
	if ok && specProp.internals != nil {
		// Any prop with "dontStore"=true we skip
		if specProp.internals.dontStore {
			return dbPropRow{}, true, nil
		}
		if specProp.internals.noDocView {
			row.DocView = false
		}
	}

	PanicIf(e.DbSID == "", "DbSID should not be empty")
	PanicIf(e.Registry == nil, "Registry should not be nil")

	// "RESOURCE" is special and is saved in it's own table
	// Need to explicitly set "RESOURCE" to nil to delete it.
	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			if IsNil(val) {
				// Remove the content
				xErr := e.Tx.Backend.SetResourceContents(e, nil)
				if xErr != nil {
					return dbPropRow{}, false, xErr
				}
			} else {
				// Update the content
				var buf []byte

				if by, ok := val.([]byte); ok {
					buf = by
				} else if str, ok := val.(string); ok {
					buf = []byte(str)
				} else {
					buf = []byte(fmt.Sprintf("%v", val))
				}

				xErr := e.Tx.Backend.SetResourceContents(e, buf)
				if xErr != nil {
					return dbPropRow{}, false, xErr
				}

				PanicIf(IsNil(e.NewObject["#contentid"]), "Missing cid")

				// Don't save "RESOURCE" in the DB, #contentid is good enough
				return dbPropRow{}, true, nil
			}
		}
	}

	// Convert specDefined BOOLEAN value "false" to "nil" so it doesn't
	// appear in the DB at all. If this is too broad then just do it for
	// "defaultversionsticky" in resources.go as we're copying attributes.
	if !IsNil(specProp) && val == false && GoToOurType(val) == BOOLEAN {
		// val = nil
	}

	if IsNil(val) {
		// Should never need this but keeping it just in case
		return row, false, nil
	}

	row.Type = GoToOurType(val)
	// Convert booleans to true/false instead of 1/0 so filter works
	// ...=true and not ...=1
	dbVal := val
	if row.Type == BOOLEAN {
		if val == true {
			dbVal = "true"
		} else {
			dbVal = "false"
		}
	}

	// row.Type already came from GoToOurType(val) above, which panics on
	// any type outside its known set - so val's concrete type here is
	// guaranteed and we can type-assert directly instead of using reflect.
	switch row.Type {
	case STRING:
		if s := val.(string); len(s) > MAX_VARCHAR {
			return dbPropRow{}, false, NewXRError("invalid_attribute",
				e.XID, "name="+pp.UI(),
				"error_detail="+
					fmt.Sprintf("must be less than %d chars",
						MAX_VARCHAR+1))
		}
	case ARRAY:
		if a := val.([]any); len(a) > 0 {
			return dbPropRow{}, false, NewXRError("invalid_attribute",
				e.XID, "name="+pp.UI(),
				"error_detail=can't set non-empty arrays")
		}
		dbVal = ""
	case MAP:
		if m := val.(map[string]any); len(m) > 0 {
			return dbPropRow{}, false, NewXRError("invalid_attribute",
				e.XID, "name= "+pp.UI(),
				"error_detail=can't set non-empty maps")
		}
		dbVal = ""
	case OBJECT:
		// GoToOurType() only ever maps OBJECT from the literal struct{}{}
		// sentinel (never a populated struct), so this is always empty.
		dbVal = ""
	}

	dbValStr := fmt.Sprintf("%v", dbVal)
	row.Value = &dbValStr
	return row, false, nil
}

// DoDBPropertyBatch flushes whatever SetDBPropertyBatch() has buffered
// into e.dbPropBatch (if anything) as one or more multi-row
// REPLACE INTO Props statements (see DBWritePropsBatch()
// in fulltree.go), then clears the buffer. Called once by Save(),
// right after its traversal loop finishes.
func doDBPropertyBatch(e *Entity) *XRError {
	tmp, ok := e.GetStuff("dbPropBatch")
	if !ok || IsNil(tmp) {
		return nil
	}
	rows := tmp.([]dbPropRow)

	if len(rows) == 0 {
		return nil
	}
	e.SetStuff("dbPropBatch", nil)

	return dbWritePropsBatch(e, rows, false)
}

// DBWritePropsBatch writes multiple own/system Props rows
// for e in as few REPLACE INTO statements as possible, chunked at
// dbPropBatchChunkSize rows/statement as a max_allowed_packet safety
// net. isSystem marks whether these rows are system-managed
// (SaveSystemProps) or plain user-set (SetDBPropertyBatch/
// DoDBPropertyBatch) - same meaning as DBWriteProp's isSystem
// param, just batched across multiple rows in one statement instead of
// one statement per row.
func dbWritePropsBatch(e *Entity, rows []dbPropRow, isSystem bool) *XRError {
	if len(rows) == 0 {
		return nil
	}

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	pos := 0
	size := len(rows)
	args := make([]any, 0, 13*max(size, dbPropBatchChunkSize))

	for pos < size {
		end := min(pos+dbPropBatchChunkSize, size)
		args = args[:0]

		for ; pos < end; pos++ {
			row := rows[pos]
			args = append(args,
				e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
				e.DbSID, e.UID, e.XID,
				row.Name, *row.Value, row.Type, e.Abstract, row.DocView)
		}

		xErr := batchUpdateProps(e, isSystem, args)
		if xErr != nil {
			return xErr
		}
	}

	return nil
}

func (sqlBE *SQLBackend) SaveChangedSystemProps(e *Entity, changed map[string]any) *XRError {

	_, propsMap := e.GetPropsOrdered()

	// "name" here is the plain top-level attribute name (matching the
	// key scheme used in e.System/e.NewSystem) - convert to the
	// trailing-DB_IN-terminated DB PropName via pp.DB() before writing.
	insertRows := make([]dbPropRow, 0, len(changed))
	deleteNames := make([]string, 0, len(changed))

	for name, val := range changed {
		docView := true
		if specProp, ok := propsMap[name]; ok && specProp.internals != nil &&
			specProp.internals.noDocView {
			docView = false
		}
		dbName := NewPPP(name).DB()

		if IsNil(val) {
			deleteNames = append(deleteNames, dbName)
			continue
		}

		propType := GoToOurType(val)
		dbVal := val
		if propType == BOOLEAN {
			if val == true {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		}

		// propType came from GoToOurType(val), so no need to re-derive
		// the Go kind via reflect - just check our own type constant.
		if propType == ARRAY || propType == MAP || propType == OBJECT {
			dbVal = ""
		}

		dbValStr := fmt.Sprintf("%v", dbVal)
		insertRows = append(insertRows, dbPropRow{
			Name: dbName, Value: &dbValStr, Type: propType, DocView: docView,
		})
	}

	if xErr := batchDeleteProps(e, deleteNames); xErr != nil {
		return xErr
	}

	if xErr := dbWritePropsBatch(e, insertRows, true); xErr != nil {
		return xErr
	}

	return nil
}
