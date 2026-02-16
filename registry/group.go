package registry

import (
	"fmt"
	"maps"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

var _ EntitySetter = &Group{}

func (g *Group) Get(name string) any {
	return g.Entity.Get(name)
}

func (g *Group) JustSet(name string, val any) *XRError {
	return g.Entity.eJustSet(NewPPP(name), val)
}

func (g *Group) SetSave(name string, val any) *XRError {
	return g.Entity.eSetSave(name, val)
}

func (g *Group) Delete() *XRError {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.UID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	// Make sure we don't have any readonly Resources
	results := Query(g.tx, `
	    SELECT EXISTS(SELECT 1 FROM FullTree
		WHERE RegSID=? AND Type=`+StrTypes(ENTITY_META)+` AND
		  Path LIKE '`+g.Path+`/%' AND
		  PropName='readonly`+string(DB_IN)+`' AND
		  PropValue='true')`,
		g.Registry.DbSID)
	defer results.Close()

	row := results.NextRow()
	if NotNilInt(row[0]) != 0 {
		return NewXRError("readonly", g.XID)
	}

	if g.Registry.Touch() {
		if xErr := g.Registry.ValidateAndSave(); xErr != nil {
			return xErr
		}
	}

	DoOne(g.tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)

	// Delete any pending changes so dirty check doesn't fail
	g.NewObject = nil
	g.tx.RemoveFromCache(&g.Entity)

	return nil
}

func (g *Group) FindResource(rType string, id string, anyCase bool, accessMode int) (*Resource, *XRError) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s,%v)", rType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindResource")

	if r := g.tx.GetResource(g, rType, id); r != nil {
		if accessMode == FOR_WRITE && r.AccessMode != FOR_WRITE {
			r.Lock()
		}
		return r, nil
	}

	ent, xErr := RawEntityFromPath(g.tx, g.Registry.DbSID,
		g.Plural+"/"+g.UID+"/"+rType+"/"+id, anyCase, accessMode)
	if xErr != nil {
		return nil, NewXRError("server_error", g.XID+"/"+rType+"/"+id).
			SetDetail(fmt.Sprintf("Error finding Resource %q(%s): %s",
				id, rType, xErr.GetTitle()))
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	r := &Resource{Entity: *ent, Group: g}
	r.Self = r
	r.tx.AddResource(r)
	return r, nil
}

func (g *Group) AddResource(rType string, id string, vID string) (*Resource, *XRError) {
	return g.AddResourceWithObject(rType, id, vID, nil, false)
}

func (g *Group) AddResourceWithObject(rType string, id string, vID string, obj Object, objIsVer bool) (*Resource, *XRError) {
	r, _, xErr := g.UpsertResource(&ResourceUpsert{
		RType:            rType,
		Id:               id,
		VID:              vID,
		Obj:              obj,
		AddType:          ADD_ADD,
		ObjIsVer:         objIsVer,
		DefaultVersionID: "",
	})
	return r, xErr
}

type ResourceUpsert struct {
	RType            string
	Id               string
	VID              string // "versionid" from json
	Obj              Object // json body
	AddType          AddType
	ObjIsVer         bool
	DefaultVersionID string // from ?setdefaultversionid
}

