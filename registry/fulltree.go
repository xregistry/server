package registry

// This file implements the incremental population of the
// FullTreeTable/FullEntities tables described in sql.md. These tables
// replace the old FullTree/Entities views (which couldn't be indexed
// and were a read-path bottleneck) for the read/query/serialization
// path (GenerateQuery, entity-lookup-by-path, filtering, etc). Every
// entity-creation site (Registry/Group/Resource/Meta/Version) calls
// FullEntityInsert() alongside its "real" table INSERT, and every
// Save() calls FullSave() at the very end to keep FullTreeTable's
// per-property rows in sync with Props. The old FullTree/Entities
// views and Props table are still maintained too (Save() itself is
// otherwise unmodified) - this is phase 1 of the sql.md migration:
// reads exclusively use the new tables, while Save()'s own internal
// property persistence still goes through Props as before.

import (
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

// FullEntityInsert adds a row to FullEntities for a newly-created
// Registry/Group/Resource/Meta/Version. Called from the same places
// that insert into the corresponding "real" entity table.
func FullEntityInsert(tx *Tx, regSID string, eType int, plural string,
	singular string, parentSID string, eSID string, uid string,
	abstract string, path string) {

	var parentArg any
	if parentSID == "" {
		parentArg = nil
	} else {
		parentArg = parentSID
	}

	Do(tx, `
        REPLACE INTO FullEntities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, Path, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?,?,false)`,
		regSID, eType, plural, singular, parentArg, eSID, uid, abstract, path)
}

// FullTreeSyncProp keeps a single own (non-cascaded) FullTreeTable row
// in sync for an entity property that's written OUTSIDE of the normal
// Save()/FullSave() flow - e.g. SetSystemDBProperty(), which is called
// by post-Save validation logic (like format-checking) well after
// FullSave() already ran for that Save(). Deleting val==nil removes the
// row (mirrors the corresponding Props DELETE); otherwise the row is
// REPLACEd with the new value. This never touches cascaded
// (IsDefaultVerCopy/IsXrefPropCopy/IsXrefVerCopy) rows, and does not by
// itself re-run any cascade - callers whose changed prop could affect
// other entities (e.g. default-version copies) still need to invoke the
// relevant cascade helper explicitly, same as FullSave() does.
func (e *Entity) FullTreeSyncProp(name string, propValue *string,
	propType string, docView bool) {

	fe := fullEntityRowFromEntity(e)

	if propValue == nil {
		Do(e.tx, `
            DELETE FROM FullTreeTable
            WHERE eSID=? AND PropName=? AND IsDefaultVerCopy=false
                  AND IsXrefPropCopy=false AND IsXrefVerCopy=false`,
			fe.eSID, name)
	} else {
		var parentArg any
		if fe.ParentSID == "" {
			parentArg = nil
		} else {
			parentArg = fe.ParentSID
		}

		Do(e.tx, `
        REPLACE INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?, ?,?,?,?,?, false,false,false)`,
			fe.RegSID, fe.Type, fe.Plural, fe.Singular, parentArg, fe.eSID,
			fe.UID, fe.Path, name, *propValue, propType, fe.Abstract, docView)
	}

	// If this is a Version, this prop change may also need to be
	// reflected in the owning Resource's IsDefaultVerCopy set (if this
	// Version happens to be the current default) - re-run that cascade
	// since Save()'s own fullSaveDefaultVerCascade already ran (and
	// won't run again) before this out-of-band write happened.
	if fe.Type == ENTITY_VERSION {
		fullSaveDefaultVerCascade(e.tx, fullEntityRowFromEntity(e.GetParent()))
	}
}

// fullEntityRow mirrors the structural (non-property) columns of a
// FullEntities row - used when generating FullTreeTable rows for an
// entity.
type fullEntityRow struct {
	RegSID    string
	Type      int
	Plural    string
	Singular  string
	ParentSID string
	eSID      string
	UID       string
	Abstract  string
	Path      string
}

func fullEntityLookup(tx *Tx, eSID string) *fullEntityRow {
	results := Query(tx, `
        SELECT RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
               Abstract, Path
        FROM FullEntities WHERE eSID=?`, eSID)
	defer results.Close()

	row := results.NextRow()
	if row == nil {
		return nil
	}

	return &fullEntityRow{
		RegSID:    NotNilString(row[0]),
		Type:      NotNilInt(row[1]),
		Plural:    NotNilString(row[2]),
		Singular:  NotNilString(row[3]),
		ParentSID: NotNilString(row[4]),
		eSID:      NotNilString(row[5]),
		UID:       NotNilString(row[6]),
		Abstract:  NotNilString(row[7]),
		Path:      NotNilString(row[8]),
	}
}

// fullEntityRowFromEntity builds a *fullEntityRow entirely from fields
// already in memory on e (plus e.GetParent(), which itself is a
// zero-DB-query traversal of e.Self's typed parent pointer) - no DB
// round-trip. Use this instead of fullEntityLookup() whenever e is an
// already-instantiated Go object (i.e. every "self" case, and every
// case where we're looking up e's own immediate parent), reserving
// fullEntityLookup() for entities we've only discovered via a SID from
// a query (e.g. xref fan-out targets not otherwise loaded in memory).
func fullEntityRowFromEntity(e *Entity) *fullEntityRow {
	if e == nil {
		return nil
	}

	parentSID := ""
	if parent := e.GetParent(); parent != nil {
		parentSID = parent.DbSID
	}

	return &fullEntityRow{
		RegSID:    e.Registry.DbSID,
		Type:      e.Type,
		Plural:    e.Plural,
		Singular:  e.Singular,
		ParentSID: parentSID,
		eSID:      e.DbSID,
		UID:       e.UID,
		Abstract:  e.Abstract,
		Path:      e.Path,
	}
}

// fullEntityExists is a lightweight existence-only check (no row
// values) - used by FullSave() as its safety-net guard against
// fullEntityExists is a lightweight existence-only check (no row
// values) - used by FullSave() as its safety-net guard against
// backfilling FullTreeTable for entities that pre-date this migration
// (see FullSave's doc comment), without paying for a full 9-column
// FullEntities row fetch when all the actual field values are already
// available in memory via fullEntityRowFromEntity().
func fullEntityExists(tx *Tx, eSID string) bool {
	results := Query(tx, `SELECT 1 FROM FullEntities WHERE eSID=?`, eSID)
	defer results.Close()
	return results.NextRow() != nil
}

// FullSave is called at the very end of Entity.Save(). It refreshes
// this entity's "own" (non-cascaded) rows in FullTreeTable, plus any
// calculated singleton attributes (xid, isdefault, RESOURCEid - these
// are cheap to recompute and don't need a cascade marker), and then
// kicks off whichever cascades are relevant given the entity's type
// (default-version-copy, xref prop/version-copy).
//
// This is intentionally forward-only: it does NOT backfill FullEntities/
// FullTreeTable for entities that existed before this migration ran -
// it relies on FullEntityInsert() having already been called when this
// entity was created. If that row isn't there yet, we just skip (log
// at high verbosity) rather than fail the real Save().
func (e *Entity) FullSave() {
	if !fullEntityExists(e.tx, e.DbSID) {
		log.VPrintf(3, "FullSave: no FullEntities row for %s, skipping", e.XID)
		return
	}
	fe := fullEntityRowFromEntity(e)

	// For a Meta, all DELETEs for this eSID (own rows + stale
	// IsXrefPropCopy rows from whatever its PREVIOUS xref state was)
	// must happen before EITHER insert runs. Otherwise, whichever
	// insert phase runs first collides with the other's not-yet-
	// deleted stale row for the same PropName/Path: fullSaveOwnProps
	// alone can't clear the xref-copy rows, and the xref-copy insert
	// alone can't clear stale own-prop rows from before xref was set.
	if e.Type == ENTITY_META {
		fullSaveOwnPropsDelete(e.tx, fe.eSID)
		e.fullSaveXrefCascadeDelete(fe)
		e.fullSaveXrefCascadeInsert(fe)
		fullSaveOwnPropsInsert(e.tx, fe)
	} else {
		fullSaveOwnProps(e.tx, fe)
	}

	switch e.Type {
	case ENTITY_VERSION:
		fullSaveVersionCalc(e.tx, fe)
		fullSaveDefaultVerCascade(e.tx, fullEntityRowFromEntity(e.GetParent()))
		e.fullSaveXrefFanOutForTargetVersion(fe.ParentSID)

	case ENTITY_RESOURCE:
		e.fullSaveResourceIsDefault(fe)

	case ENTITY_META:
		fullSaveDefaultVerCascade(e.tx, fullEntityRowFromEntity(e.GetParent()))
		e.fullSaveXrefFanOutForTargetMeta(fe.ParentSID)
	}
}

// fullSaveOwnProps deletes then re-inserts the entity's own (non-
// cascaded) Props-backed rows, plus its calculated 'xid' row. Takes tx
// explicitly (rather than being an *Entity method) so it can be reused
// to resync entities other than the one Save() is currently running
// for (e.g. all Versions of a Resource, from
// ClearResourceSystemDBProperty()).
func fullSaveOwnProps(tx *Tx, fe *fullEntityRow) {
	fullSaveOwnPropsDelete(tx, fe.eSID)
	fullSaveOwnPropsInsert(tx, fe)
}

// fullSaveOwnPropsDelete removes an entity's own (non-cascaded) rows
// from FullTreeTable. Split out from fullSaveOwnPropsInsert so a
// Meta's own-rows delete can happen up front, before any cascade
// insert/delete runs, avoiding PK collisions from ordering (see
// FullSave()'s ENTITY_META handling above).
func fullSaveOwnPropsDelete(tx *Tx, eSID string) {
	Do(tx, `
        DELETE FROM FullTreeTable
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false`, eSID)
}

// fullSaveOwnPropsInsert (re)inserts an entity's own (non-cascaded)
// Props-backed rows. Assumes fullSaveOwnPropsDelete has already run
// for this eSID.
func fullSaveOwnPropsInsert(tx *Tx, fe *fullEntityRow) {
	var parentArg any
	if fe.ParentSID == "" {
		parentArg = nil
	} else {
		parentArg = fe.ParentSID
	}

	// If this is a Meta with an active xref, its own base Props (other
	// than 'xref,', the owning Resource's '<singular>id,', and any
	// '#'-prefixed internal prop) get superseded by the target's
	// copied values (fullSaveXrefCascade's IsXrefPropCopy rows) - same
	// (RegSID,Path,PropName) key. AllProps' own UNION ALL branches
	// also both emit these, but the read path silently lets the later
	// (xref-copy) row win when building up the entity's props map.
	// FullTreeTable's real PK can't hold both, so skip the superseded
	// ones here and let the xref-copy insert own them instead. '#'
	// props are excluded from that exclusion (i.e. always kept as our
	// own) because fullSaveXrefCascadeInsert never copies them from
	// the target either (LEFT(PropName,1)<>'#' there too) - they stay
	// authoritative on the source regardless of xref state.
	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, p.PropName, p.PropValue, p.PropType, ?,
               p.DocView, false, false, false
        FROM Props AS p
        LEFT JOIN Metas AS m ON (m.SID=p.EntitySID AND m.xRefSID IS NOT NULL)
        LEFT JOIN Resources AS r ON (r.SID=m.ResourceSID)
        WHERE p.EntitySID=?
              AND (m.SID IS NULL OR LEFT(p.PropName,1)='#'
                   OR p.PropName IN (?, CONCAT(r.Singular,?)))`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, parentArg, fe.eSID,
		fe.UID, fe.Path, fe.Abstract, fe.eSID,
		"xref"+string(DB_IN), "id"+string(DB_IN))

	// xid - calculated for every entity type, doesn't need its own
	// cascade marker since it's cheap to recompute alongside the
	// entity's own base props.
	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, true, false, false, false)`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, parentArg, fe.eSID,
		fe.UID, fe.Path, "xid"+string(DB_IN), "/"+fe.Path, fe.Abstract)
}

