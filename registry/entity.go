package registry

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/xregistry/server/common"
)

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

type EntityExtensions struct {
	// dbPropBatch buffers own-property Props row info during
	// Save()'s traversal (see SetDBPropertyBatch()/DoDBPropertyBatch()
	// below) so they can be written as a single multi-row REPLACE INTO
	// instead of one round trip per property.
	dbPropBatch []dbPropRow
}

func (e *Entity) GetRequestInfo() *RequestInfo {
	if e.Tx == nil {
		return nil
	}

	if e.Tx.RequestInfo == nil {
		return nil
	}

	return e.Tx.RequestInfo
}

type EntitySetter interface {
	Get(name string) any
	JustSet(name string, val any) *XRError
	SetSave(name string, val any) *XRError
	Delete() *XRError
}

func GoToOurType(val any) string {
	// Fast path: type switch avoids the reflect.ValueOf() overhead for
	// the concrete types that make up the overwhelming majority of calls
	// (JSON-decoded values, plus our own int/uint64/struct{}{} usages).
	switch val.(type) {
	case bool:
		return BOOLEAN
	case int:
		return INTEGER
	case uint64:
		return UINTEGER
	case float64:
		return DECIMAL
	case string:
		return STRING
	case []any:
		return ARRAY
	case map[string]any:
		return MAP
	case struct{}:
		return OBJECT
	}

	panic(fmt.Sprintf("Bad type: %s", reflect.ValueOf(val).Type()))

	// Slow path: fall back to reflect.Kind for anything not covered above
	// (e.g. some other named slice/map/struct type), so behavior stays
	// identical to before for any case we didn't anticipate.
	switch reflect.ValueOf(val).Kind() {
	case reflect.Interface:
		return ANY
	case reflect.Slice:
		return ARRAY
	case reflect.Map:
		return MAP
	case reflect.Struct:
		return OBJECT
	}
	panic(fmt.Sprintf("Bad Kind: %v", reflect.ValueOf(val).Kind()))
}

func (e *Entity) ToString() string {
	str := fmt.Sprintf("%s/%s\n  Object: %s\n  NewObject: %s",
		e.Singular, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	return str
}

func (e *Entity) WasTouched() bool {
	log.FuncPrintf("tx: %s WasTouch: %s/%s", e.Tx.uuid, e.Singular, e.UID)
	return e.EpochSet
}

func (e *Entity) Touch() bool {
	log.FuncPrintf("tx: %s Touch: %s/%s", e.Tx.uuid, e.Singular, e.UID)

	// See if it's already been modified (and saved) this Tx, if so exit
	if e.ModSet && e.EpochSet {
		return false
	}

	e.Lock()
	e.EnsureNewObject()
	return true
}

// lockResourceFamily locks (FOR UPDATE) just the Resource's own row in
// the Resources table.
//
// Every write path locks the Resource FIRST - before it ever creates or
// touches any Meta/Version/Entities row for it (Group.UpsertResource()
// locks the parent Group first, then FindResource(FOR_WRITE) locks the
// Resource before any Meta/Version insert; HTTPDeleteVersions() locks
// the Resource before reading/deleting any Version, etc). That means
// this single lock already fully serializes any two Txs that would
// otherwise race on the same Resource's family: a second Tx blocks here
// before it can create/modify/delete anything else in the family.
// Verified empirically: a Tx holding only this Resources-row FOR UPDATE
// lock fully blocks a concurrent Tx's plain "INSERT INTO Versions"
// for the same ResourceSID until the first Tx commits/rolls back (even
// though Resources and Versions are different tables/no FK) - the
// second Tx can't even reach its own INSERT until it re-acquires this
// same Resources row lock first, per the locking order above.
//
// The remaining two needs handled by locking Metas/Versions/Entities
// too are each already covered elsewhere:
//   - RR-snapshot freshness for plain SELECTs against Metas/Versions
//     (newestVersionID(), GetProblematicVersions(), etc) is handled by
//     each function's own "if meta.AccessMode==FOR_WRITE" FOR UPDATE
//     hint, independent of this function.
//   - Entity.Save()'s Props DELETE-then-INSERT race is handled by
//     Refresh(FOR_WRITE)'s own "SELECT ... FROM Entities WHERE eSID=?
//     FOR UPDATE" on the specific entity being saved.
//
// The old code below locked the whole family (Metas/Versions/Entities
// rows too) - left here, commented out, as a reminder/fallback in case
// concurrency issues resurface and we need to revisit this narrowing.
func lockResourceFamily(tx *Tx, resourceSID string) {
	lock := func(query string) {
		results := Query(tx, query, resourceSID)
		results.Close()
	}

	lock(`SELECT SID FROM Resources WHERE SID=? FOR UPDATE`)

	/*
		lock(`SELECT SID FROM Metas WHERE ResourceSID=? FOR UPDATE`)
		lock(`SELECT SID FROM Versions WHERE ResourceSID=? FOR UPDATE`)

		// Entities holds both this Resource's own row (keyed by eSID) and
		// its children's rows (keyed by ParentSID) - two different indexes
		// (PRIMARY on eSID, a separate index on ParentSID). A single query
		// with "WHERE eSID=? OR ParentSID=?" can't use either index (MySQL
		// falls back to a full index scan of the WHOLE table), which - since
		// this is a locking FOR UPDATE read - ends up granting X locks on
		// every OTHER Resource's rows in the table too (verified via
		// performance_schema.data_locks), a real over-locking bug, not just
		// a missed optimization. Splitting into two separate queries lets
		// each one use its own index and lock only the rows that actually
		// belong to this Resource.
		lock(`SELECT eSID FROM Entities WHERE eSID=? FOR UPDATE`)
		lock(`SELECT eSID FROM Entities WHERE ParentSID=? FOR UPDATE`)
	*/
}

func RawEntityFromXID(tx *Tx, regID string, xid string, anyCase bool, accessMode int) (*Entity, *XRError) {
	defer log.Trace("tx: %s %s", tx.uuid, xid)()

	// RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,
	// PropName,PropValue,PropType,IsSystemProp
	//   0     1     2       3         4      5   6     7      8
	//     9        10         11         12

	XID := "XID"

	if anyCase {
		XID = "LowerXID"
		xid = strings.ToLower(xid)
	}

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
		lockExpr = " FOR UPDATE"
	}

	queryString := `
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
            e.eSID=p.eSID
            AND p.IsDefaultVerCopy=false AND p.IsXrefPropCopy=false
            AND p.IsXrefVerCopy=false AND p.IsCalcStatic=false
            AND p.IsCalcDynamic=false)
        WHERE e.RegSID=? AND e.` + XID + `=?
        ORDER BY XID` + lockExpr

	results := Query(tx, queryString, regID, xid)
	defer results.Close()

	ent, xErr := readNextEntity(tx, results, accessMode)
	if xErr != nil {
		return nil, xErr
	}

	// See lockExpr's comment above: this initial fetch already FOR
	// UPDATE-locked ent's own row (if accessMode==FOR_WRITE), but if
	// ent is a Resource/Meta/Version it also needs its sibling
	// Resource+Meta+Versions family locked - matches Entity.Lock()'s
	// same handling for already-cached entities.
	if accessMode == FOR_WRITE {
		lockEntityFamily(tx, ent)
	}

	return ent, nil
}

