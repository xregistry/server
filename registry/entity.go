package registry

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

type Object map[string]any

// type Map map[string]any
// type Array []any

type Entity struct {
	tx *Tx

	Registry  *Registry `json:"-"`
	DbSID     string    // Entity's SID
	Plural    string
	Singular  string
	UID       string         // Entity's UID
	Object    map[string]any `json:"-"`
	NewObject map[string]any `json:"-"` // updated version, save() will store

	// These were added just for convenience and so we can use the same
	// struct for traversing the SQL results
	Type     int    // ENTITY_REGISTRY(0)/GROUP(1)/RESOURCE(2)/VERSION(3)/...
	Path     string // [GROUPS/gID[/RESOURCES/rID[/versions/vID]]]
	Abstract string // [GROUPS[/RESOURCES[/versions]]]
	EpochSet bool   `json:"-"` // Has epoch changed this tx?
	ModSet   bool   `json:"-"` // Has modifiedat changed this tx?
	Self     any    `json:"-"` // ptr to typed Entity (e.g. *Group)

	// Save these values in memory so we only need to get them once
	GroupModel     *GroupModel    `json:"-"` // gModel if it's not a Registry
	ResourceModel  *ResourceModel `json:"-"` // If Res,Ver,Meta save rmModel
	VerifyVersions bool           `json:"-"` // If Res, check versions?
	LatestVersion  string         `json:"-"` // If Res (vID)

	// Debugging
	NewObjectStack []string `json:"-"` // stack when NewObj created via Ensure
}

type EntitySetter interface {
	Get(name string) any
	SetCommit(name string, val any) error // Should never be used
	JustSet(name string, val any) error
	SetSave(name string, val any) error
	Delete() error
}

func (e *Entity) GetResourceSingular() string {
	rm := e.GetResourceModel()
	if rm != nil {
		return rm.Singular
	}
	return ""
}

func (e *Entity) GetResourceModel() *ResourceModel {
	_, rm := e.GetModels()
	return rm
}

func (e *Entity) GetGroupModel() *GroupModel {
	gm, _ := e.GetModels()
	return gm
}

func GoToOurType(val any) string {
	switch reflect.ValueOf(val).Kind() {
	case reflect.Bool:
		return BOOLEAN
	case reflect.Int:
		return INTEGER
	case reflect.Interface:
		return ANY
	case reflect.Float64:
		return DECIMAL
	case reflect.String:
		return STRING
	case reflect.Uint64:
		return UINTEGER
	case reflect.Slice:
		return ARRAY
	case reflect.Map:
		return MAP
	case reflect.Struct:
		return OBJECT
	}
	panic(fmt.Sprintf("Bad Kind: %v", reflect.ValueOf(val).Kind()))
}

func (e *Entity) ToString() string {
	str := fmt.Sprintf("%s/%s\n  Object: %s\n  NewObject: %s",
		e.Singular, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	return str
}

// We use this just to make sure we can set NewObjectStack when we need to
// debug stuff
func (e *Entity) SetNewObject(newObj map[string]any) {
	e.NewObject = newObj

	// Enable the next line when we need to debug when NewObject was created
	// e.NewObjectStack = GetStack()

	// And then use e.ShowStack() to dump it

	/* Sample code to print the stack for where this NewObject was created:
	log.Printf("Stack for NewObject:")
	for _, s := range e.NewObjectStack {
		log.Printf("  %s", s)
	}
	*/
}

func (e *Entity) ShowStack() {
	log.Printf("Stack for NewObject (%s):", e.Path)
	for _, s := range e.NewObjectStack {
		log.Printf("  %s", s)
	}
}

func (e *Entity) Touch() {
	log.VPrintf(3, "Touch: %s/%s", e.Singular, e.UID)

	// See if it's already been modified (and saved) this Tx, if so exit
	if e.ModSet && e.EpochSet {
		return
	}

	e.EnsureNewObject()
}

func (e *Entity) EnsureNewObject() {
	if e.NewObject == nil {
		if e.Object == nil {
			e.SetNewObject(map[string]any{})
		} else {
			e.SetNewObject(maps.Clone(e.Object))
		}
	}
}

func (e *Entity) Get(path string) any {
	pp, err := PropPathFromUI(path)
	PanicIf(err != nil, fmt.Sprintf("%s", err))
	return e.GetPP(pp)
}

func (e *Entity) GetAsString(path string) string {
	val := e.Get(path)
	if IsNil(val) {
		return ""
	}

	if tmp := reflect.ValueOf(val).Kind(); tmp != reflect.String {
		panic(fmt.Sprintf("Not a string - got %T(%v)", val, val))
	}

	str, _ := val.(string)
	return str
}

func (e *Entity) GetAsInt(path string) int {
	val := e.Get(path)
	if IsNil(val) {
		return -1
	}
	i, ok := val.(int)
	PanicIf(!ok, fmt.Sprintf("Val: %v  T: %T", val, val))
	return i
}

func (e *Entity) GetPP(pp *PropPath) any {
	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			contentID := e.Get("#contentid")
			results, err := Query(e.tx, `
            SELECT Content FROM ResourceContents WHERE VersionSID=? `,
				contentID)
			defer results.Close()

			if err != nil {
				return fmt.Errorf("Error finding contents %q: %s", e.DbSID, err)
			}

			row := results.NextRow()
			if row == nil {
				// No data so just return
				return nil
			}

			if results.NextRow() != nil {
				panic("too many results")
			}

			return (*(row[0])).([]byte)
		}
	}

	// We used to just grab from Object, not NewObject
	/*
		// An error from ObjectGetProp is ignored because if they tried to
		// go into something incorrect/bad we should just return 'nil'.
		// This may not be the best choice in the long-run - which in case we
		// should return the 'error'
		val, _ , _ := ObjectGetProp(e.Object, pp)
	*/

	// See if we have an updated value in NewObject, if not grab from Object
	var val any
	if e.NewObject != nil {
		var ok bool
		val, ok, _ = ObjectGetProp(e.NewObject, pp)
		if !ok {
			// TODO: DUG - we should not need this
			// val, _, _ = ObjectGetProp(e.Object, pp)
		}
	} else {
		val, _, _ = ObjectGetProp(e.Object, pp)
	}

	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.UID, pp.DB(), val)
	return val
}

// Value, Found, Error
func ObjectGetProp(obj any, pp *PropPath) (any, bool, error) {
	return NestedGetProp(obj, pp, NewPP())
}

// Value, Found, Error
func NestedGetProp(obj any, pp *PropPath, prev *PropPath) (any, bool, error) {
	if log.GetVerbose() > 2 {
		log.VPrintf(0, "ObjectGetProp: %q\nobj:\n%s", pp.UI(), ToJSON(obj))
	}
	if pp == nil || pp.Len() == 0 {
		return obj, true, nil
	}
	if IsNil(obj) {
		return nil, false,
			fmt.Errorf("Can't traverse into nothing: %s", prev.UI())
	}

	objValue := reflect.ValueOf(obj)
	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		if objValue.Kind() != reflect.Slice {
			return nil, false,
				fmt.Errorf("Can't index into non-array: %s", prev.UI())
		}
		if index < 0 || index >= objValue.Len() {
			return nil, false,
				fmt.Errorf("Array reference %q out of bounds: "+
					"(max:%d-1)", prev.Append(pp.First()).UI(), objValue.Len())
		}
		objValue = objValue.Index(index)
		if objValue.IsValid() {
			obj = objValue.Interface()
		} else {
			panic("help") // Should never get here
			obj = nil
		}
		return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
	}

	// Is map/object
	if objValue.Kind() != reflect.Map {
		return nil, false, fmt.Errorf("Can't reference a non-map/object: %s",
			prev.UI())
	}
	if objValue.Type().Key().Kind() != reflect.String {
		return nil, false, fmt.Errorf("Key of %q must be a string, not %s",
			prev.UI(), objValue.Type().Key().Kind())
	}

	objValue = objValue.MapIndex(reflect.ValueOf(pp.Top()))
	if objValue.IsValid() {
		obj = objValue.Interface()
	} else {
		if pp.Next().Len() == 0 {
			return nil, false, nil
		}
		obj = nil
	}
	return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
}

func RawEntityFromPath(tx *Tx, regID string, path string, anyCase bool) (*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntityFromPath(%s)", path)
	defer log.VPrintf(3, "<Exit: RawEntityFromPath")

	// RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1     2      3       4     5    6       7         8       9    10

	caseExpr := ""
	if anyCase {
		caseExpr = " COLLATE utf8mb4_0900_ai_ci"
	}

	results, err := Query(tx, `
		SELECT
            e.RegSID as RegSID,
            e.Type as Type,
            e.Plural as Plural,
            e.Singular as Singular,
            e.eSID as eSID,
            e.UID as UID,
            p.PropName as PropName,
            p.PropValue as PropValue,
            p.PropType as PropType,
            e.Path as Path,
            e.Abstract as Abstract
        FROM Entities AS e
        LEFT JOIN Props AS p ON (e.eSID=p.EntitySID)
        WHERE e.RegSID=? AND e.Path`+caseExpr+`=? ORDER BY Path`,
		regID, path)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	return readNextEntity(tx, results)
}

