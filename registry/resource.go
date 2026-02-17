package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

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
func RemoveResourceAttributes(rm *ResourceModel, obj map[string]any) {
	attrs := maps.Clone(rm.ResourceAttributes)
	attrs.AddIfValuesAttributes(obj)

	// resobj is obj with just the resource-level attrs. We do this so that
	// when we add IfValues attributes it's just the resource-level ones.
	// Those are the ones we want to delete
	resObj := map[string]any{}

	for attrName, _ := range attrs {
		if rm.VersionAttributes[attrName] == nil {
			// build verObj with just version-level attributes.
			// Note $xxx won't work but I don't think we care about those
			resObj[attrName] = obj[attrName]
		}
	}

	attrs.AddIfValuesAttributes(resObj)

	for attrName, _ := range attrs {
		if rm.VersionAttributes[attrName] == nil { // Not sure we want this 'if'
			delete(obj, attrName)
		}
	}

	/* old stuff, but I think we need to take into account ifvalues if
	       we ever support extensions (or ifvalues) in resourceattributes
			propsOrdered, _ := rm.GetPropsOrdered()
			for _, attr := range propsOrdered {
				if attr.InType(ENTITY_RESOURCE) && !attr.InType(ENTITY_VERSION) {
					delete(obj, attr.Name)
				}
			}
	*/
}

// Not used yet but if we ever support extensions for resourceattributes
// then we may need this
func RemoveVersionAttributes(rm *ResourceModel, obj map[string]any) {
	attrs := maps.Clone(rm.VersionAttributes)

	// verObj is obj with just version-level attrs. We do this so that
	// when we add IfValues attributes it's just version-level ones.
	// Those are the ones we want to delete.
	verObj := map[string]any{}

	for attrName, _ := range rm.VersionAttributes {
		if rm.ResourceAttributes[attrName] == nil {
			// build verObj with just version-level attributes.
			// Note $xxx won't work but I don't think we care about those
			verObj[attrName] = obj[attrName]
		}
	}

	attrs.AddIfValuesAttributes(verObj)

	for attrName, _ := range attrs {
		if rm.ResourceAttributes[attrName] == nil {
			delete(obj, attrName)
		}
	}

	/*
		propsOrdered, _ := rm.GetPropsOrdered()
		for _, attr := range propsOrdered {
			if !attr.InType(ENTITY_RESOURCE) && attr.InType(ENTITY_VERSION) {
				delete(obj, attr.Name)
			}
		}
	*/
}

var _ EntitySetter = &Resource{}
var _ EntitySetter = &Meta{}

func (r *Resource) Get(name string) any {
	log.VPrintf(4, "Get: r(%s).Get(%s)", r.UID, name)

	meta, xErr := r.FindMeta(false, FOR_READ)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	xrefStr, xref, xErr := r.GetXref()
	Must(xErr)
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

	v, xErr := r.GetDefault(FOR_READ)
	if xErr != nil {
		panic(xErr)
	}
	PanicIf(v == nil, "No default version for %q", r.UID)

	return v.Get(name)
}

func (r *Resource) GetXref() (string, *Resource, *XRError) {
	meta, xErr := r.FindMeta(false, FOR_READ)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	tmp := meta.Get("xref")
	if IsNil(tmp) {
		return "", nil, nil
	}

	// TODO parse as XID
	xref := strings.TrimSpace(tmp.(string))
	if xref == "" {
		return "", nil, nil
	}

	if xref[0] != '/' {
		return "", nil, NewXRError("malformed_xref", meta.XID,
			"xref="+xref,
			"error_detail=must start with '/'")
	}

	parts := strings.Split(xref, "/")
	if len(parts) != 5 || len(parts[0]) != 0 {
		return "", nil, NewXRError("malformed_xref", meta.XID,
			"xref="+xref,
			"error_detail=must be of the form: "+
				"/GROUPS/GID/RESOURCES/RID", tmp.(string))
	}

	group, xErr := r.Registry.FindGroup(parts[1], parts[2], false, FOR_READ)
	if xErr != nil || IsNil(group) {
		return "", nil, xErr
	}
	if IsNil(group) {
		return "", nil, nil
	}
	res, xErr := group.FindResource(parts[3], parts[4], false, FOR_READ)
	if xErr != nil || IsNil(res) {
		return "", nil, xErr
	}

	// If pointing to ourselves, don't recurse, just exit
	if res.Path == r.Path {
		return xref, nil, nil
	}

	return xref, res, nil
}

func (r *Resource) IsXref() bool {
	meta, xErr := r.FindMeta(false, FOR_READ)
	Must(xErr)

	PanicIf(meta == nil, "%s: meta is gone", r.UID)

	tmp := meta.Get("xref")
	return !IsNil(tmp) && tmp != ""
}

func (m *Meta) JustSet(name string, val any) *XRError {
	log.VPrintf(4, "JustSet: m(%s).JustSet(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eJustSet(NewPPP(name), val)
}

func (r *Resource) JustSetMeta(name string, val any) *XRError {
	log.VPrintf(4, "JustSetMeta: r(%s).Set(%s,%v)", r.UID, name, val)
	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)
	return meta.Entity.eJustSet(NewPPP(name), val)
}