// lockEntityFamily locks (FOR UPDATE) the full Resource+Meta+Versions
// family that ent belongs to, if any - see lockResourceFamily()'s and
// Entity.Lock()'s doc comments for why. Safe to call with a nil ent or
// a non-Resource/Meta/Version entity (no-op).
//
// IMPORTANT: lockResourceFamily() only takes real DB-level "FOR UPDATE"
// row locks - it has no idea about this Tx's Go-level entity Cache. If
// some OTHER member of the family (e.g. the Meta, when ent is a
// Version) was already fetched FOR_READ earlier in this same Tx and is
// sitting in tx.Cache, DB-locking its row here does NOT, by itself,
// upgrade that cached Go object's AccessMode or refresh its
// Object/NewObject - it would stay FOR_READ with whatever (possibly
// now-stale, or read mid-another-Tx's-Save()) data it originally
// loaded. Every "if meta.AccessMode == FOR_WRITE { lockExpr = ...}"-
// style guard added throughout resource.go/versionmodes.go implicitly
// assumes that AccessMode faithfully reflects whether the row is
// really DB-locked - so after lockResourceFamily() runs, we must also
// walk this Tx's cache and upgrade (Refresh(FOR_WRITE)) every other
// cached Resource/Meta/Version belonging to this same family, or those
// guards (and any code that just trusts a cached Meta/Version's
// AccessMode) can silently keep using stale FOR_READ data even though
// the DB row underneath was just real-locked. This was the root cause
// of a recurring "<attr> is nil" panic under TestMiscConcurrency.
func lockEntityFamily(tx *Tx, ent *Entity) {
	if ent == nil {
		return
	}
	resourceSID := ""
	switch ent.Type {
	case ENTITY_RESOURCE:
		resourceSID = ent.DbSID
	case ENTITY_META, ENTITY_VERSION:
		resourceSID = ent.ParentSID
	}
	if resourceSID == "" {
		return
	}
	lockResourceFamily(tx, resourceSID)

	// Now upgrade any other cached family member (Resource itself, its
	// Meta, or any of its Versions) that isn't already FOR_WRITE, so the
	// Go-level cache matches the DB-level lock state we just took.
	for _, cached := range tx.Cache {
		if cached == ent || cached.AccessMode == FOR_WRITE {
			continue
		}
		isFamilyMember :=
			(cached.Type == ENTITY_RESOURCE && cached.DbSID == resourceSID) ||
				((cached.Type == ENTITY_META || cached.Type == ENTITY_VERSION) &&
					cached.ParentSID == resourceSID)
		if !isFamilyMember {
			continue
		}
		if cached.NewObject != nil || cached.NewSystem != nil {
			// This entity has buffered, not-yet-persisted edits already
			// applied to it earlier in this same Tx (e.g. an in-progress
			// SetSave()/Update() sequence). Calling Refresh() here would
			// discard e.NewObject wholesale (it only flushes NewSystem,
			// not NewObject - see Refresh()'s own doc comment), silently
			// losing those edits. Just mark it FOR_WRITE (its row is now
			// genuinely DB-locked via lockResourceFamily() above) without
			// reloading, so we don't clobber the pending edits.
			cached.AccessMode = FOR_WRITE
			continue
		}
		log.FuncPrintf("tx: %s lockEntityFamily: upgrading stale cached %q "+
			"(eSID=%s) to FOR_WRITE now that its family's DB rows are locked",
			tx.uuid, cached.XID, cached.DbSID)
		Must(cached.Refresh(FOR_WRITE))
	}
}

func (e *Entity) Query(query string, args ...any) [][]any {
	results := Query(e.Tx, query, args...)
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

	return data
}

func RawEntitiesFromQuery(tx *Tx, regID string, accessMode int, query string, args ...any) ([]*Entity, *XRError) {
	defer log.Trace("tx: %s %s", tx.uuid, query)()

	// RegSID,Type,Plural,Singular,ParentSID,eSID,UID,Abstract,XID,
	// PropName,PropValue,PropType,IsSystemProp
	//   0     1     2       3         4      5   6     7      8
	//     9        10         11         12

	if query != "" {
		query = "AND (" + query + ") "
	}

	args = append(append([]any{}, regID), args...)

	// See RawEntityFromXID()'s matching comment - without this, a
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

	// See RawEntityFromXID()'s matching comment: lock each returned
	// entity's Resource+Meta+Versions family too, if applicable.
	if accessMode == FOR_WRITE {
		for _, e := range entities {
			lockEntityFamily(tx, e)
		}
	}

	return entities, nil
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) eSetSave(path string, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.uuid, path, val)()

	pp, err := PropPathFromUI(path)
	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail="+
				fmt.Sprintf("Bad attribute path in \"%s\": %s", e.XID, err))
	}

	// Set, Validate and Save
	xErr := e.SetPP(pp, val)
	if xErr != nil {
		return xErr.SetSubject(e.XID)
	}
	return nil
}

