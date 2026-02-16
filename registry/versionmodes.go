package registry

import (
	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type VersionMode interface {
	Name() string
	CheckAncestors(r *Resource) *XRError
	NewestVersionID(r *Resource) (string, *XRError)
	WillDelete(r *Resource, vID string) *XRError
	GetOrderedVersionIDs(r *Resource) ([]*VersionAncestor, *XRError)
}

var VersionModes = map[string]VersionMode{
	"manual":    (*ManualVersionMode)(nil),
	"createdat": (*CreatedatVersionMode)(nil),
}

// MANUAL VERSION MODE

type ManualVersionMode struct{}

func (vm *ManualVersionMode) Name() string { return "manual" }

func (vm *ManualVersionMode) CheckAncestors(r *Resource) *XRError {
	newestVerID := ""

	// Problematic versions are ones that have Ancestor=ANCESTOR_TBD or
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
		if va.Ancestor != ANCESTOR_TBD {
			// Must be pointing to a non-exiting version, so error
			return NewXRError("unknown_id", r.XID,
				"singular=version",
				"id="+va.Ancestor)
		}

		// If Ancestor is ANCESTOR_TBD then assign it to the newest Ver
		if newestVerID == "" {
			// First time thru, grab the Resource's newest versionID.
			// Didn't need to get all attributes, just its ID
			VIDs, xErr := r.GetVersionMode().GetOrderedVersionIDs(r)
			if xErr != nil {
				return xErr
			}

			if len(VIDs) > 0 {
				// Grab newest non-TBD version
				for i := len(VIDs) - 1; i >= 0; i-- {
					av := VIDs[len(VIDs)-1]
					if av.Ancestor == ANCESTOR_TBD {
						continue
					}
					newestVerID = av.VID // grab its versionID
					break
				}
			}

			if newestVerID == "" {
				// No existing version is latest, so make this one a root/latest
				newestVerID = va.VID
			}
		}

		v, xErr := r.FindVersion(va.VID, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		PanicIf(v == nil, "Didn't find version %q", va.VID)

		v.SetSave("ancestor", newestVerID)
		newestVerID = v.UID // This one is now the latest
	}

	return nil
}

func (vm *ManualVersionMode) NewestVersionID(r *Resource) (string, *XRError) {
	vers, xErr := r.GetVersionMode().GetOrderedVersionIDs(r)
	Must(xErr)

	if len(vers) > 0 {
		return vers[len(vers)-1].VID, nil
	}
	return "", nil
}

func (vm *ManualVersionMode) WillDelete(r *Resource, vID string) *XRError {
	// Before we delete a version, make all versions that point to this
	// one "roots"

	vers, xErr := r.GetChildVersionIDs(vID)
	if xErr != nil {
		return xErr
	}

	for _, vid := range vers {
		ver, xErr := r.FindVersion(vid, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		ver.SetSave("ancestor", ver.UID)
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

	results := Query(r.tx, `
                SELECT VersionUID, Ancestor, Pos, CTime FROM VersionAncestors
                WHERE RegistrySID=? AND ResourceSID=? AND
                  Ancestor<>'`+ANCESTOR_TBD+`'
                ORDER BY Pos ASC, CTime ASC, VersionUID ASC`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := []*VersionAncestor{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:       NotNilString(row[0]),
			Ancestor:  NotNilString(row[1]),
			Pos:       NotNilString(row[2]),
			CreatedAt: NotNilString(row[3]),
		})
	}

	return vers, nil
}

// CREATEDAT VERSION MODE

type CreatedatVersionMode struct{}

func (vm *CreatedatVersionMode) Name() string { return "createdat" }

func (vm *CreatedatVersionMode) CheckAncestors(r *Resource) *XRError {
	// select * from (select createdat,UID,Ancestor,ifnull(lag(UID) over (order by createdat,UID),UID) as expectedAncestor from Versions) list where list.Ancestor!=list.expectedAncestor  order by createdat

	// Search the DB for all Versions of this Resource, sorted by 'createdat'
	// and return the ones that do not have the proper 'ancestor' value.
	// Meaning, they don't point to the next oldest one (based on createdat)
	results := Query(r.tx, `
                SELECT UID, ExpectedAncestor FROM (
                  SELECT CreatedAt,
                         UID,
                         Ancestor,
                         IFNULL(lag(UID) OVER (ORDER BY CreatedAt, UID),
                                UID) AS ExpectedAncestor
                  FROM Versions
                  WHERE RegistrySID=? AND ResourceSID=?) AS list
                WHERE list.Ancestor != list.ExpectedAncestor
                ORDER BY CreatedAt ASC`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vID := NotNilString(row[0])
		ancestor := NotNilString(row[1])

		v, xErr := r.FindVersion(vID, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		PanicIf(v == nil, "Didn't find version %q", vID)

		v.SetSave("ancestor", ancestor)
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

	v, xErr := r.FindVersion(vID, false, FOR_READ)
	if xErr != nil {
		return xErr
	}
	ancestor := v.GetAsString("ancestor")

	vers, xErr := r.GetChildVersionIDs(vID)
	if xErr != nil {
		return xErr
	}

	for _, vid := range vers {
		ver, xErr := r.FindVersion(vid, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		if ver.GetAsString("ancestor") != ancestor {
			ver.SetSave("ancestor", ancestor)
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

	results := Query(r.tx, `
                SELECT VersionUID, Ancestor, Pos, CTime FROM VersionAncestors
                WHERE RegistrySID=? AND ResourceSID=? AND
                  Ancestor<>'`+ANCESTOR_TBD+`'
                ORDER BY Pos ASC, CTime ASC, VersionUID ASC`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := []*VersionAncestor{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:       NotNilString(row[0]),
			Ancestor:  NotNilString(row[1]),
			Pos:       NotNilString(row[2]),
			CreatedAt: NotNilString(row[3]),
		})
	}

	return vers, nil
}
