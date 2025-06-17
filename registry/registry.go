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
		var err error
		tx, err = NewTx()
		Must(err)
	}

	reg, err := FindRegistryBySID(tx, DefaultRegDbSID, FOR_READ)
	Must(err)

	if reg != nil {
		tx.Registry = reg
	}
	PanicIf(reg == nil, "No default registry")

	return reg
}

func (r *Registry) Rollback() error {
	if r != nil {
		return r.tx.Rollback()
	}
	return nil
}

func (r *Registry) SaveAllAndCommit() error {
	if r != nil {
		if r.Model.GetChanged() {
			if err := r.SaveModel(); err != nil {
				return err
			}
		}
		return r.tx.SaveAllAndCommit()
	}
	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) SaveCommitRefresh() error {
	if r != nil {
		return r.tx.SaveCommitRefresh()
	}
	return nil
}

// ONLY CALL FROM TESTS - NEVER IN PROD
func (r *Registry) AddToCache(e *Entity) {
	r.tx.AddToCache(e)
}

func (r *Registry) Commit() error {
	if r != nil {
		return r.tx.Commit()
	}
	return nil
}

type RegOpt string

func NewRegistry(tx *Tx, id string, regOpts ...RegOpt) (*Registry, error) {
	log.VPrintf(3, ">Enter: NewRegistry %q", id)
	defer log.VPrintf(3, "<Exit: NewRegistry")

	var err error // must be used for all error checking due to defer
	newTx := false

	defer func() {
		if newTx {
			// If created just for us, close it
			tx.Conditional(err)
		}
	}()

	if tx == nil {
		tx, err = NewTx()
		if err != nil {
			return nil, err
		}
		newTx = true
	}

	if id == "" {
		id = NewUUID()
	}

	r, err := FindRegistry(tx, id, FOR_READ)
	if err != nil {
		return nil, err
	}
	if r != nil {
		return nil, fmt.Errorf("A registry with ID %q already exists", id)
	}

	dbSID := NewUUID()
	err = DoOne(tx, `
		INSERT INTO Registries(SID, UID)
		VALUES(?,?)`, dbSID, id)
	if err != nil {
		return nil, err
	}

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

	err = reg.Model.Verify()
	if err != nil {
		return nil, err
	}

	err = DoOne(tx, `
		INSERT INTO Models(RegistrySID)
		VALUES(?)`, dbSID)
	if err != nil {
		return nil, err
	}

	if err = reg.JustSet("specversion", SPECVERSION); err != nil {
		return nil, err
	}
	if err = reg.JustSet("registryid", reg.UID); err != nil {
		return nil, err
	}

	/*
		for _, regOpt := range regOpts {
			// if regOpts == RegOpt_STRING { ... }
		}
	*/

	if err = reg.SetSave("epoch", 1); err != nil {
		return nil, err
	}

	if err = reg.Model.VerifyAndSave(); err != nil {
		return nil, err
	}

	return reg, nil
}

func GetRegistryNames() []string {
	tx, err := NewTx()
	if err != nil {
		return []string{} // error!
	}
	defer tx.Rollback()

	results, err := Query(tx, ` SELECT UID FROM Registries`)
	defer results.Close()

	if err != nil {
		panic(err.Error())
	}

	res := []string{}
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		res = append(res, NotNilString(row[0]))
	}

	return res
}

var _ EntitySetter = &Registry{}

func (reg *Registry) Get(name string) any {
	return reg.Entity.Get(name)
}

func (reg *Registry) JustSet(name string, val any) error {
	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update' will just lock it automatically
	reg.Lock()
	return reg.Entity.eJustSet(NewPPP(name), val)
}

func (reg *Registry) SetSave(name string, val any) error {
	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update' will just lock it automatically
	reg.Lock()
	return reg.Entity.eSetSave(name, val)
}

func (reg *Registry) Delete() error {
	log.VPrintf(3, ">Enter: Reg.Delete(%s)", reg.UID)
	defer log.VPrintf(3, "<Exit: Reg.Delete")

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	reg.Lock()
	err := DoOne(reg.tx, `DELETE FROM Registries WHERE SID=?`, reg.DbSID)
	if err != nil {
		return err
	}
	reg.tx.EraseCache()
	return nil
}

