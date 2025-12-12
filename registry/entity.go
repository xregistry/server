package registry

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
	. "github.com/xregistry/server/common"
)

type EntityExtensions struct {
	tx         *Tx
	AccessMode int // FOR_READ, FOR_WRITE
}

func (e *Entity) GetRequestInfo() *RequestInfo {
	tx := e.tx
	if tx == nil {
		return nil
	}
	return tx.RequestInfo
}

type EntitySetter interface {
	Get(name string) any
	JustSet(name string, val any) *XRError
	SetSave(name string, val any) *XRError
	Delete() *XRError
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
	PanicIf(e.AccessMode != FOR_WRITE, "%q isn't FOR_WRITE", e.XID)
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
	log.Printf("Stack for NewObject (%s):", e.XID)
	for _, s := range e.NewObjectStack {
		log.Printf("  %s", s)
	}
}

func (e *Entity) Touch() bool {
	log.VPrintf(3, "Touch: %s/%s", e.Singular, e.UID)

	// See if it's already been modified (and saved) this Tx, if so exit
	if e.ModSet && e.EpochSet {
		return false
	}

	e.Lock()
	e.EnsureNewObject()
	return true
}

func (e *Entity) EnsureNewObject() bool {
	if e.NewObject == nil {
		if e.Object == nil {
			e.SetNewObject(map[string]any{})
		} else {
			e.SetNewObject(maps.Clone(e.Object))
		}
		return true
	}
	return false
}

func (e *Entity) Get(path string) any {
	pp, err := PropPathFromUI(path)
	PanicIf(err != nil, "%s", err)
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
	PanicIf(!ok, "Val: %v  T: %T", val, val)
	return i
}

