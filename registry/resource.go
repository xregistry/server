package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
)

type Resource struct {
	Entity
	Group *Group
}

type Meta struct {
	Entity
	Resource *Resource
}

// These attributes are on the Resource not the Version
// We used to use a "." as a prefix to know - may still need to at some point
var specialResourceAttrs = map[string]bool{
	// "id":                   true,
	"#nextversionid": true,
}

func isResourceOnly(name string) bool {
	if attr := SpecProps[name]; attr != nil {
		if (attr.InType(ENTITY_RESOURCE) || attr.InType(ENTITY_META)) &&
			!attr.InType(ENTITY_VERSION) {
			return true
		}
	}

	if specialResourceAttrs[name] {
		return true
	}

	return false
}

// Remove any attributes that appear on Resources but not Versions.
// Mainly used to prep an Obj that was directed at a Resource but will be used
// to update a Version
func RemoveResourceAttributes(singular string, obj map[string]any) {
	// attrArray, _ := CalcSpecProps(-1, singular)

	for _, attr := range OrderedSpecProps { // attrArray {
		if attr.InType(ENTITY_RESOURCE) && !attr.InType(ENTITY_VERSION) {
			name := attr.Name
			if name == "id" {
				name = singular + "id"
			}
			delete(obj, name)
		}
	}
}

var _ EntitySetter = &Resource{}
var _ EntitySetter = &Meta{}

func (r *Resource) Get(name string) any {
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.UID, name)

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	xrefStr, xref, err := r.GetXref()
	Must(err)
	if xrefStr != "" {
		// Set but target is missing
		if xref == nil {
			return nil
		}

		// Got target, so call Get() on it
		return xref.Get(name)
	}

	if isResourceOnly(name) {
		return meta.Get(name)
	}

	v, err := r.GetDefault()
	if err != nil {
		panic(err)
	}
	PanicIf(v == nil, "No default version for %q", r.UID)

	return v.Get(name)
}

func (r *Resource) GetXref() (string, *Resource, error) {
	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	tmp := meta.Get("xref")
	if IsNil(tmp) {
		return "", nil, nil
	}

	xref := strings.TrimSpace(tmp.(string))
	if xref == "" {
		return "", nil, nil
	}

	if xref[0] != '/' {
		return "", nil, fmt.Errorf("'xref' (%s) must start with '/'",
			tmp.(string))
	}

	parts := strings.Split(xref, "/")
	if len(parts) != 5 || len(parts[0]) != 0 {
		return "", nil, fmt.Errorf("'xref' (%s) must be of the form: "+
			"/GROUPS/gID/RESOURCES/rID", tmp.(string))
	}

	group, err := r.Registry.FindGroup(parts[1], parts[2], false)
	if err != nil || IsNil(group) {
		return "", nil, err
	}
	if IsNil(group) {
		return "", nil, nil
	}
	res, err := group.FindResource(parts[3], parts[4], false)
	if err != nil || IsNil(res) {
		return "", nil, err
	}

	// If pointing to ourselves, don't recurse, just exit
	if res.Path == r.Path {
		return xref, nil, nil
	}

	return xref, res, nil
}

func (r *Resource) IsXref() bool {
	meta, err := r.FindMeta(false)
	Must(err)

	return !IsNil(meta.Get("xref"))
}

func (m *Meta) SetCommit(name string, val any) error {
	log.VPrintf(4, "SetCommitMeta: m(%s).Set(%s,%v)", m.UID, name, val)

	return m.Entity.eSetCommit(name, val)
}

func (r *Resource) SetCommitMeta(name string, val any) error {
	log.VPrintf(4, "SetCommitMeta: r(%s).Set(%s,%v)", r.UID, name, val)

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)
	return meta.SetCommit(name, val)
}

func (r *Resource) SetCommit(name string, val any) error {
	return r.SetCommitDefault(name, val)
}

func (r *Resource) SetCommitDefault(name string, val any) error {
	log.VPrintf(4, "SetCommitDefault: r(%s).Set(%s,%v)", r.UID, name, val)

	v, err := r.GetDefault()
	PanicIf(err != nil, "%s", err)

	return v.SetCommit(name, val)
}

func (m *Meta) JustSet(name string, val any) error {
	log.VPrintf(4, "JustSet: m(%s).JustSet(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eJustSet(NewPPP(name), val)
}

func (r *Resource) JustSetMeta(name string, val any) error {
	log.VPrintf(4, "JustSetMeta: r(%s).Set(%s,%v)", r.UID, name, val)
	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)
	return meta.Entity.eJustSet(NewPPP(name), val)
}