// FullTreeResyncOwnProps re-derives e's own (non-cascaded) FullTreeTable
// rows from the current state of the Props table. Used by code that
// writes/clears Props directly, outside of Save()/FullSave() (e.g.
// ClearResourceSystemDBProperty, ClearEntitySystemDBProperties). e is
// assumed to already have gone through at least one FullSave() (so its
// FullEntities row already exists). If e is a Version, also re-adds its
// calculated RESOURCEid/isdefault rows (fullSaveOwnProps' delete wipes
// those too, since they share the same all-false cascade markers as
// regular own-Props rows), and refreshes the owning Resource's
// IsDefaultVerCopy set in case this Version is the current default
// (Save()'s own cascade already ran before this out-of-band write
// happened, so it won't run again).
func FullTreeResyncOwnProps(e *Entity) {
	fe := fullEntityRowFromEntity(e)
	fullSaveOwnProps(e.tx, fe)
	if fe.Type == ENTITY_VERSION {
		fullSaveVersionCalc(e.tx, fe)
		fullSaveDefaultVerCascade(e.tx, fullEntityRowFromEntity(e.GetParent()))
	}
}

// fullSaveVersionCalc adds the calculated RESOURCEid and isdefault
// attributes for a (real, non-xref-synthetic) Version. Takes tx
// explicitly (rather than being an *Entity method) so it can be reused
// by FullTreeResyncOwnProps() for entities other than the one Save()
// is currently running for.
func fullSaveVersionCalc(tx *Tx, fe *fullEntityRow) {
	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, CONCAT(r.Singular,?), r.UID, 'string', ?,
               true, false, false, false
        FROM Resources AS r WHERE r.SID=?`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, fe.ParentSID, fe.eSID,
		fe.UID, fe.Path, "id"+string(DB_IN), fe.Abstract, fe.ParentSID)

	// isdefault - true only if this Version is the owning Resource's
	// current default (via its Meta.defaultVID), or - for a Resource
	// with no defaultVID set but which is itself an xref source - if
	// it matches the xref target's default. In the common non-xref
	// case this just checks m.defaultVID.
	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, ?,
               IF(m.defaultVID IS NOT NULL AND ?=m.defaultVID, 'true', 'false'),
               'boolean', ?, true, false, false, false
        FROM Metas AS m WHERE m.ResourceSID=?`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, fe.ParentSID, fe.eSID,
		fe.UID, fe.Path, "isdefault"+string(DB_IN), fe.UID, fe.Abstract,
		fe.ParentSID)
}