func (e *Entity) ValidateAndSave(force bool) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.uuid, e.XID)()

	// Force will do a validate even if it doesn't look like anything changed.
	// BUT if after validate() nothing still hasn't changed then it doesn't
	// call save()

	// If nothing changed, then exit
	if !force && e.NewObject == nil {
		return nil
	}

	// Make sure we have a tx since Validate assumes it
	e.Tx.NewTx()

	verb := log.IsFuncVerbose()
	if verb {
		log.Printf("tx: %s "+
			"Pre validate %s/%s\ne.Object:\n%s\n\ne.NewObject:\n%s",
			e.Tx.uuid,
			e.Abstract, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	}

	if xErr := e.Validate(); xErr != nil {
		return xErr
	}

	if verb {
		log.Printf("tx: %s Post validate(%s): %s", e.Tx.uuid,
			e.XID, ToJSON(e.NewObject))
	}

	if e.NewObject == nil {
		return nil
	}

	return e.Save()
}

// This is really just an internal Setter used for testing.
// It'll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s: %s=%v", e.Tx.uuid, e.DbSID, pp.UI(), val)()
	defer func() {
		if log.IsFuncVerbose() {
			log.Printf("tx: %s exit: e.Object:\n%s", e.Tx.uuid, ToJSON(e.Object))
		}
	}()

	if xErr := e.eJustSet(pp, val); xErr != nil {
		return xErr
	}

	xErr := e.ValidateAndSave(false)
	if xErr != nil {
		// If there's an error, and we're making the assumption that we're
		// setting and saving all in one shot (and there are no other edits
		// pending), go ahead and undo the changes since they're wrong.
		// Otherwise the caller would need to call Refresh themselves.

		// Not sure why setting it to nil isn't sufficient (todo)
		// e.NewObject = nil
		e.Refresh(FOR_READ)
	}

	return xErr
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
func (e *Entity) prepDBProperty(pp *PropPath, val any) (row dbPropRow,
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
				Do(e.Tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
					e.DbSID)
			} else {
				// Update the content
				DoOneTwo(e.Tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)

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

// EnumValueToDBString encodes a single constraint "enum" value the
// same way prepDBProperty() (above) encodes a real attribute value
// before writing it to Props.PropValue (booleans as
// "true"/"false", everything else via fmt.Sprintf("%v", v)) - used by
// Group.validateEnum() to build a SQL-comparable string for each enum
// entry.
func EnumValueToDBString(v any) string {
	if b, ok := v.(bool); ok {
		if b {
			return "true"
		}
		return "false"
	}
	return fmt.Sprintf("%v", v)
}

func (e *Entity) SetDBProperty(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.uuid, pp, val)()

	row, skip, xErr := e.prepDBProperty(pp, val)
	if xErr != nil {
		return xErr
	}
	if skip {
		return nil
	}

	e.DBWriteOwnProp(row.Name, row.Value, row.Type, row.DocView)
	return nil
}

// SetDBPropertyBatch is like SetDBProperty() but, instead of writing
// the property's Props row immediately, buffers it into
// e.dbPropBatch for a later single multi-row REPLACE INTO via
// DoDBPropertyBatch() - see Save()'s traversal loop, the only caller.
// The nil/delete case is a no-op here: Save() already issues one
// blanket DELETE for all of this entity's own-prop rows before
// traversal starts, so there's never a per-prop delete to buffer.
func (e *Entity) SetDBPropertyBatch(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.uuid, pp, val)()

	row, skip, xErr := e.prepDBProperty(pp, val)
	if xErr != nil {
		return xErr
	}
	if skip || row.Value == nil {
		return nil
	}

	e.dbPropBatch = append(e.dbPropBatch, row)

	return nil
}

// dbPropBatchChunkSize caps how many rows go into a single REPLACE
// INTO statement in DoDBPropertyBatch(), purely as a max_allowed_packet
// safety net - not expected to matter for realistic per-entity
// attribute counts.
const dbPropBatchChunkSize = 200

// DoDBPropertyBatch flushes whatever SetDBPropertyBatch() has buffered
// into e.dbPropBatch (if anything) as one or more multi-row
// REPLACE INTO Props statements (see DBWritePropsBatch()
// in fulltree.go), then clears the buffer. Called once by Save(),
// right after its traversal loop finishes.
func (e *Entity) DoDBPropertyBatch() {
	if len(e.dbPropBatch) == 0 {
		return
	}

	rows := e.dbPropBatch
	e.dbPropBatch = nil

	e.DBWritePropsBatch(rows, false)
}

// Clears system prop(s) for all versions of this resource. Accepts
// multiple PropPaths so callers that need to clear several system
// props at once (e.g. EnsureCompat's format/compat validated+reason
// attrs) can do so in one pass, instead of looping over all Versions
// once per prop. Goes through the same buffered SetSystemDBProperty()
// mechanism as everything else (see Entity.System's doc comment) -
// using r.FindVersion() (not a throwaway shell object) so a clear
// issued after some other system-prop write on the same Version within
// this same request correctly composes with (and can override) that
// earlier buffered value, rather than racing an immediate DB DELETE
// against a later flush.
func (e *Entity) ClearResourceSystemDBProperty(pps ...*PropPath) {
	defer log.Trace("tx: %s %d props", e.Tx.uuid, len(pps))()

	if len(pps) == 0 {
		return
	}

	r, ok := e.Self.(*Resource)
	PanicIf(!ok, "%s isn't a Resource", e.XID)

	// FOR UPDATE to make sure we grab the latest stuff, and lock it
	lockExpr := ""
	if meta := e.Tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Query the real Versions table directly (not r.GetVersions(),
	// which reads from Entities and would also pick up synthetic
	// xref-copied version rows sharing this Resource's ParentSID).
	results := Query(e.Tx,
		`SELECT UID FROM Versions WHERE ResourceSID=?`+lockExpr,
		r.DbSID)
	defer results.Close()

	uids := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		uids = append(uids, NotNilString(row[0]))
	}

	for _, uid := range uids {
		v, xErr := r.FindVersion(uid, false)
		PanicIf(xErr != nil || v == nil, "%s/versions/%s: %s", r.XID, uid, xErr)
		for _, pp := range pps {
			v.SetSystemDBProperty(pp, nil)
		}
	}
}