func (r *Resource) JustSet(name string, val any) *XRError {
	return r.JustSetDefault(name, val)
}

func (r *Resource) JustSetDefault(name string, val any) *XRError {
	log.VPrintf(4, "JustSetDefault: r(%s).Set(%s,%v)", r.UID, name, val)

	if r.IsXref() {
		return NewXRError("extra_xref_attribute", r.XID,
			"name=defaultversionid")
	}

	v, xErr := r.GetDefault(FOR_WRITE)
	PanicIf(xErr != nil, "%s", xErr)
	return v.JustSet(name, val)
}

func (m *Meta) SetSave(name string, val any) *XRError {
	log.VPrintf(4, "SetSave: m(%s).SetSave(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eSetSave(name, val)
}

func (r *Resource) SetSaveMeta(name string, val any) *XRError {
	log.VPrintf(4, "SetSaveMeta: r(%s).Set(%s,%v)", r.UID, name, val)

	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "%s", xErr)
	return meta.Entity.eSetSave(name, val)
}

// Should only ever be used for "id"
func (r *Resource) SetSaveResource(name string, val any) *XRError {
	log.VPrintf(4, "SetSaveResource: r(%s).Set(%s,%v)", r.UID, name, val)

	PanicIf(name != r.Singular+"id", "You shouldn't be using this")

	return r.Entity.eSetSave(name, val)
}

func (r *Resource) SetSave(name string, val any) *XRError {
	return r.SetSaveDefault(name, val)
}

func (r *Resource) SetSaveDefault(name string, val any) *XRError {
	log.VPrintf(4, "SetSaveDefault: r(%s).Set(%s,%v)", r.UID, name, val)

	v, xErr := r.GetDefault(FOR_WRITE)
	PanicIf(xErr != nil, "%s", xErr)

	return v.SetSave(name, val)
}

func (r *Resource) Touch() bool {
	meta, xErr := r.FindMeta(false, FOR_WRITE)
	if xErr != nil {
		panic(xErr)
	}
	return meta.Touch()
}

