package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

var _ EntitySetter = &Version{}

func (v *Version) Get(name string) any {
	return v.Entity.Get(name)
}

func (v *Version) JustSet(name string, val any) *XRError {
	return v.Entity.eJustSet(NewPPP(name), val)
}

func (v *Version) SetSave(name string, val any) *XRError {
	return v.Entity.eSetSave(name, val)
}

func (v *Version) Delete() *XRError {
	panic("Should never call this directly - try DeleteSetNextVersion")
}

// JustDelete will delete the Version w/o any additional logic like
// "defaultversionid" manipulation.
// Used when xref on the Resource is set and we need to clear existing vers
func (v *Version) JustDelete() *XRError {
	meta, xErr := v.Resource.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "%s", xErr)

	if v.Resource.Touch() {
		if xErr := meta.ValidateAndSave(); xErr != nil {
			return xErr
		}
	}

	if meta.Get("readonly") == true {
		return NewXRError("readonly", v.XID)
	}

	// Zero is ok if it's already been deleted
	DoZeroOne(v.tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)
	v.tx.RemoveFromCache(&v.Entity)
	return nil
}

func (v *Version) DeleteSetNextVersion(nextVersionID string) *XRError {
	log.VPrintf(3, ">Enter: Version.Delete(%s, %s)", v.UID, nextVersionID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	if v.Resource.IsXref() {
		return NewXRError("bad_request", v.XID,
			"error_detail="+
				fmt.Sprintf(`can't delete "versions" of a Resource `+
					`(/%s) that uses "xref"`, v.Resource.Path))
	}

	if nextVersionID == v.UID {
		return NewXRError("bad_request", v.XID,
			"error_detail=Can't set \"defaultversionid\" to a Version that "+
				"is being deleted")
	}

	vers, xErr := v.Resource.GetChildVersionIDs(v.UID)
	if xErr != nil {
		return xErr
	}

	// Before we delete it, make all versions that point to this one "roots"
	for _, vid := range vers {
		childVersion, xErr := v.Resource.FindVersion(vid, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}
		childVersion.SetSave("ancestor", childVersion.UID)
	}

	// delete it!
	if xErr := v.JustDelete(); xErr != nil {
		return xErr
	}

	// If it was already gone we'll continue and process the nextVersionID...
	// should we?

	numVers, xErr := v.Resource.GetNumberOfVersions()
	if xErr != nil {
		return xErr
	}

	if numVers == 0 {
		// If there are no more Versions left, delete the Resource
		// TODO: Could just do this instead of deleting the Version first?
		return v.Resource.Delete()
	}

	nextVersion := (*Version)(nil)
	currentDefault := v.Resource.Get("defaultversionid")
	mustChange := (v.UID == currentDefault)

	// If they explicitly told us to unset the default version or we're
	// deleting the current default w/o a new vID being given, then unstick it
	if (nextVersionID == "" && mustChange) || nextVersionID == "null" {
		// Find the next default version
		v.Resource.SetDefault(nil)
	} else if nextVersionID != "" {
		nextVersion, xErr = v.Resource.FindVersion(nextVersionID, false,
			FOR_READ)
		if xErr != nil {
			return xErr
		}
		if nextVersion == nil {
			return NewXRError("unknown_id", v.Resource.XID,
				"singular=version",
				"id="+nextVersionID).
				SetDetailf("Can't find next default Version %q.", nextVersionID)
		}

		if xErr = v.Resource.SetDefault(nextVersion); xErr != nil {
			return xErr
		}
	}

	return nil
}

func (v *Version) SetDefault() *XRError {
	return v.Resource.SetDefault(v)
}

func (v *Version) GetChildren() ([]*Version, *XRError) {
	vIDs, xErr := v.Resource.GetChildVersionIDs(v.UID)
	if xErr != nil {
		return nil, xErr
	}

	children := ([]*Version)(nil)
	for _, vid := range vIDs {
		childVer, xErr := v.Resource.FindVersion(vid, false, FOR_READ)
		if xErr != nil {
			return nil, xErr
		}
		PanicIf(childVer == nil, "Can't find child: %s.%s", v.UID, vid)
		children = append(children, childVer)
	}

	return children, nil
}
