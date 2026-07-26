package registry

// This file implements the incremental population of the
// FullTreeTable/FullEntities tables described in sql.md. These tables
// are now the sole, authoritative store for entity properties (own,
// system, calculated, and cascaded/copied) - the old FullTree/Entities
// views and Props table are no longer read from or written to anywhere
// in the codebase (phase 2 of the sql.md migration). Every entity-
// creation site (Registry/Group/Resource/Meta/Version) calls
// FullEntityInsert() alongside its "real" table INSERT. Plain user-set
// properties are written directly by Entity.SetDBProperty() (called
// per-property during Save()'s NewObject traversal) and system-managed
// ones by Entity.SetSystemDBProperty()/FullTreeSyncProp(); FullSave(),
// called once at the very end of Save(), only needs to (re)compute
// this entity's calculated singleton attributes (xid, isdefault,
// RESOURCEid) and run whichever cascades are relevant given the
// entity's type (default-version-copy, xref prop/version-copy).

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// TEMP DEBUG: fullTreeDebug logs how long each fulltree.go func took, to
// help spot excessive/duplicate calls during a single logical operation.
// Remove once we're done profiling.
func fullTreeDebug(start time.Time, name string, extra string) {
	log.KPrintf("FullTree", "%s: %v  %s", name, time.Since(start), extra)
}

// FullEntityInsert adds a row to FullEntities for a newly-created
// Registry/Group/Resource/Meta/Version - called from the same places
// that insert into the corresponding "real" entity table, right after
// e's fields (DbSID, ParentSID, etc.) have been populated. It also
// writes e's write-once calculated ("IsCalcStatic") attributes here,
// since they're provably immutable for the rest of this entity's
// lifetime (see fullSaveCalcStaticInsert()'s doc comment) - so unlike
// FullSave(), which runs on every Save(), this only ever runs once,
// at creation.
func (e *Entity) FullEntityInsert() {
	defer log.Trace("FullTree", e.XID)()

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	// e.DbSID is always freshly generated for a brand-new entity, so
	// this REPLACE always inserts (never replaces) exactly 1 row.
	DoOne(e.tx, `
        REPLACE INTO FullEntities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, Path, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?,?,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Abstract, e.Path)

	e.fullSaveCalcStaticInsert()
}

// fullTreeWriteProp is the low-level writer for a single own
// (non-cascaded, non-calculated) FullTreeTable row. Deleting
// (propValue==nil) removes the row; otherwise it's REPLACEd with the
// new value. isSystem marks whether this is a plain user-set prop
// (SetDBProperty, false) or a system-managed one (SetSystemDBProperty,
// true). Never touches cascaded (IsDefaultVerCopy/IsXrefPropCopy/
// IsXrefVerCopy) or calculated (IsCalcStatic/IsCalcDynamic - see
// fullSaveCalcStaticInsert()/fullSaveVersionCalc()) rows - those are
// always written by their own dedicated code paths, never through
// here. Versions.AncestorID/CreatedAt and Metas.xRefPath/defaultVID are
// kept in sync by the FullTreeAncestor/FullTreeXref DB triggers
// (init.sql) whenever the corresponding own row is written/removed
// here.
func (e *Entity) fullTreeWriteProp(name string, propValue *string,
	propType string, docView bool, isSystem bool) {

	// defer log.Trace("FullTree", "%s/%s", e.Path, name)()

	if propValue == nil {
		// The prop row may or may not exist yet (e.g. deleting a prop
		// that was never set), so 0 or 1 rows is valid.
		DoZeroOne(e.tx, `
            DELETE FROM FullTreeTable
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
	DoOneTwo(e.tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsSystemProp, IsCalcStatic, IsCalcDynamic)
        VALUES(?,?,?,?,?,?,?,?, ?,?,?,?,?, false,false,false, ?,false,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Path, name, *propValue, propType, e.Abstract, docView,
		isSystem)
}

// fullTreeWritePropsBatch writes multiple own/system FullTreeTable rows
// for e in as few REPLACE INTO statements as possible, chunked at
// dbPropBatchChunkSize rows/statement as a max_allowed_packet safety
// net. isSystem marks whether these rows are system-managed
// (SaveSystemProps) or plain user-set (SetDBPropertyBatch/
// DoDBPropertyBatch) - same meaning as fullTreeWriteProp's isSystem
// param, just batched across multiple rows in one statement instead of
// one statement per row.
func (e *Entity) fullTreeWritePropsBatch(rows []dbPropRow, isSystem bool) {
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
				e.DbSID, e.UID, e.Path,
				row.Name, *row.Value, row.Type, e.Abstract, row.DocView)
		}

		Do(e.tx, `
            REPLACE INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
                IsSystemProp, IsCalcStatic, IsCalcDynamic)
            VALUES `+strings.Join(placeholders, ","), args...)
	}
}

