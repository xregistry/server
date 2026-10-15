package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
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
	Do(reg.Tx, `
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
	return nil
}

func (sqlBE *SQLBackend) ListRegistries(tx *Tx) ([]string, *XRError) {
	// Order by UID, ASC - case insensitive
	results := Query(tx, `SELECT UID FROM Registries ORDER BY UID ASC`)
	defer results.Close()

	res := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		res = append(res, NotNilString(row[0]))
	}

	return res, nil
}

func (sqlBE *SQLBackend) DeleteRegistry(r *Registry) *XRError {
	DoOne(r.Tx, `DELETE FROM Registries WHERE SID=?`, r.DbSID)

	return nil
}

func (sqlBE *SQLBackend) MapRegistryUID2SID(tx *Tx, uid string) (string, *XRError) {
	results := Query(tx, `SELECT SID FROM Registries WHERE UID=?`, uid)
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
	results := Query(r.Tx, `SELECT UsesXref FROM Registries WHERE SID=?`,
		r.DbSID)
	defer results.Close()

	if row := results.NextRow(); row != nil {
		return NotNilBoolDef(row[0], false), nil
	}

	panic("can't find registry: " + r.DbSID)
	return false, nil
}

func (sqlBE *SQLBackend) RegistrySetUsesXref(r *Registry, b bool) *XRError {
	DoZeroOne(r.Tx,
		`UPDATE Registries SET UsesXref=? WHERE SID=? AND UsesXref<>?`,
		b, r.Tx.Registry.DbSID, b)

	return nil
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
	entities, xErr := sqlBE.GetEntitiesFromQuery(r.Tx, r.DbSID, accessMode,
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

func (sqlBE *SQLBackend) DeleteGroup(g *Group) *XRError {
	DoOne(g.Tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)

	return nil
}

func (sqlBE *SQLBackend) HasReadOnlyResource(g *Group) (bool, *XRError) {
	// Make sure we don't have any readonly Resources. Callers (HTTPDelete/
	// HTTPDeleteGroups) already lock g itself FOR_WRITE, but that doesn't
	// lock the descendant Resources/Metas whose readonly Props this scans -
	// without FOR UPDATE here, a concurrent Tx setting readonly=true on a
	// descendant after our RR snapshot was established could be missed,
	// letting this Group be improperly deleted.
	results := Query(g.Tx, `
        SELECT EXISTS(SELECT 1 FROM Props
        WHERE RegSID=? AND Type=`+StrTypes(ENTITY_META)+` AND
          XID LIKE '`+g.XID+`/%' AND
          PropName='readonly`+string(DB_IN)+`' AND
          PropValue='true') FOR UPDATE`,
		g.Registry.DbSID)
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

	results := Query(g.Tx, query,
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
		enumArgs = append(enumArgs, EnumValueToDBString(v))
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

	results := Query(g.Tx, query, args...)
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
	entities, xErr := sqlBE.GetEntitiesFromQuery(g.Tx, g.Registry.DbSID, FOR_WRITE,
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

func (sqlBE *SQLBackend) RegisterEntity(e *Entity) *XRError {
	switch e.Type {
	case ENTITY_REGISTRY:
		// Add Registry and Model to the DB's tables

		DoOne(e.Tx, `INSERT INTO Registries(SID, UID) VALUES(?,?)`,
			e.DbSID, e.UID)

		DoOne(e.Tx, `INSERT INTO Models(RegistrySID) VALUES(?)`, e.DbSID)

	case ENTITY_GROUP:
		DoOne(e.Tx, `
            INSERT INTO "Groups"(
                SID, RegistrySID, UID,
                ModelSID, XID, Abstract,
                Plural, Singular)
            VALUES(?,?,?,?,?,?,?,?)`,

			e.DbSID, e.Registry.DbSID, e.UID,
			e.GroupModel.SID, e.XID, e.Abstract,
			e.Plural, e.Singular)

	case ENTITY_RESOURCE:
		DoOne(e.Tx, `
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

	case ENTITY_META:
		DoOne(e.Tx, `
            INSERT INTO Metas(SID, RegistrySID, ResourceSID,
                        XID, Abstract,
                        Plural, Singular)
            VALUES(?,?,?,?,?,?,?)`,

			e.DbSID, e.Registry.DbSID, e.ParentSID,
			e.XID, e.Abstract,
			e.ResourceModel.Plural, e.ResourceModel.Singular)

	case ENTITY_VERSION:
		DoOne(e.Tx, `
            INSERT INTO Versions(SID, UID, RegistrySID,
                ResourceSID, XID, Abstract)
            VALUES(?,?,?,?,?,?)`,

			e.DbSID, e.UID, e.Registry.DbSID, e.ParentSID,
			e.XID, e.Abstract)

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
	DoOne(e.Tx, `
        REPLACE INTO Entities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, XID, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?,?,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Abstract, e.XID)

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
	DoOne(e.Tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, true,
               false, false, false, true, false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.XID, "xid"+string(DB_IN), e.XID, e.Abstract)

	if e.Type == ENTITY_VERSION {
		// e is always a real, in-memory Version, which always has a
		// parent Resource, so e.ParentSID is never empty here. The
		// owning Resource is guaranteed to already exist (e was just
		// created as one of its Versions), so this always inserts
		// exactly 1 row.
		DoOne(e.Tx, `
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
		DoOne(e.Tx, `
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
	}

	return nil
}

func (sqlBE *SQLBackend) LoadModelJSON(r *Registry) (string, string, *XRError) {
	// Load Registry model
	results := Query(r.Tx,
		`SELECT Model,Changed FROM Models WHERE RegistrySID=?`,
		r.DbSID)
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
	DoZeroTwo(m.Registry.Tx, `
        INSERT INTO Models(RegistrySID, Model, Changed)
            VALUES(?,?,?)
            ON DUPLICATE KEY UPDATE Model=?, Changed=?`,

		m.Registry.DbSID, modelStr, changeUUID,
		modelStr, changeUUID)

	return nil
}

func (sqlBE *SQLBackend) RegisterModelEntity(me any) *XRError {
	if gm, ok := me.(*GroupModel); ok {
		DoOne(gm.Model.Registry.Tx,
			`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
			gm.SID, gm.Model.Registry.DbSID, nil,
			"/"+gm.Plural, gm.Plural, gm.Singular)
	} else if rm, ok := me.(*ResourceModel); ok {
		gm := rm.GroupModel
		gmAbs := "/" + gm.Plural

		DoOne(rm.GroupModel.Model.Registry.Tx,
			`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
			rm.SID, gm.Model.Registry.DbSID, gm.SID,
			gmAbs+"/"+rm.Plural, rm.Plural, rm.Singular)
	} else {
		panic("I don't know who I am")
	}

	return nil
}

func (sqlBE *SQLBackend) DeleteModelEnityByAbstract(reg *Registry, abstract string) *XRError {
	DoOne(reg.Tx, `DELETE FROM ModelEntities
        WHERE RegistrySID=? AND Abstract=?`,
		reg.DbSID, abstract)

	return nil
}

func (sqlBE *SQLBackend) GetModelEntitiesByAbstract(r *Registry) (map[string]string, *XRError) {
	// Load old model's ModelEntities so we can diff them
	existingModelEntities := map[string]string{} // Abstract->SID
	results := Query(r.Tx,
		`SELECT SID,Abstract FROM ModelEntities WHERE RegistrySID=?`,
		r.DbSID)
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
	results := Query(r.Tx,
		`SELECT COUNT(*) FROM "Groups" WHERE ModelSID=?`, gmSID)
	count := NotNilInt(results.NextRow()[0])
	results.Close()

	return count, nil
}

func (sqlBE *SQLBackend) CountResourceModelUsed(r *Registry, rmSID string) (int, *XRError) {
	results := Query(r.Tx,
		`SELECT COUNT(*) FROM Resources WHERE ModelSID=?`, rmSID)
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

	results := Query(r.Tx, query, rm.SID,
		names[0], names[1], names[2], names[3])
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

	results := Query(r.Tx, query, r.DbSID)
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

func (sqlBE *SQLBackend) ClearUserProps(e *Entity) *XRError {
	// Delete all user props for this entity, we assume that NewObject
	// contains everything we want going forward
	Do(e.Tx, `DELETE FROM Props
              WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                    AND IsXrefVerCopy=false AND IsSystemProp=false
                    AND IsCalcStatic=false AND IsCalcDynamic=false`,
		e.DbSID)

	return nil
}

func (sqlBE *SQLBackend) BatchUpdateProps(e *Entity, isSystem bool, args []any) *XRError {
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

	Do(e.Tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, 
            eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsSystemProp, IsCalcStatic, IsCalcDynamic)
        VALUES `+holders, args...)

	return nil
}

func (sqlBE *SQLBackend) BatchDeleteProps(e *Entity, args []any) *XRError {
	size := len(args)
	holders := strings.Repeat("?,", size-1)
	holders += "?"

	newArgs := append([]any{e.DbSID}, args...)

	Do(e.Tx, `
        DELETE FROM Props
        WHERE eSID=? AND PropName IN (`+holders+`)
            AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
            AND IsXrefVerCopy=false`, newArgs...)

	return nil
}

func (sqlBE *SQLBackend) DeleteResource(r *Resource) *XRError {
	// Any xref source's stale mirror is cleared by ResourcesTrigger
	// (init.sql), which fires for every deletion path (this, whole-
	// Group delete, whole-Registry delete) uniformly.
	DoOne(r.Tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)

	return nil
}

func (sqlBE *SQLBackend) GetResourceContents(e *Entity) ([]byte, *XRError) {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		contentID := e.Get("#contentid")

		results := Query(e.Tx, `
            SELECT Content FROM ResourceContents WHERE VersionSID=? `,
			contentID)
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
		Do(e.Tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
			e.DbSID)
	} else {
		// Update the content
		DoOneTwo(e.Tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
                VALUES(?,?)`, e.DbSID, val)
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
	Do(r.Tx, `
        UPDATE Props AS p
        JOIN Versions AS v ON (v.SID=p.eSID)
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        SET p.PropValue = IF(v.UID=m.defaultVID, 'true', 'false')
        WHERE v.ResourceSID=? AND p.PropName=?`,
		r.DbSID, "isdefault"+string(DB_IN))

	return nil
}

func (sqlBE *SQLBackend) DeleteResourceDefaultVersionProps(r *Resource) *XRError {
	Do(r.Tx, `DELETE FROM Props WHERE eSID=? AND IsDefaultVerCopy=true`,
		r.DbSID)
	return nil
}

func (sqlBE *SQLBackend) CopyResourceDefaultVersionProps(r *Resource) *XRError {
	// IsCalcDynamic isn't excluded here (unlike IsCalcStatic) so ver's
	// own "isdefault" row (just fixed up above) is mirrored into the
	// Resource too - like createdat/modifiedat, it's just copied
	// content, no special-casing needed. It's simply absent whenever
	// there's no default Version to copy from at all (see the ver ==
	// nil branch above).
	ver, xErr := r.GetDefault()
	Must(xErr)

	Do(r.Tx, `
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

	return nil
}

func (sqlBE *SQLBackend) FindBadAncestorsCreatedAt(r *Resource, lock bool) ([]*AncestorVersion, *XRError) {
	// Search the DB for all Versions of this Resource, sorted by 'createdat'
	// and return the ones that do not have the proper 'ancestorid' value.
	// Meaning, they don't point to the next oldest one (based on createdat)
	lockExpr := ""
	if lock {
		lockExpr = " FOR UPDATE"
	}

	results := Query(r.Tx, `
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

	results := Query(r.Tx, `
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

	results := Query(r.Tx, `
                SELECT UID, AncestorID FROM Versions
                WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
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

	results := Query(r.Tx, `
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
	results := Query(r.Tx, `
            SELECT UID, AncestorID, CreatedAt FROM Versions
            WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
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

	results := Query(r.Tx, `
            SELECT UID FROM Versions
            WHERE RegistrySID=? AND ResourceSID=? AND UID=AncestorID`+lockExpr,
		r.Registry.DbSID, r.DbSID)
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
	results := Query(r.Tx, `   
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
	results := Query(r.Tx, `   
            SELECT UID FROM Versions
            WHERE RegistrySID=? AND ResourceSID=? AND AncestorID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID, parentVID)
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
	results := Query(r.Tx, `
            SELECT COUNT(*) FROM Versions
            WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	row := results.NextRow()
	return NotNilInt(row[0]), nil
}

func (sqlBE *SQLBackend) GetVersions(r *Resource, accessMode int) ([]*Version, *XRError) {
	entities, xErr := sqlBE.GetEntitiesFromQuery(r.Tx, r.Registry.DbSID,
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
	results := Query(r.Tx, `
        SELECT res.XID
        FROM Metas AS m
        JOIN Resources AS res ON (res.SID=m.ResourceSID)
        WHERE m.RegistrySID=? AND m.xRefXID=?  FOR UPDATE`,
		r.Registry.DbSID, r.XID)
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

	results := Query(r.Tx, query, vAttrPath.DB(),
		r.Registry.DbSID, r.DbSID, ENTITY_VERSION)
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
	DoOne(meta.Tx, `DELETE FROM Metas WHERE SID=?`, meta.DbSID)

	return nil
}

func (sqlBE *SQLBackend) DeleteVersion(v *Version) *XRError {
	DoZeroOne(v.Tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)

	return nil
}

// Delete xref'd meta props, all Versions xref'd props (all?), xref'd Versions
func (sqlBE *SQLBackend) ClearXrefState(meta *Meta) *XRError {
	Do(meta.Tx, `DELETE FROM Props
        WHERE eSID=? AND IsXrefPropCopy=true`,
		meta.DbSID)
	Do(meta.Tx, `DELETE FROM Props
        WHERE ParentSID=? AND IsXrefVerCopy=true`,
		meta.ParentSID)
	Do(meta.Tx, `DELETE FROM Entities
        WHERE ParentSID=? AND IsXrefVerCopy=true`,
		meta.ParentSID)

	return nil
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
	tResults := Query(meta.Tx, `
        SELECT m.SID, m.ResourceSID, r.Singular FROM Resources AS r
        JOIN Metas AS m ON (m.ResourceSID=r.SID)
        WHERE r.RegistrySID=? AND r.XID=?
        FOR UPDATE`, meta.Registry.DbSID, xRefXID)
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
	Do(meta.Tx, `
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

	// Create the xref'd Versions
	if resource != nil {
		resource.SaveXrefVersionCopies(targetResourceSID)
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
	tResults := Query(r.Tx, `
            SELECT v.SID FROM Metas AS srcM
            JOIN Resources AS tr ON (tr.RegistrySID=srcM.RegistrySID AND
                                      tr.XID=srcM.xRefXID)
            JOIN Metas AS m ON (m.ResourceSID=tr.SID)
            JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                    v.UID=m.defaultVID)
            WHERE srcM.ResourceSID=? FOR UPDATE`,
		r.DbSID)
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
	Do(r.Tx, `
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

	return nil
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
	Do(srcResource.Tx, `
        DELETE ft FROM Props AS ft
        JOIN Versions AS v ON (ft.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	Do(srcResource.Tx, `
        DELETE fe FROM Entities AS fe
        JOIN Versions AS v ON (fe.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)

	// One Entities row per target Version, all at once.
	Do(srcResource.Tx, `
        REPLACE INTO Entities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, XID, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID, ?,
               CONCAT(?, '/versions/', v.UID), true
        FROM Versions AS v WHERE v.ResourceSID=? FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, synthAbstract, srcResource.XID,
		targetResourceSID)

	// Copy each target Version's own props onto its corresponding
	// synthetic eSID, for every current Version at once (excluding the
	// target's own "xref" - Versions never have one, but kept for
	// parity with the old per-row exclusion).
	Do(srcResource.Tx, `
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

	// Calculated attrs for every synthetic version at once: xid and
	// RESOURCEid (using the SOURCE resource's singular/UID, since
	// that's every synthetic version's effective parent) are static -
	// wholesale recreated here only because the whole synthetic-
	// version set itself is being recreated (the xref pointer moved),
	// not because they individually change; isdefault is genuinely
	// dynamic (mirrors the target's own per-Version isdefault).
	Do(srcResource.Tx, `
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

	Do(srcResource.Tx, `
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

	Do(srcResource.Tx, `
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

	return nil
}

func (sqlBE *SQLBackend) GetEntityFromXID(tx *Tx, regSID string, xidStr string,
	allAttrs bool, anyCase bool, accessMode int) (*Entity, *XRError) {

	defer log.Trace("tx: %s xid:%q all:%v",
		tx.uuid, xidStr, allAttrs)()

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
	e2, _ := iter.Next()
	PanicIf(e2 != nil, "too many entities: %v", e2)

	return e, nil
}

func (sqlBE *SQLBackend) xxGetEntitiesFromXID(tx *Tx, regSID string,
	xidStr string, allAttrs bool, children bool, anyCase bool,
	accessMode int) ([]*Entity, *XRError) {

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

	results := Query(tx, queryString, args...)
	defer results.Close()

	entities := []*Entity(nil)
	for {
		entity, xErr := readNextEntity(tx, results, accessMode)
		if xErr != nil {
			return nil, xErr
		}
		if entity == nil {
			break
		}

		if accessMode == FOR_WRITE {
			lockEntityFamily(tx, entity)
		}

		entities = append(entities, entity)
	}

	return entities, nil
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

	results := Query(tx, queryString, args...)
	// defer results.Close()

	iterator := &sqlEntityIterator{
		Results:    results,
		Tx:         tx,
		AccessMode: accessMode,
	}

	return iterator, nil
}

func (sqlBE *SQLBackend) GetEntitiesFromQuery(tx *Tx, regID string,
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

	results := Query(tx, `
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
			lockEntityFamily(tx, e)
		}
	}

	return entities, nil
}

type sqlEntityIterator struct {
	Results    *Result
	Tx         *Tx
	AccessMode int
}

func (sei *sqlEntityIterator) Next() (*Entity, *XRError) {
	e, xErr := readNextEntity(sei.Tx, sei.Results, sei.AccessMode)
	if xErr != nil {
		return nil, xErr
	}

	if sei.AccessMode == FOR_WRITE {
		lockEntityFamily(sei.Tx, e)
	}

	return e, nil
}