func (r *Resource) JustSet(name string, val any) error {
	return r.JustSetDefault(name, val)
}

func (r *Resource) JustSetDefault(name string, val any) error {
	log.VPrintf(4, "JustSetDefault: r(%s).Set(%s,%v)", r.UID, name, val)
	v, err := r.GetDefault()
	PanicIf(err != nil, "%s", err)
	return v.JustSet(name, val)
}

func (m *Meta) SetSave(name string, val any) error {
	log.VPrintf(4, "SetSave: m(%s).SetSave(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eSetSave(name, val)
}

func (r *Resource) SetSaveMeta(name string, val any) error {
	log.VPrintf(4, "SetSaveMeta: r(%s).Set(%s,%v)", r.UID, name, val)

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "%s", err)
	return meta.Entity.eSetSave(name, val)
}

// Should only ever be used for "id"
func (r *Resource) SetSaveResource(name string, val any) error {
	log.VPrintf(4, "SetSaveResource: r(%s).Set(%s,%v)", r.UID, name, val)

	PanicIf(name != r.Singular+"id", "You shouldn't be using this")

	return r.Entity.eSetSave(name, val)
}

func (r *Resource) SetSave(name string, val any) error {
	return r.SetSaveDefault(name, val)
}

func (r *Resource) SetSaveDefault(name string, val any) error {
	log.VPrintf(4, "SetSaveDefault: r(%s).Set(%s,%v)", r.UID, name, val)

	v, err := r.GetDefault()
	PanicIf(err != nil, "%s", err)

	return v.SetSave(name, val)
}

func (r *Resource) Touch() {
	meta, err := r.FindMeta(false)
	if err != nil {
		panic(err.Error())
	}
	meta.Touch()
}