// SetSystemDBProperty buffers a system-prop change into e.NewSystem -
// it does NOT write to the DB immediately. The actual write (and, if
// e is a Version, at most one re-run of the default-version cascade)
// happens later, via SaveSystemProps() - either at Tx-commit time
// (called from tx.WriteCache()), or earlier if a caller explicitly
// calls tx.FlushSystemProps() (e.g. EnsureCompat()'s callers do this
// right away, so the buffered values are visible in the same request's
// HTTP response, before the Tx actually commits). This lets callers
// like EnsureCompat() set several system props on the same entity
// back-to-back without each one independently re-triggering the whole
// cascade - see SaveSystemProps().
func (e *Entity) SetSystemDBProperty(pp *PropPath, val any) {
	defer log.Trace("tx: %s %s=%v", e.Tx.uuid, pp, val)()

	PanicIf(pp.UI() == "", "pp is empty")

	/* DUG FT
	name := pp.DB()

	_, propsMap := e.GetPropsOrdered()
	specProp, ok := propsMap[pp.Top()]
	if ok && specProp.internals != nil {
		// Any prop with "dontStore"=true we skip
		if specProp.internals.dontStore {
			return
		}
	}

	PanicIf(len(name) > MAX_PROPNAME, "SysProp name is too long: %s", name)
	PanicIf(e.DbSID == "", "DbSID should not be empty")
	PanicIf(e.Registry == nil, "Registry should not be nil")

	if !IsNil(val) {
		switch reflect.ValueOf(val).Kind() {
		case reflect.String:
			PanicIf(reflect.ValueOf(val).Len() > MAX_VARCHAR, "%s:too long", name)
		case reflect.Slice:
			PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
		case reflect.Map:
			PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
		case reflect.Struct:
			PanicIf(reflect.ValueOf(val).Len() > 0, "%s:non-empty", name)
		}
	}
	*/

	// Key by pp.Top() (the plain attribute name), NOT pp.DB() (which
	// has a trailing DB_IN separator) - this must match the key scheme
	// setFromDBNameInto() uses to populate e.System from the DB (via
	// ObjectSetProp(), keyed by the plain name), so a value loaded from
	// the DB and a value buffered here land under the SAME key and can
	// be diffed/overridden correctly in SaveSystemProps().
	if e.NewSystem != nil {
		if reflect.DeepEqual(e.NewSystem[pp.Top()], val) {
			return
		}
	} else if e.System != nil {
		if reflect.DeepEqual(e.System[pp.Top()], val) {
			return
		}
	}

	e.EnsureNewSystem()
	e.NewSystem[pp.Top()] = val

	e.Tx.AddToCache(e)
}

// EnsureNewSystem lazily initializes e.NewSystem (buffered, uncommitted
// system-prop changes) as a clone of e.System - mirrors
// EnsureNewObject(), but must NEVER touch epoch/modifiedat/EpochSet/
// ModSet/NewObject, since system props are, by design, invisible to
// the entity's own change-tracking (see Entity.System's doc comment).
func (e *Entity) EnsureNewSystem() {
	// Same guard as EnsureNewObject() - system-prop buffering must only
	// ever happen on an already FOR_WRITE-locked entity.
	PanicIf(e.AccessMode != FOR_WRITE, "EnsureNewSystem: %q isn't FOR_WRITE", e.XID)

	// Save pre-Tx values the first time we're about to genuinely
	// buffer a change - mirrors EnsureNewObject()'s OriginObject
	// capture. See OriginSystem's doc comment.
	if e.OriginSystem == nil {
		if e.System == nil {
			e.OriginSystem = map[string]any{}
		} else {
			e.OriginSystem = maps.Clone(e.System)
		}
	}

	if e.NewSystem != nil {
		return
	}
	e.NewSystem = map[string]any{}
	for k, v := range e.System {
		e.NewSystem[k] = v
	}
}

// This is used to take a DB entry and update the current Entity's Object
// SetFromDBName parses one Props row's PropValue/PropType and
// applies it into e.Object (own/calculated/cascaded props) or e.System
// (system props, isSystem=true) - the row-processing loop is otherwise
// identical for both, but system props go into their own bucket (see
// Entity.System's doc comment) since they must never touch Object,
// epoch, or modifiedat.
func (e *Entity) SetFromDBName(name string, val *string, propType string) *XRError {
	return e.setFromDBNameInto(&e.Object, name, val, propType)
}

func (e *Entity) SetSystemFromDBName(name string, val *string, propType string) *XRError {
	return e.setFromDBNameInto(&e.System, name, val, propType)
}

