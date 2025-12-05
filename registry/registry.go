package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

var DefaultRegDbSID string

func (r *Registry) GetTx() *Tx {
	return r.tx
}

func GetDefaultReg(tx *Tx) *Registry {
	if DefaultRegDbSID == "" {
		panic("No registry specified")
	}

	if tx == nil {
		var xErr *XRError
		tx, xErr = NewTx()
		Must(xErr)
	}

	reg, err := FindRegistryBySID(tx, DefaultRegDbSID, FOR_READ)
	Must(err)

	if reg != nil {
		tx.Registry = reg
	}
	PanicIf(reg == nil, "No default registry")

	return reg
}

func (r *Registry) Rollback() *XRError {
	if r != nil {
		return r.tx.Rollback()
	}
	return nil
}

func (r *Registry) SaveAllAndCommit() *XRError {
	if r != nil {
		if r.Model.GetChanged() {
			if xErr := r.SaveModel(); xErr != nil {
				return xErr
			}
		}
		return r.tx.SaveAllAndCommit()
	}
	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) SaveCommitRefresh() *XRError {
	if r != nil {
		return r.tx.SaveCommitRefresh()
	}
	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) AddToCache(e *Entity) {
	r.tx.AddToCache(e)
}

func (r *Registry) Commit() *XRError {
	if r != nil {
		return r.tx.Commit()
	}
	return nil
}

type RegOpt string

func NewRegistry(tx *Tx, id string, regOpts ...RegOpt) (*Registry, *XRError) {
	log.VPrintf(3, ">Enter: NewRegistry %q", id)
	defer log.VPrintf(3, "<Exit: NewRegistry")

	var xErr *XRError // must be used for all error checking due to defer
	newTx := false

	defer func() {
		if newTx {
			// If created just for us, close it
			tx.Conditional(xErr)
		}
	}()

	if tx == nil {
		tx, xErr = NewTx()
		if xErr != nil {
			return nil, xErr
		}
		newTx = true
	}

	if id == "" {
		id = NewUUID()
	}

	r, xErr := FindRegistry(tx, id, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}
	if r != nil {
		return nil, NewXRError("bad_request", "/",
			"error_detail="+
				fmt.Sprintf("A registry with ID %q already exists", id))
	}

	dbSID := NewUUID()
	DoOne(tx, `
		INSERT INTO Registries(SID, UID)
		VALUES(?,?)`, dbSID, id)

	reg := &Registry{
		Entity: Entity{
			EntityExtensions: EntityExtensions{
				tx:         tx,
				AccessMode: FOR_WRITE,
			},

			DbSID:    dbSID,
			Plural:   "registries",
			Singular: "registry",
			UID:      id,

			Type:     ENTITY_REGISTRY,
			Path:     "",
			XID:      "/",
			Abstract: "",
		},
	}

	reg.Self = reg
	reg.Entity.Registry = reg
	reg.Capabilities = DefaultCapabilities
	reg.Model = &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	tx.Registry = reg
	tx.AddRegistry(reg)

	xErr = reg.Model.Verify()
	if xErr != nil {
		return nil, xErr
	}

	DoOne(tx, `
		INSERT INTO Models(RegistrySID)
		VALUES(?)`, dbSID)

	if xErr = reg.JustSet("specversion", SPECVERSION); xErr != nil {
		return nil, xErr
	}
	if xErr = reg.JustSet("registryid", reg.UID); xErr != nil {
		return nil, xErr
	}

	/*
		for _, regOpt := range regOpts {
			// if regOpts == RegOpt_STRING { ... }
		}
	*/

	if xErr = reg.SetSave("epoch", 1); xErr != nil {
		return nil, xErr
	}

	if xErr = reg.Model.VerifyAndSave(); xErr != nil {
		return nil, xErr
	}

	return reg, nil
}

func GetRegistryNames() ([]string, *XRError) {
	tx, xErr := NewTx()
	if xErr != nil {
		return nil, xErr
	}
	defer tx.Rollback()

	results := Query(tx, `SELECT UID FROM Registries`)
	defer results.Close()

	res := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		res = append(res, NotNilString(row[0]))
	}

	return res, nil
}