func (e *Entity) Query(query string, args ...any) ([][]any, error) {
	results, err := Query(e.tx, query, args...)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	data := ([][]any)(nil)
	/*
		Ks := make([]string, len(results.colTypes))

		for i, t := range results.colTypes {
			Ks[i] = t.Kind().String()
		}
	*/

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if data == nil {
			data = [][]any{}
		}
		// row == []*any
		r := make([]any, len(row))
		for i, d := range row {
			r[i] = d
			/*
				k := Ks[i]
				if k == "slice" {
					r[i] = NotNilString(d)
				} else if k == "int64" || k == "uint64" {
					r[i] = NotNilInt(d)
				} else {
					log.Printf("%v", reflect.ValueOf(*d).Type().String())
					log.Printf("%v", reflect.ValueOf(*d).Type().Kind().String())
					log.Printf("Ks: %v", Ks)
					log.Printf("i: %d", i)
					panic("help")
				}
			*/
		}
		data = append(data, r)
	}

	return data, nil
}

func RawEntitiesFromQuery(tx *Tx, regID string, query string, args ...any) ([]*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntititiesFromQuery(%s)", query)
	defer log.VPrintf(3, "<Exit: RawEntitiesFromQuery")

	// RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1     2     3        4    5     6         7        8     9     10

	if query != "" {
		query = "AND (" + query + ") "
	}
	args = append(append([]any{}, regID), args...)
	results, err := Query(tx, `
		SELECT
            e.RegSID as RegSID,
            e.Type as Type,
            e.Plural as Plural,
            e.Singular as Singular,
            e.eSID as eSID,
            e.UID as UID,
            p.PropName as PropName,
            p.PropValue as PropValue,
            p.PropType as PropType,
            e.Path as Path,
            e.Abstract as Abstract
        FROM Entities AS e
        LEFT JOIN Props AS p ON (e.eSID=p.EntitySID)
        WHERE e.RegSID=? `+query+` ORDER BY Path`, args...)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	entities := []*Entity{}
	for {
		e, err := readNextEntity(tx, results)
		if err != nil {
			return nil, err
		}
		if e == nil {
			break
		}
		entities = append(entities, e)
	}

	return entities, nil
}

// Update the entity's Object - not the other props in Entity. Similar to
// RawEntityFromPath
func (e *Entity) Refresh() error {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbSID)
	defer log.VPrintf(3, "<Exit: Refresh")

	results, err := Query(e.tx, `
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntitySID=? `, e.DbSID)
	defer results.Close()

	if err != nil {
		return fmt.Errorf("Error refreshing props(%s): %s", e.DbSID, err)
	}

	// Erase all old props first
	e.Object = map[string]any{}
	e.NewObject = nil

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])

		if err = e.SetFromDBName(name, &val, propType); err != nil {
			return err
		}
	}

	// TODO see if we can remove this - it scares me.
	// Added when I added Touch() - touching parent on add/remove child
	e.EpochSet = false
	e.ModSet = false

	e.tx.AddToCache(e)

	return nil
}

// All in one: Set, Validate, Save to DB and Commit (or Rollback on error)
// Should never be used because the act of committing should be done
// by the caller once all of the changes are done. This is a holdover from
// before we had transaction support - once we're sure, delete it
func (e *Entity) eSetCommit(path string, val any) error {
	log.VPrintf(3, ">Enter: SetCommit(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	err := e.eSetSave(path, val)
	Must(e.tx.Conditional(err))

	return err
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) eSetSave(path string, val any) error {
	log.VPrintf(3, ">Enter: SetSave(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	pp, err := PropPathFromUI(path)
	if err == nil {
		// Set, Validate and Save
		err = e.SetPP(pp, val)
	}

	return err
}

// Set the prop in the Entity but don't Validate or Save to the DB
func (e *Entity) eJustSet(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: JustSet([%d] %s.%s=%v)", e.Type, e.UID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit: JustSet")

	// Assume no other edits are pending
	// e.Refresh() // trying not to have this here

	// If we don't have a NewObject yet then this is our first update
	// so clone the current values before adding the new prop/val
	e.EnsureNewObject()

	// Cheat a little just to make caller's life easier by converting
	// empty structs and maps need to be of the type we like (meaning 'any's)
	if !IsNil(val) {
		if val == struct{}{} {
			val = map[string]any{}
		}
		valValue := reflect.ValueOf(val)
		if valValue.Kind() == reflect.Slice && valValue.Len() == 0 {
			val = []any{}
		}
		if valValue.Kind() == reflect.Map && valValue.Len() == 0 {
			val = map[string]any{}
		}
	}
	// end of cheat

	if pp.Top() == "epoch" {
		e.EpochSet = true
	}
	if pp.Top() == "modifiedat" {
		e.ModSet = true
	}

	// Since "xref" is also a Property on the Resources table we need to
	// set it manually. We can't do it lower down (closer to the DB funcs)
	// because down there "xref" won't appear in NewObject when it's set to nil
	/*
		if e.Type == ENTITY_RESOURCE && pp.Top() == "xref" {
			// Handles both val=nil and non-nil cases
			err := DoOneTwo(e.tx, `UPDATE Resources SET xRef=? WHERE SID=?`,
				val, e.DbSID)
			if err != nil {
				return err
			}
		}
	*/

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Abstract/ID: %s/%s", e.Abstract, e.UID)
		log.VPrintf(0, "e.Object:\n%s", ToJSON(e.Object))
		log.VPrintf(0, "e.NewObject:\n%s", ToJSON(e.NewObject))
	}

	return ObjectSetProp(e.NewObject, pp, val)
}

func (e *Entity) ValidateAndSave() error {
	log.VPrintf(3, ">Enter: ValidateAndSave %s/%s", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: ValidateAndSave")

	// If nothing changed, then exit
	if e.NewObject == nil {
		return nil
	}

	// Make sure we have a tx since Validate assumes it
	e.tx.NewTx()

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Validating %s/%s\ne.Object:\n%s\n\ne.NewObject:\n%s",
			e.Abstract, e.UID, ToJSON(e.Object), ToJSON(e.NewObject))
	}

	if err := e.Validate(); err != nil {
		return err
	}

	return e.Save()
}

// This is really just an internal Setter used for testing.
// It'll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetPP(%s: %s=%v)", e.DbSID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")
	defer func() {
		if log.GetVerbose() > 2 {
			log.VPrintf(0, "SetPP exit: e.Object:\n%s", ToJSON(e.Object))
		}
	}()

	if err := e.eJustSet(pp, val); err != nil {
		return err
	}

	err := e.ValidateAndSave()
	if err != nil {
		// If there's an error, and we're making the assumption that we're
		// setting and saving all in one shot (and there are no other edits
		// pending), go ahead and undo the changes since they're wrong.
		// Otherwise the caller would need to call Refresh themselves.

		// Not sure why setting it to nil isn't sufficient (todo)
		// e.NewObject = nil
		e.Refresh()
	}

	return err
}

// This will save a single property/value in the DB. This assumes
// the caller is traversing the Object and splitting it into individual props
func (e *Entity) SetDBProperty(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetDBProperty(%s=%v)", pp, val)
	defer log.VPrintf(3, "<Exit SetDBProperty")

	PanicIf(pp.UI() == "", "pp is empty")

	var err error
	name := pp.DB()

	// Any prop with "dontStore"=true we skip
	specPropName := pp.Top()
	if specPropName == e.Singular+"id" {
		specPropName = "id"
	}

	specProp, ok := SpecProps[specPropName]
	if ok && specProp.internals.dontStore {
		return nil
	}

	PanicIf(e.DbSID == "", "DbSID should not be empty")
	PanicIf(e.Registry == nil, "Registry should not be nil")

	// "RESOURCE" is special and is saved in it's own table
	// Need to explicitly set "RESOURCE" to nil to delete it.
	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			if IsNil(val) {
				// Remove the content
				err = Do(e.tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
					e.DbSID)
				return err
			} else {
				// Update the content
				err = DoOneTwo(e.tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)
				if err != nil {
					return err
				}

				PanicIf(IsNil(e.NewObject["#contentid"]), "Missing cid")

				// Don't save "RESOURCE" in the DB, #contentid is good enough
				return nil
			}
		}
	}

	// Convert specDefined BOOLEAN value "false" to "nil" so it doesn't
	// appear in the DB at all. If this is too broad then just do it for
	// "defaultversionsticky" in resources.go as we're copying attributes.
	if !IsNil(specProp) && val == false && GoToOurType(val) == BOOLEAN {
		// val = nil
	}

	if IsNil(val) {
		// Should never need this but keeping it just in case
		err = Do(e.tx, `DELETE FROM Props WHERE EntitySID=? and PropName=?`,
			e.DbSID, name)
	} else {
		propType := GoToOurType(val)

		// Convert booleans to true/false instead of 1/0 so filter works
		// ...=true and not ...=1
		dbVal := val
		if propType == BOOLEAN {
			if val == true {
				dbVal = "true"
			} else {
				dbVal = "false"
			}
		}

		switch reflect.ValueOf(val).Kind() {
		case reflect.String:
			if reflect.ValueOf(val).Len() > MAX_VARCHAR {
				return fmt.Errorf("Value must be less that %d chars",
					MAX_VARCHAR+1)
			}
		case reflect.Slice:
			if reflect.ValueOf(val).Len() > 0 {
				return fmt.Errorf("Can't set non-empty arrays")
			}
			dbVal = ""
		case reflect.Map:
			if reflect.ValueOf(val).Len() > 0 {
				return fmt.Errorf("Can't set non-empty maps")
			}
			dbVal = ""
		case reflect.Struct:
			if reflect.ValueOf(val).NumField() > 0 {
				return fmt.Errorf("Can't set non-empty objects")
			}
			dbVal = ""
		}

		err = DoOneTwo(e.tx, `
            REPLACE INTO Props(
              RegistrySID,EntitySID,eType,PropName,PropValue,PropType,DocView)
            VALUES( ?,?,?,?,?,?, true )`,
			e.Registry.DbSID, e.DbSID, e.Type, name, dbVal, propType)
	}

	if err != nil {
		return fmt.Errorf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
	}

	return nil
}