func (e *Entity) setFromDBNameInto(dest *map[string]any, name string, val *string, propType string) *XRError {
	var err error
	pp := MustPropPathFromDB(name)

	if val == nil {
		err := ObjectSetProp(*dest, pp, val)
		if err != nil {
			return NewXRError("bad_request", e.XID,
				"error_detail=Error setting attribute: "+err.Error())
		}
		return nil
	}
	if *dest == nil {
		*dest = map[string]any{}
	}
	obj := *dest

	if IsString(propType) {
		err = ObjectSetProp(obj, pp, *val)
	} else if propType == BOOLEAN {
		// Technically the "1" check shouldn't be needed, but just in case
		err = ObjectSetProp(obj, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s: %s", *val, err))
		}
		err = ObjectSetProp(obj, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s: %s", *val, err))
		}
		err = ObjectSetProp(obj, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(obj, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(obj, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(obj, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}

	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail=Error setting attribute: "+err.Error())
	}
	return nil
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

func (e *Entity) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	switch e.Type {
	case ENTITY_REGISTRY:
		return e.Registry.Model.GetPropsOrdered()
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		return gm.GetPropsOrdered()
	case ENTITY_RESOURCE:
		_, rm := e.GetModels()
		return rm.GetPropsOrdered()
	case ENTITY_META:
		_, rm := e.GetModels()
		return rm.GetMetaPropsOrdered()
	case ENTITY_VERSION:
		_, rm := e.GetModels()
		return rm.GetVersionPropsOrdered()
	default:
		panic("What?")
	}
}

// This is used to serialize an Entity regardless of the format.
// This will:
//   - Use AddCalcProps() to fill in any missing props (eg Entity's getFn())
//   - Call that passed-in 'fn' to serialize each prop but in the right order
//     as defined by the entity's GetPropsOrdered()
func (e *Entity) SerializeProps(
	fn func(*Entity, string, any, *Attribute) *XRError) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.uuid, e.XID)()

	info := e.GetRequestInfo()
	daObj := e.AddCalcProps()
	attrs := e.GetAttributes(e.Object)

	if log.IsFuncVerbose() {
		log.Printf("tx: %s SerProps.Entity: %s", e.Tx.uuid, ToJSON(e))
		log.Printf("tx: %s SerProps.Obj: %s", e.Tx.uuid, ToJSON(e.Object))
		log.Printf("tx: %s SerProps daObj: %s", e.Tx.uuid, ToJSON(daObj))
		log.Printf("tx: %s SerProps attrs:\n%s", e.Tx.uuid, ToJSON(attrs))
	}

	resourceSingular := ""
	hasDoc := false
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
		_, rm := e.GetModels()
		resourceSingular = rm.Singular
		hasDoc = rm.GetHasDocument()
	}

	propsOrdered, propsMap := e.GetPropsOrdered()

	// Loop over the defined props - extensions are done under the "if...$ext"
	for _, prop := range propsOrdered {
		name := prop.Name

		// If hasDoc && we're in doc view &&  on the Resource (not version)
		// then skip the RESOURCE/RESOURCEbase64 attributes.
		// Other version-level attribute are automatically excluded by the
		// query. RESOURCE* aren't part of the Props table, so they're special
		if hasDoc && e.Type == ENTITY_RESOURCE && info.DoDocView() {
			if name == resourceSingular || name == resourceSingular+"base64" {
				continue
			}
		}

		log.FuncPrintf("tx: %s Ser prop(%s): %q", e.Tx.uuid, e.XID, name)

		attr, ok := attrs[name]
		if !ok {
			log.FuncPrintf("tx: %s  skipping %q, no attr", e.Tx.uuid, name)
			delete(daObj, name)
			continue // not allowed at this eType so skip it
		}

		if prop.Name == "$extensions" {
			if prop.InType(e.Type) {
				for _, objKey := range SortedKeys(daObj) {
					// Skip spec defined properties
					if propsMap[objKey] != nil {
						continue
					}

					val, _ := daObj[objKey]
					attr := attrs[objKey]
					delete(daObj, objKey)
					if attr == nil {
						attr = attrs["*"]
						PanicIf(objKey[0] != '#' && attr == nil,
							"Can't find attr for (%s) %q", e.XID, objKey)
					}
					// log.Printf("tx: %s Ser*ext(%s): %q", e.Tx.uuid, e.XID,
					//  objKey)

					if xErr := fn(e, objKey, val, attr); xErr != nil {
						return xErr
					}
				}
			}
			continue
		}

		if name[0] == '$' || (prop.internals != nil && prop.internals.alwaysSerialize) {
			log.FuncPrintf("tx: %s forced serialization of %q", e.Tx.uuid, name)
			if xErr := fn(e, name, nil, attr); xErr != nil {
				return xErr
			}
			continue
		}

		// Should be a no-op for Resources.
		if val, ok := daObj[name]; ok {
			log.FuncPrintf("tx: %s val: %v", e.Tx.uuid, val)
			if !IsNil(val) {
				xErr := fn(e, name, val, attr)
				if xErr != nil {
					return xErr
				}
			}
			delete(daObj, name)
		} else {
			log.FuncPrintf("tx: %s no value for %q", e.Tx.uuid, name)
		}
	}

	// Now do all other props (extensions) alphabetically
	/*
		for _, objKey := range SortedKeys(daObj) {
			attrKey := objKey
			if attrKey == e.Singular+"id" {
				attrKey = "id"
			}
			val, _ := daObj[objKey]
			attr := attrs[attrKey]
			if attr == nil {
				attr = attrs["*"]
				PanicIf(attrKey[0] != '#' && attr == nil,
					"Can't find attr for %q", attrKey)
			}

			if xErr := fn(e, objKey, val, attr); xErr != nil {
				return xErr
			}
		}
	*/

	return nil
}