// fullTreeDeletePropsBatch deletes multiple own/system FullTreeTable
// rows for e (identified by DB PropName) in as few
// "DELETE ... WHERE PropName IN (...)" statements as possible, chunked
// the same way as fullTreeWritePropsBatch. Matches fullTreeWriteProp's
// single-row delete filter (own rows only - never cascaded/copied
// ones), regardless of isSystem, since own vs. system PropNames never
// collide.
func (e *Entity) fullTreeDeletePropsBatch(names []string) {
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

		Do(e.tx, `
            DELETE FROM FullTreeTable
            WHERE eSID=? AND PropName IN (`+strings.Join(placeholders, ",")+`)
                  AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
                  AND IsXrefVerCopy=false`, args...)
	}
}

// FullTreeWriteOwnProp writes (or deletes, if propValue is nil) a
// plain user-set own property row for e - called by
// Entity.SetDBProperty() as part of Save()'s per-property traversal.
// Unlike FullTreeSyncProp, this never re-runs any cascade: Save()
// already calls FullSave() once at the very end of the same Save()
// call, which handles whatever cascade is relevant, so re-running it
// here for every single property would be redundant, wasted work.
func (e *Entity) FullTreeWriteOwnProp(name string, propValue *string,
	propType string, docView bool) {
	e.fullTreeWriteProp(name, propValue, propType, docView, false)
}

// FullTreeSyncProp keeps a single own (non-cascaded) FullTreeTable row
// in sync for a SYSTEM property that's written OUTSIDE of the normal
// Save()/FullSave() flow. As of the System/NewSystem buffering split,
// SetSystemDBProperty() no longer calls this directly (it buffers into
// NewSystem instead - see SaveSystemProps()); this is kept as a small,
// still-useful primitive for writing/deleting a single system-prop row
// plus its cascade in one call, in case some future caller needs an
// immediate (non-buffered) write.
func (e *Entity) FullTreeSyncProp(name string, propValue *string,
	propType string, docView bool) {

	defer log.Trace("FullTree", "%s/%s", e.XID, name)()

	e.fullTreeWriteProp(name, propValue, propType, docView, true)

	// If this is a Version, this prop change may also need to be
	// reflected in the owning Resource's IsDefaultVerCopy set (if this
	// Version happens to be the current default) - mark it for
	// deferred (re-)validation (see Tx.AddResourceToValidate()) rather
	// than running it immediately, since Save()'s own FullSave() already
	// marked it once and this out-of-band write can just piggyback on
	// that same deferred run.
	if e.Type == ENTITY_VERSION {
		v := e.Self.(*Version)
		e.tx.AddResourceToValidate(v.Resource, true, false)
	}
}