var _ EntitySetter = &Registry{}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

func (reg *Registry) JustSet(name string, val any) *XRError {
	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update' will just lock it automatically
	reg.Lock()
	return reg.Entity.eJustSet(NewPPP(name), val)
}

func (reg *Registry) SetSave(name string, val any) *XRError {
	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update' will just lock it automatically
	reg.Lock()
	return reg.Entity.eSetSave(name, val)
}

func (reg *Registry) Delete() *XRError {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.UID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	reg.Lock()
	DoOne(reg.tx, `DELETE FROM Registries WHERE SID=?`, reg.DbSID)

	reg.tx.EraseCache()
	reg.tx.Registry = nil
	return nil
}

func FindRegistryBySID(tx *Tx, sid string, accessMode int) (*Registry, *XRError) {
	log.VPrintf(3, ">Enter: FindRegistrySID(%s)", sid)
	defer log.VPrintf(3, "<Exit: FindRegistrySID")

	if tx.Registry != nil && tx.Registry.DbSID == sid {
		return tx.Registry, nil
	}

	ent, xErr := RawEntityFromPath(tx, sid, "", false, accessMode)
	if xErr != nil {
		return nil, NewXRError("server_error", "/").SetDetailf(
			"Error finding Registry %q: %s", sid, xErr.GetTitle())
	}
	if ent == nil {
		return nil, nil
	}

	reg := &Registry{Entity: *ent}
	reg.tx = tx
	reg.Self = reg
	reg.Entity.Registry = reg

	tx.Registry = reg
	tx.AddRegistry(reg)

	reg.LoadCapabilities()
	reg.LoadModel()

	return reg, nil
}

// BY UID
func FindRegistry(tx *Tx, id string, accessMode int) (*Registry, *XRError) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	if tx != nil && tx.Registry != nil && tx.Registry.UID == id {
		return tx.Registry, nil
	}

	newTx := false
	if tx == nil {
		var xErr *XRError
		tx, xErr = NewTx()
		if xErr != nil {
			return nil, xErr
		}
		newTx = true

		defer func() {
			// If we created a new Tx then assume someone is just looking
			// for the Registry and may not actually want to edit stuff, so
			// go ahead and close the Tx. It'll be reopened later if needed.
			// If a Tx was passed in then don't close it, the caller will
			if newTx { // not needed?
				tx.Rollback()
			}
		}()
	}

	defer func() {
		if newTx {
			tx.Rollback()
		}
	}()

	results := Query(tx, `
	   	SELECT SID
	   	FROM Registries
	   	WHERE UID=?`, id)

	defer results.Close()

	row := results.NextRow()

	if row == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	id = NotNilString(row[0])
	results.Close()

	ent, xErr := RawEntityFromPath(tx, id, "", false, accessMode)

	if xErr != nil {
		if newTx {
			tx.Rollback()
		}
		return nil, NewXRError("server_error", "/").SetDetailf(
			"Error finding Registry %q: %s", id, xErr.GetTitle())
	}

	PanicIf(ent == nil, "No entity but we found a reg")

	reg := &Registry{Entity: *ent}
	reg.Self = reg

	if tx.Registry == nil {
		tx.Registry = reg
	}

	reg.Entity.Registry = reg
	reg.tx = tx

	reg.tx.AddRegistry(reg)

	reg.LoadCapabilities()
	reg.LoadModel()

	return reg, nil
}

func (reg *Registry) LoadCapabilities() *Capabilities {
	capVal, ok := reg.Object["#capabilities"]
	if !ok {
		// No custom capabilities, use the default one
		reg.Capabilities = DefaultCapabilities
	} else {
		// Custom capabilities
		capStr, ok := capVal.(string)
		PanicIf(!ok, "not a byte array: %T", capVal)
		cap, xErr := ParseCapabilitiesJSON([]byte(capStr))
		Must(xErr)
		reg.Capabilities = cap
	}
	return reg.Capabilities
}

func (reg *Registry) LoadModel() *Model {
	return LoadModel(reg)
}

func (reg *Registry) SaveModel() *XRError {
	return reg.Model.VerifyAndSave()
}

