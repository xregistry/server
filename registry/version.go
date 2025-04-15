package registry

import (
	"fmt"

	log "github.com/duglin/dlog"
)

type Version struct {
	Entity
	Resource *Resource
}

var _ EntitySetter = &Version{}

func (v *Version) Get(name string) any {
	return v.Entity.Get(name)
}

func (v *Version) SetCommit(name string, val any) error {
	return v.Entity.eSetCommit(name, val)
}

func (v *Version) JustSet(name string, val any) error {
	return v.Entity.eJustSet(NewPPP(name), val)
}

func (v *Version) SetSave(name string, val any) error {
	return v.Entity.eSetSave(name, val)
}

func (v *Version) Delete() error {
	panic("Should never call this directly - try DeleteSetNextVersion")
}

// JustDelete will delete the Version w/o any additional logic like
// "defaultversionid" manipulation.
// Used when xref on the Resource is set and we need to clear existing vers
func (v *Version) JustDelete() error {
	meta, err := v.Resource.FindMeta(false)
	if err != nil {
		panic(err.Error())
	}

	if v.Resource.Touch() {
		if err := meta.ValidateAndSave(); err != nil {
			return err
		}
	}

	if meta.Get("readonly") == true {
		return fmt.Errorf("Delete operations on read-only " +
			"resources are not allowed")
	}

	// Zero is ok if it's already been deleted
	err = DoZeroOne(v.tx, `DELETE FROM Versions WHERE SID=?`, v.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
	}
	v.tx.RemoveFromCache(&v.Entity)
	return nil
}

func (v *Version) DeleteSetNextVersion(nextVersionID string) error {
	log.VPrintf(3, ">Enter: Version.Delete(%s, %s)", v.UID, nextVersionID)
	defer log.VPrintf(3, "<Exit: Version.Delete")

	if v.Resource.IsXref() {
		return fmt.Errorf(`Can't delete "versions" if "xref" is set`)
	}

	if nextVersionID == v.UID {
		return fmt.Errorf("Can't set defaultversionid to Version being deleted")
	}

	vers, err := v.Resource.GetChildVersionIDs(v.UID)
	if err != nil {
		return fmt.Errorf("Error getting version's (%s) children: %s",
			v.UID, err)
	}

	// Before we delete it, make all versions that point to this one "roots"
	for _, vid := range vers {
		childVersion, err := v.Resource.FindVersion(vid, false)
		if err != nil {
			return fmt.Errorf("Error finding version %q: %s", vid, err)
		}
		childVersion.SetSave("ancestor", childVersion.UID)
	}

	// delete it!
	if err := v.JustDelete(); err != nil {
		return err
	}

	// If it was already gone we'll continue and process the nextVersionID...
	// should we?

	numVers, err := v.Resource.GetNumberOfVersions()
	if err != nil {
		return fmt.Errorf("Error deleting Version %q: %s", v.UID, err)
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
		nextVersion, err = v.Resource.FindVersion(nextVersionID, false)
		if err != nil {
			return err
		}
		if nextVersion == nil {
			return fmt.Errorf("Can't find next default Version %q",
				nextVersionID)
		}

		if err = v.Resource.SetDefault(nextVersion); err != nil {
			return err
		}
	}

	return nil
}

func (v *Version) SetDefault() error {
	return v.Resource.SetDefault(v)
}

func (v *Version) GetChildren() ([]*Version, error) {
	vIDs, err := v.Resource.GetChildVersionIDs(v.UID)
	if err != nil {
		return nil, err
	}

	children := ([]*Version)(nil)
	for _, vid := range vIDs {
		childVer, err := v.Resource.FindVersion(vid, false)
		if err != nil {
			return nil, err
		}
		PanicIf(childVer == nil, "Can't find child: %s.%s", v.UID, vid)
		children = append(children, childVer)
	}

	return children, nil
}