// fullSaveResourceIsDefault adds the Resource.isdefault attribute, which
// per AllProps is always "true" (a Resource always shows the props of
// whichever Version is its default).
func (e *Entity) fullSaveResourceIsDefault(fe *fullEntityRow) {
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        VALUES(?,?,?,?,?,?,?,?, ?, 'true', 'boolean', ?, false,
               false, false, false)`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, fe.ParentSID, fe.eSID,
		fe.UID, fe.Path, "isdefault"+string(DB_IN), fe.Abstract)
}

// fullSaveDefaultVerCascade refreshes the IsDefaultVerCopy=true rows on
// a Resource (rfe) so they mirror whatever Version is currently its
// default. Called whenever a Version is saved (in case it's the
// current default) or a Meta is saved (in case meta.defaultversionid
// just changed). rfe is nil if the caller has no Resource to cascade
// (e.g. e.GetParent() found none) - a no-op in that case.
func fullSaveDefaultVerCascade(tx *Tx, rfe *fullEntityRow) {
	if rfe == nil {
		return
	}
	resourceSID := rfe.eSID

	Do(tx, `
        DELETE FROM FullTreeTable WHERE eSID=? AND IsDefaultVerCopy=true`,
		resourceSID)

	results := Query(tx, `
        SELECT v.SID FROM Metas AS m
        JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                v.UID=m.defaultVID)
        WHERE m.ResourceSID=?`, resourceSID)
	row := results.NextRow()
	results.Close()

	var parentArg any
	if rfe.ParentSID == "" {
		parentArg = nil
	} else {
		parentArg = rfe.ParentSID
	}

	if row == nil {
		// No real default Version - this Resource may be an xref
		// source with no Versions of its own, in which case its
		// "current default" is really the xref target's current
		// default, copied in as a synthetic Version by
		// fullSaveXrefVersionCopies(). Copy from THAT synthetic
		// FullTreeTable eSID instead of Props.
		xResults := Query(tx, `
            SELECT xRefSID FROM Metas WHERE ResourceSID=?`, resourceSID)
		xRow := xResults.NextRow()
		xResults.Close()
		if xRow == nil || NotNilString(xRow[0]) == "" {
			return
		}
		targetResourceSID := NotNilString(xRow[0])

		tResults := Query(tx, `
            SELECT v.SID FROM Metas AS m
            JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                    v.UID=m.defaultVID)
            WHERE m.ResourceSID=?`, targetResourceSID)
		tRow := tResults.NextRow()
		tResults.Close()
		if tRow == nil {
			return
		}
		targetDefVerSID := NotNilString(tRow[0])
		synthESID := fmt.Sprintf("-%s-%s", resourceSID, targetDefVerSID)

		Do(tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   true, false, false
            FROM FullTreeTable WHERE eSID=? AND IsXrefVerCopy=true
                  AND PropName NOT IN (?, ?, ?)`,
			rfe.RegSID, rfe.Type, rfe.Plural, rfe.Singular, parentArg,
			rfe.eSID, rfe.UID, rfe.Path, rfe.Abstract, synthESID,
			"xid"+string(DB_IN), "isdefault"+string(DB_IN),
			rfe.Singular+"id"+string(DB_IN))
		return
	}
	defVerSID := NotNilString(row[0])

	Do(tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               true, false, false
        FROM Props WHERE EntitySID=?`,
		rfe.RegSID, rfe.Type, rfe.Plural, rfe.Singular, parentArg, rfe.eSID,
		rfe.UID, rfe.Path, rfe.Abstract, defVerSID)

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
func (e *Entity) fullSaveXrefCascade(fe *fullEntityRow) {
	e.fullSaveXrefCascadeDelete(fe)
	e.fullSaveXrefCascadeInsert(fe)
}

// fullSaveXrefCascadeDelete clears this Meta's stale IsXrefPropCopy
// rows and its Resource's stale IsXrefVerCopy rows, from whatever the
// PREVIOUS xref state was. Split out from fullSaveXrefCascadeInsert so
// FullSave() can run ALL deletes (this plus fullSaveOwnPropsDelete)
// before either insert runs - see FullSave()'s ENTITY_META handling.
func (e *Entity) fullSaveXrefCascadeDelete(fe *fullEntityRow) {
	Do(e.tx, `DELETE FROM FullTreeTable WHERE eSID=? AND IsXrefPropCopy=true`,
		fe.eSID)
	Do(e.tx, `
        DELETE FROM FullTreeTable WHERE ParentSID=? AND IsXrefVerCopy=true`,
		fe.ParentSID)
	Do(e.tx, `
        DELETE FROM FullEntities WHERE ParentSID=? AND IsXrefVerCopy=true`,
		fe.ParentSID)
}

// fullSaveXrefCascadeInsert (re)inserts this Meta's IsXrefPropCopy and
// IsXrefVerCopy rows based on the CURRENT xref state. Assumes
// fullSaveXrefCascadeDelete (and, for the own-props exclusion to work
// correctly, fullSaveOwnPropsDelete) have already run.
func (e *Entity) fullSaveXrefCascadeInsert(fe *fullEntityRow) {
	results := Query(e.tx, `
        SELECT xRefSID FROM Metas WHERE SID=?`, fe.eSID)
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

	var parentArg any
	if fe.ParentSID == "" {
		parentArg = nil
	} else {
		parentArg = fe.ParentSID
	}

	// Copy the target's meta.* props into this (source) Meta, excluding
	// its own xref and "<singular>id" attrs, and any '#' internal props.
	Do(e.tx, `
        INSERT INTO FullTreeTable(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               false, true, false
        FROM Props
        WHERE EntitySID=? AND PropName NOT IN (?, ?) AND LEFT(PropName,1)<>'#'`,
		fe.RegSID, fe.Type, fe.Plural, fe.Singular, parentArg, fe.eSID,
		fe.UID, fe.Path, fe.Abstract, targetMetaSID,
		"xref"+string(DB_IN), targetSingular+"id"+string(DB_IN))

	e.fullSaveXrefVersionCopies(fe, fullEntityRowFromEntity(e.GetParent()),
		targetResourceSID)
}

// fullSaveXrefVersionCopies (re)creates the synthetic FullEntities/
// FullTreeTable Version rows for a source Resource (srfe) that xrefs
// targetResourceSID, one per Version the target currently has. srfe is
// the source Meta's owning Resource - callers that already hold it in
// memory (e.g. via e.GetParent()) should pass it directly; callers that
// only discovered the source Meta via a SID (xref fan-out) must build
// it with fullEntityLookup() instead.
func (e *Entity) fullSaveXrefVersionCopies(fe *fullEntityRow, srfe *fullEntityRow, targetResourceSID string) {
	if srfe == nil {
		return
	}
	sourceResourceSID := srfe.eSID

	results := Query(e.tx, `
        SELECT SID, UID FROM Versions WHERE ResourceSID=?`, targetResourceSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		verSID := NotNilString(row[0])
		verUID := NotNilString(row[1])

		synthESID := fmt.Sprintf("-%s-%s", sourceResourceSID, verSID)
		synthAbstract := srfe.Abstract + string(DB_IN) + "versions"
		synthPath := srfe.Path + "/versions/" + verUID

		// Idempotent: this is called both from fullSaveXrefCascade
		// (which already cleared out ALL of this source's xref-version
		// rows first) and directly from
		// fullSaveXrefFanOutForTargetVersion (which does not) - so
		// clear out just this specific synthetic version's rows before
		// recreating them, or a second Save() of the same target
		// Version would hit a duplicate-key error here.
		Do(e.tx, `DELETE FROM FullTreeTable WHERE eSID=?`, synthESID)
		Do(e.tx, `DELETE FROM FullEntities WHERE eSID=?`, synthESID)

		FullEntityInsert(e.tx, srfe.RegSID, ENTITY_VERSION, "versions",
			"version", sourceResourceSID, synthESID, verUID, synthAbstract,
			synthPath)
		Do(e.tx, `UPDATE FullEntities SET IsXrefVerCopy=true WHERE eSID=?`,
			synthESID)

		Do(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   false, false, true
            FROM Props WHERE EntitySID=? AND PropName<>?`,
			srfe.RegSID, ENTITY_VERSION, "versions", "version",
			sourceResourceSID, synthESID, verUID, synthPath, synthAbstract,
			verSID, "xref"+string(DB_IN))

		// Calculated attrs for the synthetic version: xid, RESOURCEid
		// (using the SOURCE resource's singular/UID, since that's this
		// synthetic version's effective parent), and isdefault.
		Do(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            VALUES(?,?,?,?,?,?,?,?, ?, ?, 'string', ?, false, false,false,true)`,
			srfe.RegSID, ENTITY_VERSION, "versions", "version",
			sourceResourceSID, synthESID, verUID, synthPath,
			"xid"+string(DB_IN), "/"+synthPath, synthAbstract)

		Do(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, CONCAT(r.Singular,?), r.UID, 'string', ?,
                   false, false,false,true
            FROM Resources AS r WHERE r.SID=?`,
			srfe.RegSID, ENTITY_VERSION, "versions", "version",
			sourceResourceSID, synthESID, verUID, synthPath,
			"id"+string(DB_IN), synthAbstract, sourceResourceSID)

		Do(e.tx, `
            INSERT INTO FullTreeTable(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, ?,
                   IF(m.defaultVID=?, 'true', 'false'), 'boolean', ?,
                   false, false,false,true
            FROM Metas AS m WHERE m.ResourceSID=?`,
			srfe.RegSID, ENTITY_VERSION, "versions", "version",
			sourceResourceSID, synthESID, verUID, synthPath,
			"isdefault"+string(DB_IN), verUID, synthAbstract,
			targetResourceSID)
	}
}