func FindRegistryBySID(tx *Tx, sid string, accessMode int) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistrySID(%s)", sid)
	defer log.VPrintf(3, "<Exit: FindRegistrySID")

	if tx.Registry != nil && tx.Registry.DbSID == sid {
		return tx.Registry, nil
	}

	ent, err := RawEntityFromPath(tx, sid, "", false, accessMode)
	if err != nil {
		return nil, fmt.Errorf("Error finding Registry %q: %s", sid, err)
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
func FindRegistry(tx *Tx, id string, accessMode int) (*Registry, error) {
	log.VPrintf(3, ">Enter: FindRegistry(%s)", id)
	defer log.VPrintf(3, "<Exit: FindRegistry")

	if tx != nil && tx.Registry != nil && tx.Registry.UID == id {
		return tx.Registry, nil
	}

	newTx := false
	if tx == nil {
		var err error
		tx, err = NewTx()
		if err != nil {
			return nil, err
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

	results, err := Query(tx, `
	   	SELECT SID
	   	FROM Registries
	   	WHERE UID=?`, id)

	defer results.Close()

	if err != nil {
		if newTx {
			tx.Rollback()
		}
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
	}

	row := results.NextRow()

	if row == nil {
		log.VPrintf(3, "None found")
		return nil, nil
	}

	id = NotNilString(row[0])
	results.Close()

	ent, err := RawEntityFromPath(tx, id, "", false, accessMode)

	if err != nil {
		if newTx {
			tx.Rollback()
		}
		return nil, fmt.Errorf("Error finding Registry %q: %s", id, err)
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
		cap, err := ParseCapabilitiesJSON([]byte(capStr))
		Must(err)
		reg.Capabilities = cap
	}
	return reg.Capabilities
}

func (reg *Registry) LoadModel() *Model {
	return LoadModel(reg)
}

func (reg *Registry) SaveModel() error {
	return reg.Model.VerifyAndSave()
}

func (reg *Registry) LoadModelFromFile(file string) error {
	log.VPrintf(3, ">Enter: LoadModelFromFile: %s", file)
	defer log.VPrintf(3, "<Exit:LoadModelFromFile")

	var err error
	buf := []byte{}
	if strings.HasPrefix(file, "http") {
		res, err := http.Get(file)
		if err == nil {
			buf, err = io.ReadAll(res.Body)
			res.Body.Close()

			if res.StatusCode/100 != 2 {
				err = fmt.Errorf("Error getting model: %s\n%s",
					res.Status, string(buf))
			}
		}
	} else {
		buf, err = os.ReadFile(file)
	}
	if err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	buf, err = ProcessIncludes(file, buf, true)
	if err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	model, err := ParseModel(buf)
	if err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	model.Registry = reg
	if err := model.Verify(); err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	if err := reg.Model.ApplyNewModel(model); err != nil {
		return fmt.Errorf("Processing %q: %s", file, err)
	}

	// reg.Model = model
	// reg.Model.VerifyAndSave()
	return nil
}

func (reg *Registry) Update(obj Object, addType AddType) error {
	if err := CheckAttrs(obj); err != nil {
		return err
	}

	// Normally we should never call Lock() directly, however Registry is
	// kind of special because we rarely know if we want to "Find" the Registry
	// for writing until later in the process. So instead of forcing the
	// code to re-Find with FOR_WRITE, we'll just make it easy and these
	// variants of 'update'  will just lock it automatically
	reg.Lock()
	reg.SetNewObject(obj)

	// Need to do it here instead of under the checkFn because doing it
	// in checkfn causes a circular reference that golang doesn't like
	val, ok := reg.NewObject["modelsource"]
	if ok {
		// Notice that "null" means erase it, not "keep it as is"
		var rawJson []byte

		if IsNil(val) {
			rawJson = []byte("{}")
		} else {
			rawJson = val.(json.RawMessage)
		}

		err := reg.Model.ApplyNewModelFromJSON(rawJson)
		if err != nil {
			return err
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
			return fmt.Errorf("Attribute %q doesn't appear to be of a "+
				"map of %q", plural, plural)
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return fmt.Errorf("Key %q in attribute %q doesn't "+
					"appear to be of type %q", key, plural, singular)
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
			_, _, err := reg.UpsertGroupWithObject(plural, key, valObj,
				addType)
			if err != nil {
				return err
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

func (reg *Registry) FindGroup(gType string, id string, anyCase bool, accessMode int) (*Group, error) {
	log.VPrintf(3, ">Enter: FindGroup(%s,%s,%v)", gType, id, anyCase)
	defer log.VPrintf(3, "<Exit: FindGroup")

	if g := reg.tx.GetGroup(reg, gType, id); g != nil {
		if accessMode == FOR_WRITE && g.AccessMode != FOR_WRITE {
			g.Lock()
		}
		return g, nil
	}

	ent, err := RawEntityFromPath(reg.tx, reg.DbSID, gType+"/"+id, anyCase,
		accessMode)
	if err != nil {
		return nil, fmt.Errorf("Error finding Group %q(%s): %s", id, gType, err)
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

func (reg *Registry) AddGroup(gType string, id string) (*Group, error) {
	g, _, err := reg.UpsertGroupWithObject(gType, id, nil, ADD_ADD)
	return g, err
}

func (reg *Registry) AddGroupWithObject(gType string, id string, obj Object) (*Group, error) {
	g, _, err := reg.UpsertGroupWithObject(gType, id, obj, ADD_ADD)
	return g, err
}

// *Group, isNew, error
func (reg *Registry) UpsertGroup(gType string, id string) (*Group, bool, error) {
	return reg.UpsertGroupWithObject(gType, id, nil, ADD_UPSERT)
}

func (reg *Registry) UpsertGroupWithObject(gType string, id string, obj Object, addType AddType) (*Group, bool, error) {
	log.VPrintf(3, ">Enter UpsertGroupWithObject(%s,%s)", gType, id)
	defer log.VPrintf(3, "<Exit UpsertGroupWithObject")

	// Need this because its parent (the registry) might not be locked, which
	// we need because we need to change stuff in it. And we don't want all
	// callers of this func to have to re-Find/lock the registry themselves.
	// The registry at this point is the generic "find the registry for read"
	// that all interactions go thru.
	reg.Lock()

	if err := reg.SaveModel(); err != nil {
		return nil, false, err
	}

	if err := CheckAttrs(obj); err != nil {
		return nil, false, err
	}

	gm := reg.Model.Groups[gType]
	if gm == nil {
		return nil, false, fmt.Errorf("Error adding Group, unknown type: %s",
			gType)
	}

	if id == "" {
		id = NewUUID()
	}

	isNew := false

	g, err := reg.FindGroup(gType, id, true, FOR_WRITE)
	if err != nil {
		return nil, false, fmt.Errorf("Error finding Group(%s) %q: %s",
			gType, id, err)
	}

	if g != nil && g.UID != id {
		return nil, false, fmt.Errorf("Attempting to create a Group "+
			"with a \"%sid\" of %q, when one already exists as %q",
			gm.Singular, id, g.UID)
	}

	if addType == ADD_ADD && g != nil {
		return nil, false, fmt.Errorf("Group %q of type %q already exists",
			id, gType)
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
				Abstract: gType,

				GroupModel: gm,
			},
			Registry: reg,
		}
		g.Self = g

		err = DoOne(reg.tx, `
			INSERT INTO "Groups"(
                SID, RegistrySID, UID,
                ModelSID, Path, Abstract,
                Plural, Singular)
			SELECT ?,?,?,?,?,?,?,?`,
			g.DbSID, g.Registry.DbSID, g.UID,
			gm.SID, g.Path, g.Abstract,
			g.Plural, g.Singular)

		if err != nil {
			err = fmt.Errorf("Error adding Group: %s", err)
			log.Print(err)
			return nil, false, err
		}

		// Use the ID passed as an arg, not from the metadata, as the true
		// ID. If the one in the metadata differs we'll flag it down below
		if err = g.JustSet(g.Singular+"id", g.UID); err != nil {
			return nil, false, err
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
				fmt.Errorf("Attribute %q doesn't appear to be of a "+
					"map of %q", plural, plural)
		}
		for key, val := range collMap {
			_, ok := val.(map[string]any)
			if !ok {
				return nil, false,
					fmt.Errorf("Key %q in attribute %q doesn't "+
						"appear to be of type %q", key, plural, singular)
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

		if err = g.ValidateAndSave(); err != nil {
			return nil, false, err
		}
	}

	// Now for each inlined Resource collection, upsert each Resource
	for plural, daMap := range objColls {
		for key, val := range daMap {
			valObj, _ := val.(map[string]any)
			_, _, err := g.UpsertResourceWithObject(plural, key, "",
				valObj, addType, false)
			if err != nil {
				return nil, false, err
			}
		}
	}

	if err = reg.ValidateAndSave(); err != nil {
		return nil, false, err
	}

	return g, isNew, nil
}

// sortKey = attribute name, -NAME means descending, no "-" means ascending
func GenerateQuery(reg *Registry, what string, paths []string, filters [][]*FilterExpr, docView bool, sortKey string) (string, []interface{}, error) {
	query := ""
	args := []any{}

	if sortKey != "" && what != "Coll" {
		return "", nil, fmt.Errorf("Can't sort on a non-collection results")
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

func (r *Registry) XID2Entity(xidStr string) (*Entity, error) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}

	g, err := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, fmt.Errorf("Cant find Group %q from xid %q", xid.Group,
			xidStr)
	}
	if xid.Type == ENTITY_GROUP {
		return &g.Entity, nil
	}

	if xid.IsEntity == false || xid.Type == ENTITY_META {
		return nil, fmt.Errorf("%q isn't an xid", xidStr)
	}

	res, err := g.FindResource(xid.Resource, xid.ResourceID, false, FOR_READ)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("Can't find Resource %q from xid %q",
			xid.Resource, xidStr)
	}
	if xid.Type == ENTITY_RESOURCE {
		return &res.Entity, nil
	}

	v, err := res.FindVersion(xid.VersionID, false, FOR_READ)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, fmt.Errorf("Can't find Version %q from xid %q",
			xid.VersionID, xidStr)
	}
	if xid.Type == ENTITY_VERSION {
		return &v.Entity, nil
	}

	return nil, fmt.Errorf("xid %q isn't valid", xidStr)
}

func (r *Registry) FindXIDGroup(xidStr string) (*Group, error) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.GroupID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"groupid\"", xidStr)
	}

	return r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
}

func (r *Registry) FindResourceByXID(xidStr string) (*Resource, error) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.GroupID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"groupid\"", xidStr)
	}
	if xid.ResourceID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"resourceid\"", xidStr)
	}
	g, err := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if err != nil || g == nil {
		return nil, err
	}
	return g.FindResource(xid.Resource, xid.ResourceID, false, FOR_READ)
}