func (r *Resource) FindMeta(anyCase bool, accessMode int) (*Meta, *XRError) {
	log.VPrintf(3, ">Enter: FindMeta(%v)", anyCase)
	defer log.VPrintf(3, "<Exit: FindMeta")

	if m := r.tx.GetMeta(r); m != nil {
		if accessMode == FOR_WRITE && m.AccessMode != FOR_WRITE {
			m.Lock()
		}
		return m, nil
	}

	ent, xErr := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/meta",
		anyCase, accessMode)
	if xErr != nil {
		return nil, NewXRError("server_error", r.XID+"/meta").
			SetDetail(fmt.Sprintf("Error finding Meta for %s: %s.",
				r.Path, xErr.GetTitle()))
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
func (r *Resource) FindVersion(id string, anyCase bool, accessMode int) (*Version, *XRError) {
	log.VPrintf(3, ">Enter: FindVersion(%s,%v)", id, anyCase)
	defer log.VPrintf(3, "<Exit: FindVersion")

	if id == "" { // just incase
		return nil, nil
	}

	if v := r.tx.GetVersion(r, id); v != nil {
		if accessMode == FOR_WRITE && v.AccessMode != FOR_WRITE {
			v.Lock()
		}
		return v, nil
	}

	ent, xErr := RawEntityFromPath(r.tx, r.Group.Registry.DbSID,
		r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+id,
		anyCase, accessMode)
	if xErr != nil {
		return nil, NewXRError("server_error", r.XID+"/versions/"+id).
			SetDetail(fmt.Sprintf("Error finding Version %s: %s.",
				r.Path+"/versions/"+id, xErr.GetTitle()))
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
func (r *Resource) GetDefault(accessMode int) (*Version, *XRError) {
	meta, xErr := r.FindMeta(false, accessMode)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	val := meta.GetAsString("defaultversionid")
	return r.FindVersion(val, false, accessMode)
}

func (r *Resource) GetVersionMode() VersionMode {
	vm := r.ResourceModel.GetVersionMode()
	apis, ok := VersionModes[vm]
	PanicIf(!ok, "Missing versionmode(%s) for: %s", vm, r.UID)

	return apis
}

func (r *Resource) GetNewestVersionID() (string, *XRError) {
	return r.GetVersionMode().NewestVersionID(r)
}

func (r *Resource) GetNewest() (*Version, *XRError) {
	vid, xErr := r.GetNewestVersionID()
	if xErr != nil {
		return nil, xErr
	}
	return r.FindVersion(vid, false, FOR_READ)
}

func (r *Resource) EnsureLatest() *XRError {
	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	currentDefault := meta.GetAsString("defaultversionid")

	// log.Printf("In %s.ensurelatest, defID: %q", r.UID, currentDefault)

	if meta.Get("defaultversionsticky") != true || currentDefault == "" {
		newDefault, xErr := r.GetNewestVersionID()
		Must(xErr)
		PanicIf(newDefault == "", "No versions")

		// Only update if it changed
		if currentDefault != newDefault {
			// log.Printf("  Setting def to: %q", newDefault)
			return meta.SetSave("defaultversionid", newDefault)
		}
	}
	return nil
}

// Note will set sticky if vID != ""
func (r *Resource) SetDefaultID(vID string) *XRError {
	if r.IsXref() {
		return NewXRError("extra_xref_attribute", r.XID,
			"name=defaultversionid")
	}

	var v *Version
	var xErr *XRError

	if vID != "" {
		v, xErr = r.FindVersion(vID, false, FOR_WRITE)
		if xErr != nil {
			return xErr
		}

		if IsNil(v) {
			return NewXRError("unknown_id", r.XID,
				"singular=version",
				"id="+vID)
		}
	}
	return r.SetDefault(v)
}

// Only call this if you want things to be sticky (when not nil).
// Creating a new version should do this directly
func (r *Resource) SetDefault(newDefault *Version) *XRError {
	if r.IsXref() {
		return NewXRError("extra_xref_attribute", r.XID,
			"name=defaultversionid")
	}

	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	newDefaultID := ""
	if newDefault != nil {
		newDefaultID = newDefault.UID
	}

	// already set
	if newDefaultID != "" && meta.Get("defaultversionid") == newDefaultID {
		// But make sure we're sticky, could just be a coincidence
		if meta.Get("defaultversionsticky") != true {
			return meta.SetSave("defaultversionsticky", true)
		}
		return nil
	}

	if newDefaultID == "" {
		if xErr := meta.JustSet("defaultversionsticky", nil); xErr != nil {
			return xErr
		}

		newDefaultID, xErr = r.GetNewestVersionID()
		if xErr != nil {
			return xErr
		}
		PanicIf(newDefaultID == "", "No newest: %s", r.UID)
	} else {
		if xErr := meta.JustSet("defaultversionsticky", true); xErr != nil {
			return xErr
		}
	}

	return meta.SetSave("defaultversionid", newDefaultID)
}

type MetaUpsert struct {
	obj                Object
	addType            AddType
	createVersion      bool
	processVersionInfo bool
	more               bool
}

// returns *Meta, isNew, error
// "createVersion" means we should create a version if there isn't already
// one there. This will only happen when the client talks directly to "meta"
// w/o the surrounding Resource object. AND, for now, we only do it when
// we're removing the 'xref' attr. Other cases, the http layer would have
// already create the Resource and default version for us.
func (r *Resource) UpsertMeta(mu *MetaUpsert) (*Meta, bool, *XRError) {
	log.VPrintf(3, ">Enter: UpsertMeta(%s,%v,%v,%v)", r.UID, mu.addType, mu.createVersion, mu.processVersionInfo)
	defer log.VPrintf(3, "<Exit: UpsertMeta")

	// log.Printf("UpsertMeta: OBJ: %s", ToJSON(mu.obj))

	if xErr := r.Registry.SaveModel(); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(mu.obj, r.XID+"/meta"); xErr != nil {
		return nil, false, xErr
	}

	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	if meta.Get("readonly") == true {
		if r.tx.RequestInfo.HasIgnore("readonly") {
			return meta, false, nil
		} else {
			return nil, false, NewXRError("readonly", r.XID)
		}
	}

	PanicIf(mu.obj == nil, "obj is nil")
	if val, ok := mu.obj[r.Singular+"id"]; ok {
		if val != r.UID {
			return nil, false, NewXRError("mismatched_id", meta.XID,
				"singular="+r.Singular,
				"invalid_id="+fmt.Sprintf("%v", val),
				"expected_id="+r.UID)
		}
	}

	// Just in case we need it, save the Resource's epoch value. If this
	// is an xref'd Resource then it'll actually be the target's epoch.
	// Use meta.Object because it's possible that upsertResource changed
	// meta.NewObject["xref"] to null and we need the xref value prior to
	// any changes due to the current operation.
	targetEpoch := 0
	if targetXref := meta.Object["xref"]; targetXref != nil {
		tStr := targetXref.(string)
		tgtR, xErr := meta.Registry.FindResourceByXID(tStr, meta.XID)
		if xErr != nil {
			return nil, false, xErr
		}
		targetEpochAny := tgtR.Get("epoch")
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

	if r.tx.RequestInfo.HasIgnore("defaultversionid") && !IsNil(mu.obj) {
		delete(mu.obj, "defaultversionid")
	}
	if r.tx.RequestInfo.HasIgnore("defaultversionsticky") && !IsNil(mu.obj) {
		delete(mu.obj, "defaultversionsticky")
	}

	// Apply properties
	existingNewObj := meta.NewObject // Should be nil when using http
	meta.SetNewObject(mu.obj)
	meta.Entity.EnsureNewObject()

	// Get new values for easy reference
	newStickyAny, newStickyok := meta.NewObject["defaultversionsticky"]
	newVerIDAny, newVerIDok := meta.NewObject["defaultversionid"]

	if meta.NewObject != nil && mu.addType == ADD_PATCH {
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

	if mu.obj != nil {
		xrefAny, hasXref = meta.NewObject["xref"]
		if hasXref {
			if IsNil(xrefAny) {
				// Do nothing - leave it there so we can null it out later
			} else {
				xref, _ = xrefAny.(string)
				xid, err := ParseXref(xref)
				if err != nil {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail="+err.Error())
				}
				if xid.ResourceID == "" {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail=must be of the "+
							"form: /GROUPS/GID/RESOURCES/RID")
				}
				xrefAbsModel, err := Xid2Abstract(xref)
				if err != nil {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail="+err.Error())
				}
				targetAbsModel := r.ResourceModel.GetOriginAbstractModel()
				if xrefAbsModel != targetAbsModel {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail="+
							fmt.Sprintf("must point to a Resource of "+
								"type %q not %q",
								targetAbsModel, xrefAbsModel))
				}
			}
		}
	}

	// If Meta doesn't exist, create it
	isNew := (meta == nil)
	if meta == nil {
		meta = &Meta{
			Entity: Entity{
				EntityExtensions: EntityExtensions{
					tx: r.tx,
				},

				Registry: r.Registry,
				DbSID:    NewUUID(),
				Plural:   "metas",
				Singular: "meta",
				UID:      r.UID,

				Type:     ENTITY_META,
				Path:     r.Path + "/meta",
				XID:      r.XID + "/meta",
				Abstract: r.Abstract + string(DB_IN) + "meta",
			},
			Resource: r,
		}
		meta.Self = meta

		DoOne(r.tx, `
        INSERT INTO Metas(SID, RegistrySID, ResourceSID,
            Path, Abstract, Plural, Singular)
        SELECT ?,?,?,?,?,?`,
			meta.DbSID, r.Registry.DbSID, r.DbSID,
			meta.Path, meta.Abstract, r.Plural, r.Singular)

		if xErr = meta.JustSet(r.Singular+"id", r.UID); xErr != nil {
			return nil, false, xErr
		}

		r.tx.AddMeta(meta)

		if xErr = meta.SetSave("#nextversionid", 1); xErr != nil {
			return nil, false, xErr
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
			if xErr = meta.JustSet("xref", nil); xErr != nil {
				return nil, false, xErr
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

			// if mu.createVersion is true, make sure we have at least one
			// version
			if mu.createVersion {
				numVers, xErr := r.GetNumberOfVersions()
				if xErr != nil {
					return nil, false, xErr
				}
				if numVers == 0 {
					// UpsertVersion might twiddle defVer, so save/reset it.
					// TODO I don't like this. I'd prefer if we add a flag
					// on the call to UpsertV to tell it NOT to muck with the
					// defaultversion stuff
					defVer := meta.Get("defaultversionid")
					_, _, xErr := r.UpsertVersion("")
					if xErr != nil {
						return nil, false, xErr
					}
					meta.JustSet("defaultversionid", defVer)
				}
			}
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
				xErr := NewXRError("extra_xref_attribute", meta.XID,
					"name="+extraAttrs[0])
				if len(extraAttrs) > 1 {
					xErr.SetDetailf("Full list: %s.",
						strings.Join(extraAttrs, ","))
				}
				return nil, false, xErr
			}

			if xErr = meta.JustSet("xref", xref); xErr != nil {
				return nil, false, xErr
			}

			// Delete all existing Versions too
			vers, xErr := r.GetVersions()
			if xErr != nil {
				return nil, false, xErr
			}

			for _, ver := range vers {
				if xErr = ver.JustDelete(); xErr != nil {
					return nil, false, xErr
				}
			}

			if xErr = meta.ValidateAndSave(); xErr != nil {
				return nil, false, xErr
			}

			return meta, isNew, nil
		}
	}

	oldSticky := meta.Object["defaultversionsticky"]
	newDefID := meta.NewObject["defaultversionid"]
	if IsNil(newDefID) {
		newDefID = ""
	}

	if oldSticky != true && newDefID == "" {
		meta.JustSet("defaultversionid", "")
	}

	if mu.processVersionInfo {
		if xErr = r.ProcessVersionInfo(); xErr != nil {
			return nil, false, xErr
		}

		// Only validate if we processed the version info since if we didn't
		// process the version info then it means we're not done setting up
		// the meta stuff yet.
		// We may not need this since ProcessVersionInfo will validate/save
		// too, but if PVI returns w/o calling save() then we should make sure
		// it is called
		if xErr = meta.ValidateAndSave(); xErr != nil {
			return nil, false, xErr
		}
	}

	return meta, isNew, nil
}

func (r *Resource) ProcessVersionInfo() *XRError {
	m, xErr := r.FindMeta(false, FOR_WRITE)
	Must(xErr)

	if !IsNil(m.Get("xref")) {
		// If xref set then don't touch any of the defaultversion stuff
		return nil
	}

	// Process "defaultversion" attributes

	stickyAny := m.Get("defaultversionsticky")
	if !IsNil(stickyAny) && stickyAny != true && stickyAny != false {
		return NewXRError("invalid_attribute", m.XID,
			"name=defaultversionsticky",
			"error_detail=must be a boolean")
	}
	sticky := (stickyAny == true)

	defaultVersionID := ""
	verIDAny := m.Get("defaultversionid")
	verID := m.GetAsString("defaultversionid")
	// if IsNil(verIDAny) || verIDAny == "" || !sticky {
	if verID == "" || !sticky {
		v, xErr := r.GetNewest()
		Must(xErr)
		if v != nil {
			defaultVersionID = v.UID
		}
	} else {
		if tmp := reflect.ValueOf(verIDAny).Kind(); tmp != reflect.String {
			return NewXRError("invalid_attribute", m.XID,
				"name=defaultversionid",
				"error_detail=must be a string")
		}
		defaultVersionID, _ = verIDAny.(string)
		if defaultVersionID == "" {
			return NewXRError("invalid_attribute", m.XID,
				"name=defaultversionid",
				"error_detail=must not be an empty string")
		}
	}

	if defaultVersionID != "" {
		// It's ok for defVerID to be "", it means we're in the middle of
		// creating a new Resource but no versions are there yet
		v, xErr := r.FindVersion(defaultVersionID, false, FOR_READ)
		Must(xErr)
		if IsNil(v) {
			return NewXRError("unknown_id", m.XID,
				"singular=version",
				"id="+defaultVersionID)
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

		// DUG do we need this "else" any more ??
	}

	return m.ValidateAndSave()
}

func (r *Resource) UpsertVersion(id string) (*Version, bool, *XRError) {
	return r.UpsertVersionWithObject(&VersionUpsert{
		Id:               id,
		Obj:              nil,
		AddType:          ADD_UPSERT,
		More:             false,
		DefaultVersionID: "",
	})
}

type VersionUpsert struct {
	Id               string
	Obj              Object
	AddType          AddType
	More             bool
	DefaultVersionID string
}

// *Version, isNew, error
func (r *Resource) UpsertVersionWithObject(vu *VersionUpsert) (*Version, bool, *XRError) {

	log.VPrintf(3, ">Enter: UpsertVersion(%s,%v,%v)", vu.Id, vu.AddType, vu.More)
	defer log.VPrintf(3, "<Exit: UpsertVersion")

	if xErr := r.Registry.SaveModel(); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(vu.Obj, r.XID+"/versions/"+vu.Id); xErr != nil {
		return nil, false, xErr
	}

	if vu.DefaultVersionID != "" && !r.ResourceModel.GetSetDefaultSticky() {
		return nil, false, NewXRError("setdefaultversionid_not_allowed", r.XID,
			"singular="+r.ResourceModel.Singular)
	}

	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	if meta.Get("readonly") == true {
		if r.tx.RequestInfo.HasIgnore("readonly") {
			return nil, false, nil
		} else {
			return nil, false, NewXRError("readonly", r.XID)
		}
	}

	if r.IsXref() {
		return nil, false,
			NewXRError("bad_request", r.XID,
				"error_detail=Cannot update Resource \""+r.XID+
					"\" in this way since it uses \"xref\"")
	}

	// Do some quick checks on the incoming vu.Obj
	if vu.Obj != nil {
		// We check for ancestor stuff here instead of in the checkFn
		// so that we allow for ANCESTOR_TBD by the system w/o allowing the
		// user to use it
		val, ok := vu.Obj["ancestor"]
		if ok && !IsNil(val) {
			valStr, ok := val.(string)
			if !ok {
				return nil, false,
					NewXRError("invalid_attribute", r.XID,
						"name=ancestor",
						"error_detail="+
							fmt.Sprintf(`must be a string, not %T`, val))
			}
			if xErr = IsValidID(valStr, "ancestor"); xErr != nil {
				xErr.Subject = r.XID
				return nil, false, xErr
			}
		}
	}

	var v *Version
	gm, rm := r.GetModels()

	if vu.Id == "" {
		// No versionID provided so grab the next available one
		tmp := meta.Get("#nextversionid")
		nextID := NotNilInt(&tmp)
		for {
			vu.Id = strconv.Itoa(nextID)
			v, xErr = r.FindVersion(vu.Id, false, FOR_WRITE)
			if xErr != nil {
				return nil, false, xErr
			}

			// Increment no matter what since it's "next" not "default"
			nextID++

			if v == nil {
				meta.JustSet("#nextversionid", nextID)
				break
			}
		}
	} else {
		v, xErr = r.FindVersion(vu.Id, true, FOR_WRITE)

		if vu.AddType == ADD_ADD && v != nil {
			return nil, false,
				NewXRError("bad_request", v.XID,
					"error_detail="+
						fmt.Sprintf("Version %q already exists", vu.Id))
		}

		if v == nil && rm.GetSetVersionId() == false {
			return nil, false, NewXRError("versionid_not_allowed", r.XID,
				"plural="+r.Plural)
		}

		if v != nil && v.UID != vu.Id {
			return nil, false,
				NewXRError("bad_request", v.XID,
					"error_detail="+
						fmt.Sprintf("Attempting to create a Version with "+
							"a \"versionid\" of %q, when one already "+
							"exists as %q", vu.Id, v.UID))
		}

		if xErr != nil {
			return nil, false, xErr
		}
	}

	// If Version doesn't exist, create it
	isNew := (v == nil)
	if v == nil {
		v = &Version{
			Entity: Entity{
				EntityExtensions: EntityExtensions{
					tx:         r.tx,
					AccessMode: FOR_WRITE,
				},

				Registry: r.Registry,
				DbSID:    NewUUID(),
				Plural:   "versions",
				Singular: "version",
				UID:      vu.Id,

				Type:     ENTITY_VERSION,
				Path:     r.Path + "/versions/" + vu.Id,
				XID:      r.XID + "/versions/" + vu.Id,
				Abstract: r.Group.Plural + string(DB_IN) + r.Plural + string(DB_IN) + "versions",

				GroupModel:    gm,
				ResourceModel: rm,
			},
			Resource: r,
		}
		v.Self = v

		DoOne(r.tx, `
        INSERT INTO Versions(SID, UID, RegistrySID, ResourceSID, Path, Abstract)
        VALUES(?,?,?,?,?,?)`,
			v.DbSID, vu.Id, r.Registry.DbSID, r.DbSID,
			r.Group.Plural+"/"+r.Group.UID+"/"+r.Plural+"/"+r.UID+"/versions/"+v.UID,
			r.Group.Plural+string(DB_IN)+r.Plural+string(DB_IN)+"versions")

		v.tx.AddVersion(v)

		if xErr = v.JustSet("versionid", vu.Id); xErr != nil {
			return nil, false, xErr
		}

		// Touch owning Resource to bump its epoch abd modifiedat timestamp
		if r.Touch() {
			if xErr = r.ValidateAndSave(); xErr != nil {
				return nil, false, xErr
			}
		}
	}

	// Apply properties
	if vu.Obj != nil {

		// Do some special processing when the Resource has a Doc
		if rm.GetHasDocument() == true {
			// Rename "RESOURCE" attrs, only if hasDoc=true
			xErr = EnsureJustOneRESOURCE(&r.Entity, vu.Obj, r.Singular)
			if xErr != nil {
				return nil, false, xErr
			}

			data, ok := vu.Obj[r.Singular]
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
					var err error
					buf, err = json.MarshalIndent(data, "", "  ")
					if err != nil {
						return nil, false,
							NewXRError("parsing_data", r.XID, err.Error())
					}
				case reflect.Invalid:
					// I think this only happens when it's "null".
					// just let 'buf' stay as nil
				default:
					str := fmt.Sprintf("%s", data)
					buf = []byte(str)
				}
				vu.Obj[rm.Singular] = buf

				// If there's a doc but no "contenttype" value then:
				// - if existing entity doesn't have one, set it
				// - if existing entity does have one then only override it
				//   if we're not doing PATCH (PUT/POST are compelte overrides)
				if _, ok := vu.Obj["contenttype"]; !ok {
					val := v.Get("contenttype")
					if IsNil(val) || vu.AddType != ADD_PATCH {
						vu.Obj["contenttype"] = "application/json"
					}
				}
			}

			if d, ok := vu.Obj[r.Singular+"base64"]; ok {
				if !IsNil(d) {
					content, err := base64.StdEncoding.DecodeString(d.(string))
					if err != nil {
						return nil, false,
							NewXRError("invalid_atributes", r.XID,
								"list="+r.Singular+"base64",
								"error_detail="+err.Error())
					}
					d = any(content)
				}
				vu.Obj[r.Singular] = d
				delete(vu.Obj, r.Singular+"base64")
			}
		}

		v.SetNewObject(vu.Obj)

		if vu.AddType == ADD_PATCH {
			// Copy existing props over if the incoming obj doesn't set them
			for k, val := range v.Object {
				if _, ok := v.NewObject[k]; !ok {
					v.NewObject[k] = val
				}
			}
		} else {
			// Just for full vu.Obj replacement.
			// the contents of any possible doc are special in that if the
			// client doesn't include it in the update we won't touch it, so
			// we need to copy it forward
			if old, ok := v.Object["#contentid"]; ok {
				if _, ok := v.NewObject["#contentid"]; !ok {
					v.NewObject["#contentid"] = old
				}
			}
		}

		if IsNil(v.NewObject["versionid"]) {
			v.NewObject["versionid"] = vu.Id
		}
	}

	if v.NewObject != nil {
		anc, ok := v.NewObject["ancestor"]
		if ok {
			// ancestor was explicitly set to null then point to latest
			// Otherwise it must be trying to point to a version, leave it
			if IsNil(anc) {
				v.NewObject["ancestor"] = ANCESTOR_TBD
			}
		} else {
			// Not there, so try to grab old value, else point to latest
			anc, ok = v.Object["ancestor"]
			if ok {
				v.NewObject["ancestor"] = anc
			} else {
				v.NewObject["ancestor"] = ANCESTOR_TBD
			}
		}
	}

	// _, touchedTS := v.NewObject["createdat"]
	// if touchedTS -> call EnsureLatest

	if xErr = v.ValidateAndSave(); xErr != nil {
		return nil, false, xErr
	}

	// If there are no more versions to be processed for this Resource in
	// this transaction, go ahead and clean-up the versions wrt the latest
	// and ancestor pointers
	if !vu.More {
		if vu.DefaultVersionID == "null" {
			if xErr := r.SetDefaultID(""); xErr != nil {
				return nil, false, xErr
			}
		} else if vu.DefaultVersionID == "request" {
			// Not sure this is 100% ok but assume request==this version
			if xErr := r.SetDefaultID(v.UID); xErr != nil {
				return nil, false, xErr
			}
		} else if vu.DefaultVersionID != "" {
			if xErr := r.SetDefaultID(vu.DefaultVersionID); xErr != nil {
				return nil, false, xErr
			}
		}

		if xErr = r.ValidateResource(); xErr != nil {
			return nil, false, xErr
		}
	}

	return v, isNew, nil
}

// This is called after all of the calls to UpsertVersionWithObject are
// done in the case where we're uploading more than one version within the
// same tx. The "more" flag on the call to Upsert will tell us whether to
// call this func or not (more=false -> call it)
func (r *Resource) ValidateResource() *XRError {
	meta, xErr := r.FindMeta(false, FOR_WRITE)
	if xErr != nil {
		return xErr
	}

	// If xref is set then we don't need to check anything
	if meta.GetAsString("xref") != "" {
		return nil
	}

	// Clean-up and verify all Ancestor attributes before we continue
	if xErr := r.CheckAncestors(); xErr != nil {
		return xErr
	}

	// Make sure latest is set properly
	if xErr := r.EnsureLatest(); xErr != nil {
		return xErr
	}

	// If we've reached the maximum # of Versions, then delete oldest
	if xErr := r.EnsureMaxVersions(); xErr != nil {
		return xErr
	}

	// Flag it if we have more than one root & the resource doesn't allow it
	if xErr := r.EnsureSingleVersionRoot(); xErr != nil {
		return xErr
	}

	// Flag it if we're left with any circular references of ancestors
	if xErr := r.EnsureCircularReferences(); xErr != nil {
		return xErr
	}

	// vs, _ := r.GetOrderedVersionIDs()
	// log.Printf("r.ancestors: %v", ToJSON(vs))

	// Only validate meta if there's a defaultversionid. Assume that
	// if it's missing then we're in the middle of recreating things
	if meta.GetAsString("defaultversionid") != "" {
		if xErr = meta.ValidateAndSave(); xErr != nil {
			return xErr
		}
	}

	return nil
}

func (r *Resource) AddVersion(id string) (*Version, *XRError) {
	v, _, xErr := r.UpsertVersionWithObject(&VersionUpsert{
		Id:               id,
		Obj:              nil,
		AddType:          ADD_ADD,
		More:             false,
		DefaultVersionID: "",
	})
	return v, xErr
}

func (r *Resource) AddVersionWithObject(id string, obj Object) (*Version, *XRError) {
	v, _, xErr := r.UpsertVersionWithObject(&VersionUpsert{
		Id:               id,
		Obj:              obj,
		AddType:          ADD_ADD,
		More:             false,
		DefaultVersionID: "",
	})
	return v, xErr
}

type VersionAncestor struct {
	VID       string
	Ancestor  string
	CreatedAt string
	Pos       string // 0-root, 1-middle, 2-leaf
}

func (r *Resource) GetVersionIDs() ([]string, *XRError) {
	// Find all version IDs for this Resource
	results := Query(r.tx, `
            SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=?`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

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

func (r *Resource) GetRootVersionIDs() ([]string, *XRError) {
	// Find all versions whose Ancestor = its vID
	results := Query(r.tx, `
            SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND UID=Ancestor`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

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

// Return all versions whose 'ancestor' is ANCESTOR_TBD or points to a missing
// version (which include pointing to null).
// Note that the results is ordered so that we can process the ones with
// a missing Ancestor in oldest->newest order
func (r *Resource) GetProblematicVersions() ([]*VersionAncestor, *XRError) {
	// Find all versions that point to non-existing versions
	results := Query(r.tx, `
            SELECT v1.UID, v1.Ancestor, v1.CreatedAt FROM Versions AS v1
			WHERE v1.RegistrySID=? AND
			      v1.ResourceSID=? AND
                  (v1.Ancestor='`+ANCESTOR_TBD+`' OR (
			          v1.UID<>v1.Ancestor AND
			          NOT EXISTS(SELECT 1 FROM Versions AS v2
				                WHERE v2.RegistrySID=v1.RegistrySID AND
							          v2.ResourceSID=v1.ResourceSID AND
							          v2.UID=v1.Ancestor)))
			ORDER BY CreatedAt ASC, UID ASC`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := ([]*VersionAncestor)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:       NotNilString(row[0]),
			Ancestor:  NotNilString(row[1]),
			CreatedAt: NotNilString(row[2]),
			Pos:       "n/a",
		})
	}

	return vers, nil
}

func (r *Resource) GetChildVersionIDs(parentVID string) ([]string, *XRError) {
	// Find all versions that point 'parentVID'.
	// Note that roots will include themselves - not sure if this is ok or not
	results := Query(r.tx, `
			SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND Ancestor=?`,
		r.Registry.DbSID, r.DbSID, parentVID)
	defer results.Close()

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

func (r *Resource) GetNumberOfVersions() (int, *XRError) {
	// Get the list of Version IDs for this Resource (oldest first)
	results := Query(r.tx, `
	        SELECT COUNT(*) FROM Versions
			WHERE RegistrySID=? AND ResourceSID=?`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	row := results.NextRow()
	return NotNilInt(row[0]), nil
}

func (r *Resource) HasCircularAncestors() ([]string, *XRError) {
	// v.RegistrySID,v.ResourceSID,v.UID
	// Get the list of Version IDs that are part of circular ancestor refs
	results := Query(r.tx, `
			SELECT UID FROM VersionCircles
			WHERE RegistrySID=? AND ResourceSID=?`,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

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

func (r *Resource) EnsureSingleVersionRoot() *XRError {
	rm := r.GetResourceModel()
	if rm.GetSingleVersionRoot() == false {
		// Requirement isn't set
		return nil
	}

	vIDs, xErr := r.GetRootVersionIDs()
	if xErr != nil {
		return xErr
	}

	if len(vIDs) > 1 {
		return NewXRError("multiple_roots", r.XID, "plural="+r.Plural)
	}

	return nil
}

func (r *Resource) EnsureMaxVersions() *XRError {
	rm := r.GetResourceModel()
	if rm.GetMaxVersions() == 0 {
		// No limit, so just exit
		return nil
	}

	verIDs, xErr := r.GetOrderedVersionIDs()
	if xErr != nil {
		return xErr
	}

	count := len(verIDs)
	PanicIf(count == 0, "Query can't be empty")

	tmp := r.Get("defaultversionid")
	defaultID := NotNilString(&tmp)
	PanicIf(defaultID == "", "No defaultid set!!")

	/*
		log.Printf("ensuremax: defID: %s", defaultID)
		log.Printf("ensuremax: sticky: %v", r.Get("defalutversionsticky"))
		log.Printf("ensuremax: ancestors: %s", ToJSON(verIDs))
	*/

	// Starting with the oldest, keep deleting until we reach the max
	// number of Versions allowed. Technically, this should always just
	// delete 1, but ya never know. Also, skip the one that's tagged
	// as "default" since that one is special
	for count > rm.GetMaxVersions() {
		// Skip the "default" Version
		if verIDs[0].VID != defaultID {
			v, xErr := r.FindVersion(verIDs[0].VID, false, FOR_WRITE)
			if xErr != nil {
				return xErr
			}
			// log.Printf("  ensuremax: Deleting: %s", v.XID)
			// ShowStack()
			xErr = v.DeleteSetNextVersion("")
			if xErr != nil {
				return xErr
			}
			count--
		}
		verIDs = verIDs[1:]
	}
	return nil
}

func (r *Resource) Delete() *XRError {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	meta, xErr := r.FindMeta(false, FOR_WRITE)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	if meta.Get("readonly") == true {
		return NewXRError("readonly", r.XID)
	}

	if xErr = meta.Delete(); xErr != nil {
		return xErr
	}

	if r.Group.Touch() {
		if xErr = r.Group.ValidateAndSave(); xErr != nil {
			return xErr
		}
	}

	DoOne(r.tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)

	// Delete any pending changes so dirty check doesn't fail
	r.NewObject = nil
	r.tx.RemoveFromCache(&r.Entity)

	return nil
}

func (m *Meta) Delete() *XRError {
	log.VPrintf(3, ">Enter: Meta.Delete(%s)", m.UID)
	defer log.VPrintf(3, "<Exit: Meta.Delete")

	// Can't use a trigger to do this because we get recusive triggers
	Do(m.tx, `DELETE FROM Props WHERE EntitySID=?`, m.DbSID)
	DoOne(m.tx, `DELETE FROM Metas WHERE SID=?`, m.DbSID)

	// Delete any pending changes so dirty check doesn't fail
	m.NewObject = nil
	m.tx.RemoveFromCache(&m.Entity)

	return nil
}

func (r *Resource) GetVersions() ([]*Version, *XRError) {
	list := []*Version{}

	entities, xErr := RawEntitiesFromQuery(r.tx, r.Registry.DbSID,
		FOR_WRITE, `ParentSID=? AND Type=?`, r.DbSID, ENTITY_VERSION)
	if xErr != nil {
		return nil, xErr
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

func (r *Resource) CheckAncestors() *XRError {
	return r.GetVersionMode().CheckAncestors(r)
}

func (r *Resource) EnsureCircularReferences() *XRError {
	vIDs, xErr := r.HasCircularAncestors()
	if xErr != nil {
		return xErr
	}

	if len(vIDs) == 0 {
		return nil
	}

	list := ""
	sort.Strings(vIDs)
	for i, vID := range vIDs {
		if i > 0 {
			list += ", "
		}
		list += vID
	}
	return NewXRError("ancestor_circular_reference", r.XID, "list="+list)
}

func (r *Resource) WillDelete(vID string) *XRError {
	return r.GetVersionMode().WillDelete(r, vID)
}

func (r *Resource) GetOrderedVersionIDs() ([]*VersionAncestor, *XRError) {
	return r.GetVersionMode().GetOrderedVersionIDs(r)
}

func (r *Resource) DumpOrderedVersions() {
	vs, xErr := r.GetOrderedVersionIDs()
	Must(xErr)
	log.Printf("Resource(%s).OrderedVersions:\n%s", r.XID, ToJSON(vs))
}
