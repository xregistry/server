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

func (g *Group) JustSet(name string, val any) error {
	return g.Entity.eJustSet(NewPPP(name), val)
}

func (g *Group) SetSave(name string, val any) error {
	return g.Entity.eSetSave(name, val)
}

func (g *Group) Delete() error {
	log.VPrintf(3, ">Enter: Group.Delete(%s)", g.UID)
	defer log.VPrintf(3, "<Exit: Group.Delete")

	// Make sure we don't have any readonly Resources
	results, err := Query(g.tx, `
	    SELECT EXISTS(SELECT 1 FROM FullTree
		WHERE RegSID=? AND Type=`+StrTypes(ENTITY_META)+` AND
		  Path LIKE '`+g.Path+`/%' AND
		  PropName='readonly`+string(DB_IN)+`' AND
		  PropValue='true')`,
		g.Registry.DbSID)
	defer results.Close()
	if err != nil {
		return err
	}
	row := results.NextRow()
	if NotNilInt(row[0]) != 0 {
		return fmt.Errorf("Delete operations on read-only " +
			"resources are not allowed")
	}

	if g.Registry.Touch() {
		if err = g.Registry.ValidateAndSave(); err != nil {
			return err
		}
	}

	err = DoOne(g.tx, `DELETE FROM "Groups" WHERE SID=?`, g.DbSID)
	if err != nil {
		return err
	}
	g.tx.RemoveFromCache(&g.Entity)
	return nil
}