func (e *Entity) Save() *XRError {
	defer log.Trace("tx: %s %s", e.Tx.uuid, e.XID)()

	PanicIf(e.AccessMode != FOR_WRITE, "%q isn't FOR_WRITE", e.XID)

	// TODO remove at some point when we're sure it's safe
	if SpecProps["epoch"].InType(e.Type) && IsNil(e.NewObject["epoch"]) {
		// Only an xref'd "meta" is allowed to not have an 'epoch'
		if e.Type != ENTITY_META || IsNil(e.NewObject["xref"]) {
			PanicIf(true, "Epoch is nil(%s):%s", e.XID, ToJSON(e.NewObject))
		}
	}

	if log.IsFuncVerbose() {
		log.Printf("tx: %s NewObject:\n%s", e.Tx.uuid, ToJSON(e.NewObject))
		// ShowStack()
	}

	// If we're saving a Group then something must have changed, so
	// we need to add it to our "validate" list (e.g. check its constraints).
	// And at the end of the tx we'll validate all of them at once.
	if e.Type == ENTITY_GROUP {
		e.Tx.AddGroupToValidate(e.Self.(*Group))
	}

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	// Delete all user props for this entity, we assume that NewObject
	// contains everything we want going forward
	Do(e.Tx, `DELETE FROM Props
              WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                    AND IsXrefVerCopy=false AND IsSystemProp=false
                    AND IsCalcStatic=false AND IsCalcDynamic=false`,
		e.DbSID)

	resSingular := ""
	resHasDoc := false
	if rm := e.GetResourceModel(); rm != nil {
		resSingular = rm.Singular
		resHasDoc = rm.GetHasDocument()
	}

	var traverse func(pp *PropPath, val any, obj map[string]any) *XRError
	traverse = func(pp *PropPath, val any, obj map[string]any) *XRError {
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
								"not %s", pp.Last().Text, keyValue.Kind().String()))
				}

				k := keyValue.Interface().(string)
				v := valValue.MapIndex(keyValue).Interface()
				// "RESOURCE" is special - call SetDBProp if it's present
				if resHasDoc && pp.Len() == 0 && k == resSingular {
					if xErr := e.SetDBPropertyBatch(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else if k[0] == '#' {
					if xErr := e.SetDBPropertyBatch(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else {
					if IsNil(v) {
						continue
					}
					if xErr := traverse(pp.P(k), v, obj); xErr != nil {
						return xErr
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return e.SetDBPropertyBatch(pp, map[string]any{})
			}

		case reflect.Slice:
			if valValue.Len() == 0 {
				valValue = reflect.MakeSlice(reflect.TypeOf(val), 0, 0)
				return e.SetDBPropertyBatch(pp, valValue.Interface())
			}
			for i := 0; i < valValue.Len(); i++ {
				v := valValue.Index(i).Interface()
				if xErr := traverse(pp.I(i), v, obj); xErr != nil {
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
				if xErr := traverse(pp.P(k), v, obj); xErr != nil {
					return xErr
				}
				count++
			}
			if count == 0 {
				return e.SetDBPropertyBatch(pp, struct{}{})
			}

		default:
			// must be scalar so add it
			return e.SetDBPropertyBatch(pp, val)
		}
		return nil
	}

	xErr := traverse(NewPP(), newObj, e.NewObject)
	if xErr != nil {
		return xErr
	}

	// Flush all buffered own-property rows from the traversal above as
	// one (or a few, if chunked) multi-row REPLACE INTO, instead of the
	// one-row-per-property writes SetDBProperty() would have done.
	e.DoDBPropertyBatch()

	// Copy 'newObj', removing all 'nil' attributes
	e.Object = map[string]any{}
	for k, v := range newObj {
		if !IsNil(v) {
			e.Object[k] = v
		}
	}
	e.NewObject = nil

	// Make sure the Resource is validated when Version or Meta is changed.
	if e.Type == ENTITY_VERSION {
		// onlyMetaChanged=false because we need the full validation code.
		v := e.Self.(*Version)
		e.Tx.AddResourceToValidate(v.Resource, false, false)
	} else if e.Type == ENTITY_META {
		meta := e.Self.(*Meta)
		e.Tx.AddResourceToValidate(meta.Resource, true, false)
	}

	// Right after we just finished writing this entity's Props
	// (DELETE-then-INSERT, above), do a locking self-check (FOR UPDATE,
	// so we see the true just-committed-by-us state, not a stale
	// snapshot) that this eSID's Entities row actually has at least one
	// matching Props row. A LEFT JOIN row with PropName IS NULL means
	// the Entities row exists but has NO Props at all - panic
	// immediately so we catch the exact Tx/stack responsible instead of
	// some later, unrelated code path.
	if log.HasVerbose("DebugSave") {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("tx: %s DIAG post-Save empty-Props query "+
						"itself failed for %s: %v", e.Tx.uuid, e.XID, r)
				}
			}()
			rows := Query(e.Tx, `
            SELECT ent.eSID, ent.XID, p.PropName
            FROM Entities AS ent
            LEFT JOIN Props AS p ON (
                ent.eSID=p.eSID AND p.IsDefaultVerCopy=false AND
                p.IsXrefPropCopy=false AND p.IsXrefVerCopy=false AND
                p.IsCalcStatic=false AND p.IsCalcDynamic=false)
            WHERE ent.eSID=? FOR UPDATE`, e.DbSID)
			defer rows.Close()

			sawNullProp := false
			for {
				row := rows.NextRow()
				if row == nil {
					break
				}
				if row[2] == nil {
					sawNullProp = true
					log.Printf("tx: %s DIAG POST-SAVE empty-Props: eSID=%v "+
						"XID=%v has NO Props row",
						e.Tx.uuid, NotNilString(row[0]), NotNilString(row[1]))
				}
			}
			if sawNullProp {
				ShowStack()
				panic(fmt.Sprintf(
					"tx: %s Save() just finished for %s (eSID=%s) but it has "+
						"NO Props rows at all - fix it!", e.Tx.uuid,
					e.XID, e.DbSID))
			}
		}()
	}

	return nil
}

// This will add in the calculated properties into the entity. This will
// normally be called after a query using FullTree view and before we serialize
// the entity we need to add the non-DB-stored properties (meaning, the
// calculated ones.
// Note that we make a copy and don't touch the entity itself. Serializing
// an entity shouldn't have side-effects.
func (e *Entity) AddCalcProps() map[string]any {
	mat := map[string]any{}
	// System props (formatvalidated, compatibilityvalidated, ...) live in
	// their own bucket (see Entity.System's doc comment) so their writes
	// don't touch epoch/modifiedat - but they're still real, serializable
	// top-level attributes, so merge them in here (Object wins on any
	// overlap, though the two buckets should never share a key).
	for k, v := range e.System {
		mat[k] = v
	}
	for k, v := range e.Object {
		mat[k] = v
	}

	// Regardless of the type of entity, set the generated properties
	propsOrdered, _ := e.GetPropsOrdered()

	for _, prop := range propsOrdered {
		// Only generate props that have a Fn
		if prop.internals != nil && prop.internals.getFn != nil {
			// Only generate/set the value if it's not already set
			if _, ok := mat[prop.Name]; !ok {
				if val := prop.internals.getFn(e); !IsNil(val) {
					// Only write it if we have a value
					// log.Printf("tx: %s Added calc prop: %q",
					// e.Tx.uuid, prop.Name)
					mat[prop.Name] = val
				}
			}
		}
	}

	return mat
}

// This will remove all Collection related attributes from the entity.
// While this is an Entity.Func, we allow callers to pass in the Object
// data to use instead of the e.Object/NewObject so that we'll use this
// Entity's Type (which tells us which collections it has), on the 'obj'.
// This is handy for cases where we need to remove the Resource's collections
// from a Version's Object - like on  a PUT to /GROUPs/gID/RESOURECEs/rID
// where we're passing in what looks like a Resource entity, but we're
// really using it to create a Version
func (e *Entity) RemoveCollections(obj Object) {
	if obj == nil {
		obj = e.NewObject
	}

	for _, coll := range e.GetCollections() {
		delete(obj, coll[0])
		delete(obj, coll[0]+"count")
		delete(obj, coll[0]+"url")
	}
}

// Array of plural/singular pairs
func (e *Entity) GetCollections() [][2]string {
	result := [][2]string{}
	switch e.Type {
	case ENTITY_REGISTRY:
		gs := e.Registry.Model.Groups
		for _, k := range Keys(gs) {
			result = append(result, [2]string{gs[k].Plural, gs[k].Singular})
		}
		return result
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		for _, rm := range gm.Resources {
			result = append(result, [2]string{rm.Plural, rm.Singular})
		}
		return result
	case ENTITY_RESOURCE:
		result = append(result, [2]string{"versions", "version"})
		return result
	case ENTITY_META:
		return nil
	case ENTITY_VERSION:
		return nil
	}
	panic(fmt.Sprintf("bad type: %d", e.Type))
	return nil
}

