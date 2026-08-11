package registry

import (
	. "github.com/xregistry/server/common"
)

type VersionMode interface {
	Name() string
	CheckAncestors(r *Resource) *XRError
	NewestVersionID(r *Resource) (string, *XRError)
	WillDelete(r *Resource, vID string) *XRError
	GetOrderedVersionIDs(r *Resource) ([]*VersionAncestor, *XRError)
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
			newestVerID, xErr = vm.newestVersionID(r, true)
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

func (vm *ManualVersionMode) NewestVersionID(r *Resource) (string, *XRError) {
	return vm.newestVersionID(r, false)
}

// newestVersionID implements the spec's manual-versionmode "Newest
// Version" rule directly: among all Versions that are NOT referenced as
// the ancestorid of any OTHER Version, pick the one with the newest
// createdat (ties broken by highest versionid, case-insensitive). This
// is intentionally independent of root status - GetOrderedVersionIDs()'s
// Pos ('0-root'/'1-middle'/'2-leaf') classification checks root-ness
// first, so a Version that just became a self-referencing root (e.g. via
// WillDelete()'s "Deleted Ancestor" handling) but is otherwise still
// unreferenced by anything else would be wrongly excluded from newest-
// candidacy if we derived the answer from that ordering instead.
//
// If excludeTBD is true, Versions whose own ancestorid is still
// ANCESTORID_TBD are left out of consideration (used by CheckAncestors()
// while it's still resolving pending orphans, so it doesn't anchor a new
// orphan to another not-yet-resolved one).
func (vm *ManualVersionMode) newestVersionID(r *Resource, excludeTBD bool) (string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE (i.e.
	// we're on a write path, like EnsureLatest()): even with
	// Entity.Lock()'s Resource+Meta+Versions family-lock in place
	// (which fixes write-conflict serialization between Txs), this
	// Tx's OWN plain SELECT here can still be pinned to its original RR
	// snapshot from before another Tx's Version INSERT committed - the
	// family-lock only guarantees this Tx now safely blocks/serializes
	// against concurrent writers, it does not retroactively refresh a
	// snapshot already established by an earlier plain read elsewhere
	// in this same Tx. FOR UPDATE here forces THIS read itself to see
	// latest-committed data. Skipped on pure-read paths (e.g.
	// GetNewest()) so we don't take unnecessary row locks there.
	// Verified necessary via TestMiscConcurrency (versionmode=manual).
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	base := `
                SELECT v.UID FROM Versions AS v
                WHERE v.RegistrySID=? AND v.ResourceSID=?`
	if excludeTBD {
		base += ` AND v.AncestorID<>'` + ANCESTORID_TBD + `'`
	}

	// NOTE: this correlated subquery must get its OWN "FOR UPDATE"
	// (lockExpr, same as the outer query) on write paths. MySQL's
	// outer-query "FOR UPDATE" does NOT implicitly force a fresh/
	// latest-committed read for rows examined only within a correlated
	// subquery - without its own lock hint the subquery can still be
	// evaluated against this Tx's original RR snapshot (e.g. established
	// by an earlier plain read elsewhere in this Tx), silently ignoring
	// a concurrently committed new leaf Version and causing the outer
	// query to treat an already-referenced (non-leaf) Version as if it
	// were still a leaf. Confirmed via a standalone repro against MySQL
	// 8.4: same-Tx plain read -> concurrent commit of a new leaf
	// elsewhere -> outer query (FOR UPDATE, no lock on subquery)
	// returned the OLD/wrong leaf; adding FOR UPDATE to the subquery too
	// fixed it. Root cause of the observed Meta.Epoch drift under
	// TestMiscConcurrency.
	notReferenced := `
                  AND NOT EXISTS (
                    SELECT 1 FROM Versions AS v2
                    WHERE v2.ResourceSID=v.ResourceSID AND
                          v2.AncestorID=v.UID AND v2.SID<>v.SID` + lockExpr + `)`
	order := `
                ORDER BY v.CreatedAt DESC, v.UID ` + FILTER_CI_COLLATE + ` DESC
                LIMIT 1`

	results := Query(r.tx, base+notReferenced+order+lockExpr, r.Registry.DbSID, r.DbSID)
	row := results.NextRow()
	results.Close()

	if row != nil {
		return NotNilString(row[0]), nil
	}

	// No Version qualifies as "not referenced by another" - this only
	// happens when every Version's ancestorid chain forms a full circle.
	// That's NOT necessarily a final error state yet though: e.g.
	// EnsureMaxVersions() (which runs later in ValidateResource(), after
	// EnsureLatest()) may still delete enough of the offending Versions
	// to break the cycle before EnsureCircularReferences() actually
	// checks for real (see TestAncestorMaxVersions, which intentionally
	// creates a temporary 2-Version cycle that's resolved once
	// maxversions=1 evicts the oldest one). So don't hard-error here -
	// just fall back to picking an arbitrary candidate amongst all of
	// them (same leniency the old Pos-based logic had, since every
	// Version always gets a Pos bucket even when circular) and let the
	// later EnsureCircularReferences() call be the one to authoritatively
	// decide if this is actually a problem once the rest of validation
	// (including any max-versions eviction) has run.
	results = Query(r.tx, base+order+lockExpr, r.Registry.DbSID, r.DbSID)
	defer results.Close()

	row = results.NextRow()
	if row == nil {
		return "", nil
	}
	return NotNilString(row[0]), nil
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

func (vm *ManualVersionMode) GetOrderedVersionIDs(r *Resource) ([]*VersionAncestor, *XRError) {
	// Get the list of Version IDs for this resource.
	// The list is sorted such that:
	// - the roots are first
	// - then non-roots and non-leaves
	// - then leaves
	// Within each group if there's more than one then it's sorted as:
	// - newest (lowest) createdat timestamp first
	// If more than one share the same timestamp, then it's sorted as:
	// - lowest versionid alphabetically (case insensitive) first

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as ManualVersionMode.newestVersionID().
	//
	// This used to query the VersionAncestors view instead of inlining
	// its definition here, but the view's own Pos-computing correlated
	// EXISTS subquery doesn't inherit this outer FOR UPDATE (same class
	// of issue as newestVersionID()'s notReferenced subquery - MySQL
	// doesn't propagate an outer query's lock hint into a correlated
	// subquery, and a view definition can't have a caller-supplied lock
	// hint baked into its own internal subqueries). Confirmed via a
	// standalone repro against MySQL 8: RR snapshot established by an
	// early plain read -> concurrent Tx commits a new leaf Version
	// elsewhere -> FOR UPDATE query via the view still returned the
	// PRE-EXISTING version's Pos as "leaf" even though it's now a
	// "middle" (the new version's ancestor) - inlining the view's SQL
	// here and adding lockExpr to the EXISTS subquery too fixes it.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	results := Query(r.tx, `
                SELECT v.UID, v.AncestorID,
                    CASE
                        WHEN v.UID=v.AncestorID THEN '0-root'
                        WHEN EXISTS(SELECT 1 FROM Versions AS v2
                                    WHERE v2.ResourceSID=v.ResourceSID AND
                                          v2.AncestorID=v.UID`+lockExpr+`)
                             THEN '1-middle'
                        ELSE '2-leaf'
                    END AS Pos,
                    v.CreatedAt
                FROM Versions AS v
                WHERE v.RegistrySID=? AND v.ResourceSID=? AND
                  v.AncestorID<>'`+ANCESTORID_TBD+`'
                ORDER BY Pos ASC, v.CreatedAt ASC, v.UID ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := []*VersionAncestor{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:        NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
			Pos:        NotNilString(row[2]),
			CreatedAt:  NotNilString(row[3]),
		})
	}

	return vers, nil
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

func (vm *CreatedatVersionMode) NewestVersionID(r *Resource) (string, *XRError) {
	vers, xErr := r.GetVersionMode().GetOrderedVersionIDs(r)
	Must(xErr)

	if len(vers) > 0 {
		return vers[len(vers)-1].VID, nil
	}
	return "", nil
}

func (vm *CreatedatVersionMode) WillDelete(r *Resource, vID string) *XRError {
	// Before we delete a version, make all versions that point to this
	// one "roots"

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

func (vm *CreatedatVersionMode) GetOrderedVersionIDs(r *Resource) ([]*VersionAncestor, *XRError) {
	// Get the list of Version IDs for this resource.
	// The list is sorted such that:
	// - the roots are first
	// - then non-roots and non-leaves
	// - then leaves
	// Within each group if there's more than one then it's sorted as:
	// - newest (lowest) createdat timestamp first
	// If more than one share the same timestamp, then it's sorted as:
	// - lowest alphabetically (case insensitive) first

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as ManualVersionMode.newestVersionID().
	//
	// See ManualVersionMode.GetOrderedVersionIDs()'s doc comment for why
	// this inlines the VersionAncestors view's definition instead of
	// querying the view directly: the view's own Pos-computing
	// correlated EXISTS subquery doesn't inherit the outer query's FOR
	// UPDATE, so it can return stale Pos values for pre-existing rows
	// after a concurrent Tx commits a new Version - confirmed via a
	// standalone MySQL repro.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	results := Query(r.tx, `
                SELECT v.UID, v.AncestorID,
                    CASE
                        WHEN v.UID=v.AncestorID THEN '0-root'
                        WHEN EXISTS(SELECT 1 FROM Versions AS v2
                                    WHERE v2.ResourceSID=v.ResourceSID AND
                                          v2.AncestorID=v.UID`+lockExpr+`)
                             THEN '1-middle'
                        ELSE '2-leaf'
                    END AS Pos,
                    v.CreatedAt
                FROM Versions AS v
                WHERE v.RegistrySID=? AND v.ResourceSID=? AND
                  v.AncestorID<>'`+ANCESTORID_TBD+`'
                ORDER BY Pos ASC, v.CreatedAt ASC, v.UID ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := []*VersionAncestor{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:        NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
			Pos:        NotNilString(row[2]),
			CreatedAt:  NotNilString(row[3]),
		})
	}

	return vers, nil
}