func (e *Entity) GetPP(pp *PropPath) any {
	if (e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION) && pp.Len() == 1 {
		rm := e.GetResourceModel()
		if rm.GetHasDocument() && pp.Top() == rm.Singular {
			contentID := e.Get("#contentid")

			results := Query(e.tx, `
            SELECT Content FROM ResourceContents WHERE VersionSID=? `,
				contentID)
			defer results.Close()

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

func RawEntityFromPath(tx *Tx, regID string, path string, anyCase bool, accessMode int) (*Entity, *XRError) {
	log.VPrintf(3, ">Enter: RawEntityFromPath(%s)", path)
	defer log.VPrintf(3, "<Exit: RawEntityFromPath")

	// RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1     2      3       4     5    6       7         8       9    10

	caseExpr := ""
	if anyCase {
		caseExpr = " COLLATE utf8mb4_0900_ai_ci"
	}

	results := Query(tx, `
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

	return readNextEntity(tx, results, accessMode)
}

func (e *Entity) Query(query string, args ...any) [][]any {
	results := Query(e.tx, query, args...)
	defer results.Close()

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

	return data
}

func RawEntitiesFromQuery(tx *Tx, regID string, accessMode int, query string, args ...any) ([]*Entity, *XRError) {
	log.VPrintf(3, ">Enter: RawEntititiesFromQuery(%s)", query)
	defer log.VPrintf(3, "<Exit: RawEntitiesFromQuery")

	// RegSID,Type,Plural,Singular,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1     2     3        4    5     6         7        8     9     10

	if query != "" {
		query = "AND (" + query + ") "
	}
	args = append(append([]any{}, regID), args...)
	results := Query(tx, `
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

	entities := []*Entity{}
	for {
		e, xErr := readNextEntity(tx, results, accessMode)
		if xErr != nil {
			return nil, xErr
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
func (e *Entity) Refresh(accessMode int) *XRError {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbSID)
	defer log.VPrintf(3, "<Exit: Refresh")

	mode := ""
	if accessMode == FOR_WRITE {
		mode = " FOR UPDATE"
	}

	results := Query(e.tx, `
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntitySID=?`+mode, e.DbSID)
	defer results.Close()

	// Erase all old props first
	e.Object = map[string]any{}
	e.NewObject = nil

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		name := NotNilString(row[0])
		val := NotNilString(row[1])
		propType := NotNilString(row[2])

		if xErr := e.SetFromDBName(name, &val, propType); xErr != nil {
			return xErr
		}
	}

	// TODO see if we can remove this - it scares me.
	// Added when I added Touch() - touching parent on add/remove child
	e.EpochSet = false
	e.ModSet = false

	if accessMode == FOR_WRITE {
		e.AccessMode = FOR_WRITE
	}
	if e.AccessMode == 0 {
		e.AccessMode = FOR_READ
	}

	e.tx.AddToCache(e)

	return nil
}

// Set, Validate and Save to DB but not Commit
func (e *Entity) eSetSave(path string, val any) *XRError {
	log.VPrintf(3, ">Enter: SetSave(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	pp, err := PropPathFromUI(path)
	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail="+
				fmt.Sprintf("Bad attribute path in \"%s\": %s", e.XID, err))
	}

	// Set, Validate and Save
	xErr := e.SetPP(pp, val)
	return xErr.SetSubject(e.XID)
}

// Set the prop in the Entity but don't Validate or Save to the DB
func (e *Entity) eJustSet(pp *PropPath, val any) *XRError {
	log.VPrintf(3, ">Enter: JustSet([%d] %s.%s=%v)", e.Type, e.UID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit: JustSet")

	PanicIf(e.AccessMode != FOR_WRITE, "ejustset: %q isn't FOR_WRITE", e.XID)

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
					xErr := DoOneTwo(e.tx,
		               `UPDATE Resources SET xRef=? WHERE SID=?`,
						val, e.DbSID)
					if xErr != nil {
						return xErr
					}
				}
	*/

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Abstract/ID: %s/%s", e.Abstract, e.UID)
		log.VPrintf(0, "e.Object:\n%s", ToJSON(e.Object))
		log.VPrintf(0, "e.NewObject:\n%s", ToJSON(e.NewObject))
	}

	err := ObjectSetProp(e.NewObject, pp, val)
	if err != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+pp.UI(),
			"error_detail="+err.Error())
	}
	return nil
}

func (e *Entity) ValidateAndSave() *XRError {
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

	if xErr := e.Validate(); xErr != nil {
		return xErr
	}

	return e.Save()
}

// This is really just an internal Setter used for testing.
// It'll set a property and then validate and save the entity in the DB
func (e *Entity) SetPP(pp *PropPath, val any) *XRError {
	log.VPrintf(3, ">Enter: SetPP(%s: %s=%v)", e.DbSID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")
	defer func() {
		if log.GetVerbose() > 2 {
			log.VPrintf(0, "SetPP exit: e.Object:\n%s", ToJSON(e.Object))
		}
	}()

	if xErr := e.eJustSet(pp, val); xErr != nil {
		return xErr
	}

	xErr := e.ValidateAndSave()
	if xErr != nil {
		// If there's an error, and we're making the assumption that we're
		// setting and saving all in one shot (and there are no other edits
		// pending), go ahead and undo the changes since they're wrong.
		// Otherwise the caller would need to call Refresh themselves.

		// Not sure why setting it to nil isn't sufficient (todo)
		// e.NewObject = nil
		e.Refresh(FOR_READ)
	}

	return xErr
}

// This will save a single property/value in the DB. This assumes
// the caller is traversing the Object and splitting it into individual props
func (e *Entity) SetDBProperty(pp *PropPath, val any) *XRError {
	log.VPrintf(3, ">Enter: SetDBProperty(%s=%v)", pp, val)
	defer log.VPrintf(3, "<Exit SetDBProperty")

	PanicIf(pp.UI() == "", "pp is empty")

	name := pp.DB()

	if len(name) > MAX_PROPNAME {
		return NewXRError("invalid_attribute", e.XID,
			"name="+name,
			"error_detail="+
				fmt.Sprintf("attribute names must not exceed %d chars",
					MAX_PROPNAME))
	}

	// Any prop with "dontStore"=true we skip
	_, propsMap := e.GetPropsOrdered()
	specProp, ok := propsMap[pp.Top()]
	if ok && specProp.internals != nil && specProp.internals.dontStore {
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
				Do(e.tx, `DELETE FROM ResourceContents WHERE VersionSID=?`,
					e.DbSID)
			} else {
				// Update the content
				DoOneTwo(e.tx, `
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)

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
		Do(e.tx, `DELETE FROM Props WHERE EntitySID=? and PropName=?`,
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
				return NewXRError("invalid_attribute", e.XID,
					"name="+pp.UI(),
					"error_detail="+
						fmt.Sprintf("must be less than %d chars",
							MAX_VARCHAR+1))
			}
		case reflect.Slice:
			if reflect.ValueOf(val).Len() > 0 {
				return NewXRError("invalid_attribute", e.XID,
					"name="+pp.UI(),
					"error_detail=can't set non-empty arrays")
			}
			dbVal = ""
		case reflect.Map:
			if reflect.ValueOf(val).Len() > 0 {
				return NewXRError("invalid_attribute", e.XID,
					"name= "+pp.UI(),
					"error_detail=can't set non-empty maps")
			}
			dbVal = ""
		case reflect.Struct:
			if reflect.ValueOf(val).NumField() > 0 {
				return NewXRError("invalid_attribute", e.XID,
					"name="+pp.UI(),
					"error_detail=can't set non-empty objects")
			}
			dbVal = ""
		}

		DoOneTwo(e.tx, `
            REPLACE INTO Props(
              RegistrySID,EntitySID,eType,PropName,PropValue,PropType,DocView)
            VALUES( ?,?,?,?,?,?, true )`,
			e.Registry.DbSID, e.DbSID, e.Type, name, dbVal, propType)
	}

	return nil
}

// This is used to take a DB entry and update the current Entity's Object
func (e *Entity) SetFromDBName(name string, val *string, propType string) *XRError {
	var err error
	pp := MustPropPathFromDB(name)

	if val == nil {
		err := ObjectSetProp(e.Object, pp, val)
		if err != nil {
			return NewXRError("bad_request", e.XID,
				"error_detail=Error setting attribute: "+err.Error())
		}
		return nil
	}
	if e.Object == nil {
		e.Object = map[string]any{}
	}

	if propType == STRING || propType == URI || propType == URI_REFERENCE ||
		propType == URI_TEMPLATE || propType == URL || propType == TIMESTAMP ||
		propType == XID || propType == XIDTYPE {
		err = ObjectSetProp(e.Object, pp, *val)
	} else if propType == BOOLEAN {
		// Technically the "1" check shouldn't be needed, but just in case
		err = ObjectSetProp(e.Object, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s: %s", *val, err))
		}
		err = ObjectSetProp(e.Object, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s: %s", *val, err))
		}
		err = ObjectSetProp(e.Object, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(e.Object, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(e.Object, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		err = ObjectSetProp(e.Object, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}

	if err != nil {
		return NewXRError("bad_request", e.XID,
			"error_detail=Error setting attribute: "+err.Error())
	}
	return nil
}

// Create a new Entity based on what's in the DB. Similar to Refresh()
func readNextEntity(tx *Tx, results *Result, accessMode int) (*Entity, *XRError) {
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
				EntityExtensions: EntityExtensions{
					tx:         tx,
					AccessMode: accessMode,
				},

				Registry: tx.Registry,
				DbSID:    NotNilString(row[4]),
				Plural:   plural,
				Singular: NotNilString(row[3]),
				UID:      uid,

				Type:     eType,
				Path:     NotNilString(row[9]),
				XID:      "/" + NotNilString(row[9]),
				Abstract: NotNilString(row[10]),
			}

			entity.GroupModel, entity.ResourceModel =
				AbstractToModels(tx.Registry, entity.Abstract)
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

		xErr := entity.SetFromDBName(propName, &propVal, propType)
		if xErr != nil {
			return nil, xErr
		}
	}

	return entity, nil
}

// This data will be merged into OrderedSpecProps during init().
// We can't put them directly into OrderedSpecProps because the client doesn't
// need them, or have access to the RequestInfo
var PropsFuncs = []*Attribute{
	{
		Name: "specversion",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				return SPECVERSION
			},
			checkFn: func(e *Entity) *XRError {
				tmp := e.NewObject["specversion"]
				if !IsNil(tmp) && tmp != "" && tmp != SPECVERSION {
					return NewXRError("invalid_attribute", e.XID,
						"name=specversion",
						"error_detail="+
							fmt.Sprintf("invalid value: %v", tmp))
				}
				return nil
			},
		},
	},
	{
		Name: "id",
		internals: &AttrInternals{
			checkFn: func(e *Entity) *XRError {
				singular := e.Singular
				// PanicIf(singular == "", "singular is '' :  %v", e)
				if e.Type == ENTITY_VERSION || e.Type == ENTITY_META {
					_, rm := e.GetModels()
					singular = rm.Singular
				}
				justSingular := singular
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
					return NewXRError("invalid_attribute", e.XID,
						"name="+singular,
						"error_detail="+"can't be an empty string")
				}

				if xErr := IsValidID(newID.(string), singular); xErr != nil {
					xErr.Subject = e.XID
					return xErr
				}

				if oldID != "" && !IsNil(oldID) && newID != oldID {
					return NewXRError("mismatched_id", e.XID,
						"singular="+justSingular,
						"expected_id="+fmt.Sprintf("%v", oldID),
						"invalid_id="+fmt.Sprintf("%v", newID))
				}
				return nil
			},
			updateFn: func(e *Entity) *XRError {
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
		Name: "versionid",
		internals: &AttrInternals{
			checkFn: func(e *Entity) *XRError {
				oldID := any(e.UID)
				newID := any(e.NewObject["versionid"])

				if IsNil(newID) {
					return nil // Not trying to be updated, so skip it
				}

				if newID == "" {
					return NewXRError("invalid_attribute", e.XID,
						"name=versionid",
						"error_detail="+"can't be an empty string")
				}

				if xErr := IsValidID(newID.(string), "versionid"); xErr != nil {
					xErr.Subject = e.XID
					return xErr
				}

				if oldID != "" && !IsNil(oldID) && newID != oldID {
					return NewXRError("mismatched_id", e.XID,
						"singular=version",
						"invalid_id="+fmt.Sprintf("%v", newID),
						"expected_id="+fmt.Sprintf("%v", oldID))
				}
				return nil
			},
			updateFn: func(e *Entity) *XRError {
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
		Name: "self",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				base := ""
				path := e.Path
				isAbs := false

				info := e.GetRequestInfo()
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
		},
	},
	/*
				{
					Name:           "shortself",
					internals: &AttrInternals{
						getFn: func(e *Entity) any {
							path := e.Path
							base := ""
		                    info := e.GetRequestInfo()
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
					},
				},
	*/
	{
		Name:      "xid",
		internals: &AttrInternals{},
	},
	{
		Name:      "xref",
		internals: &AttrInternals{},
	},
	{
		Name: "epoch",
		internals: &AttrInternals{
			checkFn: func(e *Entity) *XRError {
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
					return NewXRError("invalid_attribute", e.XID,
						"name=epoch",
						"error_detail=must be a uinteger")
				}

				if !e.tx.RequestInfo.HasIgnore("epoch") && oldEpoch != 0 && newEpoch != oldEpoch {
					return NewXRError("mismatched_epoch", e.XID,
						"bad_epoch="+fmt.Sprintf("%v", val),
						"epoch="+fmt.Sprintf("%d", oldEpoch))
				}
				return nil
			},
			updateFn: func(e *Entity) *XRError {
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
		Name:      "name",
		internals: &AttrInternals{},
	},
	{
		Name:      "isdefault",
		internals: &AttrInternals{},
	},
	{
		Name:      "description",
		internals: &AttrInternals{},
	},
	{
		Name:      "documentation",
		internals: &AttrInternals{},
	},
	{
		Name:      "labels",
		internals: &AttrInternals{},
	},
	{
		Name: "createdat",
		internals: &AttrInternals{
			updateFn: func(e *Entity) *XRError {
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

				e.NewObject["createdat"] = ca

				return nil
			},
		},
	},
	{
		Name: "modifiedat",
		internals: &AttrInternals{
			updateFn: func(e *Entity) *XRError {
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

				e.NewObject["modifiedat"] = ma
				e.ModSet = true

				return nil
			},
		},
	},
	{
		Name:      "$extensions",
		internals: &AttrInternals{},
	},
	{
		Name: "capabilities",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				// Need to explicitly ask for "capabilities", ?inline=* won't
				// do it
				info := e.GetRequestInfo()
				if info != nil && info.ShouldInline(NewPPP("capabilities").DB()) {
					capStr := e.GetAsString("#capabilities")
					if capStr == "" {
						return e.Registry.Capabilities
					}

					cap, xErr := ParseCapabilitiesJSON([]byte(capStr))
					Must(xErr)
					return cap
				}
				return nil
			},
			checkFn: func(e *Entity) *XRError {
				// Yes it's weird to store it in #capabilities but
				// it's actually easier to do it this way. Trying to convert
				// map[string]any <-> Capabilities  is really annoying
				val, ok := e.NewObject["capabilities"]

				var xErr *XRError

				if ok {
					var cap *Capabilities

					if !IsNil(val) {
						// If speed is ever a concern here, just save the raw
						// json from the input stream instead from http
						// processing
						valStr := ToJSON(val)

						cap, xErr = ParseCapabilitiesJSON([]byte(valStr))
						if xErr != nil {
							return xErr
						}
					} else {
						cap = DefaultCapabilities
					}

					if xErr = cap.Validate(); xErr != nil {
						return xErr
					}

					valStr := ToJSON(cap)

					e.NewObject["#capabilities"] = valStr
					delete(e.NewObject, "capabilities")
					e.Registry.Capabilities = cap
				}
				return nil
			},
			updateFn: func(e *Entity) *XRError {
				return nil
			},
		},
	},
	{
		Name: "model",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				// Need to explicitly ask for "model", ?inline=* won't
				// do it
				info := e.GetRequestInfo()
				if info != nil && info.ShouldInline(NewPPP("model").DB()) {
					model := info.Registry.Model
					if model == nil {
						model = &Model{}
					}
					httpModel := model // ModelToHTTPModel(model)
					return (*UserModel)(httpModel)
				}
				return nil
			},
		},
	},
	{
		Name: "modelsource",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				// Need to explicitly ask for "modelsource", ?inline=* won't
				// do it
				info := e.GetRequestInfo()
				if info != nil && info.ShouldInline(NewPPP("modelsource").DB()) {
					model := info.Registry.Model
					if model == nil || model.Source == "" {
						return struct{}{}
					}

					obj, err := ParseJSONToObject([]byte(model.Source))
					PanicIf(err != nil, "Failed: %s", err)
					return obj
				}
				return nil
			},
		},
	},
	{
		Name:      "readonly",
		internals: &AttrInternals{},
	},
	{
		Name:      "compatibility",
		internals: &AttrInternals{},
	},
	{
		Name: "compatibilityauthority",
		internals: &AttrInternals{
			updateFn: func(e *Entity) *XRError {
				if !IsNil(e.NewObject["xref"]) {
					return nil
				}
				compat, _ := e.NewObject["compatibility"]
				isDefault := (compat == SpecProps["compatibility"].Default)
				if IsNil(compat) || isDefault {
					delete(e.NewObject, "compatibilityauthority")
				} else {
					e.NewObject["compatibilityauthority"] = "external"
				}
				return nil
			},
		},
	},
	{
		Name:      "deprecated",
		internals: &AttrInternals{},
	},
	{
		Name: "ancestor",
		internals: &AttrInternals{
			updateFn: func(e *Entity) *XRError {
				_, ok := e.NewObject["ancestor"]
				PanicIf(!ok, "Missing versionid")
				if !ok {
					_, ok := e.NewObject["versionid"]
					PanicIf(!ok, "Missing versionid")
					// Just assign a placeholder to get past validation.
					// CheckAncestors() should fix this before we commit
					// the tx
					e.NewObject["ancestor"] = ANCESTOR_TBD
				}
				return nil
			},
		},
	},
	{
		Name:      "contenttype",
		internals: &AttrInternals{},
	},
	{
		Name:      "$extensions",
		internals: &AttrInternals{},
	},
	{
		Name:      "$space",
		internals: &AttrInternals{},
	},
	// For the $RESOURCE ones, make sure to use attr.Clone("newname")
	// when the $RESOURCE is substituded with the Resource's singular
	// name. Otherwise you'll be updating this shared entry.
	{
		Name: "$RESOURCEurl",
		internals: &AttrInternals{
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) *XRError {
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
		internals: &AttrInternals{
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) *XRError {
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
		internals: &AttrInternals{
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) *XRError {
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
		Name: "$RESOURCEbase64",
		internals: &AttrInternals{
			checkFn: RESOURCEcheckFn,
			updateFn: func(e *Entity) *XRError {
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
		Name:      "$space",
		internals: &AttrInternals{},
	},
	{
		Name: "metaurl",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				base := ""
				path := e.Path

				info := e.GetRequestInfo()
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
		},
	},
	{
		Name:      "meta",
		internals: &AttrInternals{},
	},
	{
		Name:      "$space",
		internals: &AttrInternals{},
	},
	{
		Name: "defaultversionid",
		internals: &AttrInternals{
			updateFn: func(e *Entity) *XRError {
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
		Name: "defaultversionurl",
		internals: &AttrInternals{
			getFn: func(e *Entity) any {
				val := e.Object["defaultversionid"]
				if IsNil(val) {
					return nil
				}
				valStr := val.(string)

				// replace "meta" with "versions/VID"
				path := e.Path[:len(e.Path)-4] + "versions/" + valStr
				result := ""
				isAbsURL := false
				suffix := ""

				info := e.GetRequestInfo()
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
						if e.GetResourceModel().GetHasDocument() == true {
							suffix = "$details"
						}
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
				result += "/" + path + suffix

				return result
			},
		},
	},
	{
		Name:      "defaultversionsticky",
		internals: &AttrInternals{},
	},
	{
		Name:      "$space",
		internals: &AttrInternals{},
	},
	{
		Name:      "$COLLECTIONS", // Implicitly creates the url and count ones
		internals: &AttrInternals{},
	},
}

func (e *Entity) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	switch e.Type {
	case ENTITY_REGISTRY:
		return e.Registry.Model.GetPropsOrdered()
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		return gm.GetPropsOrdered()
	case ENTITY_RESOURCE:
		_, rm := e.GetModels()
		return rm.GetPropsOrdered()
	case ENTITY_META:
		_, rm := e.GetModels()
		return rm.GetMetaPropsOrdered()
	case ENTITY_VERSION:
		_, rm := e.GetModels()
		return rm.GetVersionPropsOrdered()
	default:
		panic("What?")
	}
}

// This is used to serialize an Entity regardless of the format.
// This will:
//   - Use AddCalcProps() to fill in any missing props (eg Entity's getFn())
//   - Call that passed-in 'fn' to serialize each prop but in the right order
//     as defined by the entity's GetPropsOrdered()
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any, *Attribute) *XRError) *XRError {
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

	propsOrdered, propsMap := e.GetPropsOrdered()

	// Loop over the defined props - extensions are done under the "if...$ext"
	for _, prop := range propsOrdered {
		name := prop.Name

		// If hasDoc && we're in doc view &&  on the Resource (not version)
		// then skip the RESOURCE/RESOURCEbase64 attributes.
		// Other version-level attribute are automatically excluded by the
		// query. RESOURCE* aren't part of the Props table, so they're special
		if hasDoc && e.Type == ENTITY_RESOURCE && info.DoDocView() {
			if name == resourceSingular || name == resourceSingular+"base64" {
				continue
			}
		}

		log.VPrintf(4, "Ser prop(%s): %q", e.XID, name)

		attr, ok := attrs[name]
		if !ok {
			log.VPrintf(4, "  skipping %q, no attr", name)
			delete(daObj, name)
			continue // not allowed at this eType so skip it
		}

		if prop.Name == "$extensions" {
			if prop.InType(e.Type) {
				for _, objKey := range SortedKeys(daObj) {
					// Skip spec defined properties
					if propsMap[objKey] != nil {
						continue
					}

					val, _ := daObj[objKey]
					attr := attrs[objKey]
					delete(daObj, objKey)
					if attr == nil {
						attr = attrs["*"]
						PanicIf(objKey[0] != '#' && attr == nil,
							"Can't find attr for (%s) %q", e.XID, objKey)
					}
					// log.Printf("Ser*ext(%s): %q", e.XID, objKey)

					if xErr := fn(e, info, objKey, val, attr); xErr != nil {
						return xErr
					}
				}
			}
			continue
		}

		if name[0] == '$' || (prop.internals != nil && prop.internals.alwaysSerialize) {
			log.VPrintf(4, "  forced serialization of %q", name)
			if xErr := fn(e, info, name, nil, attr); xErr != nil {
				return xErr
			}
			continue
		}

		// Should be a no-op for Resources.
		if val, ok := daObj[name]; ok {
			log.VPrintf(4, "  val: %v", val)
			if !IsNil(val) {
				xErr := fn(e, info, name, val, attr)
				if xErr != nil {
					return xErr
				}
			}
			delete(daObj, name)
		} else {
			log.VPrintf(4, "  no value for %q", name)
		}
	}

	// Now do all other props (extensions) alphabetically
	/*
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

			if xErr := fn(e, info, objKey, val, attr); xErr != nil {
				return xErr
			}
		}
	*/

	return nil
}

func (e *Entity) Lock() bool { // did we lock it?
	log.VPrintf(3, ">Enter: Lock(%s)", e.XID)
	defer log.VPrintf(3, "<Exit: Lock")

	if e.AccessMode == FOR_WRITE {
		// Already locked
		return false
	}
	Must(e.Refresh(FOR_WRITE))
	return true
}

func (e *Entity) Save() *XRError {
	log.VPrintf(3, ">Enter: Save(%s/%s)", e.Abstract, e.UID)
	defer log.VPrintf(3, "<Exit: Save")

	PanicIf(e.AccessMode != FOR_WRITE, "%q isn't FOR_WRITE", e.XID)

	// TODO remove at some point when we're sure it's safe
	if SpecProps["epoch"].InType(e.Type) && IsNil(e.NewObject["epoch"]) {
		// Only an xref'd "meta" is allowed to not have an 'epoch'
		if e.Type != ENTITY_META || IsNil(e.NewObject["xref"]) {
			PanicIf(true, "Epoch is nil(%s):%s", e.XID, ToJSON(e.NewObject))
		}
	}

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Saving - %s (id:%s):\n%s\n", e.Abstract, e.UID,
			ToJSON(e.NewObject))
	}

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	// Delete all props for this entity, we assume that NewObject
	// contains everything we want going forward
	Do(e.tx, "DELETE FROM Props WHERE EntitySID=? ", e.DbSID)

	resSingular := ""
	resHasDoc := false
	if rm := e.GetResourceModel(); rm != nil {
		resSingular = rm.Singular
		resHasDoc = rm.GetHasDocument()
	}

	var traverse func(pp *PropPath, val any, obj map[string]any) *XRError
	traverse = func(pp *PropPath, val any, obj map[string]any) *XRError {
		if IsNil(val) { // Skip empty attributes
			return nil
		}

		valValue := reflect.ValueOf(val)

		switch valValue.Kind() {
		case reflect.Map:
			keys := valValue.MapKeys()
			count := 0
			for _, keyValue := range keys {
				if keyValue.Kind() != reflect.String {
					return NewXRError("invalid_attribute", e.XID,
						"name="+pp.RemoveLast().UI(),
						"error_detail="+
							fmt.Sprintf("map key (%s) needs to be a string, "+
								"not %s", pp.Last().Text, keyValue.Kind().String()))
				}

				k := keyValue.Interface().(string)
				v := valValue.MapIndex(keyValue).Interface()
				// "RESOURCE" is special - call SetDBProp if it's present
				if resHasDoc && pp.Len() == 0 && k == resSingular {
					if xErr := e.SetDBProperty(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else if k[0] == '#' {
					if xErr := e.SetDBProperty(pp.P(k), v); xErr != nil {
						return xErr
					}
				} else {
					if IsNil(v) {
						continue
					}
					if xErr := traverse(pp.P(k), v, obj); xErr != nil {
						return xErr
					}
				}
				count++
			}
			if count == 0 && pp.Len() != 0 {
				return e.SetDBProperty(pp, map[string]any{})
			}

		case reflect.Slice:
			if valValue.Len() == 0 {
				valValue = reflect.MakeSlice(reflect.TypeOf(val), 0, 0)
				return e.SetDBProperty(pp, valValue.Interface())
			}
			for i := 0; i < valValue.Len(); i++ {
				v := valValue.Index(i).Interface()
				if xErr := traverse(pp.I(i), v, obj); xErr != nil {
					return xErr
				}
			}

		case reflect.Struct:
			panic("a struct")
			// If this is ever needed, use reflect to traverse into val
			// like we do for map & slice above. The stuff below is old/wrong
			vMap := val.(map[string]any)
			count := 0
			for k, v := range vMap {
				if IsNil(v) {
					continue
				}
				if xErr := traverse(pp.P(k), v, obj); xErr != nil {
					return xErr
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

	xErr := traverse(NewPP(), newObj, e.NewObject)
	if xErr != nil {
		return xErr
	}

	// Copy 'newObj', removing all 'nil' attributes
	e.Object = map[string]any{}
	for k, v := range newObj {
		if !IsNil(v) {
			e.Object[k] = v
		}
	}
	e.NewObject = nil

	return nil
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
	propsOrdered, _ := e.GetPropsOrdered()

	for _, prop := range propsOrdered {
		// Only generate props that have a Fn
		if prop.internals != nil && prop.internals.getFn != nil {
			// Only generate/set the value if it's not already set
			if _, ok := mat[prop.Name]; !ok {
				if val := prop.internals.getFn(e); !IsNil(val) {
					// Only write it if we have a value
					// log.Printf("Added calc prop: %q", prop.Name)
					mat[prop.Name] = val
				}
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
		for _, k := range Keys(gs) {
			result = append(result, [2]string{gs[k].Plural, gs[k].Singular})
		}
		return result
	case ENTITY_GROUP:
		gm, _ := e.GetModels()
		keys := gm.GetResourceList()
		for _, k := range keys {
			rm := gm.FindResourceModel(k)
			result = append(result, [2]string{rm.Plural, rm.Singular})
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
		// This seems to work for now.
		// At some point we may want to have it only include version-level
		// attributes and not resource-level ones - like versionscount.
		// At which point we may need to add back in the code that removes
		// those resource-level attributes before we create/update a Version
		// (e.g. POST .../rID)
		return rm.GetBaseAttributes()
	}

	panic(fmt.Sprintf("Bad type: %v", e.Type))
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth checking since the server generated them.
// This is mainly used for validating input from a client.
// NOTE!!! This isn't a read-only operation. Normally it would be, but to
// avoid traversing the entity more than once, we will tweak things if needed.
// For example, if a missing attribute has a Default value then we'll add it.
func (e *Entity) Validate() *XRError {
	// Don't touch what was passed in
	attrs := e.GetAttributes(e.NewObject)
	if log.GetVerbose() > 3 {
		log.Printf("In Validate - Attrs:\n%s", ToJSON(attrs))
	}

	if e.Type == ENTITY_RESOURCE {
		// Skip Resources // TODO DUG - would prefer to not do this
		return nil
		// If we ever support extensions in resourceattributes
		/*
					RemoveVersionAttributes(e.ResourceModel, e.NewObject)

			        // Not really correct yet.
			        // should just use resourceattributes + ifvaluesattrs
					for _, k := range Keys(attrs) {
						a := attrs[k]
						if a.InType(ENTITY_VERSION) && !a.InType(ENTITY_RESOURCE) {
							delete(attrs, k)
						}
					}
		*/
	}

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
func (e *Entity) ValidateObject(val any, namecharset string, origAttrs Attributes, path *PropPath) *XRError {

	log.VPrintf(3, ">Enter: ValidateObject(path: %s)", path)
	defer log.VPrintf(3, "<Exit: ValidateObject")

	PanicIf(e.XID != "/"+e.Path, "E:%s", ToJSON(e))

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "Check Obj:\n%s", ToJSON(val))
		log.VPrintf(0, "OrigAttrs:\n%s", ToJSON(SortedKeys(origAttrs)))
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map ||
		valValue.Type().Key().Kind() != reflect.String {

		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail="+"must be a map[string] or object")
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

	// Don't touch what was passed in
	objKeys := maps.Clone(newObj)

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
				if xErr := IsValidAttributeName(path.Bottom(), e.XID, path.UI()); xErr != nil {
					return xErr
				}
			}
			*/

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (checkFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				xErr, haveReplacement, newValue := e.ValidateAttribute(val,
					attr, path.P(key))
				if xErr != nil {
					return xErr
				}
				if haveReplacement {
					val = newValue
					newObj[key] = val
					keyPresent = true
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
							return NewXRError("invalid_attribute", e.XID,
								"name="+path.P(key).UI(),
								"error_detail="+
									fmt.Sprintf(`has an "ifvalues"`+
										`(%s) that defines a conflictng `+
										`siblingattribute: %s`,
										valStr, newAttr.Name))
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
			if attr.internals != nil && attr.internals.checkFn != nil {
				if xErr := attr.internals.checkFn(e); xErr != nil {
					return xErr
				}
			}

			// Skip/remove 'dontStore' attrs
			if attr.internals != nil && attr.internals.dontStore {
				// TODO find a way to allow an admin to set the
				// meta.ReadOnly flag itself
				delete(objKeys, key) // Remove from to-process list
				delete(newObj, key)
				continue
			}

			// If this attr has a func to update its value, call it
			if attr.internals != nil && attr.internals.updateFn != nil {
				if xErr := attr.internals.updateFn(e); xErr != nil {
					return xErr
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
					return NewXRError("required_attribute_missing", e.XID,
						"list="+path.P(key).UI())
				}
			}

			// And finally check to make sure it's a valid attribute name,
			// but only if it's actually present in the object.
			if keyPresent {
				if namecharset == "extended" {
					if xErr := IsValidMapKey(key, e.XID, path.UI()); xErr != nil {
						return xErr
					}
				} else if namecharset == "" || namecharset == "strict" {
					if xErr := IsValidAttributeName(key, e.XID, path.UI()); xErr != nil {
						ShowStack()
						return xErr
					}
				} else {
					return NewXRError("bad_request", e.XID,
						"error_detail="+
							fmt.Sprintf("Unknown \"namecharset\" value: %s",
								namecharset))
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
			where += "."
		}

		xErr := NewXRError("unknown_attribute", e.XID,
			"name="+where+SortedKeys(objKeys)[0])
		if len(objKeys) > 1 {
			xErr.SetDetailf("Full list: %s.",
				strings.Join(SortedKeys(objKeys), ","))
		}
		return xErr

		/*
			list := ""
			for i, k := range SortedKeys(objKeys) {
				if i > 0 {
					list += ","
				}
				list += where + k
			}
			return NewXRError("unknown_attribute", e.XID,
				"name="+list)
		*/
	}

	return nil
}

// Return: error, haveReplaceValue, newValue
func (e *Entity) ValidateAttribute(val any, attr *Attribute, path *PropPath) (*XRError, bool, any) {
	log.VPrintf(3, ">Enter: ValidateAttribute(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateAttribute")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, " val: %v", ToJSON(val))
		log.VPrintf(0, " attr: %v", ToJSON(attr))
	}

	if attr.Type == ANY {
		// All good - let it thru
		return nil, false, nil
	} else if IsScalar(attr.Type) {
		return e.ValidateScalar(val, attr, path)
	} else if attr.Type == MAP {
		return e.ValidateMap(val, attr.Item, path), false, nil
	} else if attr.Type == ARRAY {
		return e.ValidateArray(attr, val, path), false, nil
	} else if attr.Type == OBJECT {
		/*
			attrs := e.GetBaseAttributes()
			if useNew {
				attrs.AddIfValuesAttributes(e.NewObject)
			} else {
				attrs.AddIfValuesAttributes(e.Object)
			}
		*/

		return e.ValidateObject(val, attr.NameCharSet, attr.Attributes, path),
			false, nil
	}

	ShowStack()
	panic(fmt.Sprintf("Unknown type(%s): %s", path.UI(), attr.Type))
}

func (e *Entity) ValidateMap(val any, item *Item, path *PropPath) *XRError {
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
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail=must be a map")
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type:       item.Type,
		Item:       item.Item,
		Attributes: item.Attributes,
	}

	for _, k := range valValue.MapKeys() {
		if k.Kind() != reflect.String {
			return NewXRError("invalid_attribute",
				"name="+path.RemoveLast().UI(),
				"error_detail="+
					fmt.Sprintf("map key (%s) needs to be a string, "+
						"not %s", path.Last().Text, k.Kind().String()))
		}

		keyName := k.Interface().(string)

		if path.Len() > 0 {
			if xErr := IsValidMapKey(keyName, e.XID, path.UI()); xErr != nil {
				return xErr
			}
		}

		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		xErr, haveReplacement, newValue := e.ValidateAttribute(v, attr,
			path.P(keyName))
		if xErr != nil {
			return xErr
		}
		if haveReplacement {
			valValue.SetMapIndex(k, reflect.ValueOf(newValue))
		}
	}

	return nil
}

func (e *Entity) ValidateArray(arrayAttr *Attribute, val any, path *PropPath) *XRError {
	log.VPrintf(3, ">Enter: ValidateArray(%s)", path)
	defer log.VPrintf(3, "<Exit: ValidateArray")

	if log.GetVerbose() > 2 {
		log.VPrintf(0, "item: %s", ToJSON(arrayAttr.Item))
		log.VPrintf(0, "val: %s", ToJSON(val))
	}

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return NewXRError("invalid_attribute", e.XID,
			"name="+path.UI(),
			"error_detail="+"must be an array")
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type:       arrayAttr.Item.Type,
		Item:       arrayAttr.Item.Item,
		Attributes: arrayAttr.Item.Attributes,
		Enum:       arrayAttr.Enum,
		Strict:     arrayAttr.Strict,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		xErr, haveReplacement, newValue := e.ValidateAttribute(v, attr,
			path.I(i))
		if xErr != nil {
			return xErr
		}
		if haveReplacement {
			valValue.Index(i).Set(reflect.ValueOf(newValue))
		}
	}

	return nil
}

// returns: Error, haveReplacementValue, replacementValue
func (e *Entity) ValidateScalar(val any, attr *Attribute, path *PropPath) (*XRError, bool, any) {
	if log.GetVerbose() > 2 {
		log.VPrintf(0, ">Enter: ValidateScalar(%s:%s)", path, ToJSON(val))
		defer log.VPrintf(3, "<Exit: ValidateScalar")
	}

	replace := false
	newValue := (any)(nil)

	valKind := reflect.ValueOf(val).Kind()

	switch attr.Type {
	case BOOLEAN:
		if valKind != reflect.Bool {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail=must be a boolean"), false, nil
		}
	case DECIMAL:
		if valKind != reflect.Int && valKind != reflect.Float64 {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a decimal"), false, nil
		}
	case INTEGER:
		if valKind == reflect.Float64 {
			f := val.(float64)
			if f != float64(int(f)) {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+"must be an integer"), false, nil
			}
		} else if valKind != reflect.Int {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be an integer"), false, nil
		}
	case UINTEGER:
		i := 0
		if valKind == reflect.Float64 {
			f := val.(float64)
			i = int(f)
			if f != float64(i) {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+"must be a uinteger"), false, nil
			}
		} else if valKind != reflect.Int {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uinteger"), false, nil
		} else {
			i = val.(int)
			if valKind != reflect.Int {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+"must be a uinteger"), false, nil
			}
		}
		if i < 0 {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uinteger"), false, nil
		}
	case XID:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be an xid"), false, nil
		}
		str := val.(string)

		if attr.Target != "" {
			xErr := e.MatchXID(str, attr.Target, attr.Name)
			if xErr != nil {
				return xErr, false, nil
				/*
					return NewXRError("invalid_attribute", e.XID,
					"name=" + path.UI(),
					"error_detail="+err.Error()), false, nil
				*/
			}
		}

		xid, err := ParseXid(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xid, %s",
						str, err)), false, nil
		}

		if xid.VersionID != "" && xid.Version == "meta" {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xid, "+
						"it must be in the form of: "+
						"/[GROUPS[/GID[/RESOURCES[/GID[/versions[/vid]]]]]]",
						str)), false, nil
		}

		if xid.Type != ENTITY_REGISTRY {
			gm := e.Registry.Model.FindGroupModel(xid.Group)
			if gm == nil {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+
						fmt.Sprintf("value (%s) references an unknown "+
							"Group type %q", str, xid.Group)), false, nil
			}

			if xid.Resource != "" {
				rm := gm.FindResourceModel(xid.Resource)
				if rm == nil {
					return NewXRError("invalid_attribute", e.XID,
						"name="+path.UI(),
						"error_detail="+
							fmt.Sprintf("value (%s) references an "+
								"unknown Resource type %q", str,
								xid.Resource)), false, nil
				}
			}
		}

	case XIDTYPE:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value  must be an xidtype")), false, nil
		}
		str := val.(string)

		xidType, err := ParseXidType(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%s) isn't a valid xidtype, %s",
						str, err)), false, nil
		}

		if xidType.Group != "" {
			gm := e.Registry.Model.FindGroupModel(xidType.Group)
			if gm == nil {
				return NewXRError("invalid_attribute", e.XID,
					"name="+path.UI(),
					"error_detail="+
						fmt.Sprintf("value (%s) references an unknown "+
							"Group type %q", str, xidType.Group)), false, nil
			}

			if xidType.Resource != "" {
				rm := gm.FindResourceModel(xidType.Resource)
				if rm == nil {
					return NewXRError("invalid_attribute", e.XID,
						"name="+path.UI(),
						"error_detail="+
							fmt.Sprintf("value (%s) references an "+
								"unknown Resource type %q", str,
								xidType.Resource)), false, nil
				}
			}
		}
	case STRING:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a string"), false, nil
		}
	case URI:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uri"), false, nil
		}
	case URI_REFERENCE:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uri-reference"), false, nil
		}
	case URI_TEMPLATE:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a uri-template"), false, nil
		}
	case URL:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name= "+path.UI(),
				"error_detail="+"must be a url"), false, nil
		}
	case TIMESTAMP:
		if valKind != reflect.String {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"must be a timestamp"), false, nil
		}
		str := val.(string)

		var err error
		newValue, err = NormalizeStrTime(str)
		if err != nil {
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+"is a malformed timestamp"), false, nil
		}
		replace = (newValue != str)
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
			return NewXRError("invalid_attribute", e.XID,
				"name="+path.UI(),
				"error_detail="+
					fmt.Sprintf("value (%v) must be one of the enum "+
						"values: %s", val, valids)), false, nil
		}
	}

	return nil, replace, newValue
}

func PrepUpdateEntity(e *Entity) *XRError {
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

		if attr.InType(e.Type) && attr.internals != nil && attr.internals.updateFn != nil {
			if xErr := attr.internals.updateFn(e); xErr != nil {
				return xErr
			}
		}
	}

	return nil
}

// If no match then return an error saying why
func (e *Entity) MatchXID(str string, xid string, attr string) *XRError {
	// 0=all  1=GROUPS  2=RESOURCES  3=versions|""  4=[/versions]|""
	targetParts := targetRE.FindStringSubmatch(xid)

	if len(str) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, not empty")
	}
	if str[0] != '/' {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, and start with /")
	}
	strParts := strings.Split(str, "/")
	if len(strParts) < 2 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be a valid xid")
	}
	if len(strParts[0]) > 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail=must be an xid, and start with /")
	}
	if xid == "/" {
		if str != "/" {
			return NewXRError("invalid_attribute", e.XID,
				"name="+attr,
				"error_detail="+fmt.Sprintf("must match %q target", xid))
		}
		return nil
	}
	if targetParts[1] != strParts[1] { // works for "" too
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+fmt.Sprintf("must match %q target", xid))
	}

	gm := e.Registry.Model.Groups[targetParts[1]]
	if gm == nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("uses an unknown group %q", targetParts[1]))
	}
	if len(strParts) < 3 || len(strParts[2]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"%sid\"",
					xid, str, gm.Singular))
	}
	if xErr := IsValidID(strParts[2], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the %q ID is not valid: %s`,
					gm.Singular, xErr.Args["error_detail"]))
	}

	if targetParts[2] == "" { // /GROUPS
		if len(strParts) == 3 {
			return nil
		}
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, extra stuff after %q",
					xid, strParts[2]))
	}

	// targetParts has RESOURCES
	if len(strParts) < 4 { //    /GROUPS/GID/RESOURCES
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing %q",
					xid, str, targetParts[2]))
	}

	if targetParts[2] != strParts[3] {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing %q",
					xid, str, targetParts[2]))
	}

	rm := gm.FindResourceModel(targetParts[2])
	if rm == nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("uses an unknown resource %q", targetParts[2]))
	}

	if len(strParts) < 5 || len(strParts[4]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"%sid\"",
					xid, str, rm.Singular))
	}
	if xErr := IsValidID(strParts[4], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the %q ID is not valid: %s`,
					rm.Singular, xErr.Args["error_detail"]))
	}

	if targetParts[3] == "" && targetParts[4] == "" {
		if len(strParts) == 5 {
			return nil
		}
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, extra stuff after %q",
					xid, strParts[4]))

	}

	if targetParts[4] != "" { // has [/versions]
		if len(strParts) == 5 {
			//   /GROUPS/RESOURCES[/version]  vs /GROUPS/GID/RESOURCES/RID
			return nil
		}
	}

	if len(strParts) < 6 || strParts[5] != "versions" {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing \"versions\"",
					xid, str))
	}

	if len(strParts) < 7 || len(strParts[6]) == 0 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, %q is missing a \"version\" ID",
					xid, str))
	}
	if xErr := IsValidID(strParts[6], attr); xErr != nil {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf(`the "version" ID is not valid: %s`,
					xErr.Args["error_detail"]))
	}

	if len(strParts) > 7 {
		return NewXRError("invalid_attribute", e.XID,
			"name="+attr,
			"error_detail="+
				fmt.Sprintf("must match %q target, too long", xid))
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
func CheckAttrs(obj map[string]any, source string) *XRError {
	if obj == nil {
		return nil
	}
	for k, _ := range obj {
		if xErr := IsValidAttributeName(k, source, ""); xErr != nil {
			// log.Printf("Key: %q", k)
			// ShowStack()
			return xErr
		}
	}
	return nil
}