// SaveSystemProps flushes any system-prop changes buffered by
// SetSystemDBProperty() (into e.NewSystem) since the last flush. It's
// called once per cached entity at Tx-commit time (see
// tx.WriteCache()) and is a no-op if nothing was buffered. It diffs
// NewSystem against System so only props that actually changed get
// written to the DB, and - unlike FullTreeSyncProp()'s old behavior of
// re-running the default-version cascade on every single system-prop
// write - runs that cascade AT MOST ONCE per flush, no matter how many
// system props changed on this entity since the last flush. Deletes
// and inserts are each batched into (at most, if chunked) one
// statement via fullTreeDeletePropsBatch()/fullTreeWritePropsBatch(),
// instead of one round trip per changed prop.
func (e *Entity) SaveSystemProps() {
	if e.NewSystem == nil {
		return
	}

	defer log.Trace("FullTree", e.XID)()

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

		switch reflect.ValueOf(val).Kind() {
		case reflect.Slice, reflect.Map, reflect.Struct:
			dbVal = ""
		}

		dbValStr := fmt.Sprintf("%v", dbVal)
		insertRows = append(insertRows, dbPropRow{
			Name: dbName, Value: &dbValStr, Type: propType, DocView: docView,
		})
	}

	e.fullTreeDeletePropsBatch(deleteNames)
	e.fullTreeWritePropsBatch(insertRows, true)

	// Same idea as FullTreeSyncProp(): if e is a Version, mark the
	// owning Resource for deferred (re-)validation
	// (Tx.AddResourceToValidate()) so the Resource's mirrored props get
	// refreshed once, no matter how many system props changed on this
	// entity since the last flush.
	if e.Type == ENTITY_VERSION {
		if v, ok := e.Self.(*Version); ok {
			e.tx.AddResourceToValidate(v.Resource, true, false)
		}
	}
}

// FullSave is called at the very end of Entity.Save(). Save()'s own
// property traversal already wrote this entity's own (non-cascaded,
// non-calculated) rows directly into FullTreeTable via SetDBProperty()/
// FullTreeWriteOwnProp() as it walked NewObject, so FullSave() itself
// only needs to (re)compute whichever of this entity's calculated
// singleton attributes can actually change post-creation (just a
// Version's own isdefault - see fullSaveVersionCalc()'s doc comment;
// xid/Resource.isdefault/Version.RESOURCEid are write-once, handled by
// FullEntityInsert()/fullSaveCalcStaticInsert() instead) and mark the
// owning Resource for deferred (re-)validation - see
// Tx.AddResourceToValidate()'s doc comment for why this is deferred
// rather than run immediately here.
//
// fullSaveXrefCascade() (ENTITY_META) used to run eagerly right here,
// but that's wrong: Resource.UpsertMeta()'s xref-setting path (registry/
// resource.go) calls Version.JustDelete() in a loop to remove this
// Resource's own real Versions BEFORE it's done processing, and
// JustDelete() itself can trigger a Meta save (via Resource.Touch())
// partway through that loop - i.e. while some of this Resource's own
// real Version rows still exist in FullTreeTable. Running the xref
// cascade eagerly at that point built synthetic xref-version-copy rows
// whose Path collided with those still-present real rows (same
// PropName, same Path, different eSID - a FullTreeTable PRIMARY KEY
// violation). Deferring it via AddResourceToValidate()/
// Resource.ValidateResource() instead means it only actually runs once,
// at Registry.Validate() time (called from Tx.Validate()), which is
// always after all of this Resource's own Version deletes for the
// current request have completed.
//
// This relies on FullEntityInsert() having already been called (every
// entity-creation site does so unconditionally alongside its "real"
// table INSERT - see FullEntityInsert's doc comment), so there's no
// need to re-verify the FullEntities row exists here.
func (e *Entity) FullSave() {
	defer log.Trace("FullTree", "%s", e.XID)()

	switch e.Type {
	case ENTITY_VERSION:
		e.fullSaveVersionCalc()
		v := e.Self.(*Version)
		// onlyMetaChanged=false: this fires for ANY Version save,
		// including real content/ancestor changes (e.g. WillDelete()'s
		// ancestorid relinking), which need the full CheckAncestors()/
		// EnsureMaxVersions()/etc. checks - not just the meta-only
		// subset - before EnsureLatest() can correctly determine the
		// new "newest" Version.
		e.tx.AddResourceToValidate(v.Resource, false, false)

	case ENTITY_META:
		meta := e.Self.(*Meta)
		e.tx.AddResourceToValidate(meta.Resource, true, false)
	}
}