func (e *Entity) RemoveReadOnlyImmutable(obj Object) {
	// Don't touch what was passed in
	attrs := e.GetAttributes(obj)
	singular := e.Singular

	if e.Type == ENTITY_VERSION {
		singular = e.Self.(*Version).Resource.Singular
	} else if e.Type == ENTITY_META {
		singular = e.Self.(*Meta).Resource.Singular
	}
	singular += "id"

	for _, attr := range attrs {
		if attr.ReadOnly == false && attr.Immutable == false {
			continue
		}
		key := attr.Name

		if key == "versionid" || key == singular || key == "epoch" {
			// epoch and ID are special, we do need to check their values later
			continue
		}

		if _, keyPresent := obj[key]; !keyPresent {
			continue
		}

		delete(obj, key)
	}
}

func PrepUpdateEntity(e *Entity) *XRError {
	attrs := e.GetAttributes(e.NewObject)

	for key, _ := range attrs {
		attr := attrs[key]

		// Any ReadOnly attribute in Object, but not in NewObject, must
		// be one that we want to keep around. Note that a 'nil' in NewObject
		// will not grab the one in Object - assumes we want to erase the val
		/*
			if attr.ReadOnly {
				oldVal, ok1 := e.Object[attr.Name]
				_, ok2 := e.NewObject[attr.Name]
				if ok1 && !ok2 {
					e.NewObject[attr.Name] = oldVal
				}
			}
		*/

		if e.NewObject != nil {
			if attr.InType(e.Type) && attr.internals != nil && attr.internals.updateFn != nil {
				if xErr := attr.internals.updateFn(e); xErr != nil {
					return xErr
				}
			}
		}
	}

	return nil
}

// We call this to verify that the top level attribute names are valid.
// We can't really do this during the Validation funcs because at that point
// in the process we may have added #xxx type of attribute names, and "#"
// isn't a valid char. And we need to make sure users don't try to pass in
// attributes that start with "#" to attack us.
// Now, one way around this is to move the system props (#xxx) into a separate
// map (out of Object and NewObject) but then we'd need to duplicate a lot
// of logic - but it might actually make for a cleaner design to keep
// system data out of the user data space, so worth considering in the future.
// I really would prefer to push this down in the stack though.
func CheckAttrs(obj map[string]any, source string) *XRError {
	if obj == nil {
		return nil
	}
	for k, _ := range obj {
		if xErr := IsValidAttributeName(k, source, ""); xErr != nil {
			// log.Printf("tx: %s Key: %q", e.Tx.uuid, k)
			// ShowStack()
			return xErr
		}
	}
	return nil
}

// EntityInsert adds a row to Entities for a newly-created
// Registry/Group/Resource/Meta/Version - called from the same places
// that insert into the corresponding "real" entity table, right after
// e's fields (DbSID, ParentSID, etc.) have been populated. It also
// writes e's write-once calculated ("IsCalcStatic") attributes here,
// since they're provably immutable for the rest of this entity's
// lifetime (see SaveCalcStaticInsert()'s doc comment), this only ever runs
// once, at creation.
func (e *Entity) EntityInsert() {
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

	e.SaveCalcStaticInsert()
}

// DBWriteProp is the low-level writer for a single own
// (non-cascaded, non-calculated) Props row. Deleting
// (propValue==nil) removes the row; otherwise it's REPLACEd with the
// new value. isSystem marks whether this is a plain user-set prop
// (SetDBProperty, false) or a system-managed one (SetSystemDBProperty,
// true).
func (e *Entity) DBWriteProp(name string, propValue *string,
	propType string, docView bool, isSystem bool) {

	// defer log.FuncTrace("tx: %s %s/%s", e.Tx.uuid, e.XID, name)()

	if propValue == nil {
		// The prop row may or may not exist yet (e.g. deleting a prop
		// that was never set), so 0 or 1 rows is valid.
		DoZeroOne(e.Tx, `
            DELETE FROM Props
            WHERE eSID=? AND PropName=? AND IsDefaultVerCopy=false
                  AND IsXrefPropCopy=false AND IsXrefVerCopy=false`,
			e.DbSID, name)
		return
	}

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	// REPLACE reports 1 row if this (eSID,PropName) is new, 2 if it
	// replaced an existing row.
	DoOneTwo(e.Tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsSystemProp, IsCalcStatic, IsCalcDynamic)
        VALUES(?,?,?,?,?,?,?,?, ?,?,?,?,?, false,false,false, ?,false,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.XID, name, *propValue, propType, e.Abstract, docView,
		isSystem)
}

// DBWritePropsBatch writes multiple own/system Props rows
// for e in as few REPLACE INTO statements as possible, chunked at
// dbPropBatchChunkSize rows/statement as a max_allowed_packet safety
// net. isSystem marks whether these rows are system-managed
// (SaveSystemProps) or plain user-set (SetDBPropertyBatch/
// DoDBPropertyBatch) - same meaning as DBWriteProp's isSystem
// param, just batched across multiple rows in one statement instead of
// one statement per row.
func (e *Entity) DBWritePropsBatch(rows []dbPropRow, isSystem bool) {
	if len(rows) == 0 {
		return
	}

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	isSystemStr := "false"
	if isSystem {
		isSystemStr = "true"
	}
	rowPlaceholder := "(?,?,?,?,?,?,?,?, ?,?,?,?,?, false,false,false, " +
		isSystemStr + ",false,false)"

	for len(rows) > 0 {
		n := len(rows)
		if n > dbPropBatchChunkSize {
			n = dbPropBatchChunkSize
		}
		chunk := rows[:n]
		rows = rows[n:]

		placeholders := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)*13)
		for i, row := range chunk {
			placeholders[i] = rowPlaceholder
			args = append(args,
				e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
				e.DbSID, e.UID, e.XID,
				row.Name, *row.Value, row.Type, e.Abstract, row.DocView)
		}

		Do(e.Tx, `
            REPLACE INTO Props(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, XID,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
                IsSystemProp, IsCalcStatic, IsCalcDynamic)
            VALUES `+strings.Join(placeholders, ","), args...)
	}
}