// This is used to take a DB entry and update the current Entity's Object
func (e *Entity) SetFromDBName(name string, val *string, propType string) error {
	pp := MustPropPathFromDB(name)

	if val == nil {
		return ObjectSetProp(e.Object, pp, val)
	}
	if e.Object == nil {
		e.Object = map[string]any{}
	}

	if propType == STRING || propType == URI || propType == URI_REFERENCE ||
		propType == URI_TEMPLATE || propType == URL || propType == TIMESTAMP {
		return ObjectSetProp(e.Object, pp, *val)
	} else if propType == BOOLEAN {
		// Technically the "1" check shouldn't be needed, but just in case
		return ObjectSetProp(e.Object, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s", *val))
		}
		return ObjectSetProp(e.Object, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s", *val))
		}
		return ObjectSetProp(e.Object, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}
}

// Create a new Entity based on what's in the DB. Similar to Refresh()
func readNextEntity(tx *Tx, results *Result) (*Entity, error) {
	entity := (*Entity)(nil)

	// RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1     2     3        4     5   6         7        8       9    10
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		// log.Printf("Row(%d): %#v", len(row), row)
		if log.GetVerbose() >= 4 {
			str := "("
			for _, c := range row {
				if IsNil(c) || IsNil(*c) {
					str += "nil,"
				} else {
					str += fmt.Sprintf("%s,", *c)
				}
			}
			log.Printf("Row: %s)", str)
		}
		eType := int((*row[1]).(int64))
		plural := NotNilString(row[2])
		uid := NotNilString(row[5])

		if entity == nil {
			entity = &Entity{
				tx: tx,

				Registry: tx.Registry,
				DbSID:    NotNilString(row[4]),
				Plural:   plural,
				Singular: NotNilString(row[3]),
				UID:      uid,

				Type:     eType,
				Path:     NotNilString(row[9]),
				Abstract: NotNilString(row[10]),
			}
		} else {
			// If the next row isn't part of the current Entity then
			// push it back into the result set so we'll grab it the next time
			// we're called. And exit.
			if entity.Type != eType || entity.Plural != plural || entity.UID != uid {
				results.Push()
				break
			}
		}

		propName := NotNilString(row[6])
		propVal := NotNilString(row[7])
		propType := NotNilString(row[8])

		// Edge case - no props but entity is there
		if propName == "" && propVal == "" && propType == "" {
			continue
		}

		if err := entity.SetFromDBName(propName, &propVal, propType); err != nil {
			return nil, err
		}
	}

	return entity, nil
}

func CalcSpecProps(eType int, singular string) ([]*Attribute, map[string]*Attribute) {
	// Use eType < 0 to not check the type of the prop
	// Use singular = "" to not twiddle any "singular" specific logic
	resMap := map[string]*Attribute{}
	resArr := []*Attribute{}

	for _, prop := range OrderedSpecProps {
		if eType >= 0 && !prop.InType(eType) {
			continue
		}

		var copyProp *Attribute

		// If needed, duplicate it.
		// If Attribute is ever more than scalars we'll need to deep copy
		if copyProp.Name == "id" {
			tmp := *prop
			copyProp = &tmp
			copyProp.Name = singular + "id"
		}

		resArr = append(resArr, copyProp)
		resMap[copyProp.Name] = copyProp
	}

	return resArr, resMap
}

func StrTypes(types ...int) string {
	res := strings.Builder{}
	for _, eType := range types {
		res.WriteByte('0' + byte(eType))
	}
	return res.String()
}