// fullSaveCalcStaticInsert writes e's write-once calculated attributes:
// xid (every entity type), Resource.isdefault (always "true" - per the
// old AllProps view, a Resource always shows the props of whichever
// Version is its default), and Version.RESOURCEid (e.g. "fileid",
// pointing at the owning Resource's UID). None of these can ever
// change after creation: an entity's UID/Path is immutable (no rename
// API - reusing an existing ID just errors instead of renaming), a
// Resource's isdefault is a hardcoded constant, and a Version's owning
// Resource never changes. So, unlike the genuinely-dynamic
// Version.isdefault (see fullSaveVersionCalc()), these only need to be
// computed once - here, called from FullEntityInsert() right after
// creation - and are never touched again by FullSave(). Marked
// IsCalcStatic=true so later reads/cascades can identify them and,
// e.g., exclude them when copying an entity's "real" props elsewhere.
func (e *Entity) fullSaveCalcStaticInsert() {
	defer log.Trace("FullTree", e.Path)()

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	// xid - every entity type. Plain single-row INSERT, always exactly
	// 1 (this is called once, at creation, on a brand-new eSID).
	DoOne(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, true,
               false, false, false, true, false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Path, "xid"+string(DB_IN), "/"+e.Path, e.Abstract)

	switch e.Type {
	case ENTITY_RESOURCE:
		// e is always a real, in-memory Resource, which always has a
		// parent Group, so e.ParentSID is never empty here.
		DoOne(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
                IsCalcStatic, IsCalcDynamic)
            VALUES(?,?,?,?,?,?,?,?, ?, 'true', 'boolean', ?, false,
                   false, false, false, true, false)`,
			e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
			e.DbSID, e.UID, e.Path, "isdefault"+string(DB_IN), e.Abstract)

	case ENTITY_VERSION:
		// e is always a real, in-memory Version, which always has a
		// parent Resource, so e.ParentSID is never empty here. The
		// owning Resource is guaranteed to already exist (e was just
		// created as one of its Versions), so this always inserts
		// exactly 1 row.
		DoOne(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
                IsCalcStatic, IsCalcDynamic)
            SELECT ?,?,?,?,?,?,?,?, CONCAT(r.Singular,?), r.UID, 'string', ?,
                   true, false, false, false, true, false
            FROM Resources AS r WHERE r.SID=?`,
			e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg,
			e.DbSID, e.UID, e.Path, "id"+string(DB_IN), e.Abstract,
			e.ParentSID)
	}
}

// FullTreeResyncOwnProps re-derives e's calculated FullTreeTable rows
// that can actually change post-creation. Used by code that clears
// system props directly, outside of Save()/FullSave() (e.g.
// ClearResourceSystemDBProperty, ClearEntitySystemDBProperties). e is
// assumed to already have gone through at least one FullEntityInsert()
// (so its write-once xid/isdefault/RESOURCEid rows already exist and
// never need re-deriving here). If e is a Version, re-adds its
// calculated (dynamic) isdefault row, and refreshes the owning
// Resource's IsDefaultVerCopy set in case this Version is the current
// default (Save()'s own cascade already ran before this out-of-band
// write happened, so it won't run again).
func FullTreeResyncOwnProps(e *Entity) {
	defer log.Trace("FullTree", e.XID)()

	if e.Type == ENTITY_VERSION {
		e.fullSaveVersionCalc()
		v := e.Self.(*Version)
		e.tx.AddResourceToValidate(v.Resource, true, false)
	}
}