func (reg *Registry) LoadModelFromFile(file string) *XRError {
	log.VPrintf(3, ">Enter: LoadModelFromFile: %s", file)
	defer log.VPrintf(3, "<Exit:LoadModelFromFile")

	var xErr *XRError
	var err error
	buf := []byte{}
	if strings.HasPrefix(file, "http") {
		res, err := http.Get(file)
		if err == nil {
			buf, err = io.ReadAll(res.Body)
			res.Body.Close()

			if res.StatusCode/100 != 2 {
				return NewXRError("parsing_data", file,
					"error_detail="+
						fmt.Sprintf("error getting model from %q: %s\n%s",
							file, res.Status, string(buf)))
			}
		}
	} else {
		buf, err = os.ReadFile(file)
	}
	if err != nil {
		return NewXRError("parsing_data", file,
			"error_detail="+
				fmt.Sprintf("processing %q: %s", file, err))
	}

	buf, xErr = ProcessIncludes(file, buf, true)
	if xErr != nil {
		return xErr
	}

	model, xErr := ParseModel(buf)
	if xErr != nil {
		return NewXRError("parsing_data", file,
			"error_detail="+
				fmt.Sprintf("processing %q: %s", file, err))
	}

	model.Registry = reg
	if xErr = model.Verify(); xErr != nil {
		return xErr
	}

	if xErr = reg.Model.ApplyNewModel(model, string(buf)); xErr != nil {
		return xErr
	}

	// reg.Model = model
	// reg.Model.VerifyAndSave()
	return nil
}

func (reg *Registry) Update(obj Object, addType AddType) *XRError {
	if xErr := CheckAttrs(obj, reg.XID); xErr != nil {
		return xErr
	}

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	reg.Lock()
	reg.SetNewObject(obj)

	// Ignore any incoming "model" attribute
	delete(reg.NewObject, "model")

	if reg.tx.RequestInfo.HasIgnore("capabilities") && !IsNil(reg.NewObject) {
		delete(reg.NewObject, "capabilities")
	}

	if reg.tx.RequestInfo.HasIgnore("modelsource") && !IsNil(reg.NewObject) {
		delete(reg.NewObject, "modelsource")
	}

	// Need to do it here instead of under the checkFn because doing it
	// in checkfn causes a circular reference that golang doesn't like
	val, ok := reg.NewObject["modelsource"]
	if ok {
		// Notice that "null" means erase it, not "keep it as is"
		var rawJson []byte

		if IsNil(val) {
			rawJson = []byte("{}")
		} else {
			var err error
			rawJson = val.(json.RawMessage)
			rawJson, err = RemoveSchema(rawJson)
			if err != nil {
				return NewXRError("bad_request", "/",
					"error_detail="+err.Error())
			}
		}

		xErr := reg.Model.ApplyNewModelFromJSON(rawJson)
		if xErr != nil {
			return xErr
		}

		delete(reg.NewObject, "modelsource")
	}

	// Make sure we always have an ID
	if IsNil(reg.NewObject["registryid"]) {
		reg.EnsureNewObject()
		reg.NewObject["registryid"] = reg.UID
	}

	// Remove/save all Registry level collections from NewObject
	collsMaps := map[string]map[string]any{}
	for _, coll := range reg.GetCollections() {
		plural := coll[0]
		singular := coll[1]

		collVal := obj[plural]
		if IsNil(collVal) {
			continue
		}
		collMap, ok := collVal.(map[string]any)
		if !ok {
			return NewXRError("invalid_attribute", "/",
				"name="+plural,
				"error_detail="+fmt.Sprintf("doesn't appear to be of a "+
					"map of %q", plural))
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return NewXRError("invalid_attribute", "/",
					"name="+plural,
					"error_detail="+
						fmt.Sprintf("key %q doesn't appear to be of type %q",
							key, singular))
			}
		}

		// Remove the Groups Collections attributes from the incoming obj
		collsMaps[plural] = collMap
		delete(reg.NewObject, plural)
		delete(reg.NewObject, plural+"count")
		delete(reg.NewObject, plural+"url")
	}

	// For each collection, upsert each entity
	for plural, collMap := range collsMaps {
		for key, val := range collMap {
			valObj, _ := val.(map[string]any)
			_, _, xErr := reg.UpsertGroupWithObject(plural, key, valObj,
				addType)
			if xErr != nil {
				return xErr
			}
		}
	}

	reg.EnsureNewObject()
	if addType == ADD_PATCH {
		// Copy existing props over if the incoming obj doesn't set them
		for k, val := range reg.Object {
			if _, ok := reg.NewObject[k]; !ok {
				reg.NewObject[k] = val
			}
		}
	}

	return reg.ValidateAndSave()
}

