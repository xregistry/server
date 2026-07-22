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
// e's fields (DbSID, ParentSID, etc.) have been populated.
func (e *Entity) FullEntityInsert() {
	log.KPrintf("FullTree", ">Enter: FullEntityInsert(%s)", e.Path)
	defer log.KPrintf("FullTree", "<Exit: FullEntityInsert")
	defer fullTreeDebug(time.Now(), "FullEntityInsert", "")

	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	Do(e.tx, `
        REPLACE INTO FullEntities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, Path, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?,?,false)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Abstract, e.Path)
}

// fullTreeWriteProp is the low-level writer for a single own
// (non-cascaded) FullTreeTable row. Deleting (propValue==nil) removes
// the row; otherwise it's REPLACEd with the new value. isSystem/
// isCalculated mark which kind of "own" row this is (mutually
// exclusive with each other and with the cascade markers) so later
// reads/deletes can distinguish plain user-set props (SetDBProperty,
// both false), system-managed ones (SetSystemDBProperty, isSystem
// true), and calculated singletons (xid/isdefault/RESOURCEid,
// isCalculated true). Never touches cascaded (IsDefaultVerCopy/
// IsXrefPropCopy/IsXrefVerCopy) rows. Versions.AncestorID/CreatedAt and
// Metas.xRefSID/defaultVID are kept in sync by the FullTreeAncestor/
// FullTreeXref DB triggers (init.sql) whenever the corresponding own
// row is written/removed here.
func (e *Entity) fullTreeWriteProp(name string, propValue *string,
	propType string, docView bool, isSystem bool, isCalculated bool) {

	// log.KPrintf("FullTree", ">Enter: fullTreeWriteProp(%s/%s)", e.Path,name)
	// defer log.KPrintf("FullTree", "<Exit: fullTreeWriteProp")
	// defer fullTreeDebug(time.Now(), "fullTreeWriteProp","")

	if propValue == nil {
		Do(e.tx, `
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

	Do(e.tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsSystemProp, IsCalculated)
        VALUES(?,?,?,?,?,?,?,?, ?,?,?,?,?, false,false,false, ?,?)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Path, name, *propValue, propType, e.Abstract, docView,
		isSystem, isCalculated)
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
	e.fullTreeWriteProp(name, propValue, propType, docView, false, false)
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

	log.KPrintf("FullTree", ">Enter: FullTreeSyncProp(%s,%s)", e.XID, name)
	defer log.KPrintf("FullTree", "<Exit: FullTreeSyncProp")
	defer fullTreeDebug(time.Now(), "FullTreeSyncProp", "")

	e.fullTreeWriteProp(name, propValue, propType, docView, true, false)

	// If this is a Version, this prop change may also need to be
	// reflected in the owning Resource's IsDefaultVerCopy set (if this
	// Version happens to be the current default) - re-run that cascade
	// since Save()'s own fullSaveDefaultVerCascade already ran (and
	// won't run again) before this out-of-band write happened.
	if e.Type == ENTITY_VERSION {
		v := e.Self.(*Version)
		if v.Resource != nil && fullVersionIsCurrentDefault(v) {
			fullSaveDefaultVerCascade(e.tx, v.Resource)
		}
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
// system props changed on this entity since the last flush.
func (e *Entity) SaveSystemProps() {
	if e.NewSystem == nil {
		return
	}

	log.KPrintf("FullTree", ">Enter: SaveSystemProps(%s)", e.XID)
	defer log.KPrintf("FullTree", "<Exit: SaveSystemProps")
	defer fullTreeDebug(time.Now(), "SaveSystemProps", "")

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
	for name, val := range changed {
		docView := true
		if specProp, ok := propsMap[name]; ok && specProp.internals != nil &&
			specProp.internals.noDocView {
			docView = false
		}
		dbName := NewPPP(name).DB()

		if IsNil(val) {
			e.fullTreeWriteProp(dbName, nil, "", docView, true, false)
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
		e.fullTreeWriteProp(dbName, &dbValStr, propType, docView, true, false)
	}

	// Same idea as FullTreeSyncProp(): if e is a Version and happens to
	// be the Resource's current default, re-run the default-version-
	// copy cascade once so the Resource's mirrored props reflect the
	// change(s) above - but only ONCE per flush, not once per prop.
	if e.Type == ENTITY_VERSION {
		if v, ok := e.Self.(*Version); ok && v.Resource != nil &&
			fullVersionIsCurrentDefault(v) {
			fullSaveDefaultVerCascade(e.tx, v.Resource)
		}
	}
}

// FullSave is called at the very end of Entity.Save(). Save()'s own
// property traversal already wrote this entity's own (non-cascaded,
// non-calculated) rows directly into FullTreeTable via SetDBProperty()/
// FullTreeWriteOwnProp() as it walked NewObject, so FullSave() itself
// only needs to (re)compute this entity's calculated singleton
// attributes (xid, isdefault, RESOURCEid) and kick off whichever
// cascades are relevant given the entity's type (default-version-copy,
// xref prop/version-copy).
//
// metaDefaultChanged is only meaningful when e.Type==ENTITY_META: it's
// true if this specific Save() call actually changed defaultversionid
// or xref (computed by Save() itself from a local pre-Save snapshot,
// since e.OriginObject isn't reliable for this - see Save()'s comment).
//
// This relies on FullEntityInsert() having already been called (every
// entity-creation site does so unconditionally alongside its "real"
// table INSERT - see FullEntityInsert's doc comment), so there's no
// need to re-verify the FullEntities row exists here.
func (e *Entity) FullSave(metaDefaultChanged bool) {
	log.KPrintf("FullTree", ">Enter: FullSave(%s,%v)", e.XID, metaDefaultChanged)
	defer log.KPrintf("FullTree", "<Exit: FullSave")
	defer fullTreeDebug(time.Now(), "FullSave", "")

	e.fullSaveOwnProps()

	switch e.Type {
	case ENTITY_VERSION:
		e.fullSaveVersionCalc()
		// Only refresh the Resource's IsDefaultVerCopy set if this
		// Version is actually the one currently marked as the
		// Resource's default - a Save() of any other (non-default)
		// Version can't possibly change what's mirrored there, so
		// there's no need to pay for the cascade's DELETE/REPLACE/
		// UPDATE every time any Version is touched.
		v := e.Self.(*Version)
		if v.Resource != nil && fullVersionIsCurrentDefault(v) {
			fullSaveDefaultVerCascade(e.tx, v.Resource)
		}
		e.fullSaveXrefFanOutForTargetVersion(v.Resource)

	case ENTITY_RESOURCE:
		e.fullSaveResourceIsDefault()

	case ENTITY_META:
		e.fullSaveXrefCascade()
		// Only refresh the Resource's IsDefaultVerCopy set if this
		// Save() actually changed defaultversionid or xref - those
		// are the only two Meta attrs that can affect it, so any
		// other Meta prop change (readonly, compatibility, etc.)
		// doesn't need to re-run the cascade.
		meta := e.Self.(*Meta)
		if metaDefaultChanged {
			fullSaveDefaultVerCascade(e.tx, meta.Resource)
		}
		e.fullSaveXrefFanOutForTargetMeta(meta.Resource)
	}
}

// fullVersionIsCurrentDefault reports whether v is currently its
// Resource's default Version (i.e. Metas.defaultVID == v.UID). It
// first checks whether this transaction already has the owning
// Resource's Meta loaded in-memory (tx.GetMeta) - if so, this is a
// zero-DB-query check, reading defaultversionid straight out of
// meta.Object (a root-level attr, so no need for Get()/GetAsString()'s
// path-parsing). Otherwise it falls back to a lightweight existence
// check (far cheaper than running the full cascade unconditionally).
func fullVersionIsCurrentDefault(v *Version) bool {
	log.KPrintf("FullTree", ">Enter: fullVersionIsCurrentDefault(%s)", v.XID)
	defer log.KPrintf("FullTree", "<Exit: fullVersionIsCurrentDefault")
	defer fullTreeDebug(time.Now(), "fullVersionIsCurrentDefault", "")

	meta := v.Resource.MustFindMeta(false, FOR_READ)
	// DUG FT
	// if meta := v.tx.GetMeta(v.Resource); meta != nil {
	defVID, _ := meta.Object["defaultversionid"].(string)
	return defVID == v.UID
	// }
	/*
			results := Query(v.tx, `
		        SELECT 1 FROM Metas WHERE ResourceSID=? AND defaultVID=?`,
				v.Resource.DbSID, v.UID)
			defer results.Close()
			return results.NextRow() != nil
	*/
}

// fullSaveOwnProps (re)computes an entity's calculated 'xid' singleton
// row. e is the real, in-memory Entity Save() is currently running for
// (or, via FullTreeResyncOwnProps(), the real Entity being resynced
// out-of-band) - never just a SID/raw row, so this is a plain *Entity
// method.
func (e *Entity) fullSaveOwnProps() {
	log.KPrintf("FullTree", ">Enter: fullSaveOwnProps(%s)", e.Path)
	defer log.KPrintf("FullTree", "<Exit: fullSaveOwnProps")
	defer fullTreeDebug(time.Now(), "fullSaveOwnProps", "")

	e.fullSaveOwnPropsDelete()
	e.fullSaveOwnPropsInsert()
}

// fullSaveOwnPropsDelete removes an entity's calculated (IsCalculated)
// rows from FullTreeTable, ahead of fullSaveOwnPropsInsert/
// fullSaveVersionCalc/fullSaveResourceIsDefault recomputing them. Own
// (plain user-set or system) rows are never touched here - they're
// maintained directly by SetDBProperty()/SetSystemDBProperty() as they
// change, not resynced wholesale on every Save().
func (e *Entity) fullSaveOwnPropsDelete() {
	Do(e.tx, `DELETE FROM FullTreeTable WHERE eSID=? AND IsCalculated=true`,
		e.DbSID)
}

// fullSaveOwnPropsInsert (re)inserts an entity's calculated 'xid' row.
// Assumes fullSaveOwnPropsDelete has already run for this entity.
func (e *Entity) fullSaveOwnPropsInsert() {
	var parentArg any
	if e.ParentSID != "" {
		parentArg = e.ParentSID
	}

	// xid - calculated for every entity type, doesn't need its own
	// cascade marker since it's cheap to recompute alongside the
	// entity's own base props.
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, true, false, false, false, true)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, parentArg, e.DbSID,
		e.UID, e.Path, "xid"+string(DB_IN), "/"+e.Path, e.Abstract)
}

// FullTreeResyncOwnProps re-derives e's calculated FullTreeTable rows.
// Used by code that clears system props directly, outside of Save()/
// FullSave() (e.g. ClearResourceSystemDBProperty,
// ClearEntitySystemDBProperties). e is assumed to already have gone
// through at least one FullSave() (so its FullEntities row already
// exists). If e is a Version, also re-adds its calculated
// RESOURCEid/isdefault rows, and refreshes the owning Resource's
// IsDefaultVerCopy set in case this Version is the current default
// (Save()'s own cascade already ran before this out-of-band write
// happened, so it won't run again).
func FullTreeResyncOwnProps(e *Entity) {
	log.KPrintf("FullTree", ">Enter: FullTreeResyncOwnProps(%s)", e.XID)
	defer log.KPrintf("FullTree", "<Exit: FullTreeResyncOwnProps")
	defer fullTreeDebug(time.Now(), "FullTreeResyncOwnProps", "")

	e.fullSaveOwnProps()
	if e.Type == ENTITY_VERSION {
		e.fullSaveVersionCalc()
		v := e.Self.(*Version)
		if v.Resource != nil && fullVersionIsCurrentDefault(v) {
			fullSaveDefaultVerCascade(e.tx, v.Resource)
		}
	}
}

// fullSaveVersionCalc adds the calculated RESOURCEid and isdefault
// attributes for a (real, non-xref-synthetic) Version. e is always the
// real, in-memory Entity (never just a SID/raw row) - either the one
// Save() is currently running for, or the one FullTreeResyncOwnProps()
// is resyncing out-of-band.
func (e *Entity) fullSaveVersionCalc() {
	log.KPrintf("FullTree", ">Enter: fullSaveVersionCalc(%s)", e.Path)
	defer log.KPrintf("FullTree", "<Exit: fullSaveVersionCalc")
	defer fullTreeDebug(time.Now(), "fullSaveVersionCalc", "")

	// e is always a real, in-memory Version, which always has a parent
	// Resource, so e.ParentSID is never empty here.
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        SELECT ?,?,?,?,?,?,?,?, CONCAT(r.Singular,?), r.UID, 'string', ?,
               true, false, false, false, true
        FROM Resources AS r WHERE r.SID=?`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, "id"+string(DB_IN), e.Abstract, e.ParentSID)

	// isdefault - true only if this Version is the owning Resource's
	// current default (via its Meta.defaultVID), or - for a Resource
	// with no defaultVID set but which is itself an xref source - if
	// it matches the xref target's default. In the common non-xref
	// case this just checks m.defaultVID.
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        SELECT ?,?,?,?,?,?,?,?, ?,
               IF(m.defaultVID IS NOT NULL AND ?=m.defaultVID, 'true', 'false'),
               'boolean', ?, true, false, false, false, true
        FROM Metas AS m WHERE m.ResourceSID=?`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, "isdefault"+string(DB_IN), e.UID, e.Abstract,
		e.ParentSID)
}