// fullSaveVersionCalc (re)computes the calculated 'isdefault' attribute
// for a (real, non-xref-synthetic) Version - the only Version-level
// calculated value that can actually change post-creation (xid and
// RESOURCEid are write-once - see fullSaveCalcStaticInsert(), called
// once from FullEntityInsert() instead). e is always the real,
// in-memory Entity (never just a SID/raw row) - either the one Save()
// is currently running for, or the one FullTreeResyncOwnProps() is
// resyncing out-of-band.
func (e *Entity) fullSaveVersionCalc() {
	defer log.Trace("FullTree", e.Path)()

	// Own scoped delete (rather than relying on some shared blanket
	// delete having already run) since this is the only calculated
	// value left that needs recomputing on every relevant Save(). At
	// most 1 IsCalcDynamic row ever exists per Version (the isdefault
	// row inserted below); 0 if this is the first time this func runs
	// for this Version.
	DoZeroOne(e.tx, `DELETE FROM FullTreeTable WHERE eSID=? AND IsCalcDynamic=true`,
		e.DbSID)

	// isdefault - true only if this Version is the owning Resource's
	// current default (via its Meta.defaultVID), or - for a Resource
	// with no defaultVID set but which is itself an xref source - if
	// it matches the xref target's default. In the common non-xref
	// case this just checks m.defaultVID. The owning Resource's Meta
	// is guaranteed to exist, so this always inserts exactly 1 row.
	DoOne(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?,?,?,?,?,?,?,?, ?,
               IF(m.defaultVID IS NOT NULL AND ?=m.defaultVID, 'true', 'false'),
               'boolean', ?, true, false, false, false, false, true
        FROM Metas AS m WHERE m.ResourceSID=?`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, "isdefault"+string(DB_IN), e.UID, e.Abstract,
		e.ParentSID)
}

// fullSaveDefaultVerCascade refreshes the IsDefaultVerCopy=true rows on
// r so they mirror whatever Version is currently r's default. Called
// whenever a Version is saved (in case it's the current default) or a
// Meta is saved (in case meta.defaultversionid just changed). r is nil
// if r is nil (no Resource to cascade) - a no-op in that case.
//
// r is the actual, strongly-typed *Resource (not just its SID) so we
// can follow its own in-memory fields plus tx.GetMeta(r)/tx.GetVersion(
// r,...) to find the current default Version's SID without any DB
// round-trip whenever this transaction already has them loaded.
func fullSaveDefaultVerCascade(r *Resource) {
	if r == nil {
		return
	}

	defer log.Trace("FullTree", r.XID)()

	resourceSID := r.DbSID

	Do(r.tx, `
        DELETE FROM FullTreeTable WHERE eSID=? AND IsDefaultVerCopy=true`,
		resourceSID)

	// The Resource (and its Meta) may have been deleted earlier in this
	// same Tx after being marked for a cascade run (e.g. all of its
	// Versions were deleted, which per business rules deletes the
	// Resource itself). If so there's nothing left to cascade.
	meta, xErr := r.FindMeta(false, FOR_READ)
	Must(xErr)
	if meta == nil {
		return
	}

	// Grab the default version. If there is none defined yet then
	// call EnsureLatest (if no xref) just so the processing continues.
	// If the default version needs to change based on later processing,
	// e.g. ancestor processing, then the a subsequent call to EnsureLatest
	// in resource.ValidateResource() should fix it. And hopefully, this
	// func will be called again to fix-up the default version props for
	// the resource.
	ver, xErr := r.GetDefault(FOR_READ)
	Must(xErr)
	if !r.IsXref() && ver == nil {
		Must(r.EnsureLatest())
		ver, xErr = r.GetDefault(FOR_READ)
		Must(xErr)
	}

	if ver == nil {
		// No real default Version - this Resource may be an xref
		// source with no Versions of its own, in which case its
		// "current default" is really the xref target's current
		// default, copied in as a synthetic Version by
		// fullSaveXrefVersionCopies(). Copy from THAT synthetic
		// FullTreeTable eSID instead of Props.
		tResults := Query(r.tx, `
            SELECT v.SID FROM Metas AS srcM
            JOIN Resources AS tr ON (tr.RegistrySID=srcM.RegistrySID AND
                                      tr.Path=srcM.xRefPath)
            JOIN Metas AS m ON (m.ResourceSID=tr.SID)
            JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                    v.UID=m.defaultVID)
            WHERE srcM.ResourceSID=?`, resourceSID)
		tRow := tResults.NextRow()
		tResults.Close()
		if tRow == nil {
			return
		}
		targetDefVerSID := NotNilString(tRow[0])
		synthESID := fmt.Sprintf("-%s-%s", resourceSID, targetDefVerSID)

		Do(r.tx, `
            REPLACE INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   true, false, false
            FROM FullTreeTable WHERE eSID=? AND IsXrefVerCopy=true
                  AND IsCalcStatic=false AND IsCalcDynamic=false`,
			r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID,
			r.DbSID, r.UID, r.Path, r.Abstract, synthESID)
		return
	}

	Do(r.tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               true, false, false
        FROM FullTreeTable
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false
              AND IsCalcDynamic=false`,
		r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID, r.DbSID,
		r.UID, r.Path, r.Abstract, ver.DbSID)

	// Fix up isdefault on every one of this Resource's OWN Versions -
	// they were each set based on their own state at their own last
	// Save(), so if the default just moved to a different Version, the
	// old default's row (and the new one's, if it wasn't the one that
	// triggered this call) would otherwise go stale.
	Do(r.tx, `
        UPDATE FullTreeTable AS ft
        JOIN Versions AS v ON (v.SID=ft.eSID)
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        SET ft.PropValue = IF(v.UID=m.defaultVID, 'true', 'false')
        WHERE v.ResourceSID=? AND ft.PropName=?`,
		resourceSID, "isdefault"+string(DB_IN))
}