func (reg *Registry) FindGroup(gType string, id string, anyCase bool, accessMode int) (*Group, *XRError) {
	log.VPrintf(3, ">Enter: FindGroup(%s,%s,%v)", gType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindGroup")

	if g := reg.tx.GetGroup(reg, gType, id); g != nil {
		if accessMode == FOR_WRITE && g.AccessMode != FOR_WRITE {
			g.Lock()
		}
		return g, nil
	}

	ent, xErr := RawEntityFromPath(reg.tx, reg.DbSID, gType+"/"+id, anyCase,
		accessMode)
	if xErr != nil {
		return nil, NewXRError("server_error", "/").SetDetailf(
			"Error finding Group %q(%s): %s", id, gType, xErr.GetTitle())
	}
	if ent == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	g := &Group{Entity: *ent, Registry: reg}
	g.Self = g
	g.tx.AddGroup(g)
	return g, nil
}

func (reg *Registry) AddGroup(gType string, id string) (*Group, *XRError) {
	g, _, xErr := reg.UpsertGroupWithObject(gType, id, nil, ADD_ADD)
	return g, xErr
}

func (reg *Registry) AddGroupWithObject(gType string, id string, obj Object) (*Group, *XRError) {
	g, _, xErr := reg.UpsertGroupWithObject(gType, id, obj, ADD_ADD)
	return g, xErr
}

// *Group, isNew, error
func (reg *Registry) UpsertGroup(gType string, id string) (*Group, bool, *XRError) {
	return reg.UpsertGroupWithObject(gType, id, nil, ADD_UPSERT)
}

func (reg *Registry) UpsertGroupWithObject(gType string, id string, obj Object, addType AddType) (*Group, bool, *XRError) {
	log.VPrintf(3, ">Enter UpsertGroupWithObject(%s,%s)", gType, id)
	defer log.VPrintf(3, "<Exit UpsertGroupWithObject")

	// Need this because its parent (the registry) might not be locked, which
	// we need because we need to change stuff in it. And we don't want all
	// callers of this func to have to re-Find/lock the registry themselves.
	// The registry at this point is the generic "find the registry for read"
	// that all interactions go thru.
	reg.Lock()

	if xErr := reg.SaveModel(); xErr != nil {
		return nil, false, xErr
	}

	if xErr := CheckAttrs(obj, "/"+gType+"/"+id); xErr != nil {
		return nil, false, xErr
	}

	gm := reg.Model.Groups[gType]
	if gm == nil {
		return nil, false, NewXRError("not_found", "/"+gType)
	}

	if id == "" {
		id = NewUUID()
	}

	isNew := false

	g, xErr := reg.FindGroup(gType, id, true, FOR_WRITE)
	if xErr != nil {
		return nil, false, xErr
	}

	if g != nil && g.UID != id {
		return nil, false,
			NewXRError("bad_request", "/"+g.UID,
				"error_detail="+
					fmt.Sprintf("Attempting to create a Group "+
						"with a \"%sid\" of %q, when one already exists as %q",
						gm.Singular, id, g.UID))
	}

	if addType == ADD_ADD && g != nil {
		return nil, false,
			NewXRError("bad_request", "/"+id,
				"error_detail="+
					fmt.Sprintf("Group %q of type %q already exists",
						id, gType))
	}

	for g == nil {
		// Not found, so create a new one
		g = &Group{
			Entity: Entity{
				EntityExtensions: EntityExtensions{
					tx:         reg.tx,
					AccessMode: FOR_WRITE,
				},

				Registry: reg,
				DbSID:    NewUUID(),
				Plural:   gType,
				Singular: gm.Singular,
				UID:      id,

				Type:     ENTITY_GROUP,
				Path:     gType + "/" + id,
				XID:      "/" + gType + "/" + id,
				Abstract: gType,

				GroupModel: gm,
			},
			Registry: reg,
		}
		g.Self = g

		DoOne(reg.tx, `
			INSERT INTO "Groups"(
                SID, RegistrySID, UID,
                ModelSID, Path, Abstract,
                Plural, Singular)
			SELECT ?,?,?,?,?,?,?,?`,
			g.DbSID, g.Registry.DbSID, g.UID,
			gm.SID, g.Path, g.Abstract,
			g.Plural, g.Singular)

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		if xErr = g.JustSet(g.Singular+"id", g.UID); xErr != nil {
			return nil, false, xErr
		}
		isNew = true
		g.Registry.Touch()
		g.tx.AddGroup(g)
	}

	// Remove all Resource collections from obj before we process it
	objColls := map[string]map[string]any{}
	for _, coll := range g.GetCollections() {
		plural := coll[0]
		singular := coll[1]

		collVal := obj[plural]
		if IsNil(collVal) {
			continue
		}

		collMap, ok := collVal.(map[string]any)
		if !ok {
			return nil, false,
				NewXRError("invalid_attribute", "/"+gType+"/"+id,
					"name="+plural,
					"error_detail="+
						fmt.Sprintf("doesn't appear to be of a "+
							"map of %q", plural))
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return nil, false,
					NewXRError("invalid_attribute", "/"+plural,
						"name="+plural,
						"error_detail="+
							fmt.Sprintf("Key %q doesn't appear to be of "+
								"type %q", key, singular))
			}
		}

		objColls[plural] = collMap
		delete(obj, plural)
		delete(obj, plural+"count")
		delete(obj, plural+"url")
	}

	if isNew || obj != nil {
		if obj != nil {
			g.SetNewObject(obj)
		}

		if addType == ADD_PATCH {
			// Copy existing props over if the incoming obj doesn't set them
			for k, v := range g.Object {
				if _, ok := g.NewObject[k]; !ok {
					g.NewObject[k] = v
				}
			}
		}

		// Make sure we always have an ID
		if IsNil(g.NewObject[g.Singular+"id"]) {
			g.NewObject[g.Singular+"id"] = g.UID
		}

		if xErr = g.ValidateAndSave(); xErr != nil {
			return nil, false, xErr
		}
	}

	// Now for each inlined Resource collection, upsert each Resource
	for plural, daMap := range objColls {
		for key, val := range daMap {
			valObj, _ := val.(map[string]any)
			_, _, xErr := g.UpsertResourceWithObject(plural, key, "",
				valObj, addType, false)
			if xErr != nil {
				return nil, false, xErr
			}
		}
	}

	if xErr = reg.ValidateAndSave(); xErr != nil {
		return nil, false, xErr
	}

	return g, isNew, nil
}