// This allows for us to choose the order and define custom logic per prop
var OrderedSpecProps = []*Attribute{
	{
		Name:      "specversion",
		Type:      STRING,
		ReadOnly:  true,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY),
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				return SPECVERSION
			},
			checkFn: func(e *Entity) error {
				tmp := e.NewObject["specversion"]
				if !IsNil(tmp) && tmp != "" && tmp != SPECVERSION {
					return fmt.Errorf("Invalid 'specversion': %s", tmp)
				}
				return nil
			},
			updateFn: nil,
		},
	},
	{
		Name:      "id",
		Type:      STRING,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:        "", // Yes even ENTITY_RESOURCE
			dontStore:    false,
			xrefrequired: true,
			getFn:        nil,
			checkFn: func(e *Entity) error {
				singular := e.Singular
				// PanicIf(singular == "", "singular is '' :  %v", e)
				if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
					_, rm := e.GetModels()
					singular = rm.Singular
				}
				singular += "id"

				oldID := any(e.UID)
				if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
					// Grab rID from /GROUPs/gID/RESOURCEs/rID/versions/vID
					parts := strings.Split(e.Path, "/")
					oldID = parts[3]
				}
				newID := any(e.NewObject[singular])

				if IsNil(newID) {
					return nil // Not trying to be updated, so skip it
				}

				if newID == "" {
					return fmt.Errorf(`%q can't be an empty string`,
						singular)
				}

				if err := IsValidID(newID.(string)); err != nil {
					return err
				}

				if oldID != "" && !IsNil(oldID) && newID != oldID {
					return fmt.Errorf(`The %q attribute must be set to `+
						`%q, not %q`, singular, oldID, newID)
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				// Make sure the ID is always set
				singular := e.Singular
				if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
					singular = e.GetResourceSingular()
				}
				singular += "id"

				if e.Type == ENTITY_VERSION {
					// Versions shouldn't store the RESOURCEid
					delete(e.NewObject, singular)
				} else if IsNil(e.NewObject[singular]) {
					panic(fmt.Sprintf(`%q is nil - that's bad, fix it!`,
						singular))
				}
				return nil
			},
		},
	},
	{
		Name:      "versionid",
		Type:      STRING,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn: func(e *Entity) error {
				oldID := any(e.UID)
				newID := any(e.NewObject["versionid"])

				if IsNil(newID) {
					return nil // Not trying to be updated, so skip it
				}

				if newID == "" {
					return fmt.Errorf(`"versionid" can't be an empty string`)
				}

				if err := IsValidID(newID.(string)); err != nil {
					return err
				}

				if oldID != "" && !IsNil(oldID) && newID != oldID {
					return fmt.Errorf(`The "versionid" attribute must be `+
						`set to %q, not %q`, oldID, newID)
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				// Make sure the ID is always set
				if IsNil(e.NewObject["versionid"]) {
					ShowStack()
					panic(fmt.Sprintf(`"versionid" is nil - fix it!`))
				}
				return nil
			},
		},
	},
	{
		Name:      "self",
		Type:      URL,
		ReadOnly:  true,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:        "", // Yes even ENTITY_RESOURCE
			dontStore:    true,
			xrefrequired: true,
			getFn: func(e *Entity, info *RequestInfo) any {
				base := ""
				path := e.Path
				isAbs := false

				if info != nil {
					if info.DoDocView() {
						// remove GET's base path
						path = path[len(info.Root):]
						if strings.HasPrefix(path, "/") {
							path = path[1:]
						}
						base = DOCVIEW_BASE
					} else {
						isAbs = true
						base = info.BaseURL
					}
				}

				if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
					details := info != nil && (info.ShowDetails ||
						info.ResourceUID == "" || len(info.Parts) == 5)

					if (info != nil && info.DoDocView() && !isAbs) ||
						e.GetResourceModel().GetHasDocument() == false {
						details = false
					}

					if details {
						path += "$details"
					}
				}
				return base + "/" + path
			},
			checkFn:  nil,
			updateFn: nil,
		},
	},
	/*
		{
			Name:           "shortself",
			Type:           URL,
			ReadOnly:       true,
			Immutable: true,
			Required: true,

			internals: AttrInternals{
				types:     "",
				dontStore: true,
				xrefrequired: true,
				getFn: func(e *Entity, info *RequestInfo) any {
					path := e.Path
					base := ""
					if info != nil {
						base = info.BaseURL
					}

					if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
						meta := info != nil && (info.ShowDetails ||
						info.DoDocView() ||
						info.ResourceUID == "" || len(info.Parts) == 5)

						if e.GetResourceModel().GetHasDocument() == false {
							meta = false
						}

						if meta {
							path += "$details"
						}
					}

					shortself := MD5(path)
					return base + "/r?u=" + shortself
				},
				checkFn:  nil,
				updateFn: nil,
			},
		},
	*/
	{
		Name:      "xid",
		Type:      XID,
		ReadOnly:  true,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:        "",
			dontStore:    true,
			xrefrequired: true,
			getFn:        nil,
			checkFn:      nil,
		},
	},
	{
		Name: "xref",
		Type: URL,

		internals: AttrInternals{
			types: StrTypes(ENTITY_META),
			getFn: nil,
			checkFn: func(e *Entity) error {
				return nil
			},
			updateFn: func(e *Entity) error {
				return nil
			},
		},
	},
	{
		Name:     "epoch",
		Type:     UINTEGER,
		Required: true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn: func(e *Entity) error {
				// If we explicitly setEpoch via internal API then don't check
				if e.EpochSet {
					return nil
				}

				val := e.NewObject["epoch"]
				if IsNil(val) {
					return nil
				}

				tmp := e.Object["epoch"]
				oldEpoch := NotNilInt(&tmp)
				if oldEpoch < 0 {
					oldEpoch = 0
				}

				newEpoch, err := AnyToUInt(val)
				if err != nil {
					return fmt.Errorf("Attribute \"epoch\" must be a uinteger")
				}

				if !e.tx.IgnoreEpoch && oldEpoch != 0 && newEpoch != oldEpoch {
					return fmt.Errorf("Attribute %q(%d) doesn't match "+
						"existing value (%d)", "epoch", newEpoch, oldEpoch)
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				// Very special, if we're in meta and xref set then
				// erase 'epoch'. We can't do it earlier because we need
				// the checkFn to be run to make sure any incoming value
				// was valid
				if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
					e.NewObject["epoch"] = nil
					return nil
				}

				// If we already set Epoch in this Tx, just exit
				if e.EpochSet {
					// If we already set epoch this tx but there's no value
					// then grab it from Object, otherwise we'll be missing a
					// value during Save(). This can happen when we Save()
					// more than once on this Entity during the same Tx and
					// the 2nd Save() didn't have epoch as part of the incoming
					// Object
					if IsNil(e.NewObject["epoch"]) {
						e.NewObject["epoch"] = e.Object["epoch"]
					}
					return nil
				}

				// This assumes that ALL entities must have an Epoch property
				// that we want to set. At one point this wasn't true for
				// Resources but hopefully that's no longer true

				oldEpoch := e.Object["epoch"]
				epoch := NotNilInt(&oldEpoch)

				e.NewObject["epoch"] = epoch + 1
				e.EpochSet = true
				return nil
			},
		},
	},
	{
		Name: "name",
		Type: STRING,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:     "isdefault",
		Type:     BOOLEAN,
		ReadOnly: true,
		Required: true,
		Default:  false,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_VERSION),
			dontStore: true,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				// TODO if set, set defaultversionid in the resource to this
				// guy's UID

				return nil
			},
		},
	},
	{
		Name: "description",
		Type: STRING,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "documentation",
		Type: URL,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "labels",
		Type: MAP,
		Item: &Item{
			Type: STRING,
		},

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:     "createdat",
		Type:     TIMESTAMP,
		Required: true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
					e.NewObject["createdat"] = nil

					// If for some reason there is no saved createTime
					// assume this is a new meta so save 'now'
					if IsNil(e.NewObject["#createdat"]) {
						e.NewObject["#createdat"] = e.tx.CreateTime
					}
					return nil
				}

				ca, ok := e.NewObject["createdat"]
				// If not there use the existing value, if present
				if !ok {
					ca = e.Object["createdat"]
					e.NewObject["createdat"] = ca
				}
				// Still no value, so use "now"
				if IsNil(ca) {
					ca = e.tx.CreateTime
				}

				var err error
				t := ""
				if t, err = NormalizeStrTime(ca.(string)); err != nil {
					return err
				}
				e.NewObject["createdat"] = t

				return nil
			},
		},
	},
	{
		Name:     "modifiedat",
		Type:     TIMESTAMP,
		Required: true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY, ENTITY_GROUP, ENTITY_META, ENTITY_VERSION),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				if e.Type == ENTITY_META && e.GetAsString("xref") != "" {
					e.NewObject["modifiedat"] = nil
					return nil
				}

				ma := e.NewObject["modifiedat"]

				// If we already set modifiedat in this Tx, just exit
				if e.ModSet && !IsNil(ma) && ma != "" {
					return nil
				}

				// If there's no value, or it's the same as the existing
				// value, set to "now"
				if IsNil(ma) || (ma == e.Object["modifiedat"]) {
					ma = e.tx.CreateTime
				}

				var err error
				t := ""
				if t, err = NormalizeStrTime(ma.(string)); err != nil {
					return err
				}
				e.NewObject["modifiedat"] = t
				e.ModSet = true

				return nil
			},
		},
	},
	{
		Name:     "readonly",
		Type:     BOOLEAN,
		Required: true,
		Default:  false,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_META),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "compatibility",
		Type: STRING,
		Enum: []any{"none", "backward", "backward_transitive", "forward",
			"forward_transitive", "full", "full_transitive"},
		Strict:   PtrBool(false),
		Required: true,
		Default:  "none",

		internals: AttrInternals{
			types:     StrTypes(ENTITY_META),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name:     "ancestor",
		Type:     STRING,
		Required: true,

		internals: AttrInternals{
			types:          StrTypes(ENTITY_VERSION),
			dontStore:      false,
			neverSerialize: true,
			getFn:          nil,
			checkFn:        nil,
			updateFn: func(e *Entity) error {
				_, ok := e.NewObject["ancestor"]
				if !ok {
					v, ok := e.NewObject["versionid"]
					PanicIf(!ok, "Missing versionid")
					e.NewObject["ancestor"] = v
					// Just assign a placeholder to get past validation.
					// ValidateResources() should fix this before we commit
					// the tx
					// e.NewObject["ancestor"] = "$SET_ME$"
				}
				return nil
			},
		},
	},
	{
		Name: "contenttype",
		Type: STRING,

		internals: AttrInternals{
			types:      StrTypes(ENTITY_VERSION),
			dontStore:  false,
			httpHeader: "Content-Type",
			getFn:      nil,
			checkFn:    nil,
			updateFn:   nil,
		},
	},
	{
		Name: "$extensions",
		internals: AttrInternals{
			types: "",
		},
	},
	{
		Name: "$space",
		internals: AttrInternals{
			types: "",
		},
	},
	// For the $RESOURCE ones, make sure to use attr.Clone("newname")
	// when the $RESOURCE is substituded with the Resource's singular
	// name. Otherwise you'll be updating this shared entry.
	{
		Name: "$RESOURCEurl",
		Type: URL,
		internals: AttrInternals{
			types:   StrTypes(ENTITY_VERSION),
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) error {
				singular := e.GetResourceSingular()
				v, ok := e.NewObject[singular+"url"]
				if ok && !IsNil(v) {
					e.NewObject[singular] = nil
					e.NewObject[singular+"proxyurl"] = nil
					e.NewObject["#contentid"] = nil
				}
				return nil
			},
		},
	},
	{
		Name: "$RESOURCEproxyurl",
		Type: URL,
		internals: AttrInternals{
			types:   StrTypes(ENTITY_VERSION),
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) error {
				singular := e.GetResourceSingular()
				v, ok := e.NewObject[singular+"proxyurl"]
				if ok && !IsNil(v) {
					e.NewObject[singular] = nil
					e.NewObject[singular+"url"] = nil
					e.NewObject["#contentid"] = nil
				}
				return nil
			},
		},
	},
	{
		Name: "$RESOURCE",
		Type: ANY,

		internals: AttrInternals{
			types:           StrTypes(ENTITY_VERSION),
			alwaysSerialize: true, // Will always be missing, so need this
			checkFn:         RESOURCEcheckFn,
			updateFn: func(e *Entity) error {
				singular := e.GetResourceSingular()
				v, ok := e.NewObject[singular]
				if ok {
					if !IsNil(v) {
						e.NewObject[singular+"url"] = nil
						e.NewObject[singular+"proxyurl"] = nil
						e.NewObject["#contentid"] = e.DbSID
					} else {
						e.NewObject["#contentid"] = nil
					}
				}
				return nil
			},
		},
	},
	{
		Name:      "metaurl",
		Type:      URL,
		ReadOnly:  true,
		Immutable: true,
		Required:  true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_RESOURCE),
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				base := ""
				path := e.Path

				if info != nil {
					inlineMeta := info.ShouldInline(e.Abstract +
						string(DB_IN) + "meta")

					if !info.DoDocView() || !inlineMeta {
						base = info.BaseURL
					} else {
						base = DOCVIEW_BASE

						// remove GET's base path
						path = path[len(info.Root):]
						if strings.HasPrefix(path, "/") {
							path = path[1:]
						}
					}
				}
				if path != "" {
					path = "/" + path
				}

				return base + path + "/meta"
			},
			checkFn:  nil,
			updateFn: nil,
		},
	},
	{
		Name: "$space",
		internals: AttrInternals{
			types: "",
		},
	},
	{
		Name: "defaultversionid",
		Type: STRING,
		// ReadOnly: true,
		Required: true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_META),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn: func(e *Entity) error {
				// TODO really should call Resource.EnsureLatest here

				// Make sure it has a value, if not copy from existing
				xRef := e.NewObject["xref"]
				PanicIf(xRef == "", "xref is ''")

				/* Really should check this
				newVal := e.NewObject["defaultversionid"]
				PanicIf(IsNil(xRef) && IsNil(newVal), "defverid is nil")
				*/

				/*
					if IsNil(xRef) && IsNil(newVal) {
						oldVal := e.Object["defaultversionid"]
						e.NewObject["defaultversionid"] = oldVal
					}
				*/
				return nil
			},
		},
	},
	{
		Name:     "defaultversionurl",
		Type:     URL,
		ReadOnly: true,
		Required: true,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_META),
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				val := e.Object["defaultversionid"]
				if IsNil(val) {
					return nil
				}
				valStr := val.(string)

				// place "meta" with "versions/VID"
				path := e.Path[:len(e.Path)-4] + "versions/" + valStr
				result := ""
				isAbsURL := false

				if info != nil {
					// s/meta/versions/
					abs := e.Abstract[:len(e.Abstract)-4] + "versions"
					inlineVers := info.ShouldInline(abs)
					seenDefVid := info.extras["seenDefaultVid"]

					if len(info.Parts) == 5 { // pointing directly to /meta
						isAbsURL = true
					}

					if !info.DoDocView() {
						isAbsURL = true
					}

					if !inlineVers {
						isAbsURL = true
					}

					if len(info.Filters) > 0 && seenDefVid != valStr {
						isAbsURL = true
					}

					if isAbsURL {
						result = info.BaseURL
					} else {
						if info.DoDocView() {
							result = DOCVIEW_BASE

							// remove GET's base path
							path = path[len(info.Root):]
							if strings.HasPrefix(path, "/") {
								path = path[1:]
							}
						}
					}
				}

				// remove "/meta" so we can add "/versions/vID"
				result += "/" + path

				return result
			},
			checkFn:  nil,
			updateFn: nil,
		},
	},
	{
		Name:     "defaultversionsticky",
		Type:     BOOLEAN,
		ReadOnly: true,
		Required: true,
		Default:  false,

		internals: AttrInternals{
			types:     StrTypes(ENTITY_META),
			dontStore: false,
			getFn:     nil,
			checkFn:   nil,
			updateFn:  nil,
		},
	},
	{
		Name: "$space",
		internals: AttrInternals{
			types: "",
		},
	},
	{
		Name:     "capabilities",
		Type:     OBJECT, // This ensures the client sent a map
		ReadOnly: false,
		Attributes: Attributes{
			"*": &Attribute{
				Name: "*",
				Type: ANY,
			},
		},

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY),
			dontStore: true,
			getFn: func(e *Entity, info *RequestInfo) any {
				// Need to explicitly ask for "capabilities", ?inline=* won't
				// do it
				if info != nil && info.ShouldInline(NewPPP("capabilities").DB()) {
					capStr := e.GetAsString("#capabilities")
					if capStr == "" {
						return e.Registry.Capabilities
					}

					cap, err := ParseCapabilitiesJSON([]byte(capStr))
					Must(err)
					return cap
				}
				return nil
			},
			checkFn: func(e *Entity) error {
				// Yes it's weird to store it in #capabilities but
				// it's actually easier to do it this way. Trying to covert
				// map[string]any <-> Capabilities  is really annoying
				val := e.NewObject["capabilities"]
				if !IsNil(val) {
					// If speed is ever a concern here, just save the raw
					// json from the input stream instead from http processing
					valStr := ToJSON(val)

					cap, err := ParseCapabilitiesJSON([]byte(valStr))
					if err != nil {
						return err
					}

					if err = cap.Validate(); err != nil {
						return err
					}

					valStr = ToJSON(cap)

					e.NewObject["#capabilities"] = valStr
					delete(e.NewObject, "capabilities")
					e.Registry.Capabilities = cap
				}
				return nil
			},
			updateFn: func(e *Entity) error {
				return nil
			},
		},
	},
	{
		Name:     "model",
		Type:     OBJECT,
		ReadOnly: true,
		Attributes: Attributes{
			"*": &Attribute{
				Name: "*",
				Type: ANY,
			},
		},

		internals: AttrInternals{
			types:     StrTypes(ENTITY_REGISTRY),
			dontStore: false,
			getFn: func(e *Entity, info *RequestInfo) any {
				// Need to explicitly ask for "model", ?inline=* won't
				// do it
				if info != nil && info.ShouldInline(NewPPP("model").DB()) {
					model := info.Registry.Model
					if model == nil {
						model = &Model{}
					}
					httpModel := model // ModelToHTTPModel(model)
					return httpModel
				}
				return nil
			},
			checkFn:  nil,
			updateFn: nil,
		},
	},
}