func (r *Resource) FindMeta(anyCase bool) (*Meta, error) {
	log.VPrintf(3, ">Enter: FindMeta(%v)", anyCase)
	defer log.VPrintf(3, "<Exit: FindMeta")

	if m := r.tx.GetMeta(r); m != nil {
		return m, nil
	}

	ent, err := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/meta",
		anyCase)
	if err != nil {
		return nil, fmt.Errorf("Error finding Meta for %q: %q", r.UID, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	m := &Meta{Entity: *ent, Resource: r}
	m.Self = m
	r.tx.AddMeta(m)
	return m, nil
}

// Maybe replace error with a panic? same for other finds??
func (r *Resource) FindVersion(id string, anyCase bool) (*Version, error) {
	log.VPrintf(3, ">Enter: FindVersion(%s,%v)", id, anyCase)
	defer log.VPrintf(3, "<Exit: FindVersion")

	if v := r.tx.GetVersion(r, id); v != nil {
		return v, nil
	}

	ent, err := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+id,
		anyCase)
	if err != nil {
		return nil, fmt.Errorf("Error finding Version %q: %s", id, err)
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	v := &Version{Entity: *ent, Resource: r}
	v.Self = v
	v.tx.AddVersion(v)
	return v, nil
}

// Maybe replace error with a panic?
func (r *Resource) GetDefault() (*Version, error) {
	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	val := meta.GetAsString("defaultversionid")
	if val == "" {
		return nil, nil
		// panic("No default is set")
	}

	return r.FindVersion(val, false)
}

func (r *Resource) GetNewest() (*Version, error) {
	vers, err := r.GetOrderedVersions()
	Must(err)

	if len(vers) > 0 {
		return r.FindVersion(vers[len(vers)-1].VID, false)
	}
	return nil, nil
}

func (r *Resource) EnsureLatest() error {
	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	// If it's sticky, just exit. Nothing to check
	if meta.Get("defaultversionsticky") == true {
		return nil
	}

	vers, err := r.GetOrderedVersions()
	Must(err)
	PanicIf(len(vers) == 0, "No versions")

	newDefault := vers[len(vers)-1].VID

	currentDefault := meta.GetAsString("defaultversionid")
	if currentDefault == newDefault {
		// Already set
		return nil
	}

	return meta.SetSave("defaultversionid", newDefault)
}

// Note will set sticky if vID != ""
func (r *Resource) SetDefaultID(vID string) error {
	var v *Version
	var err error

	if vID != "" {
		v, err = r.FindVersion(vID, false)
		if err != nil {
			return err
		}
	}
	return r.SetDefault(v)
}

// Only call this if you want things to be sticky (when not nil).
// Creating a new version should do this directly
func (r *Resource) SetDefault(newDefault *Version) error {
	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	// already set
	if newDefault != nil && meta.Get("defaultversionid") == newDefault.UID {
		// But make sure we're sticky, could just be a coincidence
		if meta.Get("defaultversionsticky") != true {
			return meta.SetSave("defaultversionsticky", true)
		}
		return nil
	}

	if newDefault == nil {
		if err := meta.JustSet("defaultversionsticky", nil); err != nil {
			return err
		}

		newDefault, err = r.GetNewest()
		if err != nil {
			return err
		}

		PanicIf(newDefault == nil, "No newest: %s", r.UID)
	} else {
		if err := meta.JustSet("defaultversionsticky", true); err != nil {
			return err
		}
	}

	return meta.SetSave("defaultversionid", newDefault.UID)
}

// returns *Meta, isNew, error
// "createVersion" means we should create a version if there isn't already
// one there. This will only happen when the client talks directly to "meta"
// w/o the surrounding Resource object. AND, for now, we only do it when
// we're removing the 'xref' attr. Other cases, the http layer would have
// already create the Resource and default version for us.
func (r *Resource) UpsertMetaWithObject(obj Object, addType AddType, createVersion bool, processVersionInfo bool) (*Meta, bool, error) {
	log.VPrintf(3, ">Enter: UpsertMeta(%s,%v,%v,%v)", r.UID, addType, createVersion, processVersionInfo)
	defer log.VPrintf(3, "<Exit: UpsertMeta")

	if err := CheckAttrs(obj); err != nil {
		return nil, false, err
	}

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	if meta.Get("readonly") == true {
		return nil, false, fmt.Errorf("Write operations on read-only " +
			"resources are not allowed")
	}

	if obj != nil {
		if val, ok := obj[r.Singular+"id"]; ok {
			if val != r.UID {
				return nil, false, fmt.Errorf("meta.%sid must be %q, not %q",
					r.Singular, r.UID, val)
			}
		}
	}

	// Just in case we need it, save the Resource's epoch value. If this
	// is an xref'd Resource then it'll actually be the target's epoch
	targetEpoch := 0
	if meta.Object["xref"] != nil {
		targetEpochAny := r.Get("epoch")
		targetEpoch = NotNilInt(&targetEpochAny)
	}

	var xrefAny any
	hasXref := false
	xref := ""

	attrsToKeep := map[string]bool{
		"#nextversionid": true,
		"#epoch":         true, // Last epoch so we can restore it when xref is gone
		"#createdat":     true,
	}
	attrsToKeep[r.Singular+"id"] = true

	if r.tx.IgnoreDefaultVersionID && !IsNil(obj) {
		delete(obj, "defaultversionid")
	}
	if r.tx.IgnoreDefaultVersionSticky && !IsNil(obj) {
		delete(obj, "defaultversionsticky")
	}

	// Apply properties
	existingNewObj := meta.NewObject // Should be nil when using http
	meta.SetNewObject(obj)
	meta.Entity.EnsureNewObject()

	// Get new values for easy reference
	newStickyAny, newStickyok := meta.NewObject["defaultversionsticky"]
	newVerIDAny, newVerIDok := meta.NewObject["defaultversionid"]

	if meta.NewObject != nil && addType == ADD_PATCH {
		// Do some spec checks and tweaks
		if newVerIDok && !newStickyok {
			// Just defaultversionid is present
			if !IsNil(newVerIDAny) {
				// defaultversionid=vID
				meta.NewObject["defaultversionsticky"] = true
			} else {
				// defaultversionid = null
				meta.NewObject["defaultversionsticky"] = false
			}
		}
		if !newVerIDok && newStickyok && IsNil(newStickyAny) {
			meta.NewObject["defaultversionid"] = nil
		}

		// Patching, so copy missing existing attributes.
		xr, ok := meta.NewObject["xref"]
		xrefSet := (ok && !IsNil(xr) && xr != "")

		for k, val := range meta.Object {
			// if xref isn't set, grab all #'s and just attrsToKeep ones
			if !xrefSet || k[0] == '#' || attrsToKeep[k] {
				if _, ok := meta.NewObject[k]; !ok {
					meta.NewObject[k] = val
				}
			}
		}
	}

	// Mure sure these attributes are present in NewObject, and if not
	// grab them from the previous version of NewObject or Object
	// TODO: change to just blindly copy all "#..." attributes
	for key, _ := range attrsToKeep {
		if tmp, ok := meta.NewObject[key]; !ok {
			if tmp, ok = existingNewObj[key]; ok {
				meta.NewObject[key] = tmp
			} else if tmp, ok = meta.Object[key]; ok {
				meta.NewObject[key] = tmp
			}
		}
	}

	// Make sure we always have an ID
	if IsNil(meta.NewObject[r.Singular+"id"]) {
		meta.JustSet(r.Singular+"id", r.UID)
	}

	if obj != nil {
		xrefAny, hasXref = meta.NewObject["xref"]
		if hasXref {
			if IsNil(xrefAny) {
				// Do nothing - leave it there so we can null it out later
			} else {
				xref, _ = xrefAny.(string)
				xref = strings.TrimSpace(xref)
				parts := strings.Split(xref, "/")
				if len(parts) != 5 || len(parts[0]) != 0 {
					return nil, false, fmt.Errorf("'xref' (%s) must be of the "+
						"form: /GROUPS/gID/RESOURCES/rID", xref)
				}
			}
		}
	}

	// If Meta doesn't exist, create it
	isNew := (meta == nil)
	if meta == nil {
		meta = &Meta{
			Entity: Entity{
				tx: r.tx,

				Registry: r.Registry,
				DbSID:    NewUUID(),
				Plural:   "metas",
				Singular: "meta",
				UID:      r.UID,

				Type:     ENTITY_META,
				Path:     r.Path + "/meta",
				Abstract: r.Abstract + string(DB_IN) + "meta",
			},
			Resource: r,
		}
		meta.Self = meta

		err = DoOne(r.tx, `
        INSERT INTO Metas(SID, RegistrySID, ResourceSID, Path, Abstract)
        SELECT ?,?,?,?,?`,
			meta.DbSID, r.Registry.DbSID, r.DbSID,
			meta.Path, meta.Abstract)
		if err != nil {
			return nil, false, fmt.Errorf("Error adding Meta: %s", err)
		}

		if err = meta.JustSet(r.Singular+"id", r.UID); err != nil {
			return nil, false, err
		}

		r.tx.AddMeta(meta)

		if err = meta.SetSave("#nextversionid", 1); err != nil {
			return nil, false, err
		}
	}

	// Process any xref
	if hasXref {
		if IsNil(xrefAny) || xref == "" {
			newEpochAny := meta.Object["#epoch"]
			newEpoch := NotNilInt(&newEpochAny)
			if targetEpoch > newEpoch {
				newEpoch = targetEpoch
			}
			meta.JustSet("epoch", newEpoch)
			meta.JustSet("#epoch", nil)
			// We have to fake out the updateFn to think the existing values
			// are the # values
			meta.EpochSet = false
			meta.Object["epoch"] = newEpoch

			delete(meta.NewObject, "xref")
			if err = meta.JustSet("xref", nil); err != nil {
				return nil, false, err
			}

			// If xref was previously set then make sure we reset
			// our nextversionid counter to 1
			if !IsNil(meta.Object["xref"]) {
				meta.JustSet("#nextversionid", 1)
			}

			if IsNil(meta.NewObject["createdat"]) {
				meta.JustSet("createdat", meta.Object["#createdat"])
				meta.JustSet("#createdat", nil)
				meta.Object["createdat"] = meta.Object["#createdat"]
			}

			// if createVersion is true, make sure we have at least one
			// version
			if createVersion {
				numVers, err := r.GetNumberOfVersions()
				if err != nil {
					return nil, false, err
				}
				if numVers == 0 {
					// UpsertVersion might twiddle defVer, so save/reset it.
					// TODO I don't like this. I'd prefer if we add a flag
					// to the call to UpsertV to tell it NOT to muck with the
					// defaultversion stuff
					defVer := meta.Get("defaultversionid")
					_, _, err := r.UpsertVersionWithObject("", nil, ADD_ADD)
					if err != nil {
						return nil, false, err
					}
					meta.JustSet("defaultversionid", defVer)
				}
			}

			/*
				defVerIDany := meta.NewObject["defaultversionid"]
				err = r.SetDefaultID(NotNilString(&defVerIDany))
				if err != nil {
					return nil, false, err
				}
			*/
		} else {
			// Clear all existing attributes except ID
			oldEpoch := meta.Object["epoch"]
			if IsNil(oldEpoch) {
				oldEpoch = 0
			}
			meta.JustSet("#epoch", oldEpoch)

			oldCA := meta.Object["createdat"]
			if IsNil(oldCA) {
				oldCA = meta.tx.CreateTime
			}
			meta.JustSet("#createdat", oldCA)

			// meta.JustSet("createdat", nil)

			extraAttrs := []string{}
			for k, v := range meta.NewObject {
				// Leave "epoch" in NewObject, the updateFn will delete it.
				if k[0] == '#' || k == "xref" || IsNil(v) || k == "epoch" {
					continue
				}
				if !attrsToKeep[k] {
					extraAttrs = append(extraAttrs, k)
				}
			}
			if len(extraAttrs) > 0 {
				sort.Strings(extraAttrs)
				return nil, false, fmt.Errorf("Extra attributes (%s) in "+
					"\"meta\" not allowed when \"xref\" is set",
					strings.Join(extraAttrs, ","))
			}

			if err = meta.JustSet("xref", xref); err != nil {
				return nil, false, err
			}

			// Delete all existing Versions too
			vers, err := r.GetVersions()
			if err != nil {
				return nil, false, err
			}

			for _, ver := range vers {
				if err = ver.JustDelete(); err != nil {
					return nil, false, err
				}
			}

			if err = meta.ValidateAndSave(); err != nil {
				return nil, false, err
			}

			return meta, isNew, nil
		}
	}

	if processVersionInfo {
		if err = r.ProcessVersionInfo(); err != nil {
			return nil, false, err
		}

		// Only validate if we processed the version info since if we didn't
		// process the version info then it means we're not done setting up
		// the meta stuff yet
		if err = meta.ValidateAndSave(); err != nil {
			return nil, false, err
		}
	}

	return meta, isNew, nil
}

func (r *Resource) ProcessVersionInfo() error {
	m, err := r.FindMeta(false)
	Must(err)

	if !IsNil(m.Get("xref")) {
		// If xref set then don't touch any of the defaultversion stuff
		return nil
	}

	// Process "defaultversion" attributes

	stickyAny := m.Get("defaultversionsticky")
	if !IsNil(stickyAny) && stickyAny != true && stickyAny != false {
		return fmt.Errorf("Attribute \"defaultversionsticky\" must be a " +
			"boolean")
	}
	sticky := (stickyAny == true)

	defaultVersionID := ""
	verIDAny := m.Get("defaultversionid")
	if IsNil(verIDAny) {
		v, err := r.GetNewest()
		Must(err)
		if v != nil {
			defaultVersionID = v.UID
		}
	} else {
		if tmp := reflect.ValueOf(verIDAny).Kind(); tmp != reflect.String {
			return fmt.Errorf("Attribute \"defaultversionid\" must be a string")
		}
		defaultVersionID, _ = verIDAny.(string)
		if defaultVersionID == "" {
			return fmt.Errorf("Attribute \"defaultversionid\" must not be " +
				"an empty string")
		}

		if !sticky {
			v, err := r.GetNewest()
			Must(err)
			if v != nil && v.UID != defaultVersionID {
				return fmt.Errorf("Attribute \"defaultversionid\" must be %q "+
					"since \"defaultversionsticky\" is \"false\"", v.UID)
			}
		}
	}

	if defaultVersionID != "" {
		// It's ok for defVerID to be "", it means we're in the middle of
		// creating a new Resource but no versions are there yet
		v, err := r.FindVersion(defaultVersionID, false)
		Must(err)
		if IsNil(v) {
			return fmt.Errorf("Version %q not found", defaultVersionID)
		}

		// Make sure we only "touch" meta if something changed. Calling this
		// func needs to be idempotent
		if m.Get(r.Singular+"id") != r.UID {
			m.JustSet(r.Singular+"id", r.UID)
		}
		if m.Get("defaultversionid") != defaultVersionID {
			m.JustSet("defaultversionid", defaultVersionID)
		}
	} else {
		// Bold assumption that no defaultversionid means that we're still
		// in the process of creating things (or converting from an xRef
		// resource to a non-xref resource) and the version isn't there yet
		// so just return (and skip ValidateAndSave) for now. We should call
		// this func again later in the processing though
		return nil
	}

	return m.ValidateAndSave()
}

func (r *Resource) UpsertVersion(id string) (*Version, bool, error) {
	return r.UpsertVersionWithObject(id, nil, ADD_UPSERT)
}

// *Version, isNew, error
func (r *Resource) UpsertVersionWithObject(id string, obj Object, addType AddType) (*Version, bool, error) {
	log.VPrintf(3, ">Enter: UpsertVersion(%s,%v)", id, addType)
	defer log.VPrintf(3, "<Exit: UpsertVersion")

	if err := CheckAttrs(obj); err != nil {
		return nil, false, err
	}

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	if meta.Get("readonly") == true {
		return nil, false, fmt.Errorf("Write operations on read-only " +
			"resources are not allowed")
	}

	if r.IsXref() {
		return nil, false,
			fmt.Errorf(`Can't update "versions" if "xref" is set`)
	}

	var v *Version
	gm, rm := r.GetModels()

	if id == "" {
		// No versionID provided so grab the next available one
		tmp := meta.Get("#nextversionid")
		nextID := NotNilInt(&tmp)
		for {
			id = strconv.Itoa(nextID)
			v, err = r.FindVersion(id, false)
			if err != nil {
				return nil, false,
					fmt.Errorf("Error checking for Version %q: %s", id, err)
			}

			// Increment no matter what since it's "next" not "default"
			nextID++

			if v == nil {
				meta.JustSet("#nextversionid", nextID)
				break
			}
		}
	} else {
		v, err = r.FindVersion(id, true)

		if addType == ADD_ADD && v != nil {
			return nil, false, fmt.Errorf("Version %q already exists", id)
		}

		if v != nil && v.UID != id {
			return nil, false,
				fmt.Errorf("Attempting to create a Version with "+
					"a \"versionid\" of %q, when one already exists as %q",
					id, v.UID)
		}

		if err != nil {
			return nil, false,
				fmt.Errorf("Error checking for Version %q: %s", id, err)
		}
	}

	// If Version doesn't exist, create it
	isNew := (v == nil)
	if v == nil {
		v = &Version{
			Entity: Entity{
				tx: r.tx,

				Registry: r.Registry,
				DbSID:    NewUUID(),
				Plural:   "versions",
				Singular: "version",
				UID:      id,

				Type:     ENTITY_VERSION,
				Path:     r.Group.Plural + "/" + r.Group.UID + "/" + r.Plural + "/" + r.UID + "/versions/" + id,
				Abstract: r.Group.Plural + string(DB_IN) + r.Plural + string(DB_IN) + "versions",

				GroupModel:    gm,
				ResourceModel: rm,
			},
			Resource: r,
		}
		v.Self = v

		err = DoOne(r.tx, `
        INSERT INTO Versions(SID, UID, RegistrySID, ResourceSID, Path, Abstract)
        VALUES(?,?,?,?,?,?)`,
			v.DbSID, id, r.Registry.DbSID, r.DbSID,
			r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+v.UID,
			r.Group.Plural+string(DB_IN)+r.Plural+string(DB_IN)+"versions")
		if err != nil {
			err = fmt.Errorf("Error adding Version: %s", err)
			log.Print(err)
			return nil, false, err
		}

		v.tx.AddVersion(v)

		if err = v.JustSet("versionid", id); err != nil {
			return nil, false, err
		}
	}

	// Apply properties
	if obj != nil {

		// Do some special processing when the Resource has a Doc
		if rm.GetHasDocument() == true {
			// Rename "RESOURCE" attrs, only if hasDoc=true
			if err = EnsureJustOneRESOURCE(obj, r.Singular); err != nil {
				return nil, false, err
			}

			data, ok := obj[r.Singular]
			// If there's data and it's not already just an array of bytes
			// then convert it. This is for cases where the data is raw JSON
			// and so we may need to tweak it
			// Note: ideally we should probably be doing this closer to where
			// we process things at the transport layer since by this point
			// in our processing we really shouldn't know (or care) about the
			// serialization format. However, this "contenttype" processing
			// below is kind of annoying and I wasn't in the mood to try to
			// move it up the stack. It would also require each spot that
			// got input from the transport to call a func to do this
			// conversion - not hard, but annoying in it's own way. In fact
			// at one point I had that, but other issues popped up so I moved
			// it down here for now. When we try to support more than just
			// JSON we may want to reconsider this logic.
			// This commit (https://github.com/xregistry/server/commit/c1945a061fed88f33983738010eb5c4fbdf41596)
			// removed that logic (and the #-contenttype_ attr). Look for
			// the ConvertResourceContents func and the hoops I had to just
			// thru to make sure all cases were handled.
			if ok && !IsNil(data) && reflect.ValueOf(data).Type().String() != "[]uint8" {
				// Get the raw bytes of the "rm.Singular" json attribute
				buf := []byte(nil)
				switch reflect.ValueOf(data).Kind() {
				case reflect.Float64, reflect.Map, reflect.Slice, reflect.Bool:
					buf, err = json.MarshalIndent(data, "", "  ")
					if err != nil {
						return nil, false, err
					}
				case reflect.Invalid:
					// I think this only happens when it's "null".
					// just let 'buf' stay as nil
				default:
					str := fmt.Sprintf("%s", data)
					buf = []byte(str)
				}
				obj[rm.Singular] = buf

				// If there's a doc but no "contenttype" value then:
				// - if existing entity doesn't have one, set it
				// - if existing entity does have one then only override it
				//   if we're not doing PATCH (PUT/POST are compelte overrides)
				if _, ok := obj["contenttype"]; !ok {
					val := v.Get("contenttype")
					if IsNil(val) || addType != ADD_PATCH {
						obj["contenttype"] = "application/json"
					}
				}
			}

			if v, ok := obj[r.Singular+"base64"]; ok {
				if !IsNil(v) {
					content, err := base64.StdEncoding.DecodeString(v.(string))
					if err != nil {
						return nil, false,
							fmt.Errorf("Error decoding \"%sbase64\" "+
								"attribute: "+"%s", r.Singular, err)
					}
					v = any(content)
				}
				obj[r.Singular] = v
				delete(obj, r.Singular+"base64")
			}
		}

		v.SetNewObject(obj)

		if addType == ADD_PATCH {
			// Copy existing props over if the incoming obj doesn't set them
			for k, val := range v.Object {
				if _, ok := v.NewObject[k]; !ok {
					v.NewObject[k] = val
				}
			}
		} else {
			// Just for full obj replacement.
			// the contents of any possible doc are special in that if the
			// client doesn't include it in the update we won't touch it, so
			// we need to copy it forward
			if old, ok := v.Object["#contentid"]; ok {
				if _, ok := v.NewObject["#contentid"]; !ok {
					v.NewObject["#contentid"] = old
				}
			}
		}
	}

	_, touchedTS := v.NewObject["createdat"]

	// Make sure we always have an ID
	if IsNil(v.NewObject["versionid"]) {
		v.NewObject["versionid"] = id
	}

	if err = v.ValidateAndSave(); err != nil {
		return nil, false, err
	}

	if touchedTS {
		if err = r.EnsureLatest(); err != nil {
			return nil, false, err
		}
	}

	// If we created a new Version then make sure the Resource's (meta's) epoch
	// and TS are updated
	if isNew {
		r.Touch()
	}

	// If we can only have one Version, then set the one we just created
	// as the default.
	// Also set it if we're not sticky w.r.t. default version
	if rm.MaxVersions == 1 || (isNew && meta.Get("defaultversionsticky") != true) {
		err = meta.SetSave("defaultversionid", v.UID)
		if err != nil {
			return nil, false, err
		}
	}

	// If we've reached the maximum # of Versions, then delete oldest
	if err = r.EnsureMaxVersions(); err != nil {
		return nil, false, err
	}

	// Only validate meta if there's a defaultversionid. Assume that
	// if it's missing then we're in the middle of recreating things
	if meta.GetAsString("defaultversionid") != "" {
		if err = meta.ValidateAndSave(); err != nil {
			return nil, false, err
		}
	}

	return v, isNew, nil
}

func (r *Resource) AddVersion(id string) (*Version, error) {
	v, _, err := r.UpsertVersionWithObject(id, nil, ADD_ADD)
	return v, err
}

func (r *Resource) AddVersionWithObject(id string, obj Object) (*Version, error) {
	v, _, err := r.UpsertVersionWithObject(id, obj, ADD_ADD)
	return v, err
}

type VersionAncestor struct {
	VID      string
	Ancestor string
	Pos      string // 0-root, 1-middle, 2-leaf
}

// Get the list of Version IDs for this resource.
// The list is sorted such that:
// - the roots are first
// - then non-roots and non-leaves
// - then leaves
// Within each group if there's more than one then it's sorted as:
// - newest (lowest) createdat timestamp first
// If more than one share the same timestamp, then it's sorted as:
// - lowest alphabetically (case sensitive) first
func (r *Resource) GetOrderedVersions() ([]*VersionAncestor, error) {
	results, err := Query(r.tx, `
            SELECT VersionUID, Ancestor, Pos FROM VersionAncestors
			WHERE RegistrySID=? AND ResourceSID=?
			ORDER BY Pos ASC, Time ASC, VersionUID ASC`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error getting Version IDs: %s", err)
	}

	vers := []*VersionAncestor{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:      NotNilString(row[0]),
			Ancestor: NotNilString(row[1]),
			Pos:      NotNilString(row[2]),
		})
	}

	return vers, nil
}