// fullSaveXrefFanOutForTargetMeta re-runs fullSaveXrefCascade for every
// OTHER source Meta that xrefs this Meta's Resource - used when a Meta
// that happens to be someone else's xref target changes.
func (e *Entity) fullSaveXrefFanOutForTargetMeta(resourceSID string) {
	if resourceSID == "" {
		return
	}
	results := Query(e.tx, `
        SELECT SID FROM Metas WHERE xRefSID=?`, resourceSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceMetaSID := NotNilString(row[0])
		sfe := fullEntityLookup(e.tx, sourceMetaSID)
		if sfe == nil {
			continue
		}
		e.fullSaveXrefCascade(sfe)
		fullSaveDefaultVerCascade(e.tx, fullEntityLookup(e.tx, sfe.ParentSID))
	}
}

// fullSaveXrefFanOutForTargetVersion re-runs the synthetic-version-copy
// refresh for every source Resource that xrefs resourceSID - used when
// a Version belonging to a Resource that's someone else's xref target
// is saved (added/changed).
func (e *Entity) fullSaveXrefFanOutForTargetVersion(resourceSID string) {
	if resourceSID == "" {
		return
	}
	results := Query(e.tx, `
        SELECT SID FROM Metas WHERE xRefSID=?`, resourceSID)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceMetaSID := NotNilString(row[0])
		sfe := fullEntityLookup(e.tx, sourceMetaSID)
		if sfe == nil {
			continue
		}
		srfe := fullEntityLookup(e.tx, sfe.ParentSID)
		e.fullSaveXrefVersionCopies(sfe, srfe, resourceSID)
		fullSaveDefaultVerCascade(e.tx, srfe)
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