var SpecProps = map[string]*Attribute{}

func init() {
	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[sp.Name] = sp
	}
}

// This is used to serialize an Entity regardless of the format.
// This will:
//   - Use AddCalcProps() to fill in any missing props (eg Entity's getFn())
//   - Call that passed-in 'fn' to serialize each prop but in the right order
//     as defined by OrderedSpecProps
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any, *Attribute) error) error {
	log.VPrintf(3, ">Enter: SerializeProps(%s/%s)", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: SerializeProps")

	daObj := e.AddCalcProps(info)
	attrs := e.GetAttributes(e.Object)

	if log.GetVerbose() > 3 {
		log.VPrintf(0, "SerProps.Entity: %s", ToJSON(e))
		log.VPrintf(0, "SerProps.Obj: %s", ToJSON(e.Object))
		log.VPrintf(0, "SerProps daObj: %s", ToJSON(daObj))
		log.VPrintf(0, "SerProps attrs:\n%s", ToJSON(attrs))
	}

	resourceSingular := ""
	hasDoc := false
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
		_, rm := e.GetModels()
		resourceSingular = rm.Singular
		hasDoc = rm.GetHasDocument()
	}

	// Do spec defined props first, in order
	// TODO we really should filter the OrderSpecProps based on
	// the level and whether hasDoc is true or not
	for _, prop := range OrderedSpecProps {
		name := prop.Name
		if name == "id" {
			if e.Type != ENTITY_VERSION && e.Type != ENTITY_META {
				// versionid has it's own special entry
				name = e.Singular + "id"
			} else {
				// for versions, id=RESOURCEid
				name = resourceSingular + "id"
			}
		}

		if hasDoc && strings.HasPrefix(name, "$RESOURCE") {
			name = resourceSingular + name[9:]
		}

		log.VPrintf(4, "Ser prop: %q", name)

		attr, ok := attrs[name]
		if !ok {
			log.VPrintf(4, "  skipping %q, no attr", name)
			delete(daObj, name)
			continue // not allowed at this eType so skip it
		}

		if prop.Name == "$extensions" {
			for _, objKey := range SortedKeys(daObj) {
				attrKey := objKey
				if attrKey == e.Singular+"id" {
					attrKey = "id"
				}

				// Skip spec defined properties, assume we'll add them later
				if SpecProps[attrKey] != nil {
					continue
				}

				val, _ := daObj[objKey]
				attr := attrs[attrKey]
				delete(daObj, objKey)
				if attr == nil {
					attr = attrs["*"]
					PanicIf(attrKey[0] != '#' && attr == nil,
						"Can't find attr for %q", attrKey)
				}

				if err := fn(e, info, objKey, val, attr); err != nil {
					return err
				}
			}
			continue
		}

		if name[0] == '$' || prop.internals.alwaysSerialize {
			log.VPrintf(4, "  forced serialization of %q", name)
			if err := fn(e, info, name, nil, attr); err != nil {
				return err
			}
			continue
		}

		// Should be a no-op for Resources.
		if val, ok := daObj[name]; ok {
			log.VPrintf(4, "  val: %v", val)
			if !IsNil(val) {
				cleanup := false
				var m *Model

				t := reflect.ValueOf(val).Type()
				if t.Kind() == reflect.Pointer &&
					t.Elem().String() == "registry.Model" {

					m, ok = val.(*Model)
					PanicIf(!ok, "Not a model")
					m.SetSingular()
					cleanup = true
				}

				err := fn(e, info, name, val, attr)
				if cleanup {
					m.UnsetSingular()
				}
				if err != nil {
					return err
				}
			}
			delete(daObj, name)
		} else {
			log.VPrintf(4, "  no value for %q", name)
		}
	}

	// Now do all other props (extensions) alphabetically
	for _, objKey := range SortedKeys(daObj) {
		attrKey := objKey
		if attrKey == e.Singular+"id" {
			attrKey = "id"
		}
		val, _ := daObj[objKey]
		attr := attrs[attrKey]
		if attr == nil {
			attr = attrs["*"]
			PanicIf(attrKey[0] != '#' && attr == nil,
				"Can't find attr for %q", attrKey)
		}

		if err := fn(e, info, objKey, val, attr); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) Save() error {
	log.VPrintf(3, ">Enter: Save(%s/%s)", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: Save")

	// TODO remove at some point when we're sure it's safe
	if SpecProps["epoch"].InType(e.Type) && IsNil(e.NewObject["epoch"]) {
		// Only an xref'd "meta" is allowed to not have an 'epoch'
		if e.Type != ENTITY_META || IsNil(e.NewObject["xref"]) {
			PanicIf(true, "Epoch is nil(%s):%s", e.Path, ToJSON(e.NewObject))
		}
	}

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Saving - %s (id:%s):\n%s\n", e.Abstract, e.UID,
			ToJSON(e.NewObject))
	}

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	e.RemoveCollections(newObj)

	// Delete all props for this entity, we assume that NewObject
	// contains everything we want going forward
	err := Do(e.tx, "DELETE FROM Props WHERE EntitySID=? ", e.DbSID)
	if err != nil {
		return fmt.Errorf("Error deleting all prop: %s", err)
	}

	resSingular := ""
	resHasDoc := false
	if rm := e.GetResourceModel(); rm != nil {
		resSingular = rm.Singular
		resHasDoc = rm.GetHasDocument()
	}

	var traverse func(pp *PropPath, val any, obj map[string]any) error
	traverse = func(pp *PropPath, val any, obj map[string]any) error {
		if IsNil(val) { // Skip empty attributes
			return nil
		}

		switch reflect.ValueOf(val).Kind() {
		case reflect.Map:
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				// "RESOURCE" is special - call SetDBProp if it's present
				if resHasDoc && pp.Len() == 0 && k == resSingular {
					if err := e.SetDBProperty(pp.P(k), v); err != nil {
						return err
					}
				} else if k[0] == '#' {
					if err := e.SetDBProperty(pp.P(k), v); err != nil {
						return err
					}
				} else {
					if IsNil(v) {
						continue
					}
					if err := traverse(pp.P(k), v, obj); err != nil {
						return err
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return e.SetDBProperty(pp, map[string]any{})
			}

		case reflect.Slice:
			vArray := val.([]any)
			if len(vArray) == 0 {
				return e.SetDBProperty(pp, []any{})
			}
			for i, v := range vArray {
				if err := traverse(pp.I(i), v, obj); err != nil {
					return err
				}
			}

		case reflect.Struct:
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				if IsNil(v) {
					continue
				}
				if err := traverse(pp.P(k), v, obj); err != nil {
					return err
				}
				count++
			}
			if count == 0 {
				return e.SetDBProperty(pp, struct{}{})
			}
		default:
			// must be scalar so add it
			return e.SetDBProperty(pp, val)
		}
		return nil
	}

	err = traverse(NewPP(), newObj, e.NewObject)
	if err == nil {
		// Copy 'newObj', removing all 'nil' attributes
		e.Object = map[string]any{}
		for k, v := range newObj {
			if !IsNil(v) {
				e.Object[k] = v
			}
		}
		e.NewObject = nil
	}
	return err
}