func (r *Resource) GetRootVersionIDs() ([]string, error) {
	// Find all versions whose Ancestor = its vID
	results, err := Query(r.tx, `
            SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND UID=Ancestor`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error counting Versions: %s", err)
	}

	vIDs := ([]string)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vIDs = append(vIDs, NotNilString(row[0]))
	}

	return vIDs, nil
}

func (r *Resource) GetDangingVersions() ([]*VersionAncestor, error) {
	// Find all versions that point to non-existing versions
	results, err := Query(r.tx, `
            SELECT v1.UID, v1.Ancestor FROM Versions AS v1
			WHERE v1.RegistrySID=? AND
			      v1.ResourceSID=? AND
			      v1.UID<>v1.Ancestor AND
			      NOT EXIST(SELECT 1 FROM Versions AS v2
				            WHERE v2.RegistrySID=v1.ResistrySID AND
							      v2.ResourceSID=v1.ResourceSID AND
							      v2.UID=v1.Ancestor)`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error counting Versions: %s", err)
	}

	vers := ([]*VersionAncestor)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:      NotNilString(row[0]),
			Ancestor: NotNilString(row[1]),
			Pos:      "n/a",
		})
	}

	return vers, nil
}

func (r *Resource) GetChildVersionIDs(parentVID string) ([]string, error) {
	// Find all versions that point to non-existing versions
	results, err := Query(r.tx, `
			SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND Ancestor=?`,
		r.Registry.DbSID, r.DbSID, parentVID)
	defer results.Close()

	if err != nil {
		return nil, fmt.Errorf("Error counting Versions: %s", err)
	}

	vIDs := ([]string)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vIDs = append(vIDs, NotNilString(row[0]))
	}

	return vIDs, nil
}

