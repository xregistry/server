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

func RemoveReadonlyResourceAttributes(rm *ResourceModel, obj map[string]any) {
	for _, attr := range rm.GetBaseAttributes() {
		if attr.ReadOnly {
			delete(obj, attr.Name)
		}
	}
	// delete(obj, rm.Singular+"id")
}

func RemoveReadonlyMetaAttributes(rm *ResourceModel, obj map[string]any) {
	for _, attr := range rm.GetBaseMetaAttributes() {
		if attr.ReadOnly {
			delete(obj, attr.Name)
		}
	}
	// delete(obj, rm.Singular+"id")
}

func RemoveReadonlyVersionAttributes(rm *ResourceModel, obj map[string]any) {
	for _, attr := range rm.GetBaseVersionAttributes() {
		if attr.ReadOnly {
			delete(obj, attr.Name)
		}
	}
	delete(obj, rm.Singular+"id")
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

	meta := r.MustFindMeta(false)

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

	v, xErr := r.GetDefault()
	if xErr != nil {
		panic(xErr)
	}
	PanicIf(v == nil, "No default version for %q", r.UID)

	return v.Get(name)
}

func (r *Resource) GetXref() (string, *Resource, *XRError) {
	meta := r.MustFindMeta(false)

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
	tmp := r.MustFindMeta(false).Get("xref")
	return !IsNil(tmp) && tmp != ""
}