// This will add in the calculated properties into the entity. This will
// normally be called after a query using FullTree view and before we serialize
// the entity we need to add the non-DB-stored properties (meaning, the
// calculated ones.
// Note that we make a copy and don't touch the entity itself. Serializing
// an entity shouldn't have side-effects.
func (e *Entity) AddCalcProps(info *RequestInfo) map[string]any {
	mat := maps.Clone(e.Object)

	// Regardless of the type of entity, set the generated properties
	for _, prop := range OrderedSpecProps {
		// Only generate props that are for this eType, and have a Fn
		if prop.internals.getFn == nil || !prop.InType(e.Type) {
			continue
		}

		name := prop.Name
		if name == "id" {
			name = e.Singular + "id"
		}

		// Only generate/set the value if it's not already set
		if _, ok := mat[name]; !ok {
			if val := prop.internals.getFn(e, info); !IsNil(val) {
				// Only write it if we have a value
				mat[name] = val
			}
		}
	}

	return mat
}

// This will remove all Collection related attributes from the entity.
// While this is an Entity.Func, we allow callers to pass in the Object
// data to use instead of the e.Object/NewObject so that we'll use this
// Entity's Type (which tells us which collections it has), on the 'obj'.
// This is handy for cases where we need to remove the Resource's collections
// from a Version's Object - like on  a PUT to /GROUPs/gID/RESOURECEs/rID
// where we're passing in what looks like a Resource entity, but we're
// really using it to create a Version
func (e *Entity) RemoveCollections(obj Object) {
	if obj == nil {
		obj = e.NewObject
	}

	for _, coll := range e.GetCollections() {
		delete(obj, coll[0])
		delete(obj, coll[0]+"count")
		delete(obj, coll[0]+"url")
	}
}

// Array of plural/singular pairs
func (e *Entity) GetCollections() [][2]string {
	result := [][2]string{}
	switch e.Type {
	case ENTITY_REGISTRY:
		gs := e.Registry.Model.Groups
		keys := SortedKeys(gs)
		for _, k := range keys {
			result = append(result, [2]string{gs[k].Plural, gs[k].Singular})
		}
		return result
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		rs := gm.Resources
		keys := SortedKeys(rs)
		for _, k := range keys {
			result = append(result, [2]string{rs[k].Plural, rs[k].Singular})
		}
		return result
	case ENTITY_RESOURCE:
		result = append(result, [2]string{"versions", "version"})
		return result
	case ENTITY_META:
		return nil
	case ENTITY_VERSION:
		return nil
	}
	panic(fmt.Sprintf("bad type: %d", e.Type))
	return nil
}

func (e *Entity) GetAttributes(obj Object) Attributes {
	attrs := e.GetBaseAttributes()
	if obj == nil {
		obj = e.NewObject
	}

	attrs.AddIfValuesAttributes(obj)

	return attrs
}

// Returns the initial set of attributes defined for the entity.
func (e *Entity) GetBaseAttributes() Attributes {
	// Add attributes from the model (core and user-defined)
	gm, rm := e.GetModels()

	if e.Type == ENTITY_REGISTRY {
		return e.Registry.Model.GetBaseAttributes()
	}

	if e.Type == ENTITY_GROUP {
		return gm.GetBaseAttributes()
	}

	if e.Type == ENTITY_RESOURCE {
		return rm.GetBaseAttributes()
	}

	if e.Type == ENTITY_META {
		return rm.GetBaseMetaAttributes()
	}

	if e.Type == ENTITY_VERSION {
		return rm.GetBaseAttributes()
	}

	panic(fmt.Sprintf("Bad type: %v", e.Type))
}

// Given a PropPath and a value this will add the necessary golang data
// structures to 'obj' to materialize PropPath and set the appropriate
// fields to 'val'
func ObjectSetProp(obj map[string]any, pp *PropPath, val any) error {
	log.VPrintf(4, "ObjectSetProp(%s=%v)", pp, val)
	if pp.Len() == 0 && IsNil(val) {
		// A bit of a special case, not 100% sure if this is ok.
		// Treat nil val as a request to delete all properties.
		// e.g. obj={}
		for k, _ := range obj {
			delete(obj, k)
		}
		return nil
	}
	PanicIf(pp.Len() == 0, "Can't be zero w/non-nil val")

	_, err := MaterializeProp(obj, pp, val, nil)
	return err
}

func MaterializeProp(current any, pp *PropPath, val any, prev *PropPath) (any, error) {
	log.VPrintf(4, ">Enter: MaterializeProp(%s)", pp)
	log.VPrintf(4, "<Exit: MaterializeProp")

	// current is existing value, used for adding to maps/arrays
	if pp == nil {
		return val, nil
	}

	var ok bool
	var err error

	if prev == nil {
		prev = NewPP()
	}

	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		// TODO look for cases where Kind(val) == array too - maybe?
		var daArray []any

		if current != nil {
			daArray, ok = current.([]any)
			if !ok {
				return nil, fmt.Errorf("Attribute %q isn't an array",
					prev.Append(pp.First()).UI())
			}
		}

		// Resize if needed
		if diff := (1 + index - len(daArray)); diff > 0 {
			daArray = append(daArray, make([]any, diff)...)
		}

		// Trim the end of the array if there are nil's
		daArray[index], err = MaterializeProp(daArray[index], pp.Next(), val,
			prev.Append(pp.First()))
		for len(daArray) > 0 && daArray[len(daArray)-1] == nil {
			daArray = daArray[:len(daArray)-1]
		}
		return daArray, err
	}

	// Is a map/object
	// TODO look for cases where Kind(val) == obj/map too - maybe?

	daMap := map[string]any{}
	if !IsNil(current) {
		daMap, ok = current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("Current isn't a map: %T", current)
		}
	}

	res, err := MaterializeProp(daMap[pp.Top()], pp.Next(), val,
		prev.Append(pp.First()))
	if err != nil {
		return nil, err
	}
	if IsNil(res) {
		delete(daMap, pp.Top())
	} else {
		daMap[pp.Top()], err = res, err
	}

	return daMap, err
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth checking since the server generated them.
// This is mainly used for validating input from a client.
// NOTE!!! This isn't a read-only operation. Normally it would be, but to
// avoid traversing the entity more than once, we will tweak things if needed.
// For example, if a missing attribute has a Default value then we'll add it.
func (e *Entity) Validate() error {
	if e.Type == ENTITY_RESOURCE {
		// Skip Resources // TODO DUG - would prefer to not do this
		return nil
	}

	// Don't touch what was passed in
	attrs := e.GetAttributes(e.NewObject)

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "========")
		log.VPrintf(0, "Validating(%d/%s):\n%s",
			e.Type, e.UID, ToJSON(e.NewObject))
		log.VPrintf(0, "Attrs: %v", SortedKeys(attrs))
	}
	return e.ValidateObject(e.NewObject, "strict", attrs, NewPP())
}