func (g *Group) FindResource(rType string, id string, anyCase bool, accessMode int) (*Resource, error) {
	log.VPrintf(3, ">Enter: FindResource(%s,%s,%v)", rType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindResource")

	if r := g.tx.GetResource(g, rType, id); r != nil {
		if accessMode == FOR_WRITE && r.AccessMode != FOR_WRITE {
			r.Lock()
		}
		return r, nil
	}

	ent, err := RawEntityFromPath(g.tx, g.Registry.DbSID,
		g.Plural+"/"+g.UID+"/"+rType+"/"+id, anyCase, accessMode)
	if err != nil {
		return nil, fmt.Errorf("Error finding Resource %q(%s): %s",
			id, rType, err)
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

func (g *Group) AddResource(rType string, id string, vID string) (*Resource, error) {
	return g.AddResourceWithObject(rType, id, vID, nil, false)
}

func (g *Group) AddResourceWithObject(rType string, id string, vID string, obj Object, objIsVer bool) (*Resource, error) {

	r, _, err := g.UpsertResourceWithObject(rType, id, vID, obj,
		ADD_ADD, objIsVer)
	return r, err
}

func (g *Group) UpsertResource(rType string, id string, vID string) (*Resource, bool, error) {
	return g.UpsertResourceWithObject(rType, id, vID, nil, ADD_ADD, false)
}

// Return: *Resource, isNew, error
func (g *Group) UpsertResourceWithObject(rType string, id string, vID string, obj Object, addType AddType, objIsVer bool) (*Resource, bool, error) {
	log.VPrintf(3, ">Enter: UpsertResourceWithObject(%s,%s)", rType, id)
	defer log.VPrintf(3, "<Exit: UpsertResourceWithObject")

	if err := g.Registry.SaveModel(); err != nil {
		return nil, false, err
	}

	// vID is the version ID we want to use for the update/create.
	// A value of "" means just use the default Version

	if err := CheckAttrs(obj); err != nil {
		return nil, false, err
	}

	gModel := g.GetGroupModel()
	rModel := gModel.FindResourceModel(rType)
	if rModel == nil {
		return nil, false, fmt.Errorf("Unknown Resource type (%s) for Group %q",
			rType, g.Plural)
	}

	r, err := g.FindResource(rType, id, true, FOR_WRITE)
	if err != nil {
		return nil, false, fmt.Errorf("Error checking for Resource(%s) %q: %s",
			rType, id, err)
	}

	// Can this ever happen??
	if r != nil && r.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Resource with "+
			"a \"%sid\" of %q, when one already exists as %q",
			rModel.Singular, id, r.UID)
	}

	if obj != nil && !IsNil(obj[rModel.Singular+"id"]) && !objIsVer {
		if id != obj[rModel.Singular+"id"] {
			return nil, false,
				fmt.Errorf(`The "%sid" attribute must be set to %q, not %q`,
					rModel.Singular, id, obj[rModel.Singular+"id"])
		}
	}

	if addType == ADD_ADD && r != nil {
		return nil, false, fmt.Errorf("Resource %q of type %q already exists",
			id, rType)
	}

	metaObj := (map[string]any)(nil)
	metaObjAny, hasMeta := obj["meta"]

	if hasMeta && !objIsVer {
		delete(obj, "meta")
	}

	if hasMeta {
		if objIsVer {
			return nil, false, fmt.Errorf("Can't include a Version with a " +
				"\"meta\" attribute")
		}

		if IsNil(metaObjAny) {
			// Convert "null" to empty {}
			metaObjAny = map[string]any{}
		}

		metaObj = metaObjAny.(map[string]any)
	}

	// List of versions in the incoming request
	versions := map[string]any(nil)

	if !objIsVer {
		// If obj is for the resource then save and delete the versions
		// collection (and it's attributes) so we don't try to save them
		// as extensions on the Resource
		var ok bool
		val, _ := obj["versions"]
		if !IsNil(val) {
			versions, ok = val.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Attribute %q doesn't appear to be of a "+
						"map of %q", "versions", "versions")
			}
		}

		// Remove the "versions" collection attributes
		delete(obj, "versions")
		delete(obj, "versionscount")
		delete(obj, "versionsurl")
	} else {
		if _, ok := obj["versions"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versions\" attribute")
		}
		if _, ok := obj["versionscount"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versionscount\" attribute")
		}
		if _, ok := obj["versionsurl"]; ok {
			return nil, false, fmt.Errorf("Can't create a Version with a " +
				"\"versionsurl\" attribute")
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
				Plural:   rType,
				Singular: rModel.Singular,
				UID:      id,

				Type:     ENTITY_RESOURCE,
				Path:     g.Plural + "/" + g.UID + "/" + rType + "/" + id,
				Abstract: g.Plural + string(DB_IN) + rType,

				GroupModel:    gModel,
				ResourceModel: rModel,
			},
			Group: g,
		}
		r.Self = r

		err = DoOne(r.tx, `
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
			rType)
		// When we delete entities due to their model def being deleted
		// then I think we can use rModel.SID in the above sql stmt
		// instead of the sub-query
		if err != nil {
			return nil, false, fmt.Errorf("Error adding Resource: %s", err)
		}
		isNew = true
		r.tx.AddResource(r)
		g.Touch()

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		err = r.SetSaveResource(r.Singular+"id", r.UID)
		if err != nil {
			return nil, false, err
		}

		m, err := r.FindMeta(false, FOR_WRITE)
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
				Abstract: r.Abstract + string(DB_IN) + "meta",

				GroupModel:    gModel,
				ResourceModel: rModel,
			},
			Resource: r,
		}
		m.Self = m

		err = DoOne(r.tx, `
                INSERT INTO Metas(SID, RegistrySID, ResourceSID, Path,
                            Abstract, Plural, Singular)
                SELECT ?,?,?,?,?,?,?`,
			m.DbSID, g.Registry.DbSID, r.DbSID,
			m.Path, m.Abstract, r.Plural, r.Singular)
		if err != nil {
			return nil, false, fmt.Errorf("Error adding Meta: %s", err)
		}

		err = m.JustSet(r.Singular+"id", r.UID)
		if err != nil {
			return nil, false, err
		}

		r.tx.AddMeta(m)
		err = m.JustSet("#nextversionid", 1)
		if err != nil {
			return nil, false, err
		}
	}

	// Process the "meta" sub-object if there - but NOT versioninfo yet
	var meta *Meta

	if !IsNil(metaObj) {
		meta, _, err = r.UpsertMetaWithObject(metaObj, addType, false, false)
		if err != nil {
			return nil, false, err
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
			fmt.Errorf(`Can't update "versions" if "xref" is set`)
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
					fmt.Errorf("Key %q in attribute %q doesn't "+
						"appear to be of type %q", verID, plural, singular)
			}

			_, _, err := r.UpsertVersionWithObject(verID, verObj, addType,
				count != len(versions))
			if err != nil {
				return nil, false, err
			}
		}

		if err := r.EnsureLatest(); err != nil {
			return nil, false, err
		}
	}

	// Process the "meta" sub-object if there
	if !IsNil(metaObj) {
		err := r.ProcessVersionInfo()
		if err != nil {
			if isNew {
				// Needed if doing local func calls to create the Resource
				// and we don't commit/rollback the tx upon failure
				r.Delete()
			}
			return nil, false, err
		}
	}

	meta, err = r.FindMeta(false, FOR_READ)
	PanicIf(err != nil, "No meta %q: %s", r.UID, err)

	// Kind of late in the process but oh well
	if meta.Get("readonly") == true {
		return nil, false, fmt.Errorf("Write operations on read-only " +
			"resources are not allowed")
	}

	if !IsNil(meta.Get("xref")) {
		delete(obj, "meta")
		delete(obj, r.Singular+"id")
		if len(obj) > 0 {
			return nil, false,
				fmt.Errorf("Extra attributes (%s) not allowed when "+
					"\"xref\" is set", strings.Join(SortedKeys(obj), ","))
		}

		if err = g.ValidateAndSave(); err != nil {
			return nil, false, err
		}

		if err = r.ProcessVersionInfo(); err != nil {
			return nil, false, err
		}

		// All versions should have been deleted already so just return
		return r, isNew, nil
	}

	defVerID := meta.GetAsString("defaultversionid")

	if !objIsVer {
		// Clear any ID there since it's the Resource's
		delete(obj, r.Singular+"id")
	}

	attrVersionID := ""
	if val, ok := obj["versionid"]; ok {
		attrVersionID = NotNilString(&val)
	}

	// If both vID and attrVersionID are set, they MUST match if obj is
	// the Resource, not a new Version.
	// Not sure this can ever happen, but just in case...
	if !objIsVer && vID != "" && attrVersionID != "" {
		return nil, false, fmt.Errorf("The desired \"versionid\"(%s) must "+
			"match the \"versionid\" attribute(%s)", vID, attrVersionID)
	}

	// If the passed-in vID is empty, and we're new, look for "versionid"
	if vID == "" && isNew && attrVersionID != "" {
		vID = attrVersionID
	}

	// if vID is still empty, then use the defaultversionid
	if vID == "" {
		vID = defVerID
	}

	if defVerID != "" && attrVersionID != "" && attrVersionID != defVerID {
		return nil, false, fmt.Errorf("When \"versionid\"(%s) is "+
			"present it must match the \"defaultversionid\"(%s)",
			attrVersionID, defVerID)
	}

	// Update the appropriate Version (vID), but only if the versionID
	// doesn't match a Version ID from the "versions" collection (if there).
	// If both Resource attrs and Version attrs are present, use the Version's
	vObj := maps.Clone(obj)

	if vID != "" {
		if _, ok := versions[defVerID]; !ok {
			RemoveResourceAttributes(rModel, vObj)
			_, _, err := r.UpsertVersionWithObject(vID, vObj, addType, false)
			if err != nil {
				return nil, false, err
			}
		}
	} else {
		RemoveResourceAttributes(rModel, vObj)
		_, _, err := r.UpsertVersionWithObject(vID, vObj, addType, false)
		if err != nil {
			return nil, false, err
		}
	}

	/* If we ever have extension resourceattributes
	RemoveVersionAttributes(rModel, obj)
	r.SetNewObject(obj)
	err = r.SetSaveResource(r.Singular+"id", r.UID)
	if err != nil {
		return nil, false, err
	}
	*/

	if err = g.ValidateAndSave(); err != nil {
		return nil, false, err
	}

	// Re-process the defaultversion info in case things changed
	if err = r.ProcessVersionInfo(); err != nil {
		return nil, false, err
	}

	return r, isNew, err
}