// sortKey = attribute name, -NAME means descending, no "-" means ascending
func GenerateQuery(reg *Registry, what string, paths []string, filters [][]*FilterExpr, docView bool, sortKey string) (string, []interface{}, *XRError) {
	query := ""
	args := []any{}

	if sortKey != "" && what != "Coll" {
		return "", nil, NewXRError("bad_sort", "",
			"sort_value="+sortKey,
			"error_detail=can't sort on a non-collection results")
	}

	ascDesc := "ASC"
	sortJoin := ""
	sortOrder := ""

	if sortKey != "" {
		if sortKey[0] == '-' {
			ascDesc = "DESC"
			sortKey = sortKey[1:]
		}

		count := strings.Count(paths[0], "/")
		if count == 0 {
			count = 2
		} else if count == 2 {
			count = 4
		} else {
			count = 6
		}
		slashCount := fmt.Sprintf("%d", count)

		/*
					sortOrder = `
			    sj.PropValue IS NOT NULL,
			    CASE WHEN sj.PropType IN ('integer','decimal','uinteger')
			      THEN CAST(sj.PropValue AS DECIMAL) END ` + ascDesc + `,
			    CASE WHEN sj.PropType NOT IN ('integer','decimal','uinteger')
			      THEN sj.PropValue END ` + ascDesc + `,
			`
		*/

		sortOrder = `
    sj.PropValue IS NOT NULL ` + ascDesc + `,
    CASE
      WHEN sj.PropType IN ('integer','decimal','uinteger') THEN
        IFNULL(CAST(sj.PropValue AS DECIMAL), 0)
      WHEN sj.PropType NOT IN ('integer','decimal','uinteger') THEN
        sj.PropValue
      ELSE NULL
    END COLLATE utf8mb4_general_ci ` + ascDesc + `,
`

		sortJoin = `
  LEFT JOIN FullTree AS sj ON (
    sj.RegSID = ft.RegSID AND
    sj.Path = substring_index(ft.Path, '/', ` + slashCount + `) AND
    sj.PropName = '` + sortKey + `')
`
	}

	args = []interface{}{reg.DbSID}
	query = `
SELECT
  ft.RegSID,ft.Type,ft.Plural,ft.Singular,ft.eSID,ft.UID,ft.PropName,ft.PropValue,ft.PropType,ft.Path,ft.Abstract
  FROM FullTree AS ft` + sortJoin + `
  WHERE ft.RegSID=?
`

	// Exclude generated attributes/entities if 'doc view' is turned on.
	// Meaning, only grab Props that have 'DocView' set to 'true'. These
	// should be (mainly) just the ones we set explicitly.
	if docView {
		query += `  AND ft.DocView=true
`
	}

	// Remove entities that are higher than the GET PATH specified
	if what != "Registry" && len(paths) > 0 {
		query += "  AND ("
		for i, p := range paths {
			if i > 0 {
				query += " OR "
			}
			query += "ft.Path=? OR ft.Path LIKE ?"
			args = append(args, p, p+"/%")
		}
		query += ")\n"

	}

	if len(filters) != 0 {
		query += `
AND
(
eSID IN ( -- eSID from query
  -- Find all entities that match the filters, and then grab all parents
  -- This "RECURSIVE" stuff finds all parents
  WITH RECURSIVE cte(eSID,Type,ParentSID,Path) AS (
    -- This defines the init set of rows of the query. We'll recurse later on
    SELECT eSID,Type,ParentSID,Path FROM Entities
    WHERE eSID in ( -- start of the OR Filter groupings`
		// This section will find all matching entities
		firstOr := true
		for _, OrFilters := range filters {
			if !firstOr {
				query += `
      UNION -- Adding another OR`
			}
			firstOr = false
			query += `
      -- start of one Filter AND grouping (expr1 AND expr2).
      -- Find all SIDs for the leaves for entities (SIDs) of interest.
      SELECT list.eSID FROM (
        SELECT count(*) as cnt,e2.eSID,e2.Path FROM Entities AS e1
        RIGHT JOIN (
          -- start of expr1 - below finds SearchNodes/SIDs of interest`
			firstAnd := true
			andCount := 0
			for _, filter := range OrFilters { // AndFilters
				andCount++
				if !firstAnd {
					query += `
          UNION ALL`
				}
				firstAnd = false

				if filter.Operator == FILTER_PRESENT { // ?filter=xxx
					// BINARY means case-sensitive for that operand
					check := "(BINARY Abstract=? AND PropName=? AND "

					args = append(args, reg.DbSID, filter.Abstract,
						filter.PropName)
					check += "PropValue IS NOT NULL)"
					query += `
          SELECT eSID,Type,Path FROM FullTree WHERE RegSID=? AND ` + check

				} else if filter.Operator == FILTER_ABSENT { // ?filter=xxx=null
					// Look for non-existing prop
					args = append(args, reg.DbSID, filter.Abstract,
						filter.PropName)

					// BINARY means case-sensitive for that operand
					query += `
          -- Entities that don't have the specified prop
          SELECT e.eSID,e.Type,e.Path FROM Entities AS e
          WHERE e.RegSID=? AND e.Abstract=? AND
            NOT EXISTS (SELECT 1 FROM FullTree WHERE
              RegSID=e.RegSID AND eSID=e.eSID AND (BINARY PropName=?))`

				} else if filter.Operator == FILTER_EQUAL { // ?filter=xxx=zzz
					// BINARY means case-sensitive for that operand
					check := "(BINARY Abstract=? AND PropName=? AND "

					args = append(args, reg.DbSID, filter.Abstract,
						filter.PropName)
					value, wildcard := WildcardIt(filter.Value)
					args = append(args, value)
					if !wildcard {
						check += "PropValue=?"
					} else {
						args = append(args, value)
						check += "((PropType<>'string' AND PropValue=?) " +
							" OR (PropType='string' AND PropValue LIKE ?))"
					}
					check += ")"
					query += `
          SELECT eSID,Type,Path FROM FullTree
            WHERE RegSID=? AND ` + check

				} else if filter.Operator == FILTER_NOT_EQUAL { // ?filter=x!=z
					args = append(args, reg.DbSID, filter.Abstract,
						filter.PropName)
					// BINARY means case-sensitive for that operand
					query += `
          -- Entities that don't have the specified prop
          SELECT e.eSID,e.Type,e.Path FROM Entities AS e
          WHERE e.RegSID=? AND e.Abstract=? AND
            NOT EXISTS (SELECT 1 FROM FullTree WHERE
              RegSID=e.RegSID AND eSID=e.eSID AND (BINARY PropName=? AND `

					value, wildcard := WildcardIt(filter.Value)
					args = append(args, value)
					if !wildcard {
						query += "PropValue=?"
					} else {
						args = append(args, value)
						query += "((PropType<>'string' AND PropValue=?) " +
							" OR (PropType='string' AND PropValue LIKE ?))"
					}
					query += "))"

				} else {
					PanicIf(true, "Bad filter.op: %#v", filter)
				}
			} // end of AndFilter
			query += `
          -- end of expr1
        ) AS result ON ( result.eSID=e1.eSID )
        -- For each result found, find all Leaves under the matching entity.
        -- The Leaves that show up 'cnt' times, where cnt is the # of
        -- expressions in each filter (the ANDs), are branches to return.
        -- Note we return the Path of each Leaf, not the path of the matching
        -- entity. The entity that matches isn't important.
        JOIN Entities AS e2 ON (
          (
            (
              -- Non-meta objects, just compare the Path
              result.Type<>` + StrTypes(ENTITY_META) + ` AND
              ( e2.Path=result.Path OR
                e2.Path LIKE CONCAT(IF(result.Path<>'',CONCAT(result.Path,'/'),''),'%')
              )
            )
            OR
            (
              -- For 'meta' objects, compare it's parent's Path
              result.Type=` + StrTypes(ENTITY_META) + ` AND
              ( e2.Path=TRIM(TRAILING '/meta' FROM result.Path) OR
                e2.Path LIKE CONCAT(TRIM(TRAILING 'meta' FROM result.Path),'%')
              )
            )
          )
          AND e2.eSID IN (SELECT * from Leaves)
        ) GROUP BY e2.eSID
        -- end of RIGHT JOIN
      ) as list
      WHERE list.cnt=?   -- cnt is the # of operands in the AND filter
      -- end of one Filter AND grouping (expr1 AND expr2 ...)`
			args = append(args, andCount)
		} // end of OrFilter

		query += `
    ) -- end of all OR Filter groupings

    -- This is the recusive part of the query.
    -- Find all of the parents (and 'meta' sub-objects) of the found
    -- entities, up to root of Reg.
    UNION DISTINCT SELECT
      e.eSID,e.Type,e.ParentSID,e.Path
    FROM Entities AS e
    INNER JOIN cte ON
      (
        -- Find its parent
        e.eSID=cte.ParentSID
        OR
        -- If this is a Resource, grab its 'meta' sub-object
        ( cte.Type=` + StrTypes(ENTITY_RESOURCE) + ` AND
          e.Type=` + StrTypes(ENTITY_META) + ` AND
          e.ParentSID=cte.eSID
        )
      )
  )
  SELECT DISTINCT eSID FROM cte )
)
`
	}

	query += `  ORDER BY ` + sortOrder +
		`    ft.Path COLLATE utf8mb4_general_ci ASC;`

	log.VPrintf(3, "Query:\n%s\n\n", SubQuery(query, args))
	return query, args, nil
}