// This should be called after all type-specific calculated properties have
// been removed - such as collections
func (e *Entity) ValidateObject(val any, namecharset string, origAttrs Attributes, path *PropPath) error {

	log.VPrintf(3, ">Enter: ValidateObject(path: %s)", path)
	defer log.VPrintf(3, "<Exit: ValidateObject")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Check Obj:\n%s", ToJSON(val))
		log.VPrintf(0, "OrigAttrs:\n%s", ToJSON(SortedKeys(origAttrs)))
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map ||
		valValue.Type().Key().Kind() != reflect.String {

		return fmt.Errorf("Attribute %q must be a map[string] or object",
			path.UI())
	}
	newObj := val.(map[string]any)

	// Convert origAttrs to a slice of *Attribute where "*" is first, if there
	attrs := make([]*Attribute, len(origAttrs))
	allAttrNames := map[string]bool{}
	count := 1
	for _, attr := range origAttrs {
		allAttrNames[attr.Name] = true
		if attr.Name == "*" {
			attrs[0] = attr // "*" must appear first in the slice
		} else if count == len(attrs) {
			attrs[0] = attr // at last one and no "*" so use [0]
		} else {
			attrs[count] = attr
			count++
		}
	}

	// For Registry entities, get the list of possible collections
	collections := [][2]string{}
	if path.Len() == 0 {
		collections = e.GetCollections()
	}

	// Don't touch what was passed in by just saving the keys and then
	// removing the ones we don't want from it (ie. the collections ones)
	objKeys := map[string]bool{}
	for k, _ := range newObj {
		// Skip collection related attributes
		isColl := false
		for _, coll := range collections {
			if k == coll[0] || k == coll[0]+"count" || k == coll[0]+"url" {
				isColl = true
				break
			}
		}
		if !isColl {
			objKeys[k] = true
		}
	}

	attr := (*Attribute)(nil)
	key := ""
	for len(attrs) > 0 {
		l := len(attrs)
		attr = attrs[l-1] // grab last one & remove it
		attrs = attrs[:l-1]

		// Keys are all of the attribute names in newObj we need to check.
		// Normally there's just one (attr.Name) but if attr.Name is "*"
		// then we'll have a list of all remaining attribute names in newObj to
		// check, hence it's a slice not a single string
		keys := []string{}
		if attr.Name != "*" {
			keys = []string{attr.Name}
		} else {
			keys = SortedKeys(objKeys) // no need to be sorted, just grab keys
		}

		// For each attribute (key) in newObj, check its type
		for _, key = range keys {
			if len(key) > 0 && key[0] == '#' && path.Len() == 0 {
				// Skip system attributes, but only at top level
				continue
			}

			if key == "id" {
				tmp := origAttrs["$singular"]
				if tmp != nil {
					if e.Type == ENTITY_VERSION {
						key = "versionid"
					} else {
						key = tmp.Description + "id"
					}
				}
			}

			val, keyPresent := newObj[key]

			// A Default value is defined but there's no value, so set it
			// and then let normal processing continue
			if !IsNil(attr.Default) && (!keyPresent || IsNil(val)) {
				// When meta.xref is set we skip any attributes with default
				// values. However, if this ever changes where some do need
				// to be set, add a flag to the OrderedSpecProps stuff
				// so we don't need to special case each one
				if e.Type != ENTITY_META || e.GetAsString("xref") == "" {
					val = attr.Default
					newObj[key] = val
					keyPresent = true
				}
			}

			/* Not sure what this was for :-)  save for now
			if path.Len() > 0 {
				if err := IsValidAttributeName(path.Bottom()); err != nil {
					return err
				}
			}
			*/

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (checkFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				err := e.ValidateAttribute(val, attr, path.P(key))
				if err != nil {
					return err
				}
			}

			// GetAttributes already added IfValues for Registry attributes
			if path.Len() >= 1 && len(attr.IfValues) > 0 {
				valStr := fmt.Sprintf("%v", val)
				for ifValStr, ifValueData := range attr.IfValues {
					if valStr != ifValStr {
						continue
					}

					for _, newAttr := range ifValueData.SiblingAttributes {
						if _, ok := allAttrNames[newAttr.Name]; ok {
							return fmt.Errorf(`Attribute %q has an "ifvalues"`+
								`(%s) that defines a conflictng `+
								`siblingattribute: %s`, path.P(key).UI(),
								valStr, newAttr.Name)
						}
						// add new attr to the list so we can check its ifValues
						if newAttr.Name == "*" {
							attrs = append([]*Attribute{newAttr}, attrs...)
						} else {
							attrs = append(attrs, newAttr)
						}
						allAttrNames[newAttr.Name] = true
					}
				}
			}

			// Call the attr's checkFn if there to make sure any
			// incoming value is ok
			if attr.internals.checkFn != nil {
				if err := attr.internals.checkFn(e); err != nil {
					return err
				}
			}

			// Skip read-only attrs
			if attr.ReadOnly {
				delete(objKeys, key) // Remove from to-process list
				continue
			}

			// If this attr has a func to update its value, call it
			if attr.internals.updateFn != nil {
				if err := attr.internals.updateFn(e); err != nil {
					return err
				}

				// grab value in case it changed
				val, keyPresent = newObj[key]
			}

			// Required but not present - note that nil means will be deleted
			if attr.Required && (!keyPresent || IsNil(val)) {
				flagit := true // Assume we'll err

				// Most "meta" attribute aren't actually required when xref
				// is set, so only flag the ones w/o 'xrefrequired=true'
				if e.Type == ENTITY_META && e.GetAsString("xref") != "" &&
					!attr.internals.xrefrequired {
					flagit = false
				}

				// Version.RESOURCEid MUST be missing, so don't flag it
				// All other entities need that attribute though
				if path.Len() == 0 && e.Type == ENTITY_VERSION &&
					key == e.GetResourceSingular()+"id" {
					flagit = false
				}

				if flagit {
					return fmt.Errorf("Required property %q is missing",
						path.P(key).UI())
				}
			}

			// And finally check to make sure it's a valid attribute name,
			// but only if it's actually present in the object.
			if keyPresent {
				if namecharset == "extended" {
					if err := IsValidMapKey(key); err != nil {
						return err
					}
				} else if namecharset == "" || namecharset == "strict" {
					if err := IsValidAttributeName(key); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("Unknown \"namecharset\" value: %s",
						namecharset)
				}
			}

			// Everything is good, so remove it from to-process list
			delete(objKeys, key)
		}
	}

	// See if we have any extra keys and if so, generate an error
	del := []string{}
	for k, _ := range objKeys {
		if k[0] == '#' {
			del = append(del, k)
		}
	}
	for _, k := range del {
		delete(objKeys, k)
	}
	if len(objKeys) != 0 {
		where := path.UI()
		if where != "" {
			where = " in \"" + where + "\""
		}
		return fmt.Errorf("Invalid extension(s)%s: %s", where,
			strings.Join(SortedKeys(objKeys), ","))
	}

	return nil
}

func (e *Entity) ValidateAttribute(val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateAttribute(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateAttribute")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, " val: %v", ToJSON(val))
		log.VPrintf(0, " attr: %v", ToJSON(attr))
	}

	if attr.Type == ANY {
		// All good - let it thru
		return nil
	} else if IsScalar(attr.Type) {
		return e.ValidateScalar(val, attr, path)
	} else if attr.Type == MAP {
		return e.ValidateMap(val, attr.Item, path)
	} else if attr.Type == ARRAY {
		return e.ValidateArray(val, attr.Item, path)
	} else if attr.Type == OBJECT {
		/*
			attrs := e.GetBaseAttributes()
			if useNew {
				attrs.AddIfValuesAttributes(e.NewObject)
			} else {
				attrs.AddIfValuesAttributes(e.Object)
			}
		*/

		return e.ValidateObject(val, attr.NameCharSet, attr.Attributes, path)
	}

	ShowStack()
	panic(fmt.Sprintf("Unknown type(%s): %s", path.UI(), attr.Type))
}