func (m *Meta) JustSet(name string, val any) *XRError {
	log.VPrintf(4, "JustSet: m(%s).JustSet(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eJustSetPath(name, val)
}

func (r *Resource) JustSetMeta(name string, val any) *XRError {
	log.VPrintf(4, "JustSetMeta: r(%s).Set(%s,%v)", r.UID, name, val)
	meta := r.MustFindMeta(false)
	return meta.Entity.eJustSetPath(name, val)
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

	v, xErr := r.GetDefault()
	PanicIf(xErr != nil, "%s", xErr)
	return v.JustSet(name, val)
}

func (m *Meta) SetSave(name string, val any) *XRError {
	log.VPrintf(4, "SetSave: m(%s).SetSave(%s,%v)", m.Resource.UID, name, val)
	return m.Entity.eSetSave(name, val)
}

func (r *Resource) SetSaveMeta(name string, val any) *XRError {
	log.VPrintf(4, "SetSaveMeta: r(%s).Set(%s,%v)", r.UID, name, val)

	meta := r.MustFindMeta(false)
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

	v, xErr := r.GetDefault()
	PanicIf(xErr != nil, "%s", xErr)

	return v.SetSave(name, val)
}

func (r *Resource) Touch() bool {
	return r.MustFindMeta(false).Touch()
}

func (r *Resource) MustFindMeta(anyCase bool) *Meta {
	meta, xErr := r.FindMeta(anyCase)
	if xErr != nil {
		panic(xErr)
	}
	if IsNil(meta) {
		panic(fmt.Sprintf("tx: %s Meta is missing: %s (rXID=%s)",
			r.tx.uuid, r.UID, r.XID))
	}
	return meta
}

func (r *Resource) FindMeta(anyCase bool) (*Meta, *XRError) {
	log.VPrintf(3, ">Enter: FindMeta(%v)", anyCase)
	defer log.VPrintf(3, "<Exit: FindMeta")

	// Resource/Meta/Version are locked together as one family (see
	// lockEntityFamily()) - Meta's effective access mode is always
	// derived from its parent Resource's lock state, never chosen
	// independently by the caller. If you need a FOR_WRITE Meta, call
	// r.Lock() first (which locks the whole family), then call
	// FindMeta()/MustFindMeta() - there's no separate "accessMode" to
	// think about. This removes the caller-mistake class of bug fixed
	// in newestVersionID()/HTTPDeleteVersions() (passing FOR_READ by
	// mistake, or simply not realizing the family is already locked,
	// and silently getting a stale RR snapshot for a row that's
	// conceptually already locked in this Tx).
	accessMode := FOR_READ
	if r.AccessMode == FOR_WRITE {
		accessMode = FOR_WRITE
	}

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
func (r *Resource) FindVersion(id string, anyCase bool) (*Version, *XRError) {
	log.VPrintf(3, ">Enter: FindVersion(%s,%v)", id, anyCase)
	defer log.VPrintf(3, "<Exit: FindVersion")

	if id == "" { // just incase
		return nil, nil
	}

	// Same reasoning as FindMeta() above - Resource/Meta/Version are
	// locked together as one family, so a Version's effective access
	// mode always mirrors its parent Resource's - no separate
	// accessMode arg for callers to get wrong.
	accessMode := FOR_READ
	if r.AccessMode == FOR_WRITE {
		accessMode = FOR_WRITE
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
func (r *Resource) GetDefault() (*Version, *XRError) {
	meta, xErr := r.FindMeta(false)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)
	if meta == nil {
		// Resource (and its Meta) no longer exists - e.g. it was
		// deleted earlier in this same Tx after being marked for a
		// cascade run. Nothing to do.
		return nil, nil
	}

	val := meta.GetAsString("defaultversionid")
	return r.FindVersion(val, false)
}

func (r *Resource) GetVersionMode() VersionMode {
	vm := r.ResourceModel.GetVersionMode()
	apis, ok := VersionModes[strings.ToLower(vm)]
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
	return r.FindVersion(vid, false)
}

func (r *Resource) EnsureLatest() *XRError {
	log.Trace("EnsureLatest", r.XID)()
	meta := r.MustFindMeta(false)

	currentDefault := meta.GetAsString("defaultversionid")

	// log.Printf("In %s.ensurelatest, defID: %q", r.UID, currentDefault)

	// Since defaultversionid and defaultversionsticky are so closely related
	// we need to make sure that defaultversionsticky's default value is
	// applied before we calculate the defaultversionid if it's missing.
	if meta.NewObject != nil && meta.NewObject["defaultversionsticky"] == nil {
		def := r.ResourceModel.MetaAttributes["defaultversionsticky"].Default

		// Should never be nil but just in case
		if !IsNil(def) {
			meta.NewObject["defaultversionsticky"] = def
		}
	}

	if meta.Get("defaultversionsticky") != true || currentDefault == "" {
		newDefault, xErr := r.GetNewestVersionID()
		Must(xErr)
		PanicIf(newDefault == "", "No versions")

		// Only update if it changed
		if currentDefault != newDefault {
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
		v, xErr = r.FindVersion(vID, false)
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

	meta := r.MustFindMeta(false)

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

	var xErr *XRError
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
	obj           Object
	addType       AddType
	createVersion bool
	more          bool
}

// returns *Meta, isNew, error
// "createVersion" means we should create a version if there isn't already
// one there. This will only happen when the client talks directly to "meta"
// w/o the surrounding Resource object. AND, for now, we only do it when
// we're removing the 'xref' attr. Other cases, the http layer would have
// already create the Resource and default version for us.
func (r *Resource) UpsertMeta(mu *MetaUpsert) (*Meta, bool, *XRError) {
	log.VPrintf(3, ">Enter: UpsertMeta(%s,%v,%v,%v)", r.UID, mu.addType, mu.createVersion, mu.more)
	defer log.VPrintf(3, "<Exit: UpsertMeta")

	// log.Printf("UpsertMeta: OBJ: %s", ToJSON(mu.obj))

	if xErr := r.Registry.SaveModel(false); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(mu.obj, r.XID+"/meta"); xErr != nil {
		return nil, false, xErr
	}

	meta := r.MustFindMeta(false)

	if meta.Get("readonly") == true {
		if r.tx.RequestInfo.HasIgnore("readonly") {
			return meta, false, nil
		} else {
			return nil, false, NewXRError("readonly", r.XID)
		}
	}

	PanicIf(mu.obj == nil, "obj is nil")

	// Just in case we need it, save the Resource's epoch value. If this
	// is an xref'd Resource then it'll actually be the target's epoch.
	// Use meta.Object because it's possible that upsertResource changed
	// meta.NewObject["xref"] to null and we need the xref value prior to
	// any changes due to the current operation.
	targetEpoch := 0
	if targetXref := meta.Object["xref"]; targetXref != nil {
		tStr := targetXref.(string)
		tgtR, xErr := meta.Registry.FindResourceByXID(tStr, meta.XID, FOR_READ)
		if xErr != nil {
			return nil, false, xErr
		}
		// xref might point to a non-existing resource
		if tgtR != nil {
			targetEpochAny := tgtR.Get("epoch")
			targetEpoch = NotNilInt(&targetEpochAny)
		}
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
				def := r.ResourceModel.MetaAttributes["defaultversionsticky"].Default
				meta.NewObject["defaultversionsticky"] = def
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

				// Find the RM origin of our target
				parts := strings.Split(xrefAbsModel, "/")
				gm := r.ResourceModel.GroupModel.Model.Groups[parts[1]]
				if gm == nil {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail="+
							fmt.Sprintf("points to a non-existing Group "+
								"Type: %s", parts[1]))
				}

				rm := gm.Resources[parts[2]]
				if rm == nil {
					return nil, false, NewXRError("malformed_xref", meta.XID,
						"xref="+xref,
						"error_detail="+
							fmt.Sprintf("points to a non-existing Resource "+
								"Type: %s", parts[2]))
				}

				xrefAbsModel = rm.GetOriginAbstractModel()

				// Find RM origin of _this_ Resource
				targetAbsModel := r.ResourceModel.GetOriginAbstractModel()

				// They need to match
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

				Registry:  r.Registry,
				DbSID:     NewUUID(),
				ParentSID: r.DbSID,
				Plural:    "metas",
				Singular:  "meta",
				UID:       r.UID,

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

		meta.EntityInsert()

		if xErr := meta.JustSet(r.Singular+"id", r.UID); xErr != nil {
			return nil, false, xErr
		}

		r.tx.AddMeta(meta)

		if xErr := meta.SetSave("#nextversionid", 1); xErr != nil {
			return nil, false, xErr
		}
	}

	// Process any xref if there, or if it used to have it but now doesn't.
	// Basically if xref changed, or is set then do this...
	if hasXref || !IsNil(meta.Object["xref"]) {
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
			if xErr := meta.JustSet("xref", nil); xErr != nil {
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
				// This Resource was just created (in this same request)
				// and is becoming an xref before it ever had a chance to
				// be independently validated/saved as a normal Resource
				// - so it never really had an epoch of its own, and no
				// caller could have observed one yet. Treat it as if it
				// had already been through its normal first-epoch(1)
				// lifecycle anyway, so the eventual "restore from xref"
				// epoch (oldEpoch+1, see above) stays consistent with
				// every other Resource's numbering scheme instead of
				// needing a special case.
				oldEpoch = 1
			}
			meta.JustSet("#epoch", oldEpoch)

			oldCA := meta.Object["createdat"]
			if IsNil(oldCA) {
				oldCA = meta.tx.CreateTime
			}
			meta.JustSet("#createdat", oldCA)

			// meta.JustSet("createdat", nil)

			// DUG
			// RemoveReadonlyMetaAttributes(r.ResourceModel, meta.NewObject)

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

			if xErr := meta.JustSet("xref", xref); xErr != nil {
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

			if xErr = meta.ValidateAndSave(false); xErr != nil {
				return nil, false, xErr
			}

			return meta, isNew, nil
		}
	}

	/* DUG FT - old stuff
	oldSticky := meta.Object["defaultversionsticky"]
	newDefID := meta.NewObject["defaultversionid"]
	if IsNil(newDefID) {
		newDefID = ""
	}

	if oldSticky != true && newDefID == "" {
		meta.JustSet("defaultversionid", "")
	}
	*/

	if !mu.more {
		// If there's no more processing, go ahead and verify everything

		// Clear the defautversionid if we're not sticky

		newSticky := meta.NewObject["defaultversionsticky"]
		// If it's not set then we need to find out what it's default
		// value is (per the model). And if its not "true" then we should
		// erase any "defaultversionid" value since it's not going to be
		// used anyway
		if IsNil(newSticky) {
			stickyAttr := r.ResourceModel.MetaAttributes["defaultversionsticky"]
			if stickyAttr != nil && !IsNil(stickyAttr.Default) {
				newSticky = (stickyAttr.Default == true)
			}
		}

		if newSticky != true {
			meta.JustSet("defaultversionid", "")
		}

		r.tx.AddResourceToValidate(r, true, false)
	}

	return meta, isNew, nil
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

	if xErr := r.Registry.SaveModel(false); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(vu.Obj, r.XID+"/versions/"+vu.Id); xErr != nil {
		return nil, false, xErr
	}

	meta := r.MustFindMeta(false)

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
		// so that we allow for ANCESTORID_TBD by the system w/o allowing the
		// user to use it
		val, ok := vu.Obj["ancestorid"]
		if ok && !IsNil(val) {
			valStr, ok := val.(string)
			if !ok {
				return nil, false,
					NewXRError("invalid_attribute", r.XID,
						"name=ancestorid",
						"error_detail="+
							fmt.Sprintf(`must be a string, not %T`, val))
			}
			if xErr := IsValidID(valStr, "ancestorid"); xErr != nil {
				xErr.Subject = r.XID
				return nil, false, xErr
			}
		}
	}

	var v *Version
	gm, rm := r.GetModels()

	var xErr *XRError

	if vu.Id == "" {
		// No versionID provided so grab the next available one
		tmp := meta.Get("#nextversionid")
		nextID := NotNilInt(&tmp)
		for {
			vu.Id = strconv.Itoa(nextID)
			v, xErr := r.FindVersion(vu.Id, false)
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
		v, xErr = r.FindVersion(vu.Id, true)

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

				Registry:  r.Registry,
				DbSID:     NewUUID(),
				ParentSID: r.DbSID,
				Plural:    "versions",
				Singular:  "version",
				UID:       vu.Id,

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

		v.EntityInsert()

		v.tx.AddVersion(v)

		if xErr = v.JustSet("versionid", vu.Id); xErr != nil {
			return nil, false, xErr
		}

		// Touch owning Resource to bump its epoch abd modifiedat timestamp
		if r.Touch() {
			if xErr = r.ValidateAndSave(false); xErr != nil {
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
		anc, ok := v.NewObject["ancestorid"]
		if ok {
			// ancestorid was explicitly set to null then point to latest
			// Otherwise it must be trying to point to a version, leave it
			if IsNil(anc) {
				v.NewObject["ancestorid"] = ANCESTORID_TBD
			}
		} else {
			// Not there, so try to grab old value, else point to latest
			anc, ok = v.Object["ancestorid"]
			if ok {
				v.NewObject["ancestorid"] = anc
			} else {
				v.NewObject["ancestorid"] = ANCESTORID_TBD
			}
		}
	}

	// _, touchedTS := v.NewObject["createdat"]
	// if touchedTS -> call EnsureLatest

	if xErr = v.ValidateAndSave(false); xErr != nil {
		return nil, false, xErr
	}

	// If there are no more versions to be processed for this Resource in
	// this transaction, go ahead and clean-up the versions wrt the latest
	// and ancestorid pointers
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

		/*
			if xErr = meta.ValidateAndSave(false); xErr != nil {
				return nil, false, xErr
			}
		*/

		r.tx.AddResourceToValidate(r, false, false)
	}

	return v, isNew, nil
}

// checkHasDocumentViolation returns non-nil XRError if hasdocument=false
// but any versions have document content stored in the ResourceContents DB
// table. Note: this deliberately does NOT look at whether
// singular/singularurl/singularproxyurl Props are set - those are only
// reserved attribute names when hasdocument=true (see AddAttribute() in
// shared_model.go), so when hasdocument=false a user may have legitimately
// defined one of those exact names as their own extension attribute (or
// used a "*" wildcard extension). That data is 100% valid and must not be
// flagged here. If no such extension is defined, the generic attribute
// validator (run elsewhere, on every PUT) already correctly rejects it as
// an unknown_attribute - there's no need to duplicate that check here.
// The one thing generic attribute validation can never see is actual
// stored document bytes, which can never be legal once hasdocument=false -
// that's the only thing left to check for.
func (r *Resource) checkHasDocumentViolation() *XRError {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from ValidateResource()
	// on the write path.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// The nested EXISTS subquery below reads ResourceContents, a
	// different table than the outer query's Versions - MySQL does NOT
	// propagate the outer query's FOR UPDATE into a correlated
	// subquery's own table reads (confirmed empirically elsewhere in
	// this investigation, e.g. the VersionAncestors view bug), so
	// lockExpr must be applied to the subquery explicitly too.
	query := `
		SELECT v.Path FROM Versions v
		WHERE v.ResourceSID = ?
		AND EXISTS (
			SELECT 1 FROM ResourceContents rc
			WHERE rc.VersionSID = v.SID` + lockExpr + `
		)
		LIMIT 1` + lockExpr

	results := Query(r.tx, query, r.DbSID)
	defer results.Close()

	row := results.NextRow()
	if row != nil {
		// Found a version with document content
		versionPath := "/" + string((*(row[0])).([]byte))
		return NewXRError("hasdocument_violation", versionPath)
	}

	return nil
}

// Run all constrait check on the Resource - see:
//
//	spec.md#resource-processing-algorithm
func (r *Resource) ValidateResource(onlyMetaChanged bool, force bool) *XRError {
	// onlyMetaChanged indicates whether we need to do ALL checks or just
	// ones that might have changed due to a meta.* attribute.
	// If any Version actually changed we should run all checks.
	// "force" will check things even if they haven't changed.

	log.VPrintf(3, ">Enter: ValidateResource(r:%s only:%v, force:%v)", r.UID, onlyMetaChanged, force)
	defer log.VPrintf(3, "<Exit: ValiateResource")

	// We're about to fully (re-)validate r ourselves right now, so drop
	// any pending AddResourceToValidate() mark for it - otherwise
	// Registry.Validate() would redundantly run ValidateResource() for
	// r again later in this same Tx (e.g. a Version/Meta save that
	// happened as part of this very call already marked r via
	// Entity.Save()). Also drop it from ResourcesValidatingBatch
	// (if present) - see that field's doc comment - so any other
	// xref source's runCascade() checking "is my target still pending
	// in this batch" correctly sees that r's own validation (and its
	// unconditional xref fan-out) has now started/is about to run.
	if r.tx.ResourcesToValidate != nil {
		delete(r.tx.ResourcesToValidate, r.DbSID)
	}
	delete(r.tx.ResourcesValidatingBatch, r.DbSID)

	// On the way out, delete any mark this call's OWN body may have
	// re-added to ResourcesToValidate (e.g. EnsureLatest()'s
	// meta.SetSave("defaultversionid", ...) -> Entity.Save()
	// -> AddResourceToValidate()) - this call is about to account for
	// that change itself (via runCascade() below), so leaving that
	// self-mark in place would cause a second, fully redundant
	// ValidateResource() run later when Registry.Validate() drains the
	// Tx.
	defer func() {
		delete(r.tx.ResourcesToValidate, r.DbSID)
	}()

	// If anything changed in the Resource, causing us to validate it, then
	// assume we need to run its Group's Validate() func as well to ensure
	// the constraints are still valid
	r.tx.AddGroupToValidate(r.Group)

	meta := r.MustFindMeta(false)

	// If xref is set then we don't need to check anything, but this
	// Resource (as an xref source) may still need its default-version-
	// copy cascade/xref fan-out (re-)run (e.g. its own Meta.xref was
	// just set/changed) - run it before returning so its mirrored data
	// isn't left stale.
	if meta.GetAsString("xref") != "" {
		r.runCascade()
		return nil
	}

	/* DUG strict
	if xErr := meta.ValidateAndSave(false, force); xErr != nil {
		return xErr
	}
	*/

	// Check hasdocument violations when force=true (model changed)
	if force && !r.ResourceModel.GetHasDocument() {
		if xErr := r.checkHasDocumentViolation(); xErr != nil {
			return xErr
		}
	}

	// Clean-up and verify all AncestorID attributes before we continue
	if !onlyMetaChanged {
		if xErr := r.CheckAncestors(); xErr != nil {
			return xErr
		}
	}

	// Make sure latest is set properly
	if xErr := r.EnsureLatest(); xErr != nil {
		return xErr
	}

	// Validate compat/format between Versions if needed
	if xErr := r.EnsureCompat(force); xErr != nil {
		return xErr
	}
	// Flush any system props EnsureCompat() buffered (on this or other
	// Versions of this Resource) so they're visible to the response
	// that's about to be serialized, well before this Tx actually
	// commits (see Tx.FlushSystemProps()).
	r.tx.FlushSystemProps()

	// Make sure all attribtues with matchversions=true are the same for
	// all versions
	if xErr := r.EnsureMatchVersions(force); xErr != nil {
		return xErr
	}

	if !onlyMetaChanged {
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
	}

	// Save meta if needed
	if xErr := meta.ValidateAndSave(force); xErr != nil {
		return xErr
	}

	if xErr := r.ValidateAndSave(force); xErr != nil {
		return xErr
	}

	// All of this Resource's own Version/Meta processing for the
	// current request is done - (re-)run its default-version-copy
	// cascade/xref fan-out exactly once, against the final state.
	r.runCascade()

	return nil
}

// runCascade (re)builds r's default-version-copy cascade and xref
// fan-out (IsXrefPropCopy/IsXrefVerCopy synthetic rows, plus mirrored
// data on any xref target). Idempotent - safe to call even if nothing
// about r changed since the last run. Called once per Resource per Tx
// as the final step of ValidateResource() - see
// Tx.AddResourceToValidate()'s doc comment for why this is deferred
// until then rather than run immediately every time a Version/Meta
// belonging to r is saved.
//
// Both the xref-cascade and default-version-cascade steps below are
// skipped whenever nothing relevant to them changed this Tx, to avoid
// unnecessary DB round trips - see plan/design notes for the "cascade-
// skip-when-unchanged" optimization. r's Meta is guaranteed non-nil
// here: ValidateResource() (runCascade()'s only caller) already calls
// r.MustFindMeta() before ever reaching this point.
func (r *Resource) runCascade() {
	meta := r.MustFindMeta(false)

	// (Re)build r's own IsXrefPropCopy/IsXrefVerCopy rows first, in
	// case r's own Meta.xref was just set/changed/cleared -
	// SaveDefaultVersionCascade (next) reads those synthetic Version
	// rows when r has no real Versions of its own (r is an xref
	// source - see its "No real default Version" branch), so this
	// must run before it. Only run the Delete/Insert halves that could
	// actually find/produce rows: if xref was unset, there are no
	// stale rows to delete; if it's now unset, the insert would just
	// re-check and no-op.
	oldXref := meta.GetOriginAsString("xref")
	newXref := meta.GetAsString("xref")
	if oldXref != newXref {
		if oldXref != "" {
			meta.SaveXrefCascadeDelete()
		}
		if newXref != "" {
			// This Registry now has at least one xref - flip the
			// fast-path flag immediately (DB + in-memory) if it isn't
			// already set, so later cascades THIS SAME Tx (e.g. a
			// batch-created sibling that's this xref's own target)
			// correctly see UsesXref=true rather than a stale false
			// from before this Tx started. Safe/cheap: only runs the
			// one time this flips from false to true, ever, per
			// Registry (see init.sql's Registries.UsesXref comment for
			// the full design, including why clearing it back to false
			// is instead handled lazily via DB triggers).
			if !r.tx.Registry.UsesXref {
				DoZeroOne(r.tx,
					`UPDATE Registries SET UsesXref=true WHERE SID=? AND UsesXref=false`,
					r.tx.Registry.DbSID)
				r.tx.Registry.UsesXref = true
			}

			// If the xref target is itself still pending validation in
			// the SAME drain-loop batch as r (i.e. its ValidateResource()
			// hasn't started yet - see Tx.ResourcesValidatingBatch's doc
			// comment), skip our own insert: the target's own
			// runCascade(), whenever it runs (later in this same
			// batch), unconditionally fans out to every current xref
			// source via SaveXrefFanOutForTarget (which re-queries
			// Metas.xRefPath fresh, so it'll pick us up), making our own
			// insert here redundant work that would just get
			// immediately rebuilt anyway.
			skipInsert := false
			if _, target, xErr := r.GetXref(); xErr == nil && target != nil {
				if r.tx.ResourcesValidatingBatch[target.DbSID] {
					skipInsert = true
				}
			}
			if !skipInsert {
				meta.SaveXrefCascadeInsert()
			}
		}
	}

	// The default-version cascade needs to (re)run if: xref changed
	// (flips r between having a real default Version and mirroring the
	// xref target's synthetic one), defaultversionid itself changed, or
	// the final default Version's content actually changed this Tx
	// (EpochSet - bumped by Save() whenever a real content change
	// happens), or a system prop on it actually changed value
	// (OriginSystem != System - OriginSystem is the pre-Tx snapshot,
	// captured once by EnsureNewSystem(); comparing it against the
	// current System catches a real diff even across multiple
	// SetSystemDBProperty()/SaveSystemProps() flush cycles this Tx,
	// and - unlike a "changed" flag - survives a mid-Tx Refresh() since
	// Refresh() never resets OriginSystem, same as OriginObject).
	defaultVerCascadeNeeded := oldXref != newXref ||
		meta.GetOriginAsString("defaultversionid") != meta.GetAsString("defaultversionid")
	if !defaultVerCascadeNeeded {
		finalDefVer, _ := r.GetDefault()
		// finalDefVer can be nil for an xref source with no real
		// Versions of its own (its "default version" is really the
		// xref target's synthetic copy) - nothing of r's OWN to have
		// changed in that case, so nothing extra needed here (any
		// change to the target itself is handled independently by the
		// target's own unconditional fan-out).
		defaultVerCascadeNeeded = finalDefVer != nil &&
			(finalDefVer.EpochSet ||
				(finalDefVer.OriginSystem != nil &&
					!reflect.DeepEqual(finalDefVer.OriginSystem, finalDefVer.System)))
	}
	if defaultVerCascadeNeeded {
		r.SaveDefaultVersionCascade()
	}

	// Skip entirely if this Registry has never used xref - see
	// init.sql's Registries.UsesXref comment for the full design.
	if r.tx.Registry.UsesXref {
		r.SaveXrefFanOutForTarget()
	}
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
	VID        string
	AncestorID string
	CreatedAt  string
	Pos        string // 0-root, 1-middle, 2-leaf
}

func (r *Resource) GetVersionIDs() ([]string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID().
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// Find all version IDs for this Resource
	results := Query(r.tx, `
            SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
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
	// Find all versions whose AncestorID = its vID

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// GetOrderedVersionIDs().
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	results := Query(r.tx, `
            SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND UID=AncestorID`+lockExpr,
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

// Return all versions whose 'ancestorid' is ANCESTORID_TBD or points to a
// missing version (which include pointing to null).
// Note that the results is ordered so that we can process the ones with
// a missing AncestorID in oldest->newest order
func (r *Resource) GetProblematicVersions() ([]*VersionAncestor, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). This is called from
	// CheckAncestors() on the write path, and its correlated subquery
	// needs its own lock hint too (see HasCircularAncestors()).
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// Find all versions that point to non-existing versions
	results := Query(r.tx, `
            SELECT v1.UID, v1.AncestorID, v1.CreatedAt FROM Versions AS v1
			WHERE v1.RegistrySID=? AND
			      v1.ResourceSID=? AND
                  (v1.AncestorID='`+ANCESTORID_TBD+`' OR (
			          v1.UID<>v1.AncestorID AND
			          NOT EXISTS(SELECT 1 FROM Versions AS v2
				                WHERE v2.RegistrySID=v1.RegistrySID AND
							          v2.ResourceSID=v1.ResourceSID AND
							          v2.UID=v1.AncestorID`+lockExpr+`)))
			ORDER BY CreatedAt ASC, UID ASC`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	vers := ([]*VersionAncestor)(nil)
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		vers = append(vers, &VersionAncestor{
			VID:        NotNilString(row[0]),
			AncestorID: NotNilString(row[1]),
			CreatedAt:  NotNilString(row[2]),
			Pos:        "n/a",
		})
	}

	return vers, nil
}

func (r *Resource) GetChildVersionIDs(parentVID string) ([]string, *XRError) {
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from WillDelete() on
	// the write path.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// Find all versions that point 'parentVID'.
	// Note that roots will include themselves - not sure if this is ok or not
	results := Query(r.tx, `
			SELECT UID FROM Versions
			WHERE RegistrySID=? AND ResourceSID=? AND AncestorID=?`+lockExpr,
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
	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// ManualVersionMode.newestVersionID(). Called from write paths
	// (e.g. Version.Delete(), UpsertVersion()).
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	// Get the list of Version IDs for this Resource (oldest first)
	results := Query(r.tx, `
	        SELECT COUNT(*) FROM Versions
			WHERE RegistrySID=? AND ResourceSID=?`+lockExpr,
		r.Registry.DbSID, r.DbSID)
	defer results.Close()

	row := results.NextRow()
	return NotNilInt(row[0]), nil
}

func (r *Resource) HasCircularAncestors() ([]string, *XRError) {
	// Get the list of Version IDs that are part of circular ancestor refs

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as ManualVersionMode.newestVersionID()
	// / GetOrderedVersionIDs(): otherwise this query runs against this
	// tx's original REPEATABLE-READ snapshot and can miss Version rows
	// committed by other Txs after that snapshot was established,
	// producing false "circular reference" errors.
	//
	// NOTE: we deliberately do NOT run the previous recursive-CTE/VIEW-
	// based query here anymore. MySQL doesn't propagate an outer FOR
	// UPDATE into a recursive CTE's internal correlated subqueries, AND
	// (worse) it doesn't even allow a "FOR UPDATE" clause inside one of
	// a recursive CTE's own UNION branches (syntax error) - so there's
	// no way to make every part of that query honor lockExpr. Instead,
	// fetch ALL of this Resource's (UID, AncestorID) pairs with a
	// single flat, fully lockExpr'd query, and do the cycle-detection
	// logic itself in Go, where the correctness of "did we see every
	// Version row as of the lock" only depends on this one query.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}
	results := Query(r.tx, `
		SELECT UID, AncestorID FROM Versions
		WHERE ResourceSID=?`+lockExpr,
		r.DbSID)
	defer results.Close()

	ancestorOf := map[string]string{}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		ancestorOf[NotNilString(row[0])] = NotNilString(row[1])
	}

	// A Version is "OK" (not circular) if, by walking its AncestorID
	// chain, we reach a self-referencing root (AncestorID==UID) without
	// re-visiting a Version we've already seen on this same walk. Any
	// Version whose chain loops back on itself before reaching such a
	// root is part of a cycle.
	vIDs := ([]string)(nil)
	for uid := range ancestorOf {
		seen := map[string]bool{}
		cur := uid
		circular := false
		for {
			if cur == uid && seen[cur] {
				// Walked all the way back to our own starting Version -
				// uid itself is part of the cycle.
				circular = true
				break
			}
			if seen[cur] {
				// Looped back to some OTHER already-visited Version -
				// that means uid merely points (directly or
				// transitively) INTO a cycle that doesn't include uid
				// itself (e.g. 3->2->1->2->...): uid is not itself
				// circular, just unreachable/problematic, which is a
				// separate concern (not reported here).
				break
			}
			seen[cur] = true
			anc, ok := ancestorOf[cur]
			if !ok {
				// Points at a Version that doesn't exist - not our
				// concern here, GetProblematicVersions() handles that
				// case. Treat as non-circular so we don't double-report.
				break
			}
			if anc == cur {
				// Self-referencing root - end of chain, not circular
				break
			}
			cur = anc
		}
		if circular {
			vIDs = append(vIDs, uid)
		}
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
	// xref resource have no versios, so exit
	if r.IsXref() {
		return nil
	}

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
			v, xErr := r.FindVersion(verIDs[0].VID, false)
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

	meta := r.MustFindMeta(false)
	if rm.GetMaxVersions() == 1 && meta.Get("defaultversionsticky") == true {
		return NewXRError("setdefaultversionsticky_false", meta.XID)
	}

	return nil
}

func (r *Resource) Delete() *XRError {
	log.VPrintf(3, ">Enter: Resource.Delete(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: Resource.Delete")

	meta := r.MustFindMeta(false)

	if meta.Get("readonly") == true {
		return NewXRError("readonly", r.XID)
	}

	if xErr := meta.Delete(); xErr != nil {
		return xErr
	}

	if r.Group.Touch() {
		if xErr := r.Group.ValidateAndSave(false); xErr != nil {
			return xErr
		}
	}

	// Any xref source's stale mirror is cleared by ResourcesTrigger
	// (init.sql), which fires for every deletion path (this, whole-
	// Group delete, whole-Registry delete) uniformly.
	DoOne(r.tx, `DELETE FROM Resources WHERE SID=?`, r.DbSID)

	// No longer anything to validate - drop any pending mark so
	// Registry.Validate() doesn't try to (re-)validate a Resource whose
	// Meta (and now Resource row) no longer exist.
	if r.tx.ResourcesToValidate != nil {
		delete(r.tx.ResourcesToValidate, r.DbSID)
	}

	// Delete any pending changes so dirty check doesn't fail
	r.NewObject = nil
	r.tx.RemoveFromCache(&r.Entity)

	return nil
}

func (m *Meta) Delete() *XRError {
	log.VPrintf(3, ">Enter: Meta.Delete(%s)", m.UID)
	defer log.VPrintf(3, "<Exit: Meta.Delete")

	// Props/Entities rows for this Meta are cleaned up by
	// ResourcesTrigger (ParentSID=OLD.SID) when the owning Resource is
	// deleted right after this.
	DoOne(m.tx, `DELETE FROM Metas WHERE SID=?`, m.DbSID)

	// Delete any pending changes so dirty check doesn't fail
	m.NewObject = nil
	m.tx.RemoveFromCache(&m.Entity)

	return nil
}

func (r *Resource) GetVersions() ([]*Version, *XRError) {
	list := []*Version{}

	entities, xErr := RawEntitiesFromQuery(r.tx, r.Registry.DbSID,
		FOR_WRITE, `e.ParentSID=? AND e.Type=?`, r.DbSID, ENTITY_VERSION)
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

type FormatChecker interface {
	// 1st return arg: bool - did we do the check?
	// 2nd return arg: if no check done, then why?
	// 3rd return arg: the error to return if we need to return an error

	IsValid(version *Version) (bool, string, *XRError)
	// 'direction' == backward, forward
	IsCompatible(direction string, oldVersion, newVersion *Version) (bool, string, *XRError)
}

// case insensitive 'format' values'
var SupportedFormatCheckers = map[string]FormatChecker{}

func RegisterFormat(name string, format FormatChecker) {
	SupportedFormatCheckers[strings.ToLower(name)] = format
	AddSupportedFormat(name, []string{
		"backward",
		"backward_transitive",
		"forward",
		"forward_transitive",
		"full",
		"full_transitive",
	})
}

// checker, registered format string
func GetFormatChecker(format string) (FormatChecker, string) {
	// Look for an exact match first - we choose those over wildcards
	format = strings.ToLower(format)
	checker := SupportedFormatCheckers[format]
	if checker != nil {
		return checker, format
	}

	// Just grab the first format whose pattern matches - not determinant
	for pattern, checker := range SupportedFormatCheckers {
		if Match(strings.ToLower(pattern), format) {
			return checker, pattern
		}
	}

	return nil, ""
}

// This will check "format" as well.
// "force" check all Verisons even if we don't think we need to.
func (r *Resource) EnsureCompat(force bool) *XRError {
	log.VPrintf(3, ">Enter: EnsureCompat(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: EnsureCompat")

	meta := r.MustFindMeta(false)

	validateFormat := r.ResourceModel.GetValidateFormat()
	validateCompat := r.ResourceModel.GetValidateCompatibility()

	oldCompat := meta.GetOrigin("compatibility")
	newCompat := meta.Get("compatibility")

	if validateCompat && newCompat == "" {
		return NewXRError("invalid_attribute", meta.XID,
			"name=compatibility",
			"error_detail=can't be an empty string")
	}

	// Doing neither so just return. No need to clear the *validated/
	// *validatedreason props here - Model.ApplyNewModel() already
	// bulk-clears them registry-wide, once, at the moment validation
	// was turned off for this ResourceType (see
	// Registry.clearValidationSystemProps()), so there's nothing stale
	// left to worry about on every single save.
	if !validateCompat && !validateFormat {
		return nil
	}

	strict := r.ResourceModel.GetStrictValidation()

	// Check all versions, not just changed ones?
	// Check all versions if we've changed compat & we're validating
	doAll := force ||
		(validateCompat && oldCompat != newCompat && newCompat != "")

	// Get the complete list of Versions and ancestor orders.
	// We'll use this to build our easy look-ups as we process things.
	orderedVAs, xErr := r.GetOrderedVersionIDs()
	if xErr != nil {
		return xErr
	}

	childrenMap := map[string][]string{} // v.UID -> []child.UID
	changedVersions := []string{}        // v.UID

	doneChecks := map[string]bool{}    // "direction>oldID">"newID" -> true
	ancestorMap := map[string]string{} // v.UID -> v.ancestorID

	// 'direction' = 'backward', 'forward'
	doCheckCompat := func(direction string, oldVID string, newVID string) *XRError {
		PanicIf(oldVID == "", "can't be empty")
		PanicIf(newVID == "", "can't be empty")

		key := direction + ">" + oldVID + ">" + newVID
		if doneChecks[key] {
			// Already checked
			return nil
		}
		oldV, xErr := r.FindVersion(oldVID, false)
		PanicIf(!IsNil(xErr) || IsNil(oldV), "%s: %s", oldVID, ToJSON(xErr))
		newV, xErr := r.FindVersion(newVID, false)
		PanicIf(!IsNil(xErr) || IsNil(newV), "%s: %s", newVID, ToJSON(xErr))

		// I'm always compatible with myself. Just in case caller doesn't check
		if oldVID == newVID {
			newV.SetSystemDBProperty(NewPPP("compatibilityvalidated"), true)
			newV.SetSystemDBProperty(NewPPP("compatibilityvalidatedreason"),
				nil)
			return nil
		}

		// Do actual check here
		format := newV.GetAsString("format")
		checker, formatPattern := GetFormatChecker(format)

		// Shouldn't be needed, but just in case
		if !r.Registry.Capabilities.CompatibilityEnabled(formatPattern,
			newCompat.(string)) {
			return NewXRError("compatibility_unknown", r.XID+"/meta",
				"compat="+newCompat.(string),
				"format="+formatPattern)
		}

		checked, reason, xErr := checker.IsCompatible(direction, oldV, newV)
		PanicIf(!checked && reason == "", "Bad state")

		if xErr != nil && (checked || strict) {
			return xErr
		}

		newV.SetSystemDBProperty(NewPPP("compatibilityvalidated"), checked)

		if reason == "" {
			newV.SetSystemDBProperty(NewPPP("compatibilityvalidatedreason"),
				nil)
		} else {
			newV.SetSystemDBProperty(NewPPP("compatibilityvalidatedreason"),
				reason)
		}

		doneChecks[key] = true
		return nil
	}

	// Loop over all of the Resource's Versions
	for _, va := range orderedVAs {
		ver, xErr := r.FindVersion(va.VID, false)
		if xErr != nil {
			return xErr
		}

		// For each Version, save it's list of ancestors for easy lookup later.
		// Note that we may need this even if the Version didn't change
		oldList := childrenMap[va.AncestorID]
		if va.AncestorID != va.VID {
			// Don't add roots to themselves
			childrenMap[va.AncestorID] = append(oldList, va.VID)
		}

		// Save for easy look-up later
		ancestorMap[va.VID] = va.AncestorID
		PanicIf(va.AncestorID == "", "Not good")

		// Build our list of changed Versions.
		// So, either doAll=true, or version's epoch was changed, otherwise
		// skip it, it didn't change.
		// doAll would be true in cases like changing the 'compat' value
		if doAll || ver.EpochSet {
			if validateFormat {
				newFormat := ver.GetAsString("format")
				if newFormat == "" {
					// Regardless of being strict or not, turn off checks
					// and don't show any of the *validated attributes
					ver.SetSystemDBProperty(NewPPP("formatvalidated"), nil)
					ver.SetSystemDBProperty(NewPPP("formatvalidatedreason"),
						nil)
					ver.SetSystemDBProperty(NewPPP("compatibilityvalidated"),
						nil)
					ver.SetSystemDBProperty(NewPPP(
						"compatibilityvalidatedreason"), nil)
					continue
				}

				checker, formatPattern := GetFormatChecker(newFormat)

				if IsNil(checker) {
					if strict {
						return NewXRError("format_unknown", ver.XID,
							"format="+newFormat)
					}
					ver.SetSystemDBProperty(NewPPP("formatvalidated"), false)
					ver.SetSystemDBProperty(NewPPP("formatvalidatedreason"),
						"Unknown format")
					if validateCompat {
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidated"), false)
						ver.SetSystemDBProperty(NewPPP(
							"compatibilityvalidatedreason"), "Unknown format")
					} else {
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidated"), nil)
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidatedreason"), nil)
					}
					continue
				}

				if !r.Registry.Capabilities.FormatEnabled(formatPattern) {
					return NewXRError("format_unknown", ver.XID,
						"format="+newFormat)
				}

				// Validate that the Version is valid per the "format"
				checked, reason, xErr := checker.IsValid(ver)
				PanicIf(!checked && reason == "", "Bad state")
				if xErr != nil && (checked || strict) {
					return xErr
				}

				ver.SetSystemDBProperty(NewPPP("formatvalidated"), checked)
				if reason == "" {
					ver.SetSystemDBProperty(NewPPP("formatvalidatedreason"),
						nil)
				} else {
					ver.SetSystemDBProperty(NewPPP("formatvalidatedreason"),
						reason)
				}

				if reason != "" {
					if validateCompat {
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidated"), false)
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidatedreason"), reason)
					} else {
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidated"), nil)
						ver.SetSystemDBProperty(
							NewPPP("compatibilityvalidatedreason"), nil)
					}
					continue
				}
			}

			// Only add to the list if we're checking for compat.
			// We don't do Compat checking here because we need to populate
			// our cache of data first (maps, arrays, etc)
			if validateCompat {
				changedVersions = append(changedVersions, va.VID)
			} else {
				ver.SetSystemDBProperty(NewPPP("compatibilityvalidated"), nil)
				ver.SetSystemDBProperty(NewPPP("compatibilityvalidatedreason"),
					nil)
			}
		}
	}

	// If compat isn't enabled at the model level, skip entirely - no
	// need to clear here, Model.ApplyNewModel()'s bulk sweep already
	// owns clearing stale values registry-wide the moment validation
	// was turned off (see clearValidationSystemProps()).
	if !validateCompat {
		return nil
	}

	// Model-level validation is on, but THIS Resource's "compatibility"
	// attribute isn't set - a per-instance condition, not a model
	// transition, so it isn't covered by the bulk sweep above. Still
	// need to clear it here.
	if IsNil(newCompat) {
		// clear compatvalidated attr for all versions
		r.ClearResourceSystemDBProperty(
			NewPPP("compatibilityvalidated"),
			NewPPP("compatibilityvalidatedreason"))
		return nil
	}

	// compat is case-insensitive
	newCompat = strings.ToLower(newCompat.(string))

	// for all changed versions do compat checking
	compatFound := false
	for _, verID := range changedVersions {
		// Already checked newFormat & checker in previous loop
		ver, xErr := r.FindVersion(verID, false)
		PanicIf(!IsNil(xErr) || IsNil(verID), "%s: %s", verID, ToJSON(xErr))

		newFormat := ver.GetAsString("format")
		_, formatPattern := GetFormatChecker(newFormat)

		if !r.Registry.Capabilities.CompatibilityEnabled(formatPattern,
			newCompat.(string)) {

			return NewXRError("compatibility_unknown", r.XID+"/meta",
				"compat="+newCompat.(string),
				"format="+formatPattern)
		}

		if newCompat == "backward" || newCompat == "full" {
			compatFound = true
			// compatible w/ the next oldest Ver
			xErr := doCheckCompat("backward", ancestorMap[verID], verID)
			if xErr != nil {
				return xErr
			}

			// compatible w/ all children
			for _, childUID := range childrenMap[verID] {
				xErr := doCheckCompat("backward", verID, childUID)
				if xErr != nil {
					return xErr
				}
			}
		}

		if newCompat == "backward_transitive" || newCompat == "full_transitive" {
			compatFound = true
			// compatible w/ all older Ver
			currentID := verID
			for {
				prevID := ancestorMap[currentID]
				if prevID == currentID { // root, so stop
					break
				}

				// Compatible with our next ancestor
				xErr := doCheckCompat("backward", prevID, currentID)
				if xErr != nil {
					return xErr
				}
				currentID = prevID
			}

			// Make sure we didn't break our children's compat
			for _, childUID := range childrenMap[verID] {
				xErr := doCheckCompat("backward", verID, childUID)
				if xErr != nil {
					return xErr
				}
			}
		}

		if newCompat == "forward" || newCompat == "full" {
			compatFound = true
			// compatible w/ the next newest Ver
			for _, childUID := range childrenMap[verID] {
				// Compatible with a descendent
				xErr := doCheckCompat("forward", verID, childUID)
				if xErr != nil {
					return xErr
				}
			}

			// Compatible w/ our ancestor
			xErr := doCheckCompat("forward", ancestorMap[verID], verID)
			if xErr != nil {
				return xErr
			}
		}

		if newCompat == "forward_transitive" || newCompat == "full_transitive" {
			compatFound = true
			// compatible w/ all newer Versions
			list := [][2]string{} // [old,new]
			// Start our psuedo-recursive list of old/new pairs to check
			for _, childID := range childrenMap[verID] {
				list = append(list, [2]string{verID, childID})
			}

			for len(list) != 0 {
				item := list[0] // [old,new]
				list = list[1:]

				xErr := doCheckCompat("forward", item[0], item[1])
				if xErr != nil {
					return xErr
				}

				// Now be recursive by adding this item's children to "list"
				for _, childID := range childrenMap[item[1]] {
					list = append(list, [2]string{item[1], childID})
				}
			}

			// Now check our ancestor
			xErr := doCheckCompat("forward", ancestorMap[verID], verID)
			if xErr != nil {
				return xErr
			}
		}

		if !compatFound {
			// Should we check this in the checkFn stuff instead???
			panic("should never get here")
			return NewXRError("compatibility_unknown", r.XID+"/meta",
				"compat="+newCompat.(string),
				"format=n/a")
		}
	}

	return nil
}

// Check to make sure all attributes with matchversions=true are validated
// to be the same across all Versions
func (r *Resource) EnsureMatchVersions(force bool) *XRError {
	log.VPrintf(3, ">Enter: EnsureMatchVersions(%s)", r.UID)
	defer log.VPrintf(3, "<Exit: MatchVersions")

	mvs := r.ResourceModel.GetMatchVersionAttributes()

	// FOR UPDATE only when r's Meta is already locked FOR_WRITE - same
	// RR-snapshot-staleness reasoning as HasCircularAncestors() /
	// GetOrderedVersionIDs(): otherwise this can miss sibling Version
	// rows committed by other Txs after this tx's snapshot was taken.
	lockExpr := ""
	if meta := r.tx.GetMeta(r); meta != nil && meta.AccessMode == FOR_WRITE {
		lockExpr = " FOR UPDATE"
	}

	for _, mv := range mvs {
		binary := ""
		if mv.MatchCase {
			binary = "BINARY"
		}

		query := fmt.Sprintf(`
            SELECT count(*),p.PropName,p.PropValue FROM Entities e
            LEFT JOIN Props AS p ON ( p.eSID=e.eSID AND p.PropName=?)
            WHERE e.RegSID = ?  AND e.ParentSID = ?  AND e.Type = ?
            GROUP BY %s PropValue`+lockExpr, binary)

		results := Query(r.tx, query, mv.Path.DB(),
			r.Registry.DbSID, r.DbSID, ENTITY_VERSION)
		defer results.Close()

		numEmpty := 0
		numNonEmpty := 0
		numValues := 0
		for {
			row := results.NextRow()
			if row == nil {
				break
			}
			if IsNil(*row[2]) {
				// if NotNilString(row[2]) == ""
				numEmpty = NotNilInt(row[0])
			} else {
				numNonEmpty++
			}
			numValues++
		}

		if numValues > 1 {
			return NewXRError("mismatched_version_attribute", r.XID,
				"name="+mv.Path.UI()).
				SetDetailf("Unique values: %d. Versions w/o values: %d.",
					numNonEmpty, numEmpty)
		}
	}

	return nil
}

// See entity.go's above EntityInsert()
// for the overall Props/Entities incremental-population design.

// SaveDefaultVersionCascade refreshes the IsDefaultVerCopy=true rows on
// r so they mirror whatever Version is currently r's default. Called
// whenever a Version is saved (in case it's the current default) or a
// Meta is saved (in case meta.defaultversionid just changed). r is nil
// if r is nil (no Resource to cascade) - a no-op in that case.
//
// r is the actual, strongly-typed *Resource (not just its SID) so we
// can follow its own in-memory fields plus tx.GetMeta(r)/tx.GetVersion(
// r,...) to find the current default Version's SID without any DB
// round-trip whenever this transaction already has them loaded.
func (r *Resource) SaveDefaultVersionCascade() {
	if r == nil {
		return
	}

	defer log.Trace("FullTree", r.XID)()

	resourceSID := r.DbSID

	Do(r.tx, `
        DELETE FROM Props WHERE eSID=? AND IsDefaultVerCopy=true`,
		resourceSID)

	// The Resource (and its Meta) may have been deleted earlier in this
	// same Tx after being marked for a cascade run (e.g. all of its
	// Versions were deleted, which per business rules deletes the
	// Resource itself). If so there's nothing left to cascade.
	meta, xErr := r.FindMeta(false)
	Must(xErr)
	if meta == nil {
		return
	}

	// Grab the default version. If there is none defined yet then
	// call EnsureLatest (if no xref) just so the processing continues.
	// If the default version needs to change based on later processing,
	// e.g. ancestor processing, then the a subsequent call to EnsureLatest
	// in resource.ValidateResource() should fix it. And hopefully, this
	// func will be called again to fix-up the default version props for
	// the resource.
	ver, xErr := r.GetDefault()
	Must(xErr)
	if !r.IsXref() && ver == nil {
		Must(r.EnsureLatest())
		ver, xErr = r.GetDefault()
		Must(xErr)
	}

	if ver == nil {
		// No real default Version - this Resource may be an xref
		// source with no Versions of its own, in which case its
		// "current default" is really the xref target's current
		// default, copied in as a synthetic Version by
		// SaveXrefVersionCopies(). Copy from THAT synthetic
		// Props eSID instead of Props.
		//
		// This is a source reading its xref TARGET's row (same
		// direction as SaveXrefCascadeInsert()'s tResults query), so it
		// needs its own FOR UPDATE too: a plain SELECT here would still
		// be pinned to this Tx's original RR snapshot, and could miss
		// the target entirely (or see a stale defaultVID) even if the
		// target Resource/Meta/Version was created AND committed by a
		// concurrent Tx after this Tx began - silently leaving this
		// source's mirrored default-version Props missing/stale.
		tResults := Query(r.tx, `
            SELECT v.SID FROM Metas AS srcM
            JOIN Resources AS tr ON (tr.RegistrySID=srcM.RegistrySID AND
                                      tr.Path=srcM.xRefPath)
            JOIN Metas AS m ON (m.ResourceSID=tr.SID)
            JOIN Versions AS v ON (v.ResourceSID=m.ResourceSID AND
                                    v.UID=m.defaultVID)
            WHERE srcM.ResourceSID=?
            FOR UPDATE`, resourceSID)
		tRow := tResults.NextRow()
		tResults.Close()
		if tRow == nil {
			return
		}
		targetDefVerSID := NotNilString(tRow[0])
		synthESID := fmt.Sprintf("-%s-%s", resourceSID, targetDefVerSID)

		// IsCalcDynamic isn't excluded here (unlike IsCalcStatic) so
		// the synthetic version's own "isdefault" row (already
		// correctly computed by SaveXrefVersionCopies(), since this
		// synthESID always corresponds to the target's CURRENT
		// default version) is copied in too - just like createdat/
		// modifiedat, it's simply mirrored content, not a special
		// case. If the xref is dangling (tRow == nil, above) nothing
		// gets copied at all, so "isdefault" - along with every other
		// mirrored attribute - is naturally absent, exactly like a
		// Resource with no default Version at all.
		Do(r.tx, `
            REPLACE INTO Props(
                RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
                PropName, PropValue, PropType, Abstract, DocView,
                IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
            SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
                   true, false, false
            FROM Props WHERE eSID=? AND IsXrefVerCopy=true
                  AND IsCalcStatic=false`,
			r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID,
			r.DbSID, r.UID, r.Path, r.Abstract, synthESID)
		return
	}

	// Fix up isdefault on every one of this Resource's OWN Versions -
	// they were each set based on their own state at their own last
	// Save(), so if the default just moved to a different Version, the
	// old default's row (and the new one's, if it wasn't the one that
	// triggered this call) would otherwise go stale. This must run
	// BEFORE the copy below, so ver's own "isdefault" row is already
	// correct by the time it gets mirrored into the Resource.
	Do(r.tx, `
        UPDATE Props AS ft
        JOIN Versions AS v ON (v.SID=ft.eSID)
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        SET ft.PropValue = IF(v.UID=m.defaultVID, 'true', 'false')
        WHERE v.ResourceSID=? AND ft.PropName=?`,
		resourceSID, "isdefault"+string(DB_IN))

	// IsCalcDynamic isn't excluded here (unlike IsCalcStatic) so ver's
	// own "isdefault" row (just fixed up above) is mirrored into the
	// Resource too - like createdat/modifiedat, it's just copied
	// content, no special-casing needed. It's simply absent whenever
	// there's no default Version to copy from at all (see the ver ==
	// nil branch above).
	Do(r.tx, `
        REPLACE INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?,?,?,?,?,?,?,?, PropName, PropValue, PropType, ?, false,
               true, false, false
        FROM Props
        WHERE eSID=? AND IsDefaultVerCopy=false AND IsXrefPropCopy=false
              AND IsXrefVerCopy=false AND IsCalcStatic=false`,
		r.Registry.DbSID, r.Type, r.Plural, r.Singular, r.ParentSID, r.DbSID,
		r.UID, r.Path, r.Abstract, ver.DbSID)
}

// SaveXrefVersionCopies (re)creates the synthetic Entities/
// Props Version rows for srcResource (a real, in-memory
// *Resource - either e.Self.(*Meta).Resource in the direct-Save() case,
// or one resolved via Registry.FindResourceByXID() in the xref fan-out
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
//
// Every INSERT...SELECT below that reads the target's Versions/Props
// uses FOR UPDATE: this is srcResource (a source) reading its xref
// TARGET's rows, the mirror image of SaveXrefFanOutForTarget's
// target-reads-sources direction (which already locks each source
// FOR_WRITE). Without FOR UPDATE here, a plain SELECT would still be
// pinned to this Tx's RR snapshot and could copy stale target Version/
// Prop data into the source's mirror even after a concurrent Tx
// already committed a newer Version, which would then feed this
// source's own Group constraint validation with stale mirrored data.
// (The DELETE...JOIN statements above don't need this: DELETE/UPDATE
// searches always read latest-committed data in InnoDB, unlike plain
// SELECTs - only these INSERT...SELECTs need the explicit FOR UPDATE.)
func (srcResource *Resource) SaveXrefVersionCopies(targetResourceSID string) {
	if srcResource == nil {
		return
	}

	defer log.Trace("FullTree", "%s,%s", srcResource.Path, targetResourceSID)()

	sourceResourceSID := srcResource.DbSID
	synthAbstract := srcResource.Abstract + string(DB_IN) + "versions"

	// Idempotent: this is called both from SaveXrefCascadeInsert
	// (which already cleared out ALL of this source's xref-version
	// rows first) and directly from SaveXrefFanOutForTarget
	// (which does not) - so clear out just the synthetic versions that
	// correspond to the target's CURRENT version set before
	// recreating them, or a second Save() of the same target Version
	// would hit a duplicate-key error here.
	Do(srcResource.tx, `
        DELETE ft FROM Props AS ft
        JOIN Versions AS v ON (ft.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)
	Do(srcResource.tx, `
        DELETE fe FROM Entities AS fe
        JOIN Versions AS v ON (fe.eSID=CONCAT('-', ?, '-', v.SID))
        WHERE v.ResourceSID=?`, sourceResourceSID, targetResourceSID)

	// One Entities row per target Version, all at once.
	Do(srcResource.tx, `
        REPLACE INTO Entities(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID,
            Abstract, Path, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID, ?,
               CONCAT(?, '/versions/', v.UID), true
        FROM Versions AS v WHERE v.ResourceSID=?
        FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, synthAbstract, srcResource.Path,
		targetResourceSID)

	// Copy each target Version's own props onto its corresponding
	// synthetic eSID, for every current Version at once (excluding the
	// target's own "xref" - Versions never have one, but kept for
	// parity with the old per-row exclusion).
	Do(srcResource.tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ft.PropName, ft.PropValue, ft.PropType, ?, false,
               false, false, true
        FROM Versions AS v
        JOIN Props AS ft ON (ft.eSID=v.SID)
        WHERE v.ResourceSID=? AND ft.IsDefaultVerCopy=false
              AND ft.IsXrefPropCopy=false AND ft.IsXrefVerCopy=false
              AND ft.IsCalcStatic=false AND ft.IsCalcDynamic=false
              AND ft.PropName<>?
        FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path, synthAbstract,
		targetResourceSID, "xref"+string(DB_IN))

	// Calculated attrs for every synthetic version at once: xid and
	// RESOURCEid (using the SOURCE resource's singular/UID, since
	// that's every synthetic version's effective parent) are static -
	// wholesale recreated here only because the whole synthetic-
	// version set itself is being recreated (the xref pointer moved),
	// not because they individually change; isdefault is genuinely
	// dynamic (mirrors the target's own per-Version isdefault).
	Do(srcResource.tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, CONCAT('/', ?, '/versions/', v.UID), 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v WHERE v.ResourceSID=?
        FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"xid"+string(DB_IN), srcResource.Path, synthAbstract, targetResourceSID)

	Do(srcResource.tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               CONCAT(r.Singular, ?), r.UID, 'string', ?, false,
               false, false, true, true, false
        FROM Versions AS v
        JOIN Resources AS r ON (r.SID=?)
        WHERE v.ResourceSID=?
        FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"id"+string(DB_IN), synthAbstract, sourceResourceSID,
		targetResourceSID)

	Do(srcResource.tx, `
        INSERT INTO Props(
            RegSID, Type, Plural, Singular, ParentSID, eSID, UID, Path,
            PropName, PropValue, PropType, Abstract, DocView,
            IsDefaultVerCopy, IsXrefPropCopy, IsXrefVerCopy,
            IsCalcStatic, IsCalcDynamic)
        SELECT ?, ?, ?, ?, ?, CONCAT('-', ?, '-', v.SID), v.UID,
               CONCAT(?, '/versions/', v.UID),
               ?, IF(m.defaultVID=v.UID, 'true', 'false'), 'boolean', ?,
               false, false, false, true, false, true
        FROM Versions AS v
        JOIN Metas AS m ON (m.ResourceSID=v.ResourceSID)
        WHERE v.ResourceSID=?
        FOR UPDATE`,
		srcResource.Registry.DbSID, ENTITY_VERSION, "versions", "version",
		sourceResourceSID, sourceResourceSID, srcResource.Path,
		"isdefault"+string(DB_IN), synthAbstract, targetResourceSID)
}

// SaveXrefFanOutForTarget re-runs SaveXrefCascade and the
// synthetic-version-copy refresh for every OTHER source Resource that
// xrefs r - used whenever either r's Meta or one of r's Versions
// (something that makes r someone else's xref target) is saved. r is
// the real, in-memory *Resource whose Meta/Version was just saved
// (always available at the Entity.Save() call site as meta.Resource or
// v.Resource). This combines what used to be two separate functions
// (fullSaveXrefFanOutForTargetMeta/fullSaveXrefFanOutForTargetVersion)
// - they were always called back-to-back from the same runCascade()
// call site, each running its own copy of the identical "who xrefs me"
// query and its own SaveDefaultVersionCascade(sourceResource) call, so
// merging them halves both the query count and the redundant
// per-source default-version cascade work. The query below joins
// straight through to Resources to grab each source's own Path
// (already exactly "groupPlural/groupUID/resPlural/resUID" - see
// where it's written at Resource creation), rather than just
// returning its SID and needing a separate lookup query to turn that
// SID back into the Group/Resource plural+UID FindGroup()/
// FindResource() need (what Registry.FindResourceBySID() used to do)
// - so each source is resolved via Registry.FindResourceByXID()/
// FindMeta() (cache-checked, so repeat fan-out hits for the same
// source within one Tx are free) with no extra DB round trip beyond
// this one query.
func (r *Resource) SaveXrefFanOutForTarget() {
	if r == nil {
		return
	}

	defer log.Trace("FullTree", r.XID)()

	// FOR UPDATE: this is r (the target) reading the "who xrefs me" set
	// from Metas.xRefPath - a plain SELECT here would still be pinned
	// to this Tx's original RR snapshot, and could miss a source Meta
	// that set its xref to r AND committed after this Tx began (e.g. a
	// brand new source Resource created and xref'd to r by a concurrent
	// Tx that's already committed by the time we run this). Missing it
	// here means that source's mirrored Props never get refreshed to
	// reflect r's just-saved change, even though r is committing last.
	results := Query(r.tx, `
        SELECT res.Path
        FROM Metas AS m
        JOIN Resources AS res ON (res.SID=m.ResourceSID)
        WHERE m.RegistrySID=? AND m.xRefPath=?  FOR UPDATE`,
		r.Registry.DbSID, r.Path)
	defer results.Close()

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		sourceXID := "/" + NotNilString(row[0])
		sourceResource, xErr := r.tx.Registry.FindResourceByXID(
			sourceXID, r.Path, FOR_WRITE)
		if xErr != nil || sourceResource == nil {
			continue
		}
		sourceMeta, xErr := sourceResource.FindMeta(false)
		if xErr != nil || sourceMeta == nil {
			continue
		}
		sourceMeta.SaveXrefCascade()
		sourceResource.SaveXrefVersionCopies(r.DbSID)
		sourceResource.SaveDefaultVersionCascade()

		// The mirrored data we just (re-)copied into sourceResource may
		// now violate sourceResource's own Group's "equals"/"enum"
		// constraints (e.g. this xref target's attribute value changed
		// to something the source's group doesn't allow) - mark that
		// Group for constraint re-validation so GroupsToValidate
		// catches it before this Tx commits/responds. Without this,
		// a target update could silently leave a xref source's mirror
		// in a group-non-compliant state.
		r.tx.AddGroupToValidate(sourceResource.Group)
	}
}

// NOTE: cleaning up stale xref-source mirror rows when a target
// Resource is deleted is handled entirely by ResourcesTrigger (see
// init.sql) now, not here - that trigger fires uniformly for every
// deletion path (direct Resource delete, whole-Group delete, whole-
// Registry delete), so there's no Go-level call site to remember.