func WildcardIt(str string) (string, bool) {
	wild := false
	res := strings.Builder{}

	prevch := '\000'
	for _, ch := range str {
		if ch == '*' && prevch != '\\' {
			res.WriteRune('%')
			wild = true
		} else {
			res.WriteRune(ch)
		}
		prevch = ch
	}

	return res.String(), wild
}

func (r *Registry) XID2Entity(xidStr string, path string) (*Entity, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}

	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}
	if g == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("cant find Group %q", xid.Group))
	}
	if xid.Type == ENTITY_GROUP {
		return &g.Entity, nil
	}

	if xid.IsEntity == false || xid.Type == ENTITY_META {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("%q isn't an xid", xidStr))
	}

	res, xErr := g.FindResource(xid.Resource, xid.ResourceID, false, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}

	if res == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("unknown Resource %q", xid.Resource))
	}
	if xid.Type == ENTITY_RESOURCE {
		return &res.Entity, nil
	}

	v, xErr := res.FindVersion(xid.VersionID, false, FOR_READ)
	if xErr != nil {
		return nil, xErr
	}
	if v == nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+fmt.Sprintf("unknown Version %q", xid.VersionID))
	}
	if xid.Type == ENTITY_VERSION {
		return &v.Entity, nil
	}

	return nil, NewXRError("malformed_xid", path,
		"xid="+xidStr,
		"error_detail=not a valid XID")
}