func (e *Entity) ValidateMap(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateMap(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateMap")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, " item: %v", ToJSON(item))
		log.VPrintf(0, " val: %v", ToJSON(val))
	}

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map {
		return fmt.Errorf("Attribute %q must be a map", path.UI())
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type:       item.Type,
		Item:       item.Item,
		Attributes: item.Attributes,
	}

	for _, k := range valValue.MapKeys() {
		keyName := k.Interface().(string)

		if path.Len() > 0 {
			if err := IsValidMapKey(keyName); err != nil {
				return err
			}
		}

		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		if err := e.ValidateAttribute(v, attr, path.P(keyName)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) ValidateArray(val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateArray(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateArray")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "item: %s", ToJSON(item))
		log.VPrintf(0, "val: %s", ToJSON(val))
	}

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return fmt.Errorf("Attribute %q must be an array", path.UI())
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type:       item.Type,
		Item:       item.Item,
		Attributes: item.Attributes,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		if err := e.ValidateAttribute(v, attr, path.I(i)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) ValidateScalar(val any, attr *Attribute, path *PropPath) error {
	if log.GetVerbose() > 2 {
		log.VPrintf(0, ">Enter: ValidateScalar(%s:%s)", path, ToJSON(val))
		defer log.VPrintf(3, "<Exit: ValidateScalar")
	}

	valKind := reflect.ValueOf(val).Kind()

	switch attr.Type {
	case BOOLEAN:
		if valKind != reflect.Bool {
			return fmt.Errorf("Attribute %q must be a boolean", path.UI())
		}
	case DECIMAL:
		if valKind != reflect.Int && valKind != reflect.Float64 {
			return fmt.Errorf("Attribute %q must be a decimal", path.UI())
		}
	case INTEGER:
		if valKind == reflect.Float64 {
			f := val.(float64)
			if f != float64(int(f)) {
				return fmt.Errorf("Attribute %q must be an integer", path.UI())
			}
		} else if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be an integer", path.UI())
		}
	case UINTEGER:
		i := 0
		if valKind == reflect.Float64 {
			f := val.(float64)
			i = int(f)
			if f != float64(i) {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		} else if valKind != reflect.Int {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		} else {
			i = val.(int)
			if valKind != reflect.Int {
				return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
			}
		}
		if i < 0 {
			return fmt.Errorf("Attribute %q must be a uinteger", path.UI())
		}
	case XID:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be an xid", path.UI())
		}
		str := val.(string)

		if attr.Target != "" {
			err := e.MatchXID(str, attr.Target)
			if err != nil {
				return fmt.Errorf("Attribute %q %s", path.UI(), err.Error())
			}
		}
	case STRING:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a string", path.UI())
		}
	case URI:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri", path.UI())
		}
	case URI_REFERENCE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-reference", path.UI())
		}
	case URI_TEMPLATE:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a uri-template", path.UI())
		}
	case URL:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a url", path.UI())
		}
	case TIMESTAMP:
		if valKind != reflect.String {
			return fmt.Errorf("Attribute %q must be a timestamp", path.UI())
		}
		str := val.(string)

		_, err := ConvertStrToTime(str)
		if err != nil {
			return fmt.Errorf("Attribute %q is a malformed timestamp",
				path.UI())
		}
	default:
		panic(fmt.Sprintf("Unknown type: %v", attr.Type))
	}

	// don't "return nil" above, we may need to check enum values
	if len(attr.Enum) > 0 && attr.GetStrict() {
		foundOne := false
		valStr := fmt.Sprintf("%v", val)
		for _, enumVal := range attr.Enum {
			enumValStr := fmt.Sprintf("%v", enumVal)
			if enumValStr == valStr {
				foundOne = true
				break
			}
		}
		if !foundOne {
			valids := ""
			for i, v := range attr.Enum {
				if i > 0 {
					valids += ", "
				}
				valids += fmt.Sprintf("%v", v)
			}
			return fmt.Errorf("Attribute %q(%v) must be one of the enum "+
				"values: %s", path.UI(), val, valids)
		}
	}

	return nil
}

func (e *Entity) GetModels() (*GroupModel, *ResourceModel) {
	if e.GroupModel != nil {
		return e.GroupModel, e.ResourceModel
	}

	if e.Type == ENTITY_REGISTRY || e.Type == ENTITY_MODEL {
		return nil, nil
	}

	e.GroupModel, e.ResourceModel = AbstractToModels(e.Registry, e.Abstract)
	return e.GroupModel, e.ResourceModel
}

func PrepUpdateEntity(e *Entity) error {
	attrs := e.GetAttributes(e.NewObject)

	for key, _ := range attrs {
		attr := attrs[key]

		// Any ReadOnly attribute in Object, but not in NewObject, must
		// be one that we want to keep around. Note that a 'nil' in NewObject
		// will not grab the one in Object - assumes we want to erase the val
		/*
			if attr.ReadOnly {
				oldVal, ok1 := e.Object[attr.Name]
				_, ok2 := e.NewObject[attr.Name]
				if ok1 && !ok2 {
					e.NewObject[attr.Name] = oldVal
				}
			}
		*/

		if attr.InType(e.Type) && attr.internals.updateFn != nil {
			if err := attr.internals.updateFn(e); err != nil {
				return err
			}
		}
	}

	return nil
}

// If no match then return an error saying why
func (e *Entity) MatchXID(str string, xid string) error {
	// 0=all  1=GROUPS  2=RESOURCES  3=versions|""  4=[/versions]|""
	targetParts := targetRE.FindStringSubmatch(xid)

	if len(str) == 0 {
		return fmt.Errorf("must be an xid, not empty")
	}
	if str[0] != '/' {
		return fmt.Errorf("must be an xid, and start with /")
	}
	strParts := strings.Split(str, "/")
	if len(strParts) < 2 {
		return fmt.Errorf("must be a valid xid")
	}
	if len(strParts[0]) > 0 {
		return fmt.Errorf("must be an xid, and start with /")
	}
	if xid == "/" {
		if str != "/" {
			return fmt.Errorf("must match %q target", xid)
		}
		return nil
	}
	if targetParts[1] != strParts[1] { // works for "" too
		return fmt.Errorf("must match %q target", xid)
	}

	gm := e.Registry.Model.Groups[targetParts[1]]
	if gm == nil {
		return fmt.Errorf("uses an unknown group %q", targetParts[1])
	}
	if len(strParts) < 3 || len(strParts[2]) == 0 {
		return fmt.Errorf("must match %q target, missing \"%sid\"",
			xid, gm.Singular)
	}
	if err := IsValidID(strParts[2]); err != nil {
		return fmt.Errorf("must match %q target: %s", xid, err)
	}

	if targetParts[2] == "" { // /GROUPS
		if len(strParts) == 3 {
			return nil
		}
		return fmt.Errorf("must match %q target, extra stuff after %q",
			xid, strParts[2])
	}

	// targetParts has RESOURCES
	if len(strParts) < 4 { //    /GROUPS/gID/RESOURCES
		return fmt.Errorf("must match %q target, missing %q",
			xid, targetParts[2])
	}

	if targetParts[2] != strParts[3] {
		return fmt.Errorf("must match %q target, missing %q",
			xid, targetParts[2])
	}

	rm := gm.Resources[targetParts[2]]
	if rm == nil {
		return fmt.Errorf("uses an unknown resource %q", targetParts[2])
	}

	if len(strParts) < 5 || len(strParts[4]) == 0 {
		return fmt.Errorf("must match %q target, missing \"%sid\"",
			xid, rm.Singular)
	}
	if err := IsValidID(strParts[4]); err != nil {
		return fmt.Errorf("must match %q target: %s", xid, err)
	}

	if targetParts[3] == "" && targetParts[4] == "" {
		if len(strParts) == 5 {
			return nil
		}
		return fmt.Errorf("must match %q target, extra stuff after %q",
			xid, strParts[4])

	}

	if targetParts[4] != "" { // has [/versions]
		if len(strParts) == 5 {
			//   /GROUPS/RESOURCES[/version]  vs /GROUPS/gID/RESOURCES/rID
			return nil
		}
	}

	if len(strParts) < 6 || strParts[5] != "versions" {
		return fmt.Errorf("must match %q target, missing \"versions\"", xid)
	}

	if len(strParts) < 7 || len(strParts[6]) == 0 {
		return fmt.Errorf("must match %q target, missing a \"versionid\"", xid)
	}
	if err := IsValidID(strParts[6]); err != nil {
		return fmt.Errorf("must match %q target: %s", xid, err)
	}

	if len(strParts) > 7 {
		return fmt.Errorf("must match %q target, too long", xid)
	}

	return nil
}

// We call this to verify that the top level attribute names are valid.
// We can't really do this during the Validation funcs because at that point
// in the process we may have added #xxx type of attribute names, and "#"
// isn't a valid char. And we need to make sure users don't try to pass in
// attributes that start with "#" to attack us.
// Now, one way around this is to move the system props (#xxx) into a separate
// map (out of Object and NewObject) but then we'd need to duplicate a lot
// of logic - but it might actually make for a cleaner design to keep
// system data out of the user data space, so worth considering in the future.
// I really would prefer to push this down in the stack though.
func CheckAttrs(obj map[string]any) error {
	if obj == nil {
		return nil
	}
	for k, _ := range obj {
		if err := IsValidAttributeName(k); err != nil {
			// log.Printf("Key: %q", k)
			// ShowStack()
			return err
		}
	}
	return nil
}