// Return: *Resource, isNew, error
func (g *Group) UpsertResource(ru *ResourceUpsert) (*Resource, bool, *XRError) {
	log.VPrintf(3, ">Enter: UpsertResource(%s,%s)", ru.RType, ru.Id)
	defer log.VPrintf(3, "<Exit: UpsertResource")

	if xErr := g.Registry.SaveModel(); xErr != nil {
		return nil, false, xErr
	}

	// ru.VID is the version ID we want to use for the update/create.
	// A value of "" means just use the default Version

	if xErr := CheckAttrs(ru.Obj, g.XID+"/"+ru.RType+"/"+ru.Id); xErr != nil {
		return nil, false, xErr
	}

	gModel := g.GetGroupModel()
	rModel := gModel.FindResourceModel(ru.RType)
	if rModel == nil {
		return nil, false, NewXRError("unknown_resource_type", g.XID,
			"group="+g.Plural,
			"name="+ru.RType)
	}

	if ru.DefaultVersionID != "" && rModel.GetSetDefaultSticky() == false {
		return nil, false,
			NewXRError("setdefaultversionid_not_allowed",
				g.XID+"/"+rModel.Plural+"/"+ru.Id,
				"singular="+rModel.Singular)
	}

	r, xErr := g.FindResource(ru.RType, ru.Id, true, FOR_WRITE)
	if xErr != nil {
		return nil, false, xErr
	}

	// calc rXID so we can use it even if r == nil
	rXID := g.XID + "/" + ru.RType + "/" + ru.Id

	if r != nil {
		meta, xErr := r.FindMeta(false, FOR_READ)
		PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)
		if meta.Get("readonly") == true {
			if r.tx.RequestInfo.HasIgnore("readonly") {
				// Ignoring that it's read-only but also stopping
				return r, false, nil
			} else {
				return nil, false, NewXRError("readonly", r.XID)
			}
		}
	}

	// Can this ever happen??
	if r != nil && r.UID != ru.Id {
		return nil, false, NewXRError("bad_request", r.XID,
			"error_detail="+fmt.Sprintf("Attempting to create a Resource "+
				"with a \"%sid\" of %q, when one already exists as %q",
				rModel.Singular, ru.Id, r.UID))
	}

	if ru.Obj != nil && !IsNil(ru.Obj[rModel.Singular+"id"]) && !ru.ObjIsVer {
		if ru.Id != ru.Obj[rModel.Singular+"id"] {
			return nil, false, NewXRError("mismatched_id",
				g.XID+"/"+rModel.Plural+"/"+ru.Id,
				"singular="+rModel.Singular,
				"invalid_id="+fmt.Sprintf("%v", ru.Obj[rModel.Singular+"id"]),
				"expected_id="+ru.Id)
		}
	}

	if ru.AddType == ADD_ADD && r != nil {
		return nil, false, NewXRError("bad_request", r.XID,
			"error_detail="+
				fmt.Sprintf("Resource %q of type %q already exists",
					ru.Id, ru.RType))
	}

	// "versionid" from the incoming Obj, if present
	objVersionID := ""
	if val, ok := ru.Obj["versionid"]; ok {
		objVersionID = NotNilString(&val)
	}

	// Will hold the Resource's meta obj
	meta := (*Meta)(nil)

	// Will hold the versionID that we're assuming the Resource.* attributes
	// are for. We'll try to figure it out from things like Resource.versionid
	// or meta.defaultversionid - see the spec for how to determine this
	resourceDefaultVersionID := ""
	if r != nil {
	}

	// Is the Resource new
	isNew := false

	metaObj := (map[string]any)(nil)
	metaObjAny, hasMeta := ru.Obj["meta"]

	if hasMeta && !ru.ObjIsVer {
		delete(ru.Obj, "meta")
	}

	if hasMeta {
		if IsNil(metaObjAny) {
			// Convert "null" to empty {}
			metaObjAny = map[string]any{}
		}

		var ok bool
		metaObj, ok = metaObjAny.(map[string]any)
		if !ok {
			return nil, false, NewXRError("invalid_attribute", rXID,
				"name=meta",
				"error_detail=\"meta\" must be an object")
		}
	}

	// List of versions in the incoming request
	versions := map[string]any(nil)

	if !ru.ObjIsVer {
		// If ru.Obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Resource
		var ok bool
		val, _ := ru.Obj["versions"]
		if !IsNil(val) {
			versions, ok = val.(map[string]any)
			if !ok {
				return nil, false, NewXRError("invalid_attribute",
					g.XID+"/"+rModel.Plural+"/"+ru.Id,
					"name=versions",
					"error_detail=doesn't appear to be of a map of Versions")
			}
		}

	}

	if ru.DefaultVersionID == "request" {
		if len(versions) == 0 && r != nil { // DUG not sure what this is
			/*
				meta, xErr := r.FindMeta(false, FOR_READ)
				PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)
				ru.DefaultVersionID = meta.GetAsString("defaultversionid")
			*/
			/*
				return nil, false, NewXRError("defaultversionid_request",
					g.XID+"/"+rModel.Plural+"/"+ru.Id)
			*/
		}
		if len(versions) > 1 {
			return nil, false, NewXRError("too_many_versions",
				g.XID+"/"+rModel.Plural+"/"+ru.Id)
		}
	}

	// Set incoming metaObj defaultversionid base on ru.DefaultVersionID
	metaAddType := ru.AddType
	if IsNil(metaObj) {
		if ru.DefaultVersionID != "" {
			metaAddType = ADD_PATCH
			metaObj = map[string]any{}
		}
	}

	if r == nil {
		// If Resource doesn't exist, go ahead and create it.
		// This will not create any Versions yet, just the Resource
		r = &Resource{
			Entity: Entity{
				EntityExtensions: EntityExtensions{
					tx:         g.tx,
					AccessMode: FOR_WRITE,
				},

				Registry: g.Registry,
				DbSID:    NewUUID(),
				Plural:   ru.RType,
				Singular: rModel.Singular,
				UID:      ru.Id,

				Type:     ENTITY_RESOURCE,
				Path:     g.Path + "/" + ru.RType + "/" + ru.Id,
				XID:      g.XID + "/" + ru.RType + "/" + ru.Id,
				Abstract: g.Plural + string(DB_IN) + ru.RType,

				GroupModel:    gModel,
				ResourceModel: rModel,
			},
			Group: g,
		}
		r.Self = r

		DoOne(r.tx, `
        INSERT INTO Resources(
            SID, UID, RegistrySID,
            GroupSID, ModelSID,
            Path, Abstract,
            Plural, Singular)
        SELECT ?,?,?,?,SID,?,?,?,?
        FROM ModelEntities
        WHERE RegistrySID=?
          AND ParentSID IN (
            SELECT SID FROM ModelEntities
            WHERE RegistrySID=?
            AND ParentSID IS NULL
            AND Plural=?)
            AND Plural=?`,

			r.DbSID, r.UID, g.Registry.DbSID,
			g.DbSID, /* , ModelSID */
			r.Path, r.Abstract,
			r.Plural, r.Singular,

			g.Registry.DbSID, g.Registry.DbSID, g.Plural,
			ru.RType)
		// When we delete entities due to their model def being deleted
		// then I think we can use rModel.SID in the above sql stmt
		// instead of the sub-query

		isNew = true
		r.tx.AddResource(r)
		g.Touch()

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		xErr = r.SetSaveResource(r.Singular+"id", r.UID)
		if xErr != nil {
			return nil, false, xErr
		}

		/*
			meta, xErr = r.FindMeta(false, FOR_WRITE)
			PanicIf(meta != nil, "Should not be nil")
		*/

		meta = &Meta{
			Entity: Entity{
				EntityExtensions: EntityExtensions{
					tx:         g.tx,
					AccessMode: FOR_WRITE,
				},

				Registry: g.Registry,
				DbSID:    NewUUID(),
				Plural:   "metas",
				Singular: "meta",
				UID:      r.UID,

				Type:     ENTITY_META,
				Path:     r.Path + "/meta",
				XID:      r.XID + "/meta",
				Abstract: r.Abstract + string(DB_IN) + "meta",

				GroupModel:    gModel,
				ResourceModel: rModel,
			},
			Resource: r,
		}
		meta.Self = meta

		DoOne(r.tx, `
                INSERT INTO Metas(SID, RegistrySID, ResourceSID, Path,
                            Abstract, Plural, Singular)
                SELECT ?,?,?,?,?,?,?`,
			meta.DbSID, g.Registry.DbSID, r.DbSID,
			meta.Path, meta.Abstract, r.Plural, r.Singular)

		xErr = meta.JustSet(r.Singular+"id", r.UID)
		if xErr != nil {
			return nil, false, xErr
		}

		r.tx.AddMeta(meta)
		xErr = meta.JustSet("#nextversionid", 1)
		if xErr != nil {
			return nil, false, xErr
		}

		// Grab Resource's defaultVersionID from either the incoming
		// "versionid" attribute (if present) or meta.defaultvertsionid
		resourceDefaultVersionID = objVersionID

		if resourceDefaultVersionID == "" {
			if !IsNil(metaObj) {
				if val, ok := metaObj["defaultversionid"]; ok {
					resourceDefaultVersionID = NotNilString(&val)
				}
			}
		}

		// If still not found/set then leave it blank since "" will use
		// #nextversionid automatically when we create the new version
	} else {
		meta, xErr = r.FindMeta(false, FOR_WRITE)
		PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

		resourceDefaultVersionID = meta.GetAsString("defaultversionid")
	}

	if ru.DefaultVersionID != "" {
		meta.JustSet("defaultversionid", ru.DefaultVersionID)
		meta.JustSet("defaultversionsticky", true)
	}

	// Now we have a Resource.
	// Order of processing:
	// - "versions" collection if there
	// - Resource level properties applied to default version IFF default
	//   version wasn't already uploaded as part of the "versions" collection
	// - Process 'meta' if there

	// If xRef is set we can flag extra attrs quickly
	xrefValue := meta.Get("xref")
	hasXref := !IsNil(xrefValue) && xrefValue != "" // !IsNil(xrefValue)
	objXref, okObj := metaObj["xref"]

	// Need to set meta.xref early because some processing depends on it
	// being set to the final value before we actually process "meta"
	if hasXref {
		if objXref == false || (okObj && IsNil(objXref)) {
			hasXref = false
			meta.JustSet("xref", nil)
			// This is also done in the upsertMeta call but we need it
			// now so that the upsertVersion below uses "1" and not "2+"
			// TODO see if we can clean-up this dup
			// DUG
			meta.JustSet("#nextversionid", 1)

			// Treat it like a new resource // DUG clean up dup with above
			// Grab Resource's defaultVersionID from either the incoming
			// "versionid" attribute (if present) or meta.defaultvertsionid
			resourceDefaultVersionID = objVersionID

			if resourceDefaultVersionID == "" {
				if !IsNil(metaObj) {
					if val, ok := metaObj["defaultversionid"]; ok {
						resourceDefaultVersionID = NotNilString(&val)
					}
				}
			}
		}
	} else {
		if okObj && objXref != "" {
			hasXref = true
			meta.JustSet("xref", objXref)
		}
	}
	// log.Printf("hasxref: %v", hasXref)
	// log.Printf("   xref: %v", objXref)

	if hasXref { // DUG!! may need these checks
		/* Should be covered by the stuff below
		if versions != nil {
			return nil, false,
				NewXRError("extra_xref_attribute", r.XID, "name=versions")
		}
		*/

		// delete(ru.Obj, "meta")          // DUG commented out
		delete(ru.Obj, r.Singular+"id") // DUG commented out
		if len(ru.Obj) > 0 {
			xErr := NewXRError("extra_xref_attribute", r.XID,
				"name="+SortedKeys(ru.Obj)[0])
			if len(ru.Obj) > 1 {
				xErr.SetDetailf("Full list: %s.",
					strings.Join(SortedKeys(ru.Obj), ","))
			}
			return nil, false, xErr
		}

		/*
			if xErr = g.ValidateAndSave(); xErr != nil {
				return nil, false, xErr
			}

			if xErr = r.ProcessVersionInfo(); xErr != nil {
				return nil, false, xErr
			}

			// All versions should have been deleted already so just return
			return r, isNew, nil
		*/
	}

	defaultVersion := (*Version)(nil)

	// Process any Versions in the request
	if len(versions) > 0 {
		plural := "versions"
		singular := "version"

		count := 0
		for verID, val := range versions {
			count++

			// Make sure that each entry in "versions" is an Object
			verObj, ok := val.(map[string]any)
			if !ok {
				return nil, false,
					NewXRError("invalid_attribute", r.XID,
						"name="+plural,
						"error_detail="+
							fmt.Sprintf("key %q in attribute %q doesn't "+
								"appear to be of type %q", verID, plural,
								singular))
			}

			// Update or create each Version
			v, _, xErr := r.UpsertVersionWithObject(&VersionUpsert{
				Id:               verID,
				Obj:              verObj,
				AddType:          ru.AddType,
				More:             true, // count != len(versions),
				DefaultVersionID: "",
			})
			if xErr != nil {
				return nil, false, xErr
			}

			// must only be one so save it for later
			if ru.DefaultVersionID == "request" {
				defaultVersion = v
			}
		}

		/*
			if xErr := r.EnsureLatest(); xErr != nil {
				return nil, false, xErr
			}
		*/
	}

	/*
		// Process the "meta" sub-object if there
		if !IsNil(metaObj) {
			xErr := r.ProcessVersionInfo()
			if xErr != nil {
				if isNew {
					// Needed if doing local func calls to create the Resource
					// and we don't commit/rollback the tx upon failure
					r.Delete()
				}
				return nil, false, xErr
			}
		}
	*/

	/* Should be old
	meta, xErr = r.FindMeta(false, FOR_READ)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)
	*/

	/*
		// Kind of late in the process but oh well
		if meta.Get("readonly") == true {
			return nil, false, NewXRError("readonly", r.XID)
		}
	*/

	// old: defVerID := meta.GetAsString("defaultversionid")

	if !ru.ObjIsVer {
		// Clear any ID there since it's the Resource's
		delete(ru.Obj, r.Singular+"id")
	}

	// If both ru.VID and objVersionID are set, they MUST match if ru.Obj is
	// the Resource, not a new Version.
	// Not sure this can ever happen, but just in case...
	/* OLD check
	if !ru.ObjIsVer && ru.VID != "" && objVersionID != "" && ru.VID != objVersionID {
		return nil, false, NewXRError("mismatched_id",
			r.XID+"/versions/"+ru.VID,
			"singular=version",
			"invalid_id="+objVersionID,
			"expected_id="+ru.VID).SetDetailf(
			"The desired \"versionid\"(%s) must "+
				"match the \"versionid\" attribute(%s).", ru.VID, objVersionID)
	}
	*/

	// If the passed-in ru.VID is empty, and we're new, look for "versionid"
	/* OLD?
	if ru.VID == "" && isNew && resourceDefaultVersionID != "" { // objVersionID != "" {
		// The call to create the version will complain about passing in a vid
		// if SetVersionID is 'false'. No need to check here too
		ru.VID = resourceDefaultVersionID // objVersionID
	}
	*/

	if !IsNil(metaObj) {
		// Skip "request" for now
		if ru.DefaultVersionID == "null" {
			metaObj["defaultversionid"] = ""
			metaObj["defaultversionsticky"] = false
		} else if ru.DefaultVersionID == "request" {
			// Do nothing right now
		} else if ru.DefaultVersionID != "" {
			metaObj["defaultversionid"] = ru.DefaultVersionID
			metaObj["defaultversionsticky"] = true
		}
	}

	// log.Printf("ru: %s", ToJSON(ru))
	// log.Printf("metaobj: %s", ToJSON(metaObj))
	// log.Printf("resourceDefaultVersionID: %s", resourceDefaultVersionID)

	if ru.VID == "" {
		// ru.VID = defVerID
		ru.VID = resourceDefaultVersionID // objVersionID

		// If still "" then it must be new, look for meta.defaultversionid
		/*
			if ru.VID == "" {
				ru.VID = ru.DefaultVersionID
			}
		*/

		// If still "", look for meta.defaultversionid
		if ru.VID == "" && !IsNil(metaObj) {
			tmp := metaObj["defaultversionid"]
			ru.VID = NotNilString(&tmp)
		}
	}

	// if defVerID != "" && objVersionID != "" && objVersionID != defVerID {
	if resourceDefaultVersionID != "" && objVersionID != "" && objVersionID != resourceDefaultVersionID {
		return nil, false, NewXRError("mismatched_id", r.XID,
			"singular=version",
			"invalid_id="+objVersionID,
			"expected_id="+resourceDefaultVersionID).
			SetDetail("Must match the \"defaultversionid\" value.")
	}

	// Update the appropriate Version (ru.VID), but only if the versionID
	// doesn't match a Version ID from the "versions" collection (if there).
	// If both Resource attrs and Version attrs are present, use the Version's
	vObj := maps.Clone(ru.Obj)

	// log.Printf("ru: %s", ToJSON(ru))
	// log.Printf("resdefverid: %s", resourceDefaultVersionID)
	// log.Printf("objVersionID: %s", objVersionID)
	// ShowStack()
	if !hasXref && ru.VID != "" { // DUG clean-up this use of hasXref - it's hacky
		// Skip if ru.VID is in "versions" collection
		if _, ok := versions[ru.VID]; !ok {
			RemoveResourceAttributes(rModel, vObj)
			defaultVersion, _, xErr = r.UpsertVersionWithObject(&VersionUpsert{
				Id:               ru.VID,
				Obj:              vObj,
				AddType:          ru.AddType,
				More:             true, // false,
				DefaultVersionID: "",
			})
			if xErr != nil {
				return nil, false, xErr
			}
		}
	} else if !hasXref { // DUG clean-up this use of hasXref - it's hacky
		// Creating a new version w/o an ID, must be a new resource
		RemoveResourceAttributes(rModel, vObj)
		defaultVersion, _, xErr = r.UpsertVersionWithObject(&VersionUpsert{
			Id:               ru.VID,
			Obj:              vObj,
			AddType:          ru.AddType,
			More:             true, // false,
			DefaultVersionID: "",
		})
		if xErr != nil {
			return nil, false, xErr
		}
		// log.Printf("Created a new version.obj: %s", ToJSON(defaultVersion.Object))
		// log.Printf("Created a new version.new: %s", ToJSON(defaultVersion.NewObject))
	}
	// vs, _ := r.GetVersionIDs()
	// log.Printf("r.vers: %s", ToJSON(vs))

	if !IsNil(metaObj) {
		if ru.DefaultVersionID == "null" {
			metaObj["defaultversionid"] = ""
			metaObj["defaultversionsticky"] = false
		} else if ru.DefaultVersionID == "request" {
			// MUST only be one, so grab its ID
			metaObj["defaultversionid"] = defaultVersion.GetAsString("versionid")
			metaObj["defaultversionsticky"] = true
		} else if ru.DefaultVersionID != "" {
			metaObj["defaultversionid"] = ru.DefaultVersionID
			metaObj["defaultversionsticky"] = true
		}

		// Uncommented
		meta, _, xErr = r.UpsertMeta(&MetaUpsert{
			obj:                metaObj,
			addType:            metaAddType, // ru.AddType,
			createVersion:      false,
			processVersionInfo: false,
			more:               true,
		})

		if xErr != nil {
			return nil, false, xErr
		}
	}

	/* If we ever have extension resourceattributes
	RemoveVersionAttributes(rModel, ru.Obj)
	r.SetNewObject(ru.Obj)
	xErr = r.SetSaveResource(r.Singular+"id", r.UID)
	if xErr != nil {
		return nil, false, xErr
	}
	*/

	// Process the "meta" sub-object if there
	// OLD: if !IsNil(metaObj) {
	/*
		xErr = r.ProcessVersionInfo()
		if xErr != nil {
			if isNew {
				// Needed if doing local func calls to create the Resource
				// and we don't commit/rollback the tx upon failure
				r.Delete()
			}
			return nil, false, xErr
		}
	*/
	// OLD: }

	if xErr = g.ValidateAndSave(); xErr != nil {
		return nil, false, xErr
	}

	// Re-process the defaultversion info in case things changed
	/*
		if xErr = r.ProcessVersionInfo(); xErr != nil {
			return nil, false, xErr
		}
	*/

	if xErr = r.ValidateResource(); xErr != nil {
		return nil, false, xErr
	}

	return r, isNew, xErr
}
