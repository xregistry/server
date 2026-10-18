package registry

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type EntityExtensions struct{}

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
	log.FuncPrintf("tx: %s WasTouch: %s/%s", e.Tx.UUID, e.Singular, e.UID)
	return e.EpochSet
}

func (e *Entity) Touch() (bool, *XRError) {
	log.FuncPrintf("tx: %s Touch: %s/%s", e.Tx.UUID, e.Singular, e.UID)

	// See if it's already been modified (and saved) this Tx, if so exit
	if e.ModSet && e.EpochSet {
		return false, nil
	}

	if _, xErr := e.Lock(); xErr != nil {
		return false, xErr
	}

	e.EnsureNewObject()
	return true, nil
}

// LockResourceFamily locks (FOR UPDATE) just the Resource's own row in
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
func LockResourceFamily(tx *Tx, resourceSID string) *XRError {
	return tx.Backend.LockResourceBySID(tx, resourceSID)

	/*
		lock := func(query string) {
			results := Q uery(tx, query, resourceSID)
			results.Close()
		}

		lock(`SELECT SID FROM Resources WHERE SID=? FOR UPDATE`)
	*/

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

	return nil
}

// lockEntityFamily locks (FOR UPDATE) the full Resource+Meta+Versions
// family that ent belongs to, if any - see LockResourceFamily()'s and
// Entity.Lock()'s doc comments for why. Safe to call with a nil ent or
// a non-Resource/Meta/Version entity (no-op).
//
// IMPORTANT: LockResourceFamily() only takes real DB-level "FOR UPDATE"
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
// really DB-locked - so after LockResourceFamily() runs, we must also
// walk this Tx's cache and upgrade (Refresh(FOR_WRITE)) every other
// cached Resource/Meta/Version belonging to this same family, or those
// guards (and any code that just trusts a cached Meta/Version's
// AccessMode) can silently keep using stale FOR_READ data even though
// the DB row underneath was just real-locked. This was the root cause
// of a recurring "<attr> is nil" panic under TestMiscConcurrency.
func LockEntityFamily(tx *Tx, ent *Entity) *XRError {
	if ent == nil {
		return nil
	}
	resourceSID := ""
	switch ent.Type {
	case ENTITY_RESOURCE:
		resourceSID = ent.DbSID
	case ENTITY_META, ENTITY_VERSION:
		resourceSID = ent.ParentSID
	}
	if resourceSID == "" {
		return nil
	}
	if xErr := LockResourceFamily(tx, resourceSID); xErr != nil {
		return xErr
	}

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
			// genuinely DB-locked via LockResourceFamily() above) without
			// reloading, so we don't clobber the pending edits.
			cached.AccessMode = FOR_WRITE
			continue
		}
		log.FuncPrintf("tx: %s lockEntityFamily: upgrading stale cached %q "+
			"(eSID=%s) to FOR_WRITE now that its family's DB rows are locked",
			tx.UUID, cached.XID, cached.DbSID)
		if xErr := cached.Refresh(FOR_WRITE); xErr != nil {
			return xErr
		}
	}

	return nil
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) eSetSave(path string, val any) *XRError {
	defer log.Trace("tx: %s %s=%v", e.Tx.UUID, path, val)()

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

// This is really just an internal Setter used for testing.
// It'll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) *XRError {
	defer log.Trace("tx: %s %s: %s=%v", e.Tx.UUID, e.DbSID, pp.UI(), val)()
	defer func() {
		if log.IsFuncVerbose() {
			log.Printf("tx: %s exit: e.Object:\n%s", e.Tx.UUID, ToJSON(e.Object))
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
		if tmpxErr := e.Refresh(FOR_READ); tmpxErr != nil {
			panic(tmpxErr)
		}
	}

	return xErr
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
func (e *Entity) ClearResourceSystemDBProperty(pps ...*PropPath) *XRError {
	defer log.Trace("tx: %s %d props", e.Tx.UUID, len(pps))()

	if len(pps) == 0 {
		return nil
	}

	r, ok := e.Self.(*Resource)
	PanicIf(!ok, "%s isn't a Resource", e.XID)

	vers, xErr := r.GetVersions(FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	for _, v := range vers {
		for _, pp := range pps {
			v.SetSystemDBProperty(pp, nil)
		}
	}

	return nil
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
	defer log.Trace("tx: %s %s=%v", e.Tx.UUID, pp, val)()

	PanicIf(pp.UI() == "", "pp is empty")

	/* DUG FT
	name := pp.DB()

	_, propsMap := e.GetPropsOrdered()
	specProp, ok := propsMap[pp.Top()]
	if ok && specProp.Internals != nil {
		// Any prop with "dontStore"=true we skip
		if specProp.Internals.dontStore {
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
//   - Use AddCalcProps() to fill in any missing props (eg Entity's GetFn())
//   - Call that passed-in 'fn' to serialize each prop but in the right order
//     as defined by the entity's GetPropsOrdered()
func (e *Entity) SerializeProps(
	fn func(*Entity, string, any, *Attribute) *XRError) *XRError {
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	info := e.GetRequestInfo()
	daObj := e.AddCalcProps()
	attrs := e.GetAttributes(e.Object)

	if log.IsFuncVerbose() {
		log.Printf("tx: %s SerProps.Entity: %s", e.Tx.UUID, ToJSON(e))
		log.Printf("tx: %s SerProps.Obj: %s", e.Tx.UUID, ToJSON(e.Object))
		log.Printf("tx: %s SerProps daObj: %s", e.Tx.UUID, ToJSON(daObj))
		log.Printf("tx: %s SerProps attrs:\n%s", e.Tx.UUID, ToJSON(attrs))
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

		log.FuncPrintf("tx: %s Ser prop(%s): %q", e.Tx.UUID, e.XID, name)

		attr, ok := attrs[name]
		if !ok {
			log.FuncPrintf("tx: %s  skipping %q, no attr", e.Tx.UUID, name)
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
					// log.Printf("tx: %s Ser*ext(%s): %q", e.Tx.UUID, e.XID,
					//  objKey)

					if xErr := fn(e, objKey, val, attr); xErr != nil {
						return xErr
					}
				}
			}
			continue
		}

		if name[0] == '$' || (prop.Internals != nil && prop.Internals.AlwaysSerialize) {
			log.FuncPrintf("tx: %s forced serialization of %q", e.Tx.UUID, name)
			if xErr := fn(e, name, nil, attr); xErr != nil {
				return xErr
			}
			continue
		}

		// Should be a no-op for Resources.
		if val, ok := daObj[name]; ok {
			log.FuncPrintf("tx: %s val: %v", e.Tx.UUID, val)
			if !IsNil(val) {
				xErr := fn(e, name, val, attr)
				if xErr != nil {
					return xErr
				}
			}
			delete(daObj, name)
		} else {
			log.FuncPrintf("tx: %s no value for %q", e.Tx.UUID, name)
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
	defer log.Trace("tx: %s %s", e.Tx.UUID, e.XID)()

	PanicIf(e.AccessMode != FOR_WRITE, "%q isn't FOR_WRITE", e.XID)

	if log.IsFuncVerbose() {
		log.Printf("tx: %s NewObject:\n%s", e.Tx.UUID, ToJSON(e.NewObject))
		// ShowStack()
	}

	xErr := e.Tx.Backend.SaveEntity(e)
	if xErr != nil {
		return xErr
	}

	// Set e.Object to e.NewObject, skipping all 'nil' attributes
	e.Object = map[string]any{}
	for k, v := range e.NewObject {
		if !IsNil(v) {
			e.Object[k] = v
		}
	}
	e.NewObject = nil

	// Update tx-wide Validate lists based on e.Type - processed at end of tx
	if e.Type == ENTITY_GROUP {
		e.Tx.AddGroupToValidate(e.Self.(*Group))
	} else if e.Type == ENTITY_VERSION {
		// onlyMetaChanged=false because we need the full validation code
		e.Tx.AddResourceToValidate(e.Self.(*Version).Resource, false, false)
	} else if e.Type == ENTITY_META {
		e.Tx.AddResourceToValidate(e.Self.(*Meta).Resource, true, false)
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
		if prop.Internals != nil && prop.Internals.GetFn != nil {
			// Only generate/set the value if it's not already set
			if _, ok := mat[prop.Name]; !ok {
				if val := prop.Internals.GetFn(e); !IsNil(val) {
					// Only write it if we have a value
					// log.Printf("tx: %s Added calc prop: %q",
					// e.Tx.UUID, prop.Name)
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
			if attr.InType(e.Type) && attr.Internals != nil && attr.Internals.UpdateFn != nil {
				if xErr := attr.Internals.UpdateFn(e); xErr != nil {
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
			// log.Printf("tx: %s Key: %q", e.Tx.UUID, k)
			// ShowStack()
			return xErr
		}
	}
	return nil
}

// SaveSystemProps flushes any system-prop changes buffered by
// SetSystemDBProperty() (into e.NewSystem) since the last flush. It's
// called once per cached entity at Tx-commit time (see
// tx.WriteCache()) and is a no-op if nothing was buffered. It diffs
// NewSystem against System so only props that actually changed get
// written to the DB.
func (e *Entity) SaveSystemProps() *XRError {
	if e.NewSystem == nil {
		return nil
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
		return nil
	}

	xErr := e.Tx.Backend.SaveChangedSystemProps(e, changed)
	if xErr != nil {
		return xErr
	}

	// If this is a Version, make sure we fully validate its owning Resource
	if e.Type == ENTITY_VERSION {
		if v, ok := e.Self.(*Version); ok {
			e.Tx.AddResourceToValidate(v.Resource, true, false)
		}
	}

	return nil
}
