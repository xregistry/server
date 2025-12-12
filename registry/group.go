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
		rType:            rType,
		id:               id,
		vID:              vID,
		obj:              obj,
		addType:          ADD_ADD,
		objIsVer:         objIsVer,
		defaultVersionID: "",
	})
	return r, xErr
}

type ResourceUpsert struct {
	rType            string
	id               string
	vID              string
	obj              Object
	addType          AddType
	objIsVer         bool
	defaultVersionID string
}

// Return: *Resource, isNew, error
func (g *Group) UpsertResource(ru *ResourceUpsert) (*Resource, bool, *XRError) {
	log.VPrintf(3, ">Enter: UpsertResource(%s,%s)", ru.rType, ru.id)
	defer log.VPrintf(3, "<Exit: UpsertResource")

	if xErr := g.Registry.SaveModel(); xErr != nil {
		return nil, false, xErr
	}

	// ru.vID is the version ID we want to use for the update/create.
	// A value of "" means just use the default Version

	if xErr := CheckAttrs(ru.obj, g.XID+"/"+ru.rType+"/"+ru.id); xErr != nil {
		return nil, false, xErr
	}

	gModel := g.GetGroupModel()
	rModel := gModel.FindResourceModel(ru.rType)
	if rModel == nil {
		return nil, false, NewXRError("unknown_resource_type", g.XID,
			"group="+g.Plural,
			"name="+ru.rType)
	}

	r, xErr := g.FindResource(ru.rType, ru.id, true, FOR_WRITE)
	if xErr != nil {
		return nil, false, xErr
	}

	// calc rXID so we can use it even if r == nil
	rXID := g.XID + "/" + ru.rType + "/" + ru.id

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
	if r != nil && r.UID != ru.id {
		return nil, false, NewXRError("bad_request", r.XID,
			"error_detail="+fmt.Sprintf("Attempting to create a Resource "+
				"with a \"%sid\" of %q, when one already exists as %q",
				rModel.Singular, ru.id, r.UID))
	}

	if ru.obj != nil && !IsNil(ru.obj[rModel.Singular+"id"]) && !ru.objIsVer {
		if ru.id != ru.obj[rModel.Singular+"id"] {
			return nil, false, NewXRError("mismatched_id",
				g.XID+"/"+rModel.Plural+"/"+ru.id,
				"singular="+rModel.Singular,
				"invalid_id="+fmt.Sprintf("%v", ru.obj[rModel.Singular+"id"]),
				"expected_id="+ru.id)
		}
	}

	if ru.addType == ADD_ADD && r != nil {
		return nil, false, NewXRError("bad_request", r.XID,
			"error_detail="+
				fmt.Sprintf("Resource %q of type %q already exists",
					ru.id, ru.rType))
	}

	metaObj := (map[string]any)(nil)
	metaObjAny, hasMeta := ru.obj["meta"]

	if hasMeta && !ru.objIsVer {
		delete(ru.obj, "meta")
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

	if !ru.objIsVer {
		// If ru.obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Resource
		var ok bool
		val, _ := ru.obj["versions"]
		if !IsNil(val) {
			versions, ok = val.(map[string]any)
			if !ok {
				return nil, false, NewXRError("invalid_attribute",
					g.Path+"/"+rModel.Plural+"/"+ru.id,
					"name=versions",
					"error_detail=doesn't appear to be of a map of Versions")
			}
		}

	}

	isNew := false

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
				Plural:   ru.rType,
				Singular: rModel.Singular,
				UID:      ru.id,

				Type:     ENTITY_RESOURCE,
				Path:     g.Path + "/" + ru.rType + "/" + ru.id,
				XID:      g.XID + "/" + ru.rType + "/" + ru.id,
				Abstract: g.Plural + string(DB_IN) + ru.rType,

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
			ru.rType)
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

		m, xErr := r.FindMeta(false, FOR_WRITE)
		PanicIf(m != nil, "Should not be nil")

		m = &Meta{
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
		m.Self = m

		DoOne(r.tx, `
                INSERT INTO Metas(SID, RegistrySID, ResourceSID, Path,
                            Abstract, Plural, Singular)
                SELECT ?,?,?,?,?,?,?`,
			m.DbSID, g.Registry.DbSID, r.DbSID,
			m.Path, m.Abstract, r.Plural, r.Singular)

		xErr = m.JustSet(r.Singular+"id", r.UID)
		if xErr != nil {
			return nil, false, xErr
		}

		r.tx.AddMeta(m)
		xErr = m.JustSet("#nextversionid", 1)
		if xErr != nil {
			return nil, false, xErr
		}
	}

	// Process the "meta" sub-object if there - but NOT versioninfo yet
	var meta *Meta

	if !IsNil(metaObj) {
		meta, _, xErr = r.UpsertMeta(metaObj, ru.addType, false, false)
		if xErr != nil {
			return nil, false, xErr
		}
	}

	// Now we have a Resource.
	// Order of processing:
	// - "versions" collection if there
	// - "defaultversionsticky" flag if there
	// - "defaultversionid" flag if sticky is set
	// - Resource level properties applied to default version IFF default
	//   version wasn't already uploaded as part of the "versions" collection

	if r.IsXref() && versions != nil {
		return nil, false,
			NewXRError("extra_xref_attribute", r.XID, "name=versions")
	}

	// If we're processing children, and have a versions collection, process it
	if len(versions) > 0 {
		plural := "versions"
		singular := "version"

		count := 0
		for verID, val := range versions {
			count++
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

			_, _, xErr := r.UpsertVersionWithObject(verID, verObj, ru.addType,
				count != len(versions))
			if xErr != nil {
				return nil, false, xErr
			}
		}

		if xErr := r.EnsureLatest(); xErr != nil {
			return nil, false, xErr
		}
	}

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

	meta, xErr = r.FindMeta(false, FOR_READ)
	PanicIf(xErr != nil, "No meta %q: %s", r.UID, xErr)

	/*
		// Kind of late in the process but oh well
		if meta.Get("readonly") == true {
			return nil, false, NewXRError("readonly", r.XID)
		}
	*/

	if !IsNil(meta.Get("xref")) {
		delete(ru.obj, "meta")
		delete(ru.obj, r.Singular+"id")
		if len(ru.obj) > 0 {
			xErr := NewXRError("extra_xref_attribute", r.XID,
				"name="+SortedKeys(ru.obj)[0])
			if len(ru.obj) > 1 {
				xErr.SetDetailf("Full list: %s.",
					strings.Join(SortedKeys(ru.obj), ","))
			}
			return nil, false, xErr
		}

		if xErr = g.ValidateAndSave(); xErr != nil {
			return nil, false, xErr
		}

		if xErr = r.ProcessVersionInfo(); xErr != nil {
			return nil, false, xErr
		}

		// All versions should have been deleted already so just return
		return r, isNew, nil
	}

	defVerID := meta.GetAsString("defaultversionid")

	if !ru.objIsVer {
		// Clear any ID there since it's the Resource's
		delete(ru.obj, r.Singular+"id")
	}

	attrVersionID := ""
	if val, ok := ru.obj["versionid"]; ok {
		attrVersionID = NotNilString(&val)
	}

	// If both ru.vID and attrVersionID are set, they MUST match if ru.obj is
	// the Resource, not a new Version.
	// Not sure this can ever happen, but just in case...
	if !ru.objIsVer && ru.vID != "" && attrVersionID != "" {
		return nil, false, NewXRError("mismatched_id",
			r.Path+"/versions/"+ru.vID,
			"singular=version",
			"invalid_id="+attrVersionID,
			"expected_id="+ru.vID).SetDetailf(
			"The desired \"versionid\"(%s) must "+
				"match the \"versionid\" attribute(%s).", ru.vID, attrVersionID)
	}

	// If the passed-in ru.vID is empty, and we're new, look for "versionid"
	if ru.vID == "" && isNew && attrVersionID != "" {
		// The call to create the version will complain about passing in a vid
		// if SetVersionID is 'false'. No need to check here too
		ru.vID = attrVersionID
	}

	// if ru.vID is still empty, then use the defaultversionid
	if ru.vID == "" {
		ru.vID = defVerID
	}

	if defVerID != "" && attrVersionID != "" && attrVersionID != defVerID {
		return nil, false, NewXRError("mismatched_id", r.XID,
			"singular=version",
			"invalid_id="+attrVersionID,
			"expected_id="+defVerID).
			SetDetail("Must match the \"defaultversionid\" value.")
	}

	// Update the appropriate Version (ru.vID), but only if the versionID
	// doesn't match a Version ID from the "versions" collection (if there).
	// If both Resource attrs and Version attrs are present, use the Version's
	vObj := maps.Clone(ru.obj)
	if ru.vID != "" {
		// Skip if ru.vID is in "versions" collection
		if _, ok := versions[defVerID]; !ok {
			RemoveResourceAttributes(rModel, vObj)
			_, _, xErr := r.UpsertVersionWithObject(ru.vID, vObj, ru.addType, false)
			if xErr != nil {
				return nil, false, xErr
			}
		}
	} else {
		// Creating a new version w/o an ID, must be a new resource
		RemoveResourceAttributes(rModel, vObj)
		_, _, xErr := r.UpsertVersionWithObject(ru.vID, vObj, ru.addType, false)
		if xErr != nil {
			return nil, false, xErr
		}
	}

	/* If we ever have extension resourceattributes
	RemoveVersionAttributes(rModel, ru.obj)
	r.SetNewObject(ru.obj)
	xErr = r.SetSaveResource(r.Singular+"id", r.UID)
	if xErr != nil {
		return nil, false, xErr
	}
	*/

	if xErr = g.ValidateAndSave(); xErr != nil {
		return nil, false, xErr
	}

	// Re-process the defaultversion info in case things changed
	if xErr = r.ProcessVersionInfo(); xErr != nil {
		return nil, false, xErr
	}

	return r, isNew, xErr
}
