package registry

import (
	. "github.com/xregistry/server/common"
)

type VersionMode interface {
	Name() string
	CheckAncestors(r *Resource) *XRError
	WillDelete(r *Resource, vID string) *XRError
}

// keys MUST be lowercase
var VersionModes = map[string]VersionMode{
	"manual":    (*ManualVersionMode)(nil),
	"createdat": (*CreatedatVersionMode)(nil),
}

// MANUAL VERSION MODE

type ManualVersionMode struct{}

func (vm *ManualVersionMode) Name() string { return "manual" }

func (vm *ManualVersionMode) CheckAncestors(r *Resource) *XRError {
	newestVerID := ""

	// Problematic versions are ones that have Ancestor=ANCESTORID_TBD or
	// point to a non-existing Version
	badVAs, xErr := r.GetProblematicVersions()
	if xErr != nil {
		return xErr
	}

	// Loop over the problem versions, checking/fixing each.
	// Note that we're processing them from oldest to newest so that
	// if we need to assign them a parent/ancestor, they'll be ordered
	// correctly.
	for _, va := range badVAs {
		if va.AncestorID != ANCESTORID_TBD {
			// Must be pointing to a non-exiting version, so error
			return NewXRError("unknown_id", r.XID,
				"singular=version",
				"id="+va.AncestorID)
		}

		// If AncestorID is ANCESTORID_TBD then assign it to the newest Ver
		if newestVerID == "" {
			// First time thru, grab the Resource's newest (already
			// resolved, i.e. non-TBD) versionID to anchor this orphan to.
			var xErr *XRError
			newestVerID, xErr = r.GetNewestVersionID(true)
			if xErr != nil {
				return xErr
			}

			if newestVerID == "" {
				// No existing version is latest, so make this one a root/latest
				newestVerID = va.VID
			}
		}

		v, xErr := r.FindVersion(va.VID, false)
		if xErr != nil {
			return xErr
		}
		PanicIf(v == nil, "Didn't find version %q", va.VID)

		v.SetSave("ancestorid", newestVerID)
		newestVerID = v.UID // This one is now the latest
	}

	return nil
}

func (vm *ManualVersionMode) WillDelete(r *Resource, vID string) *XRError {
	// Before we delete a version, make all versions that point to this
	// one become "roots"

	vers, xErr := r.GetChildVersionIDs(vID)
	if xErr != nil {
		return xErr
	}

	for _, vid := range vers {
		ver, xErr := r.FindVersion(vid, false)
		if xErr != nil {
			return xErr
		}
		ver.SetSave("ancestorid", ver.UID)
	}

	return nil
}

// CREATEDAT VERSION MODE

type CreatedatVersionMode struct{}

func (vm *CreatedatVersionMode) Name() string { return "createdat" }

func (vm *CreatedatVersionMode) CheckAncestors(r *Resource) *XRError {
	// select * from (select createdat,UID,AncestorID,ifnull(lag(UID) over (order by createdat,UID),UID) as expectedAncestorID from Versions) list where list.AncestorID!=list.expectedAncestorID  order by createdat

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as ManualVersionMode.newestVersionID()
	// / HasCircularAncestors(): this runs on the write path (via
	// ValidateResource()) and without a lock hint can be pinned to this
	// tx's original RR snapshot, missing Version rows committed by other
	// Txs after that snapshot was established.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Search the DB for all Versions of this Resource, sorted by 'createdat'
	// and return the ones that do not have the proper 'ancestorid' value.
	// Meaning, they don't point to the next oldest one (based on createdat)
	results := Query(r.tx, `
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

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vID := NotNilString(row[0])
		ancestorID := NotNilString(row[1])

		v, xErr := r.FindVersion(vID, false)
		if xErr != nil {
			return xErr
		}
		PanicIf(v == nil, "Didn't find version %q", vID)

		v.SetSave("ancestorid", ancestorID)
	}

	return nil
}

func (vm *CreatedatVersionMode) WillDelete(r *Resource, vID string) *XRError {
	// Before we delete a version, make all versions that point to this
	// one become "roots"

	v, xErr := r.FindVersion(vID, false)
	if xErr != nil {
		return xErr
	}
	ancestorID := v.GetAsString("ancestorid")

	vers, xErr := r.GetChildVersionIDs(vID)
	if xErr != nil {
		return xErr
	}

	for _, vid := range vers {
		ver, xErr := r.FindVersion(vid, false)
		if xErr != nil {
			return xErr
		}
		if ver.GetAsString("ancestorid") != ancestorID {
			ver.SetSave("ancestorid", ancestorID)
		}
	}

	return nil
}
