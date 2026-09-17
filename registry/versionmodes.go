package registry

import (
	"sort"
	"strconv"
	"strings"

	. "github.com/xregistry/server/common"
)

type VersionMode interface {
	Name() string
	CheckAncestors(r *Resource) *XRError
	WillDelete(r *Resource, vID string) *XRError
}

// keys MUST be lowercase
var VersionModes = map[string]VersionMode{
	"manual":     (*ManualVersionMode)(nil),
	"createdat":  (*CreatedatVersionMode)(nil),
	"modifiedat": (*ModifiedatVersionMode)(nil),
	"semver":     (*SemverVersionMode)(nil),
}

// chainWillDelete implements the "WillDelete" ancestor fix-up shared by all
// single-chain-ordering versionmodes (createdat, modifiedat, semver): before
// a Version is deleted, every Version that pointed to it as its ancestor
// must instead point to ITS ancestor (i.e. they get spliced out of the
// chain), so the chain stays connected without needing a full re-run of
// CheckAncestors(). This is independent of whatever key (createdat,
// modifiedat, semver precedence) was used to originally order the chain,
// since it only ever follows the already-stored 'ancestorid' pointers.
func chainWillDelete(r *Resource, vID string) *XRError {
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
	// RR-snapshot-staleness reasoning as HasCircularAncestors(): this runs on
	// the write path (via ValidateResource()) and without a lock hint can be
	// pinned to this tx's original RR snapshot, missing Version rows
	// committed by other Txs after that snapshot was established.
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
	return chainWillDelete(r, vID)
}

// MODIFIEDAT VERSION MODE
//
// Identical to CreatedatVersionMode above except the chain is ordered by
// each Version's 'modifiedat' attribute instead of 'createdat' (see the
// ModifiedAt column, mirrored from the 'modifiedat' Prop the same way
// CreatedAt mirrors 'createdat' - see init.sql's FullTreeAncestor trigger).

type ModifiedatVersionMode struct{}

func (vm *ModifiedatVersionMode) Name() string { return "modifiedat" }

func (vm *ModifiedatVersionMode) CheckAncestors(r *Resource) *XRError {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// reasoning as CreatedatVersionMode.CheckAncestors().
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	// Search the DB for all Versions of this Resource, sorted by
	// 'modifiedat' and return the ones that do not have the proper
	// 'ancestorid' value. Meaning, they don't point to the next oldest
	// one (based on modifiedat)
	results := Query(r.tx, `
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

func (vm *ModifiedatVersionMode) WillDelete(r *Resource, vID string) *XRError {
	return chainWillDelete(r, vID)
}

// SEMVER VERSION MODE
//
// Ancestor Processing: the 'ancestorid' value of each Version MUST either
// be its own 'versionid' value (if it is the oldest Version), or the
// 'versionid' of the next oldest Version per the Semantic Versioning
// (https://semver.org/) specification's "precedence" ordering rules.
// Unlike createdat/modifiedat, the ordering key here is each Version's
// 'versionid' itself (interpreted as a semver string), not a timestamp -
// there's no portable way to express semver precedence comparisons in SQL,
// so (unlike the other two modes) this is computed in Go after fetching
// every Version's id/ancestorid pair.

type SemverVersionMode struct{}

func (vm *SemverVersionMode) Name() string { return "semver" }

func (vm *SemverVersionMode) CheckAncestors(r *Resource) *XRError {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// reasoning as CreatedatVersionMode.CheckAncestors().
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	results := Query(r.tx, `
                SELECT UID, AncestorID FROM Versions
                WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	type verRow struct {
		vID        string
		ancestorID string
	}

	vers := ([]*verRow)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &verRow{
			vID:        NotNilString(row[0]),
			ancestorID: NotNilString(row[1]),
		})
	}

	// Sort oldest->newest by semver precedence, falling back to a
	// case-insensitive 'versionid' comparison to break ties (same
	// tie-break convention used elsewhere for "newest"/"oldest" - see
	// the versionmode doc in model.md), so results are deterministic.
	sort.Slice(vers, func(i, j int) bool {
		if c := CompareSemver(vers[i].vID, vers[j].vID); c != 0 {
			return c < 0
		}
		return strings.ToLower(vers[i].vID) < strings.ToLower(vers[j].vID)
	})

	for i, ver := range vers {
		expectedAncestorID := ver.vID // oldest/root points to itself
		if i > 0 {
			expectedAncestorID = vers[i-1].vID
		}

		if ver.ancestorID == expectedAncestorID {
			continue
		}

		v, xErr := r.FindVersion(ver.vID, false)
		if xErr != nil {
			return xErr
		}
		PanicIf(v == nil, "Didn't find version %q", ver.vID)

		v.SetSave("ancestorid", expectedAncestorID)
	}

	return nil
}