// fullSaveResourceIsDefault adds the Resource.isdefault attribute, which
// per AllProps is always "true" (a Resource always shows the props of
// whichever Version is its default).
func (e *Entity) fullSaveResourceIsDefault() {
	log.KPrintf("FullTree", ">Enter: fullSaveResourceIsDefault(%s)", e.Path)
	defer log.KPrintf("FullTree", "<Exit: fullSaveResourceIsDefault")
	defer fullTreeDebug(time.Now(), "fullSaveResourceIsDefault", "")

	// e is always a real, in-memory Resource, which always has a parent
	// Group, so e.ParentSID is never empty here.
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        VALUES(?,?,?,?,?,?,?,?, ?, 'true', 'boolean', ?, false,
               false, false, false, true)`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, "isdefault"+string(DB_IN), e.Abstract)
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
func fullSaveDefaultVerCascade(tx *Tx, r *Resource) {
	if r == nil {
		return
	}

	log.KPrintf("FullTree", ">Enter: fullSaveDefaultVerCascade(%s)", r.XID)
	defer log.KPrintf("FullTree", "<Exit: fullSaveDefaultVerCascade")
	defer fullTreeDebug(time.Now(), "fullSaveDefaultVerCascade", "")

	resourceSID := r.DbSID

	Do(tx, `
        DELETE FROM FullTreeTable WHERE eSID=? AND IsDefaultVerCopy=true`,
		resourceSID)

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
		tResults := Query(tx, `
            SELECT v.SID FROM Metas AS srcM
            JOIN Metas AS m ON (m.ResourceSID=srcM.xRefSID)
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

		Do(tx, `
            REPLACE INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   true, false, false
            FROM FullTreeTable WHERE eSID=? AND IsXrefVerCopy=true
                  AND IsCalculated=false`,
			r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID,
			r.DbSID, r.UID, r.Path, r.Abstract, synthESID)
		return
	}

	Do(tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               true, false, false
        FROM FullTreeTable
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalculated=false`,
		r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID, r.DbSID,
		r.UID, r.Path, r.Abstract, ver.DbSID)

	// Fix up isdefault on every one of this Resource's OWN Versions -
	// they were each set based on their own state at their own last
	// Save(), so if the default just moved to a different Version, the
	// old default's row (and the new one's, if it wasn't the one that
	// triggered this call) would otherwise go stale.
	Do(tx, `
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
	log.KPrintf("FullTree", ">Enter: fullSaveXrefCascade(%s)", e.Path)
	defer log.KPrintf("FullTree", "<Exit: fullSaveXrefCascade")
	defer fullTreeDebug(time.Now(), "fullSaveXrefCascade", "")

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
        SELECT xRefSID FROM Metas WHERE SID=?`, e.DbSID)
	row := results.NextRow()
	results.Close()
	if row == nil || NotNilString(row[0]) == "" {
		return
	}
	targetResourceSID := NotNilString(row[0])

	tResults := Query(e.tx, `
        SELECT m.SID, r.Singular FROM Metas AS m
        JOIN Resources AS r ON (r.SID=m.ResourceSID)
        WHERE m.ResourceSID=?`, targetResourceSID)
	tRow := tResults.NextRow()
	tResults.Close()
	if tRow == nil {
		return
	}
	targetMetaSID := NotNilString(tRow[0])
	targetSingular := NotNilString(tRow[1])

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
              AND IsXrefVerCopy=false AND IsCalculated=false
              AND PropName NOT IN (?, ?) AND LEFT(PropName,1)<>'#'`,
		e.Registry.DbSID, e.Type, e.Plural, e.Singular, e.ParentSID, e.DbSID,
		e.UID, e.Path, e.Abstract, targetMetaSID,
		"xref"+string(DB_IN), targetSingular+"id"+string(DB_IN))

	if resource != nil {
		fullSaveXrefVersionCopies(e.tx, resource, targetResourceSID)
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
func fullSaveXrefVersionCopies(tx *Tx, srcResource *Resource, targetResourceSID string) {
	if srcResource == nil {
		return
	}

	log.KPrintf("FullTree", ">Enter: fullSaveXrefVersionCopies(%s,%s)",
		srcResource.Path, targetResourceSID)
	defer log.KPrintf("FullTree", "<Exit: fullSaveXrefVersionCopies")
	defer fullTreeDebug(time.Now(), "fullSaveXrefVersionCopies", "")

	sourceResourceSID := srcResource.DbSID
	synthAbstract := srcResource.Abstract + string(DB_IN) + "versions"

	// Idempotent: this is called both from fullSaveXrefCascadeInsert
	// (which already cleared out ALL of this source's xref-version
	// rows first) and directly from fullSaveXrefFanOutForTargetVersion
	// (which does not) - so clear out just the synthetic versions that
	// correspond to the target's CURRENT version set before
	// recreating them, or a second Save() of the same target Version
	// would hit a duplicate-key error here.
	Do(tx, `
        DELETE ft FROM FullTreeTable AS ft
        JOIN Versions AS v ON (ft.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	Do(tx, `
        DELETE fe FROM FullEntities AS fe
        JOIN Versions AS v ON (fe.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)

	// One FullEntities row per target Version, all at once.
	Do(tx, `
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
	Do(tx, `
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
              AND ft.IsCalculated=false AND ft.PropName<>?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path, synthAbstract,
		targetResourceSID, "xref"+string(DB_IN))

	// Calculated attrs for every synthetic version at once: xid,
	// RESOURCEid (using the SOURCE resource's singular/UID, since
	// that's every synthetic version's effective parent), and
	// isdefault.
	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, CONCAT('/', ?, '/versions/', v.UID), 'string', ?, false,
               false, false, true, true
        FROM Versions AS v WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"xid"+string(DB_IN), srcResource.Path, synthAbstract, targetResourceSID)

	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               CONCAT(r.Singular, ?), r.UID, 'string', ?, false,
               false, false, true, true
        FROM Versions AS v
        JOIN Resources AS r ON (r.SID=?)
        WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"id"+string(DB_IN), synthAbstract, sourceResourceSID,
		targetResourceSID)

	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy, IsCalculated)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, IF(m.defaultVID=v.UID, 'true', 'false'), 'boolean', ?,
               false, false, false, true, true
        FROM Versions AS v
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        WHERE v.ResourceSID=?`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"isdefault"+string(DB_IN), synthAbstract, targetResourceSID)
}

// fullSaveXrefFanOutForTargetMeta re-runs fullSaveXrefCascade for every
// OTHER source Meta that xrefs r - used when a Meta that happens to be
// someone else's xref target changes. r is the real, in-memory
// *Resource whose Meta was just saved (always available at the
// FullSave() call site as meta.Resource). Each source discovered here
// is resolved to its own real *Resource/*Meta via
// Registry.FindResourceBySID()/FindMeta() (cache-checked, so repeat
// fan-out hits for the same source within one Tx are free) rather than
// a raw fullEntityLookup() row.
func (e *Entity) fullSaveXrefFanOutForTargetMeta(r *Resource) {
	if r == nil {
		return
	}

	log.KPrintf("FullTree", ">Enter: fullSaveXrefFanOutForTargetMeta(%s)", r.XID)
	defer log.KPrintf("FullTree", "<Exit: fullSaveXrefFanOutForTargetMeta")
	defer fullTreeDebug(time.Now(), "fullSaveXrefFanOutForTargetMeta", "")

	results := Query(e.tx, `
        SELECT ResourceSID FROM Metas WHERE xRefSID=?`, r.DbSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceResourceSID := NotNilString(row[0])
		sourceResource, xErr := e.tx.Registry.FindResourceBySID(
			sourceResourceSID, FOR_WRITE)
		if xErr != nil || sourceResource == nil {
			continue
		}
		sourceMeta, xErr := sourceResource.FindMeta(false, FOR_WRITE)
		if xErr != nil || sourceMeta == nil {
			continue
		}
		sourceMeta.fullSaveXrefCascade()
		fullSaveDefaultVerCascade(e.tx, sourceResource)
	}
}

// fullSaveXrefFanOutForTargetVersion re-runs the synthetic-version-copy
// refresh for every source Resource that xrefs r - used when a Version
// belonging to a Resource that's someone else's xref target is saved
// (added/changed). r is the real, in-memory *Resource that owns the
// saved Version (always available at the FullSave() call site as
// v.Resource). As with fullSaveXrefFanOutForTargetMeta, each source is
// resolved to its real *Resource via Registry.FindResourceBySID().
func (e *Entity) fullSaveXrefFanOutForTargetVersion(r *Resource) {
	if r == nil {
		return
	}

	log.KPrintf("FullTree", ">Enter: fullSaveXrefFanOutForTargetVersion(%s)", r.XID)
	defer log.KPrintf("FullTree", "<Exit: fullSaveXrefFanOutForTargetVersion")
	defer fullTreeDebug(time.Now(), "fullSaveXrefFanOutForTargetVersion", "")

	results := Query(e.tx, `
        SELECT ResourceSID FROM Metas WHERE xRefSID=?`, r.DbSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceResourceSID := NotNilString(row[0])
		sourceResource, xErr := e.tx.Registry.FindResourceBySID(
			sourceResourceSID, FOR_WRITE)
		if xErr != nil || sourceResource == nil {
			continue
		}
		fullSaveXrefVersionCopies(e.tx, sourceResource, r.DbSID)
		fullSaveDefaultVerCascade(e.tx, sourceResource)
	}
}

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