func (r *Resource) GetNumberOfVersions() (int, error) {
	// Get the list of Version IDs for this Resource (oldest first)
	results, err := Query(r.tx, `
	        SELECT COUNT(*) FROM Versions
			WHERE RegistrySID=? AND ResourceSID=?`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	if err != nil {
		return 0, fmt.Errorf("Error counting Versions: %s", err)
	}

	row := results.NextRow()
	return NotNilInt(row[0]), nil
}

func (r *Resource) EnsureMaxVersions() error {
	rm := r.GetResourceModel()
	if rm.MaxVersions == 0 {
		// No limit, so just exit
		return nil
	}

	vers, err := r.GetOrderedVersions()
	if err != nil {
		return err
	}

	count := len(vers)
	PanicIf(count == 0, "Query can't be empty")

	tmp := r.Get("defaultversionid")
	defaultID := NotNilString(&tmp)

	// Starting with the oldest, keep deleting until we reach the max
	// number of Versions allowed. Technically, this should always just
	// delete 1, but ya never know. Also, skip the one that's tagged
	// as "default" since that one is special
	for count > rm.MaxVersions {
		// Skip the "default" Version
		if vers[0].VID != defaultID {
			err = DoOne(r.tx, `DELETE FROM Versions
					WHERE ResourceSID=? AND UID=?`, r.DbSID, vers[0].VID)
			if err != nil {
				return fmt.Errorf("Error deleting Version %q: %s",
					vers[0].VID, err)
			}
			count--
		}
		vers = vers[1:]
	}
	return nil
}

func (r *Resource) Delete() error {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	meta, err := r.FindMeta(false)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	if meta.Get("readonly") == true {
		return fmt.Errorf("Delete operations on read-only " +
			"resources are not allowed")
	}

	if err = meta.Delete(); err != nil {
		return err
	}

	r.Group.Touch()

	err = DoOne(r.tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)
	if err != nil {
		return err
	}
	r.tx.RemoveFromCache(&r.Entity)
	return nil
}

func (m *Meta) Delete() error {
	log.VPrintf(3, ">Enter: Meta.Delete(%s)", m.UID)
	defer log.VPrintf(3, "<Exit: Meta.Delete")

	// Can't use a trigger to do this because we get recusive triggers
	err := Do(m.tx, `DELETE FROM Props WHERE EntitySID=?`, m.DbSID)
	if err != nil {
		return err
	}

	err = DoOne(m.tx, `DELETE FROM Metas WHERE SID=?`, m.DbSID)
	if err != nil {
		return err
	}
	m.tx.RemoveFromCache(&m.Entity)
	return nil
}

func (r *Resource) GetVersions() ([]*Version, error) {
	list := []*Version{}

	entities, err := RawEntitiesFromQuery(r.tx, r.Registry.DbSID,
		`ParentSID=? AND Type=?`, r.DbSID, ENTITY_VERSION)
	if err != nil {
		return nil, err
	}

	for _, e := range entities {
		v := r.tx.GetVersion(r, e.UID)
		if v == nil {
			v = &Version{Entity: *e, Resource: r}
			v.Self = v
			v.tx.AddVersion(v)
		}
		list = append(list, v)
	}

	return list, nil
}

func (r *Resource) GetHasDocument() bool {
	return r.GetResourceModel().GetHasDocument()
}