func (r *Registry) FindXIDGroup(xidStr string, path string) (*Group, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}

	return r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
}

func (r *Registry) FindResourceByXID(xidStr string, path string) (*Resource, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if xErr != nil || g == nil {
		return nil, xErr
	}
	return g.FindResource(xid.Resource, xid.ResourceID, false, FOR_READ)
}

func (r *Registry) FindXIDVersion(xidStr string, path string) (*Version, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	if xid.VersionID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"versionid\"")
	}
	if xid.Version != "versions" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=not a \"versions\" XID")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if xErr != nil || g == nil {
		return nil, xErr
	}
	resource, xErr := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if xErr != nil || resource == nil {
		return nil, xErr
	}
	return resource.FindVersion(xid.VersionID, false, FOR_READ)
}

func (r *Registry) FindXIDMeta(xidStr string, path string) (*Meta, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.GroupID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"groupid\"")
	}
	if xid.ResourceID == "" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=missing a \"resourceid\"")
	}
	if xid.Version != "meta" {
		return nil, NewXRError("malformed_xid", path,
			"xid="+xidStr,
			"error_detail=not a \"meta\" XID")
	}
	g, xErr := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if !IsNil(err) || g == nil {
		return nil, xErr
	}
	resource, xErr := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if xErr != nil || resource == nil {
		return nil, xErr
	}
	return resource.FindMeta(false, FOR_READ)
}

func LoadRemoteRegistry(host string) (*Registry, *XRError) {
	reg := &Registry{}

	// Download model
	data, err := DownloadURL(host + "/model")
	if err != nil {
		return nil, NewXRError("bad_request", host+"/model",
			"error_detail="+
				fmt.Sprintf("Error getting model (%s/model): %s", host, err))
	}

	var xErr *XRError
	reg.Model, xErr = ParseModel(data)
	if xErr != nil {
		return nil, xErr
	}

	// Download capabilities
	data, err = DownloadURL(host + "/capabilities")
	if err == nil {
		var xErr *XRError
		reg.Capabilities, xErr = ParseCapabilitiesJSON(data)

		if xErr != nil {
			return nil, xErr
		}
	} else {
		return nil, NewXRError("bad_request", host+"/capabilities",
			"error_detail="+
				fmt.Sprintf("Error getting capabilities "+
					"(%s/capabilities): %s", host, err))
	}

	return reg, nil
}