// DBDeletePropsBatch deletes multiple own/system Props
// rows for e (identified by DB PropName) in as few
// "DELETE ... WHERE PropName IN (...)" statements as possible, chunked
// the same way as DBWritePropsBatch. Matches DBWriteProp's
// single-row delete filter (own rows only - never cascaded/copied
// ones), regardless of isSystem, since own vs. system PropNames never
// collide.
func (e *Entity) DBDeletePropsBatch(names []string) {
	if len(names) == 0 {
		return
	}

	for len(names) > 0 {
		n := len(names)
		if n > dbPropBatchChunkSize {
			n = dbPropBatchChunkSize
		}
		chunk := names[:n]
		names = names[n:]

		placeholders := make([]string, len(chunk))
		args := make([]any, 0, len(chunk)+1)
		args = append(args, e.DbSID)
		for i, name := range chunk {
			placeholders[i] = "?"
			args = append(args, name)
		}

		Do(e.Tx, `
            DELETE FROM Props
            WHERE eSID=? AND PropName IN (`+strings.Join(placeholders, ",")+`)
                  AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                  AND IsXrefVerCopy=false`, args...)
	}
}

// DBWriteOwnProp writes (or deletes, if propValue is nil) a
// plain user-set own property row for e - called by
// Entity.SetDBProperty() as part of Save()'s per-property traversal.
func (e *Entity) DBWriteOwnProp(name string, propValue *string,
	propType string, docView bool) {
	e.DBWriteProp(name, propValue, propType, docView, false)
}

// SaveSystemProps flushes any system-prop changes buffered by
// SetSystemDBProperty() (into e.NewSystem) since the last flush. It's
// called once per cached entity at Tx-commit time (see
// tx.WriteCache()) and is a no-op if nothing was buffered. It diffs
// NewSystem against System so only props that actually changed get
// written to the DB.
func (e *Entity) SaveSystemProps() {
	if e.NewSystem == nil {
		return
	}

	newSystem := e.NewSystem
	e.NewSystem = nil

	changed := map[string]any{}
	for name, newVal := range newSystem {
		oldVal, existed := e.System[name]
		if IsNil(newVal) {
			if existed && !IsNil(oldVal) {
				changed[name] = nil
			}
			continue
		}
		if !existed || !reflect.DeepEqual(oldVal, newVal) {
			changed[name] = newVal
		}
	}

	e.System = newSystem

	if len(changed) == 0 {
		return
	}

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

	e.DBDeletePropsBatch(deleteNames)
	e.DBWritePropsBatch(insertRows, true)

	// If this is a Version, make sure we fully validate its owning Resource
	if e.Type == ENTITY_VERSION {
		if v, ok := e.Self.(*Version); ok {
			e.Tx.AddResourceToValidate(v.Resource, true, false)
		}
	}
}

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
func (e *Entity) SaveCalcStaticInsert() {
	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

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
		e.SaveVersionCalc()
	}
}

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
func (e *Entity) SaveVersionCalc() {
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

// SaveXrefCascade refreshes the IsXrefPropCopy (this Meta's own
// copied meta.* attrs) and IsXrefVerCopy (synthetic Version rows) sets
// for a source Meta entity whose xref may have just been set, changed,
// or cleared.
// SaveXrefCascade refreshes the IsXrefPropCopy (this Meta's own
// copied meta.* attrs) and IsXrefVerCopy (synthetic Version rows) sets
// for a source Meta entity (e) whose xref may have just been set,
// changed, or cleared. e is always the real, in-memory Meta - either
// the one Save() is currently running for, or (via xref fan-out) one
// resolved through Registry.FindResourceByXID()+FindMeta() rather than
// a raw row.
func (e *Entity) SaveXrefCascade() {
	e.SaveXrefCascadeDelete()
	e.SaveXrefCascadeInsert()
}

// SaveXrefCascadeDelete clears this Meta's stale IsXrefPropCopy
// rows and its Resource's stale IsXrefVerCopy rows, from whatever the
// PREVIOUS xref state was.
func (e *Entity) SaveXrefCascadeDelete() {
	// e is always a real, in-memory Meta, which always has a parent
	// Resource, so e.ParentSID is never empty here.
	Do(e.Tx, `DELETE FROM Props WHERE eSID=? AND IsXrefPropCopy=true`,
		e.DbSID)
	Do(e.Tx, `
        DELETE FROM Props
        WHERE RegSID=? AND ParentSID=? AND IsXrefVerCopy=true`,
		e.Registry.DbSID, e.ParentSID)
	Do(e.Tx, `
        DELETE FROM Entities
        WHERE RegSID=? AND ParentSID=? AND IsXrefVerCopy=true`,
		e.Registry.DbSID, e.ParentSID)
}

// SaveXrefCascadeInsert (re)inserts this Meta's IsXrefPropCopy and
// IsXrefVerCopy rows based on the CURRENT xref state. Assumes
// SaveXrefCascadeDelete (and, for the own-props exclusion to work
// correctly, fullSaveOwnPropsDelete) have already run.
func (e *Entity) SaveXrefCascadeInsert() {
	results := Query(e.Tx, `
        SELECT xRefXID FROM Metas WHERE SID=?`, e.DbSID)
	row := results.NextRow()
	results.Close()
	if row == nil || NotNilString(row[0]) == "" {
		return
	}
	xRefXID := NotNilString(row[0])

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
	tResults := Query(e.Tx, `
        SELECT m.SID, m.ResourceSID, r.Singular FROM Resources AS r
        JOIN Metas AS m ON (m.ResourceSID=r.SID)
        WHERE r.RegistrySID=? AND r.XID=?
        FOR UPDATE`, e.Registry.DbSID, xRefXID)
	tRow := tResults.NextRow()
	tResults.Close()
	if tRow == nil {
		return
	}
	targetMetaSID := NotNilString(tRow[0])
	targetResourceSID := NotNilString(tRow[1])
	targetSingular := NotNilString(tRow[2])

	// e is always the real Meta entity, so its owning Resource is
	// directly accessible via e.Self.(*Meta).Resource - no need to
	// look it up as a "parent" entity at all. e always has a parent
	// Resource, so e.ParentSID is never empty here.
	resource := e.Self.(*Meta).Resource

	// Copy the target's meta.* props into this (source) Meta, excluding
	// its own xref and "<singular>id" attrs, and any '#' internal props.
	Do(e.Tx, `
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
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.XID, e.Abstract, targetMetaSID,
		"xref"+string(DB_IN), targetSingular+"id"+string(DB_IN))

	if resource != nil {
		resource.SaveXrefVersionCopies(targetResourceSID)
	}
}