// fullSaveXrefCascade refreshes the IsXrefPropCopy (this Meta's own
// copied meta.* attrs) and IsXrefVerCopy (synthetic Version rows) sets
// for a source Meta entity whose xref may have just been set, changed,
// or cleared.
// fullSaveXrefCascade refreshes the IsXrefPropCopy (this Meta's own
// copied meta.* attrs) and IsXrefVerCopy (synthetic Version rows) sets
// for a source Meta entity (e) whose xref may have just been set,
// changed, or cleared. e is always the real, in-memory Meta - either
// the one Save() is currently running for, or (via xref fan-out) one
// resolved through Registry.FindResourceBySID()+FindMeta() rather than
// a raw row.
func (e *Entity) fullSaveXrefCascade() {
	defer log.Trace("FullTree", e.Path)()

	e.fullSaveXrefCascadeDelete()
	e.fullSaveXrefCascadeInsert()
}

// fullSaveXrefCascadeDelete clears this Meta's stale IsXrefPropCopy
// rows and its Resource's stale IsXrefVerCopy rows, from whatever the
// PREVIOUS xref state was. Split out from fullSaveXrefCascadeInsert so
// FullSave() can run ALL deletes (this plus fullSaveOwnPropsDelete)
// before either insert runs - see FullSave()'s ENTITY_META handling.
func (e *Entity) fullSaveXrefCascadeDelete() {
	// e is always a real, in-memory Meta, which always has a parent
	// Resource, so e.ParentSID is never empty here.
	Do(e.tx, `DELETE FROM FullTreeTable WHERE eSID=? AND IsXrefPropCopy=true`,
		e.DbSID)
	Do(e.tx, `
        DELETE FROM FullTreeTable
        WHERE RegSID=? AND ParentSID=? AND IsXrefVerCopy=true`,
		e.Registry.DbSID, e.ParentSID)
	Do(e.tx, `
        DELETE FROM FullEntities
        WHERE RegSID=? AND ParentSID=? AND IsXrefVerCopy=true`,
		e.Registry.DbSID, e.ParentSID)
}