func (vm *SemverVersionMode) WillDelete(r *Resource, vID string) *XRError {
	return chainWillDelete(r, vID)
}

// CompareSemver compares two version strings using the Semantic
// Versioning (https://semver.org/#spec-item-11) "precedence" rules, used
// by SemverVersionMode to order a Resource's Versions. Returns <0, 0 or
// >0 if a is respectively lower than, equal to, or higher precedence
// than b.
//
// This is deliberately lenient rather than a strict validating semver
// parser: a 'versionid' is a general xRegistry ID (see RegexpID) and this
// must never panic/error on one that isn't a fully well-formed semver
// string. Any non-numeric major/minor/patch component is simply compared
// as a plain string instead of being rejected.
func CompareSemver(a, b string) int {
	aCore, aPre := splitSemver(a)
	bCore, bPre := splitSemver(b)

	if c := compareSemverCore(aCore, bCore); c != 0 {
		return c
	}

	// Per spec: a pre-release version has LOWER precedence than the
	// associated normal version, e.g. "1.0.0-alpha" < "1.0.0"
	if aPre == "" && bPre == "" {
		return 0
	}
	if aPre == "" {
		return 1
	}
	if bPre == "" {
		return -1
	}

	return compareSemverPrerelease(aPre, bPre)
}

// splitSemver splits off (and discards) any build-metadata suffix
// ("+...") and separates the core "major.minor.patch"-style prefix from
// any pre-release suffix ("-...").
func splitSemver(v string) (core string, prerelease string) {
	if i := strings.IndexByte(v, '+'); i >= 0 {
		v = v[:i]
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

// compareSemverCore compares the dot-separated "major.minor.patch"-style
// core components numerically, left to right. Missing trailing
// components (e.g. comparing "1.2" to "1.2.3") are treated as 0, per
// semver's numeric-identifier comparison rules.
func compareSemverCore(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}

	for i := 0; i < max; i++ {
		ap, bp := "0", "0"
		if i < len(aParts) {
			ap = aParts[i]
		}
		if i < len(bParts) {
			bp = bParts[i]
		}
		if c := compareSemverIdentifier(ap, bp); c != 0 {
			return c
		}
	}

	return 0
}

// compareSemverPrerelease compares two dot-separated pre-release
// identifier lists per semver's rules: identifiers are compared pairwise
// (numeric identifiers numerically, alphanumeric ones lexically, with
// numeric always lower precedence than alphanumeric), and if all shared
// identifiers are equal, the list with MORE identifiers has higher
// precedence.
func compareSemverPrerelease(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	n := len(aParts)
	if len(bParts) < n {
		n = len(bParts)
	}

	for i := 0; i < n; i++ {
		if c := compareSemverIdentifier(aParts[i], bParts[i]); c != 0 {
			return c
		}
	}

	return len(aParts) - len(bParts)
}

// compareSemverIdentifier compares a single dot-separated identifier from
// either the core version or a pre-release string.
func compareSemverIdentifier(a, b string) int {
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)

	if aErr == nil && bErr == nil {
		if aNum == bNum {
			return 0
		}
		if aNum < bNum {
			return -1
		}
		return 1
	}

	// Numeric identifiers always have lower precedence than alphanumeric
	// ones
	if aErr == nil {
		return -1
	}
	if bErr == nil {
		return 1
	}

	return strings.Compare(a, b)
}