func (r *Registry) FindXIDVersion(xidStr string) (*Version, error) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.GroupID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"groupid\"", xidStr)
	}
	if xid.ResourceID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"resourceid\"", xidStr)
	}
	if xid.VersionID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"versionid\"", xidStr)
	}
	if xid.Version != "versions" {
		return nil, fmt.Errorf("XID %q is \"versions\"", xidStr)
	}
	g, err := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if err != nil || g == nil {
		return nil, err
	}
	resource, err := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if err != nil || resource == nil {
		return nil, err
	}
	return resource.FindVersion(xid.VersionID, false, FOR_READ)
}

func (r *Registry) FindXIDMeta(xidStr string) (*Meta, error) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.GroupID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"groupid\"", xidStr)
	}
	if xid.ResourceID == "" {
		return nil, fmt.Errorf("XID %q is missing a \"resourceid\"", xidStr)
	}
	if xid.Version != "meta" {
		return nil, fmt.Errorf("XID %q is \"meta\"", xidStr)
	}
	g, err := r.FindGroup(xid.Group, xid.GroupID, false, FOR_READ)
	if err != nil || g == nil {
		return nil, err
	}
	resource, err := g.FindResource(xid.Resource, xid.ResourceID, false,
		FOR_READ)
	if err != nil || resource == nil {
		return nil, err
	}
	return resource.FindMeta(false, FOR_READ)
}

func LoadRemoteRegistry(host string) (*Registry, error) {
	reg := &Registry{}

	// Download model
	data, err := DownloadURL(host + "/model")
	if err == nil {
		reg.Model, err = ParseModel(data)
	}
	if err != nil {
		return nil, fmt.Errorf("Error getting model (%s/model): %s", host, err)
	}

	// Download capabilities
	data, err = DownloadURL(host + "/capabilities")
	if err == nil {
		reg.Capabilities, err = ParseCapabilitiesJSON(data)
	}
	if err != nil {
		return nil, fmt.Errorf("Error getting capabilities "+
			"(%s/capabilities): %s", host, err)
	}

	return reg, nil
}