// fullSaveXrefCascadeInsert (re)inserts this Meta's IsXrefPropCopy and
// IsXrefVerCopy rows based on the CURRENT xref state. Assumes
// fullSaveXrefCascadeDelete (and, for the own-props exclusion to work
// correctly, fullSaveOwnPropsDelete) have already run.
func (e *Entity) fullSaveXrefCascadeInsert() {
	results := Query(e.tx, `
        SELECT xRefPath FROM Metas WHERE SID=?`, e.DbSID)
	row := results.NextRow()
	results.Close()
	if row == nil || NotNilString(row[0]) == "" {
		return
	}
	xRefPath := NotNilString(row[0])

	// Resolve the target live, by RegistrySID+Path (Path alone isn't
	// unique across the whole DB, only within one Registry) - never by
	// a cached SID, so this always reflects reality even if the
	// target didn't exist (or existed under a different SID) the last
	// time this ran.
	tResults := Query(e.tx, `
        SELECT m.SID, m.ResourceSID, r.Singular FROM Resources AS r
        JOIN Metas AS m ON (m.ResourceSID=r.SID)
        WHERE r.RegistrySID=? AND r.Path=?`, e.Registry.DbSID, xRefPath)
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
	Do(e.tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               false, true, false
        FROM FullTreeTable
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false
              AND IsCalcDynamic=false
              AND PropName NOT IN (?, ?) AND LEFT(PropName,1)<>'#'`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, e.Abstract, targetMetaSID,
		"xref"+string(DB_IN), targetSingular+"id"+string(DB_IN))

	if resource != nil {
		fullSaveXrefVersionCopies(resource, targetResourceSID)
	}
}

// fullSaveXrefVersionCopies (re)creates the synthetic FullEntities/
// FullTreeTable Version rows for srcResource (a real, in-memory
// *Resource - either e.Self.(*Meta).Resource in the direct-Save() case,
// or one resolved via Registry.FindResourceBySID() in the xref fan-out
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
func fullSaveXrefVersionCopies(srcResource *Resource, targetResourceSID string) {
	if srcResource == nil {
		return
	}

	defer log.Trace("FullTree", "%s,%s", srcResource.Path, targetResourceSID)()

	sourceResourceSID := srcResource.DbSID
	synthAbstract := srcResource.Abstract + string(DB_IN) + "versions"

	// Idempotent: this is called both from fullSaveXrefCascadeInsert
	// (which already cleared out ALL of this source's xref-version
	// rows first) and directly from fullSaveXrefFanOutForTarget
	// (which does not) - so clear out just the synthetic versions that
	// correspond to the target's CURRENT version set before
	// recreating them, or a second Save() of the same target Version
	// would hit a duplicate-key error here.
	Do(srcResource.tx, `
        DELETE ft FROM FullTreeTable AS ft
        JOIN Versions AS v ON (ft.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	Do(srcResource.tx, `
        DELETE fe FROM FullEntities AS fe
        JOIN Versions AS v ON (fe.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)

	// One FullEntities row per target Version, all at once.
	Do(srcResource.tx, `
        REPLACE INTO FullEntities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, Path, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID, ?,
               CONCAT(?, '/versions/', v.UID), true
        FROM Versions AS v WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, synthAbstract, srcResource.Path,
		targetResourceSID)

	// Copy each target Version's own props onto its corresponding
	// synthetic eSID, for every current Version at once (excluding the
	// target's own "xref" - Versions never have one, but kept for
	// parity with the old per-row exclusion).
	Do(srcResource.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ft.PropName, ft.PropValue, ft.PropType, ?, false,
               false, false, true
        FROM Versions AS v
        JOIN FullTreeTable AS ft ON (ft.eSID=v.SID)
        WHERE v.ResourceSID=? AND ft.IsDefaultVerCopy=false
              AND ft.IsXrefPropCopy=false AND ft.IsXrefVerCopy=false
              AND ft.IsCalcStatic=false AND ft.IsCalcDynamic=false
              AND ft.PropName<>?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path, synthAbstract,
		targetResourceSID, "xref"+string(DB_IN))

	// Calculated attrs for every synthetic version at once: xid and
	// RESOURCEid (using the SOURCE resource's singular/UID, since
	// that's every synthetic version's effective parent) are static -
	// wholesale recreated here only because the whole synthetic-
	// version set itself is being recreated (the xref pointer moved),
	// not because they individually change; isdefault is genuinely
	// dynamic (mirrors the target's own per-Version isdefault).
	Do(srcResource.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, CONCAT('/', ?, '/versions/', v.UID), 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"xid"+string(DB_IN), srcResource.Path, synthAbstract, targetResourceSID)

	Do(srcResource.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               CONCAT(r.Singular, ?), r.UID, 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v
        JOIN Resources AS r ON (r.SID=?)
        WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"id"+string(DB_IN), synthAbstract, sourceResourceSID,
		targetResourceSID)

	Do(srcResource.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, IF(m.defaultVID=v.UID, 'true', 'false'), 'boolean', ?,
               false, false, false, true, false, true
        FROM Versions AS v
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"isdefault"+string(DB_IN), synthAbstract, targetResourceSID)
}

// fullSaveXrefFanOutForTarget re-runs fullSaveXrefCascade and the
// synthetic-version-copy refresh for every OTHER source Resource that
// xrefs r - used whenever either r's Meta or one of r's Versions
// (something that makes r someone else's xref target) is saved. r is
// the real, in-memory *Resource whose Meta/Version was just saved
// (always available at the FullSave() call site as meta.Resource or
// v.Resource). This combines what used to be two separate functions
// (fullSaveXrefFanOutForTargetMeta/fullSaveXrefFanOutForTargetVersion)
// - they were always called back-to-back from the same runCascade()
// call site, each running its own copy of the identical "who xrefs me"
// query and its own fullSaveDefaultVerCascade(sourceResource) call, so
// merging them halves both the query count and the redundant
// per-source default-version cascade work. Each source discovered here
// is resolved to its own real *Resource/*Meta via
// Registry.FindResourceBySID()/FindMeta() (cache-checked, so repeat
// fan-out hits for the same source within one Tx are free) rather than
// a raw fullEntityLookup() row.
func fullSaveXrefFanOutForTarget(r *Resource) {
	if r == nil {
		return
	}

	defer log.Trace("FullTree", r.XID)()

	results := Query(r.tx, `
        SELECT ResourceSID FROM Metas WHERE RegistrySID=? AND xRefPath=?`,
		r.Registry.DbSID, r.Path)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceResourceSID := NotNilString(row[0])
		sourceResource, xErr := r.tx.Registry.FindResourceBySID(
			sourceResourceSID, FOR_WRITE)
		if xErr != nil || sourceResource == nil {
			continue
		}
		sourceMeta, xErr := sourceResource.FindMeta(false, FOR_WRITE)
		if xErr != nil || sourceMeta == nil {
			continue
		}
		sourceMeta.fullSaveXrefCascade()
		fullSaveXrefVersionCopies(sourceResource, r.DbSID)
		fullSaveDefaultVerCascade(sourceResource)
	}
}

// NOTE: cleaning up stale xref-source mirror rows when a target
// Resource is deleted is handled entirely by ResourcesTrigger (see
// init.sql) now, not here - that trigger fires uniformly for every
// deletion path (direct Resource delete, whole-Group delete, whole-
// Registry delete), so there's no Go-level call site to remember.

// DiffFullTree is a basic diff-check (step 4, partial, per sql.md/plan
// scope): it compares FullTreeTable's contents against what the FullTree
// view produces for the given registry and returns a human-readable
// summary of any mismatches. It does NOT modify anything and is not on
// any read path - it's meant to be called ad hoc (e.g. from a test or a
// debug tool) while validating FullSave()'s incremental logic.
func DiffFullTree(tx *Tx, regSID string) string {
	viewRows := fullTreeRowsFrom(tx, "FullTree", regSID)
	tableRows := fullTreeRowsFrom(tx, "FullTreeTable", regSID)

	missing := []string{}
	for k, v := range viewRows {
		if tv, ok := tableRows[k]; !ok {
			missing = append(missing, fmt.Sprintf("missing in table: %s=%s", k, v))
		} else if tv != v {
			missing = append(missing, fmt.Sprintf(
				"value mismatch: %s view=%q table=%q", k, v, tv))
		}
	}
	for k, v := range tableRows {
		if _, ok := viewRows[k]; !ok {
			missing = append(missing, fmt.Sprintf("extra in table: %s=%s", k, v))
		}
	}

	if len(missing) == 0 {
		return ""
	}
	return strings.Join(missing, "\n")
}

func fullTreeRowsFrom(tx *Tx, table string, regSID string) map[string]string {
	out := map[string]string{}
	results := Query(tx, `
        SELECT Path, PropName, PropValue FROM `+table+` WHERE RegSID=?`, regSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		key := NotNilString(row[0]) + "|" + NotNilString(row[1])
		out[key] = NotNilString(row[2])
	}
	return out
}
