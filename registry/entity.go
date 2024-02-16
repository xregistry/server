package registry

import (
	"encoding/base64"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
	_ "github.com/go-sql-driver/mysql"
)

type Entity struct {
	RegistrySID string
	DbSID       string // Entity's SID
	Plural      string
	UID         string // Entity's UID
	Props       map[string]any
	Object      map[string]any `json:"-"`
	NewObject   map[string]any `json:"-"` // updated version, save() will store

	// These were added just for convenience and so we can use the same
	// struct for traversing the SQL results
	Level     int // 0=registry, 1=group, 2=resource, 3=version
	Path      string
	Abstract  string
	SkipEpoch bool `json:"-"`
}

type EntitySetter interface {
	Get(name string) any
	Set(name string, val any) error
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

func ToGoType(s string) reflect.Type {
	switch s {
	case ANY:
		return reflect.TypeOf(any(true))
	case BOOLEAN:
		return reflect.TypeOf(true)
	case DECIMAL:
		return reflect.TypeOf(float64(1.1))
	case INTEGER:
		return reflect.TypeOf(int(1))
	case STRING, TIMESTAMP, URI, URI_REFERENCE, URI_TEMPLATE, URL:
		return reflect.TypeOf("")
	case UINTEGER:
		return reflect.TypeOf(uint(0))
	}
	panic("ToGoType - not supported: " + s)
}

func (e *Entity) GetPropFromUI(name string) any {
	pp, err := PropPathFromUI(name)
	PanicIf(err != nil, fmt.Sprintf("%s", err))
	return e.GetPropPP(pp)
}

func (e *Entity) GetPropPP(pp *PropPath) any {
	name := pp.DB()
	if pp.Len() == 1 && pp.Top() == "#resource" {
		// if name == "#resource" {
		results, err := Query(`
            SELECT Content
            FROM ResourceContents
            WHERE VersionSID=? OR
			      VersionSID=(SELECT eSID FROM FullTree WHERE ParentSID=? AND
				  PropName=? and PropValue='true')
			`, e.DbSID, e.DbSID, NewPPP("latest").DB())
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

	val, _ := ObjectGetProp(e.Object, pp)
	log.VPrintf(4, "%s(%s).Get(%s) -> %v", e.Plural, e.UID, name, val)
	return val
}

func ObjectGetProp(obj any, pp *PropPath) (any, error) {
	return NestedGetProp(obj, pp, NewPP())
}

func NestedGetProp(obj any, pp *PropPath, prev *PropPath) (any, error) {
	log.VPrintf(3, "ObjectGetProp: %q\nobj:\n%s", pp.UI(), ToJSON(obj))
	if pp == nil || pp.Len() == 0 {
		return obj, nil
	}
	if IsNil(obj) {
		return nil, fmt.Errorf("Can't traverse into nothing: %s", prev.UI())
	}

	objValue := reflect.ValueOf(obj)
	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		if objValue.Kind() != reflect.Slice {
			return nil, fmt.Errorf("Can't index into non-array: %s", prev.UI())
		}
		if index < 0 || index >= objValue.Len() {
			return nil, fmt.Errorf("Array reference %q out of bounds: "+
				"(max:%d-1)", prev.Append(pp.First()).UI(), objValue.Len())
		}
		objValue = objValue.Index(index)
		if objValue.IsValid() {
			obj = objValue.Interface()
		} else {
			obj = nil
		}
		return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
	}

	// Is map/object
	if objValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("Can't reference a non-map/object: %s",
			prev.UI())
	}
	if objValue.Type().Key().Kind() != reflect.String {
		return nil, fmt.Errorf("Key of %q must be a string, not %s",
			prev.UI(), objValue.Type().Key().Kind())
	}

	objValue = objValue.MapIndex(reflect.ValueOf(pp.Top()))
	if objValue.IsValid() {
		obj = objValue.Interface()
	} else {
		obj = nil
	}
	return NestedGetProp(obj, pp.Next(), prev.Append(pp.First()))
}

func RawEntityFromPath(regID string, path string) (*Entity, error) {
	log.VPrintf(3, ">Enter: RawEntityFromPath(%s)", path)
	defer log.VPrintf(3, "<Exit: RawEntityFromPath")

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9

	results, err := Query(`
		SELECT
            e.RegSID as RegSID,
            e.Level as Level,
            e.Plural as Plural,
            e.eSID as eSID,
            e.UID as UID,
            p.PropName as PropName,
            p.PropValue as PropValue,
            p.PropType as PropType,
            e.Path as Path,
            e.Abstract as Abstract
        FROM Entities AS e
        LEFT JOIN Props AS p ON (e.eSID=p.EntitySID)
        WHERE e.RegSID=? AND e.Path=? ORDER BY Path`, regID, path)
	defer results.Close()

	if err != nil {
		return nil, err
	}

	return readNextEntity(results)
}

func (e *Entity) Find() (bool, error) {
	log.VPrintf(3, ">Enter: Find(%s)", e.UID)
	defer log.VPrintf(3, "<Exit: Find")

	// TODO NEED REGID

	results, err := Query(`
		SELECT
			p.RegistrySID AS RegistrySID,
			p.EntitySID AS DbSID,
			e.Plural AS Plural,
			e.UID AS UID,
			p.PropName AS PropName,
			p.PropValue AS PropValue,
			p.PropType AS PropType
		FROM Props AS p
		LEFT JOIN Entities AS e ON (e.eSID=p.EntitySID)
		WHERE e.UID=?`, e.UID)
	defer results.Close()

	if err != nil {
		return false, err
	}

	first := true
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		if first {
			e.RegistrySID = NotNilString(row[0])
			e.DbSID = NotNilString(row[1])
			e.Plural = NotNilString(row[2])
			e.UID = NotNilString(row[3])
			first = false
		}
	}

	return !first, nil
}

func (e *Entity) Refresh() error {
	log.VPrintf(3, ">Enter: Refresh(%s)", e.DbSID)
	defer log.VPrintf(3, "<Exit: Refresh")

	results, err := Query(`
        SELECT PropName, PropValue, PropType
        FROM Props WHERE EntitySID=? `, e.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error refreshing props(%s): %s", e.DbSID, err)
		return fmt.Errorf("Error refreshing props(%s): %s", e.DbSID, err)
	}

	// Erase all old props first
	e.Props = map[string]any{}
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
	return nil
}

func (e *Entity) Set(path string, val any) error {
	log.VPrintf(3, ">Enter: Set(%s=%v)", path, val)
	defer log.VPrintf(3, "<Exit Set")

	pp, err := PropPathFromUI(path)
	if err == nil {
		err = e.SetPP(pp, val)
	}

	return err
}

func (e *Entity) JustSet(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: JustSet(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit: JustSet")

	// Assume no other edits are pending
	// e.Refresh() // trying not to have this here
	if e.Object == nil {
		e.Object = map[string]any{}
	}
	e.NewObject = maps.Clone(e.Object)

	// Cheat a little just to make caller's life easier
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
	// end of cheat

	if pp.Top() == "epoch" {
		save := e.SkipEpoch
		e.SkipEpoch = true
		defer func() {
			e.SkipEpoch = save
		}()
	}

	log.VPrintf(3, "Abstract/ID: %s/%s", e.Abstract, e.UID)
	log.VPrintf(3, "e.Object:\n%s", ToJSON(e.Object))
	log.VPrintf(3, "e.NewObject:\n%s", ToJSON(e.NewObject))

	if err := e.SetPPValidate(pp, val, true, nil); err != nil {
		return err
	}

	return ObjectSetProp(e.NewObject, pp, val)
}

func (e *Entity) ValidateAndSave() error {
	log.VPrintf(3, ">Enter: ValidateAndSave")
	defer log.VPrintf(3, "<Exit: ValidateAndSave")

	log.VPrintf(3, "e.NewObject:\n%s", ToJSON(e.NewObject))

	if err := e.Validate(true); err != nil {
		return err
	}

	if err := PrepUpdateEntity(e, false); err != nil {
		return err
	}

	return e.Save()
}

func (e *Entity) SetPP(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetPP(%s: %s=%v)", e.DbSID, pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")
	defer func() {
		log.VPrintf(3, "SetPP exit: e.Object:\n%s", ToJSON(e.Object))
	}()

	if err := e.JustSet(pp, val); err != nil {
		return err
	}

	// Make the bold assumption that we we're setting and saving all in one
	// that a user who is explicitly setting 'epoc' via an interenal
	// set() knows what they're doing
	save := e.SkipEpoch
	e.SkipEpoch = true
	defer func() { e.SkipEpoch = save }()

	return e.ValidateAndSave()
}

func (e *Entity) SetPPValidate(pp *PropPath, val any, validate bool, obj map[string]any) error {
	log.VPrintf(3, ">Enter: SetPP(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetPP")

	name := pp.DB()

	// Make sure the attribute is defined in the model and has valid chars
	attrType, err := GetAttributeType(e, e.RegistrySID, obj, e.Abstract, pp)
	if err != nil {
		return err
	}
	if attrType == "" {
		return fmt.Errorf("Can't find attribute %q", pp.UI())
	}

	if !IsNil(val) && validate {
		if err = ValidatePropValue(val, attrType); err != nil {
			return fmt.Errorf("%q: %s", pp.UI(), err)
		}
	}

	if val == nil {
		delete(e.Props, name)
	} else {
		if name == "#resource," {
			val = ""
		}
		if e.Props == nil {
			e.Props = map[string]any{}
		}
		e.Props[name] = val
	}

	return nil
}

func (e *Entity) SetDBProperty(pp *PropPath, val any) error {
	log.VPrintf(3, ">Enter: SetDBProperty(%s=%v)", pp.UI(), val)
	defer log.VPrintf(3, "<Exit SetDBProperty")

	var err error
	name := pp.DB()

	// Any prop with "dontStore"=true we skip
	if sp, ok := SpecProps[pp.Top()]; ok && sp.dontStore {
		return nil
	}

	PanicIf(e.DbSID == "", "DbSID should not be empty")
	PanicIf(e.RegistrySID == "", "RegistrySID should not be empty")

	// #resource is special and is saved in it's own table
	// Need to explicitly set #resoure to nil to delete it.
	if pp.Len() == 1 && pp.Top() == "#resource" {
		if IsNil(val) {
			err = Do(`DELETE FROM ResourceContents WHERE VersionSID=?`, e.DbSID)
			return err
		} else {
			if val == "" {
				return nil
			}
			// The actual contents
			err = DoOneTwo(`
                REPLACE INTO ResourceContents(VersionSID, Content)
            	VALUES(?,?)`, e.DbSID, val)
			if err != nil {
				return err
			}
			val = ""
			// Fall thru to normal processing so we save a placeholder
			// attribute in the resource
		}
	}

	if IsNil(val) {
		// Should never use this but keeping it just in case
		err = Do(`DELETE FROM Props WHERE EntitySID=? and PropName=?`,
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

		err = DoOneTwo(`
            REPLACE INTO Props(
              RegistrySID, EntitySID, PropName, PropValue, PropType)
            VALUES( ?,?,?,?,? )`,
			e.RegistrySID, e.DbSID, name, dbVal, propType)
	}

	if err != nil {
		log.Printf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
		return fmt.Errorf("Error updating prop(%s/%v): %s", pp.UI(), val, err)
	}

	return nil
}

func (e *Entity) SetFromDBName(name string, val *string, propType string) error {
	pp := MustPropPathFromDB(name)

	if val == nil {
		delete(e.Props, name)
		return ObjectSetProp(e.Object, pp, val)
	}
	if e.Props == nil {
		e.Props = map[string]any{}
	}
	if e.Object == nil {
		e.Object = map[string]any{}
	}

	if propType == STRING || propType == URI || propType == URI_REFERENCE ||
		propType == URI_TEMPLATE || propType == URL || propType == TIMESTAMP {
		e.Props[name] = *val
		return ObjectSetProp(e.Object, pp, *val)
	} else if propType == BOOLEAN {
		// Technically "1" check shouldn't be needed, but just in case
		e.Props[name] = (*val == "1") || (*val == "true")
		return ObjectSetProp(e.Object, pp, (*val == "1" || (*val == "true")))
	} else if propType == INTEGER || propType == UINTEGER {
		tmpInt, err := strconv.Atoi(*val)
		if err != nil {
			panic(fmt.Sprintf("error parsing int: %s", *val))
		}
		e.Props[name] = tmpInt
		return ObjectSetProp(e.Object, pp, tmpInt)
	} else if propType == DECIMAL {
		tmpFloat, err := strconv.ParseFloat(*val, 64)
		if err != nil {
			panic(fmt.Sprintf("error parsing float: %s", *val))
		}
		e.Props[name] = tmpFloat
		return ObjectSetProp(e.Object, pp, tmpFloat)
	} else if propType == MAP {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = map[string]any{}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else if propType == ARRAY {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = []any{}
		return ObjectSetProp(e.Object, pp, []any{})
	} else if propType == OBJECT {
		if *val != "" {
			panic(fmt.Sprintf("MAP value should be empty string"))
		}
		e.Props[name] = map[string]any{}
		return ObjectSetProp(e.Object, pp, map[string]any{})
	} else {
		panic(fmt.Sprintf("bad type(%s): %v", propType, name))
	}
}

// This validates a single attribute (leaf) of the object.
// That's why it only supports empty maps/arrays/objects as values.
// It assumes the caller has walked down to a leaf/attribute.
func ValidatePropValue(val any, attrType string) error {
	vKind := reflect.ValueOf(val).Kind()

	switch attrType {
	case ANY:
		return nil
	case BOOLEAN:
		if vKind != reflect.Bool {
			return fmt.Errorf(`"%v" should be a boolean`, val)
		}
	case DECIMAL:
		if vKind != reflect.Int && vKind != reflect.Float64 {
			return fmt.Errorf(`"%v" should be a decimal`, val)
		}
	case INTEGER:
		if vKind == reflect.Float64 {
			f := val.(float64)
			if f != float64(int(f)) {
				return fmt.Errorf(`"%v" must be an integer`, val)
			}
			return nil
		}
		if vKind != reflect.Int {
			return fmt.Errorf(`"%v" should be an integer`, val)
		}
	case UINTEGER:
		i := 0
		if vKind == reflect.Float64 {
			f := val.(float64)
			i = int(f)
			if f != float64(i) {
				return fmt.Errorf("%q must be a uinteger", val)
			}
		} else {
			i = val.(int)
			if vKind != reflect.Int {
				return fmt.Errorf("%q must be a uinteger", val)
			}
		}
		if i < 0 {
			return fmt.Errorf(`"%v" should be a uinteger`, val)
		}
	case STRING, URI, URI_REFERENCE, URI_TEMPLATE, URL: // cheat
		if vKind != reflect.String {
			return fmt.Errorf(`"%v" should be a string`, val)
		}
	case TIMESTAMP:
		if vKind != reflect.String {
			return fmt.Errorf(`"%v" should be a timestamp`, val)
		}
		str := val.(string)
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return fmt.Errorf("Malformed timestamp %q: %s", str, err)
		}

	// For the non-scalar types, these should only be used when someone
	// passing in something like:
	//    "foo": {}
	// and we need to save an empty (non-scalar) value. Hence the "if" below.
	case MAP:
		// anything but an empty map means we did something wrong before this
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Map || v.Len() > 0 {
			return fmt.Errorf(`%q must be an empty map`, val)
		}
		val = ""

	case ARRAY:
		// anything but an empty array means we did something wrong before this
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Slice || v.Len() > 0 {
			return fmt.Errorf(`%q must be an empty array`, val)
		}
		val = ""

	case OBJECT:
		// Anything but an empty struct means we did something wrong before this
		// Allow for a map since we can't tell sometimes
		v := reflect.ValueOf(val)
		if (v.Kind() != reflect.Struct || v.NumField() > 0) &&
			(v.Kind() != reflect.Map || v.Len() > 0) {
			// ShowStack()
			return fmt.Errorf(`%q must be an empty object`, val)
		}
		val = ""

	default:
		ShowStack()
		log.Printf("AttrType: %q  Val: %#q", attrType, val)
		return fmt.Errorf("unsupported type: %s", attrType)
	}
	return nil
}

func readNextEntity(results *Result) (*Entity, error) {
	entity := (*Entity)(nil)

	// RegSID,Level,Plural,eSID,UID,PropName,PropValue,PropType,Path,Abstract
	//   0     1      2     3    4     5         6         7     8      9
	for row := results.NextRow(); row != nil; row = results.NextRow() {
		// log.Printf("Row(%d): %#v", len(row), row)
		level := int((*row[1]).(int64))
		plural := NotNilString(row[2])
		uid := NotNilString(row[4])

		if entity == nil {
			entity = &Entity{
				RegistrySID: NotNilString(row[0]),
				DbSID:       NotNilString(row[3]),
				Plural:      plural,
				UID:         uid,
				Props:       map[string]any{},

				Level:    level,
				Path:     NotNilString(row[8]),
				Abstract: NotNilString(row[9]),
			}
		} else {
			// If the next row isn't part of the current Entity then
			// push it back into the result set so we'll grab it the next time
			// we're called. And exit.
			if entity.Level != level || entity.Plural != plural || entity.UID != uid {
				results.Push()
				break
			}
		}

		propName := NotNilString(row[5])
		propVal := NotNilString(row[6])
		propType := NotNilString(row[7])

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

type SpecProp struct {
	name      string // prop name
	daType    string
	levels    string // only show for these levels
	mutable   bool   // user editable
	dontStore bool
	getFn     func(*Entity, *RequestInfo) any // return its value
	checkFn   func(e *Entity) error
	// prep newObj for an update to the DB
	updateFn       func(*Entity, bool) error
	modelAttribute *Attribute
}

func (sp *SpecProp) InLevel(level int) bool {
	return sp.levels == "" || strings.ContainsRune(sp.levels, rune('0'+level))
}

// This allows for us to choose the order and define custom logic per prop
var OrderedSpecProps = []*SpecProp{
	{"specversion", STRING, "0", false, false,
		func(e *Entity, info *RequestInfo) any {
			return SPECVERSION
		},
		func(e *Entity) error {
			tmp := e.NewObject["specversion"]
			if !IsNil(tmp) && tmp != "" && tmp != SPECVERSION {
				return fmt.Errorf("Invalid 'specversion': %s", tmp)
			}
			return nil
		},
		nil,
		&Attribute{
			Name:           "specversion",
			Type:           STRING,
			ServerRequired: true,
			ReadOnly:       true,
		}},
	{"id", STRING, "", false, false, nil,
		func(e *Entity) error {
			if e.Object != nil {
				oldID := any(e.Object["id"])
				newID := any(e.NewObject["id"])

				if IsNil(oldID) {
					oldID = ""
				}
				if IsNil(newID) {
					newID = ""
				}

				if newID != "" && oldID != "" && newID != oldID {
					return fmt.Errorf("Can't change the ID of an "+
						"entity(%s->%s)", oldID, newID)
				}
				/*
					v := e.NewObject["id"]
					if !IsNil(v) && v != e.Object["id"] {
						return fmt.Errorf("Can't change the ID of an "+
							"entity(%v->%s)", e.Object["id"], v)
					}
				*/
			}
			return nil
		},
		func(e *Entity, isNew bool) error {
			if e.Object != nil {
				if IsNil(e.NewObject["id"]) && !IsNil(e.Object["id"]) {
					e.NewObject["id"] = e.Object["id"]
					return nil
				}
			}
			return nil
		},
		&Attribute{
			Name:           "id",
			Type:           STRING,
			ServerRequired: true,
		}},
	{"name", STRING, "", true, false, nil, nil, nil, &Attribute{
		Name: "name",
		Type: STRING,
	}},
	{"epoch", UINTEGER, "", false, false, nil,
		func(e *Entity) error {
			if e.SkipEpoch {
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

			if oldEpoch != 0 && newEpoch != oldEpoch {
				return fmt.Errorf("Attribute %q(%d) doesn't match existing "+
					"value (%d)", "epoch", newEpoch, oldEpoch)
			}
			return nil
		},
		func(e *Entity, isNew bool) error {
			if e.SkipEpoch {
				return nil
			}
			tmp := e.Object["epoch"]
			if IsNil(tmp) {
				return nil
			}
			epoch := NotNilInt(&tmp)
			if epoch < 0 {
				epoch = 0
			}
			if isNew {
				epoch = 0
			}
			e.NewObject["epoch"] = epoch + 1
			return nil
		},
		&Attribute{
			Name:     "epoch",
			Type:     UINTEGER,
			ReadOnly: true,
		}},
	{"self", STRING, "", false, false, func(e *Entity, info *RequestInfo) any {
		base := ""
		if info != nil {
			base = info.BaseURL
		}
		if e.Level > 1 {
			if info != nil && (info.ShowMeta || info.ResourceUID == "") {
				return base + "/" + e.Path + "?meta"
			} else {
				return base + "/" + e.Path
			}
		}
		return base + "/" + e.Path
	}, nil, nil, &Attribute{
		Name:           "self",
		Type:           STRING,
		ServerRequired: true,
		ReadOnly:       true,
	}},
	{"latest", BOOLEAN, "3", false, false, nil, nil,
		func(e *Entity, isNew bool) error {
			// TODO is set, set latestvesionid in the resource to this
			// guy's UID

			return nil
		},
		&Attribute{
			Name: "latest",
			Type: BOOLEAN,
		}},
	{"latestversionid", STRING, "2", false, false, nil, nil, nil, &Attribute{
		Name:           "latestversionid",
		Type:           STRING,
		ServerRequired: true,
		ReadOnly:       true,
	}},
	{"latestversionurl", URL, "2", false, false,
		func(e *Entity, info *RequestInfo) any {
			val := e.Object["latestversionid"]
			if IsNil(val) {
				return nil
			}
			base := ""
			if info != nil {
				base = info.BaseURL
			}

			tmp := base + "/" + e.Path + "/versions/" + val.(string)
			if info != nil && (info.ShowMeta || info.ResourceUID == "") {
				tmp += "?meta"
			}
			return tmp
		}, nil, nil, &Attribute{
			Name:           "latestversionurl",
			Type:           URL,
			ServerRequired: true,
			ReadOnly:       true,
		}},
	{"description", STRING, "", true, false, nil, nil, nil, &Attribute{
		Name: "description",
		Type: STRING,
	}},
	{"documentation", STRING, "", true, false, nil, nil, nil, &Attribute{
		Name: "documentation",
		Type: STRING,
	}},
	{"labels", MAP, "", true, false, nil, nil, nil, &Attribute{
		Name: "labels",
		Type: MAP,
		Item: &Item{
			Type: STRING,
		},
	}},
	{"origin", URI, "123", true, false, nil, nil, nil, &Attribute{
		Name: "origin",
		Type: STRING,
	}},
	{"createdby", STRING, "", false, false, nil, nil, nil, &Attribute{
		Name:     "createdby",
		Type:     STRING,
		ReadOnly: true,
	}},
	{"createdon", TIMESTAMP, "", false, false, nil, nil, nil, &Attribute{
		Name:     "createdon",
		Type:     TIMESTAMP,
		ReadOnly: true,
	}},
	{"modifiedby", STRING, "", false, false, nil, nil, nil, &Attribute{
		Name:     "modifiedby",
		Type:     STRING,
		ReadOnly: true,
	}},
	{"modifiedon", TIMESTAMP, "", false, false, nil, nil, nil, &Attribute{
		Name:     "modifiedon",
		Type:     TIMESTAMP,
		ReadOnly: true,
	}},
	{"model", OBJECT, "0", false, false,
		func(e *Entity, info *RequestInfo) any {
			if info != nil && info.ShowModel {
				model := info.Registry.Model
				if model == nil {
					model = &Model{}
				}
				httpModel := model // ModelToHTTPModel(model)
				return httpModel
			}
			return nil
		}, nil, nil, &Attribute{
			Name:     "model",
			Type:     ANY,
			ReadOnly: true,
		}},
}

var SpecProps = map[string]*SpecProp{}

func init() {
	// Load map via lower-case version of prop name
	for _, sp := range OrderedSpecProps {
		SpecProps[strings.ToLower(sp.name)] = sp
		PanicIf(sp.modelAttribute != nil && sp.name != sp.modelAttribute.Name,
			"Key & name mismatch in OrderedSpecProps: %s", sp.name)
		if sp.checkFn != nil && sp.modelAttribute != nil {
			sp.modelAttribute.checkFn = sp.checkFn
			sp.modelAttribute.updateFn = sp.updateFn
		}
	}
}

// This is used to serialize Prop regardless of the format.
func (e *Entity) SerializeProps(info *RequestInfo,
	fn func(*Entity, *RequestInfo, string, any, *Attribute) error) error {

	daObj := e.Materialize(info)
	attrs := e.GetAttributes(false)

	// Do spec defined props first, in order
	for _, prop := range OrderedSpecProps {
		attr, ok := attrs[prop.name]
		if !ok {
			delete(daObj, prop.name)
			continue // not allowed at this level so skip it
		}

		if val, ok := daObj[prop.name]; ok {
			if err := fn(e, info, prop.name, val, attr); err != nil {
				return err
			}
			delete(daObj, prop.name)
		}
	}

	// Now do all other props (extensions) alphabetically
	for _, key := range SortedKeys(daObj) {
		val, _ := daObj[key]
		attr := attrs[key]
		if attr == nil {
			attr = attrs["*"]
			PanicIf(key[0] != '#' && attr == nil, "Can't find attr for %q", key)
		}

		if err := fn(e, info, key, val, attr); err != nil {
			return err
		}
	}

	return nil
}

func (e *Entity) Save() error {
	log.VPrintf(3, ">Enter: Save(%s/%s)", e.Plural, e.UID)
	defer log.VPrintf(3, "<Exit: Save")

	log.VPrintf(3, "Saving - %s (id:%s):\n%s\n", e.Abstract, e.UID,
		ToJSON(e.NewObject))

	// make a dup so we can delete some attributes
	newObj := maps.Clone(e.NewObject)

	// TODO calculate which to delete based on attr properties
	delete(newObj, "self")

	for _, coll := range e.GetCollections() {
		delete(newObj, coll)
		delete(newObj, coll+"count")
		delete(newObj, coll+"url")
	}

	if err := Do(`DELETE FROM Props WHERE EntitySID=?`, e.DbSID); err != nil {
		log.Printf("Error deleting all props %s", err)
		return fmt.Errorf("Error deleting all prop: %s", err)
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
				if k[0] == '#' {
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
			if count == 0 {
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

	err := traverse(NewPP(), newObj, e.NewObject)
	if err == nil {
		e.Object = newObj
		e.NewObject = nil
	}
	return err
}

// Note that this will copy the latest version props to the resource.
// This is mainly used for end-user facing serialization of the entity
func (e *Entity) Materialize(info *RequestInfo) map[string]any {
	mat := maps.Clone(e.Object)

	// Copy all Version props into the Resource (except for a few)
	if e.Level == 2 {
		// On Resource grab latest Version attrs
		paths := strings.Split(e.Path, "/")
		reg, _ := FindRegistryBySID(e.RegistrySID)
		group, _ := reg.FindGroup(paths[0], paths[1])
		resource, _ := group.FindResource(paths[2], paths[3])
		ver, _ := resource.GetLatest()

		if ver != nil { // can be nil during resource.create()
			// Copy version specific attributes not found in Resources
			for k, v := range ver.Object {
				if k == "id" { // Retain Resource ID
					continue
				}
				// exclude props that appear in vers, not resource.latest
				if prop, ok := SpecProps[k]; ok {
					if prop.InLevel(3) && !prop.InLevel(2) {
						continue
					}
				}

				mat[k] = v
			}
		}
	}

	// Regardless of the type of entity, set the generated properties
	for _, prop := range OrderedSpecProps {
		// Only generate props that are for this level, and have a Fn
		if prop.getFn == nil || !prop.InLevel(e.Level) {
			continue
		}

		// Only generate/set the value if it's not already set
		if _, ok := mat[prop.name]; !ok {
			if val := prop.getFn(e, info); !IsNil(val) {
				// Only write it if we have a value
				mat[prop.name] = val
			}
		}
	}

	return mat
}

func (e *Entity) GetCollections() []string {
	reg, err := FindRegistryBySID(e.RegistrySID)
	PanicIf(reg == nil, "Can't find registry(%s): %s", e.RegistrySID, err)

	paths := strings.Split(e.Abstract, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		return SortedKeys(reg.Model.Groups)
	} else {
		gm := reg.Model.Groups[paths[0]]
		PanicIf(gm == nil, "Can't find Group %q", paths[0])

		if len(paths) == 1 {
			return SortedKeys(gm.Resources)
		} else if len(paths) == 2 {
			return []string{"versions"}
		}
	}

	return nil
}

func (e *Entity) GetAttributes(useNew bool) Attributes {
	attrs := e.GetBaseAttributes()
	if useNew {
		attrs.AddIfValueAttributes(e.NewObject)
	} else {
		attrs.AddIfValueAttributes(e.Object)
	}
	return attrs
}

// Returns the initial set of attributes defined for the entity. So
// no IfValue attributes yet as we need the current set of properties
// to calculate that
func (e *Entity) GetBaseAttributes() Attributes {
	reg, err := FindRegistryBySID(e.RegistrySID)
	Must(err)

	attrs := Attributes{}
	level := 0
	singular := ""

	// Add user-defined attributes
	// TODO check for conflicts with xReg defined ones
	paths := strings.Split(e.Abstract, string(DB_IN))
	if len(paths) == 0 || paths[0] == "" {
		maps.Copy(attrs, reg.Model.Attributes)
	} else {
		level = len(paths)
		gm := reg.Model.Groups[paths[0]]
		PanicIf(gm == nil, "Can't find Group %q", paths[0])
		if len(paths) == 1 {
			maps.Copy(attrs, gm.Attributes)
		} else {
			rm := gm.Resources[paths[1]]
			PanicIf(rm == nil, "Cant find Resource %q", paths[1])
			maps.Copy(attrs, rm.Attributes)
			singular = rm.Singular
		}
	}

	// Add xReg defied attributes
	// TODO Check for conflicts
	for _, specProp := range OrderedSpecProps {
		if specProp.InLevel(level) {
			if specProp.modelAttribute != nil {
				attrs[specProp.name] = specProp.modelAttribute
			}
		}
	}

	// Add the RESOURCExxx attributes (for resources and versions)
	if singular != "" {
		checkFn := func(e *Entity) error {
			list := []string{
				singular,
				singular + "url",
				singular + "base64",
				singular + "proxyurl",
			}
			count := 0
			for _, name := range list {
				if v, ok := e.NewObject[name]; ok && !IsNil(v) {
					count++
				}
			}
			if count > 1 {
				return fmt.Errorf("Only one of %s can be present at a time",
					strings.Join(list[:3], ",")) // exclude proxy
			}
			return nil
		}

		// Add resource content attributes
		attrs[singular] = &Attribute{
			Name:    singular,
			Type:    ANY,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular]
				if ok {
					e.NewObject["#resource"] = v
					// e.NewObject["#resourceURL"] = nil
					delete(e.NewObject, singular)
				}
				return nil
			},
		}
		attrs[singular+"url"] = &Attribute{
			Name:    singular + "url",
			Type:    URL,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"url"]
				if !ok {
					return nil
				}
				e.NewObject["#resource"] = nil
				e.NewObject["#resourceURL"] = v
				delete(e.NewObject, singular+"url")
				return nil
			},
		}
		attrs[singular+"proxyurl"] = &Attribute{
			Name:    singular + "proxyurl",
			Type:    URL,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"proxyurl"]
				if !ok {
					return nil
				}
				e.NewObject["#resource"] = nil
				e.NewObject["#resourceProxyURL"] = v
				delete(e.NewObject, singular+"proxyurl")
				return nil
			},
		}
		attrs[singular+"base64"] = &Attribute{
			Name:    singular + "base64",
			Type:    STRING,
			checkFn: checkFn,
			updateFn: func(e *Entity, isNew bool) error {
				v, ok := e.NewObject[singular+"base64"]
				if !ok {
					return nil
				}
				if !IsNil(v) {
					data := v.(string)
					content, err := base64.StdEncoding.DecodeString(data)
					if err != nil {
						return fmt.Errorf("Error decoding \"%sbase64\" "+
							"attribute: "+"%s", singular, err)
					}
					v = any(content)
				}
				e.NewObject["#resource"] = v
				// e.NewObject["#resourceURL"] = nil
				delete(e.NewObject, singular+"base64")
				return nil
			},
		}
	}

	return attrs
}

func ObjectSetProp(obj map[string]any, pp *PropPath, val any) error {
	// TODO see if we can move this into MaterializeProp
	if pp.Len() == 0 && IsNil(val) {
		// A bit of a special case, not 100% sure if this is ok
		for k, _ := range obj {
			delete(obj, k)
		}
		return nil
	}
	PanicIf(pp.Len() == 0, "Can't be zero w/non-nil val")

	_, err := MaterializeProp(obj, pp, val)
	if err != nil {
		return err
	}
	return nil
}

func MaterializeProp(current any, pp *PropPath, val any) (any, error) {
	// current is existing value, used for adding to maps/arrays
	if pp == nil {
		return val, nil
	}

	var ok bool
	var err error

	part := pp.Parts[0]
	if index := part.Index; index >= 0 {
		// Is an array
		// TODO look for cases where Kind(val) == array too - maybe?
		var daArray []any

		if current != nil {
			daArray, ok = current.([]any)
			if !ok {
				return nil, fmt.Errorf("Current isn't an array: %T", current)
			}
		}

		// Resize if needed
		if diff := (1 + index - len(daArray)); diff > 0 {
			daArray = append(daArray, make([]any, diff)...)
		}

		// Trim the end of the array if there are nil's
		daArray[index], err = MaterializeProp(daArray[index], pp.Next(), val)
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

	res, err := MaterializeProp(daMap[pp.Top()], pp.Next(), val)
	if err != nil {
		return nil, err
	}
	if IsNil(res) {
		delete(daMap, pp.Top())
	} else {
		daMap[pp.Top()], err = MaterializeProp(daMap[pp.Top()], pp.Next(), val)
	}

	return daMap, err
}

// Doesn't fully validate in the sense that it'll assume read-only fields
// are not worth cheching since the server generated them.
// This is mainly used for validating input from a client
func (e *Entity) Validate(useNew bool) error {
	// Don't touch what was passed in
	dupObj := (map[string]any)(nil)
	if useNew {
		dupObj = maps.Clone(e.NewObject)
	} else {
		dupObj = maps.Clone(e.Object)
	}

	for _, coll := range e.GetCollections() {
		log.VPrintf(3, "Deleting collection: %q", coll)
		delete(dupObj, coll)
		delete(dupObj, coll+"count")
		delete(dupObj, coll+"url")
	}

	attrs := e.GetAttributes(true)
	log.VPrintf(3, "========")
	log.VPrintf(3, "Validating:\n%s", ToJSON(dupObj))
	return ValidateObject(e, dupObj, e.Object, attrs, NewPP())
}

func PrepUpdateEntity(e *Entity, isNew bool) error {
	attrs := e.GetAttributes(true)

	for key, _ := range attrs {
		attr := attrs[key]
		if attr != nil && attr.updateFn != nil {
			if err := attr.updateFn(e, isNew); err != nil {
				return err
			}
		}
	}

	return nil
}

// This should be called after all level-specific calculated properties have
// been removed - such as collections
func ValidateObject(e *Entity, val any, oldObj map[string]any,
	origAttrs Attributes, path *PropPath) error {

	log.VPrintf(3, ">Enter: ValidateObject(path: %s)", path)
	defer log.VPrintf(3, "<Exit: ValidateObject")

	log.VPrintf(3, "NewObj:\n%s", ToJSON(val))
	log.VPrintf(3, "OrigAttrs:\n%s", ToJSON(origAttrs))

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map ||
		valValue.Type().Key().Kind() != reflect.String {

		return fmt.Errorf("Attribute %q must be an object", path.UI())
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
	objKeys := map[string]bool{}
	for k, _ := range newObj {
		objKeys[k] = true
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
			val, ok := newObj[key]

			// Based on the attribute's type check the incoming 'val'.
			// This will check for adherence to the model (eg type),
			// the next section (checkFn) will allow for more detailed
			// checking, like for valid values
			if !IsNil(val) {
				err := ValidateAttribute(e, val, attr, path.P(key))
				if err != nil {
					return err
				}
			}

			// GetAttributes already added IfValue for top-level attributes
			if path.Len() > 1 && len(attr.IfValue) > 0 {
				valStr := fmt.Sprintf("%v", val)
				for ifValStr, ifValueData := range attr.IfValue {
					if valStr != ifValStr {
						continue
					}

					for _, newAttr := range ifValueData.SiblingAttributes {
						if _, ok := allAttrNames[newAttr.Name]; ok {
							return fmt.Errorf(`Attribute %q has an "ifvalue"`+
								`(%s) that conflicts with an existing `+
								`attribute`, path.P(key).UI(), newAttr.Name)
						}
						if newAttr.Name == "*" {
							attrs = append([]*Attribute{newAttr}, attrs...)
						} else {
							attrs = append(attrs, newAttr)
						}
						allAttrNames[newAttr.Name] = true
					}
				}
			}

			// We normally skip read-only attrs, but if it has a checkFn
			// then allow for that to be called
			if attr.ReadOnly {
				// Call the attr's checkFn if there
				if attr.checkFn != nil {
					if err := attr.checkFn(e); err != nil {
						return err
					}
				}

				delete(objKeys, key)
				continue
			}

			if attr.ClientRequired && !ok { // Required but not present
				return fmt.Errorf("Required property %q is missing",
					path.P(key).UI())
			}

			if !attr.ClientRequired && (!ok || IsNil(val)) { // treat nil as absent
				delete(objKeys, key)
				continue
			}

			// Call the attr's checkFn if there - for more refined checks
			if attr.checkFn != nil {
				if err := attr.checkFn(e); err != nil {
					return err
				}
			}

			// Everything is good, so remove it
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

func ValidateAttribute(e *Entity, val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateAttribute(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateAttribute")

	log.VPrintf(3, "ValidateAttribute:")
	log.VPrintf(3, " val: %v", ToJSON(val))
	log.VPrintf(3, " attr: %v", ToJSON(attr))

	if attr.Type == ANY {
		// All good - let it thru
		return nil
	} else if IsScalar(attr.Type) {
		return ValidateScalar(e, val, attr, path)
	} else if attr.Type == MAP {
		return ValidateMap(e, val, attr.Item, path)
	} else if attr.Type == ARRAY {
		return ValidateArray(e, val, attr.Item, path)
	} else if attr.Type == OBJECT {
		attrs := Attributes(nil)
		if attr.Item != nil {
			attrs = attr.Item.Attributes
		}
		return ValidateObject(e, val, nil, attrs, path)
	}

	ShowStack()
	panic(fmt.Sprintf("Unknown type(%s): %s", path.UI(), attr.Type))
}

func ValidateMap(e *Entity, val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateMap(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateMap")

	log.VPrintf(3, " item: %v", ToJSON(item))
	log.VPrintf(3, " val: %v", ToJSON(val))

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Map {
		return fmt.Errorf("Attribute %q must be a map", path.UI())
	}

	// All values in the map must be of the same type
	attr := &Attribute{
		Type: item.Type,
		Item: item,
	}

	for _, k := range valValue.MapKeys() {
		keyName := k.Interface().(string)
		v := valValue.MapIndex(k).Interface()
		if IsNil(v) {
			continue
		}
		if err := ValidateAttribute(e, v, attr, path.P(keyName)); err != nil {
			return err
		}
	}

	return nil
}

func ValidateArray(e *Entity, val any, item *Item, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateArray(%s)", path.UI())
	defer log.VPrintf(3, "<Exit: ValidateArray")

	log.VPrintf(3, "item: %s", ToJSON(item))
	log.VPrintf(3, "val: %s", ToJSON(val))

	if IsNil(val) {
		return nil
	}

	valValue := reflect.ValueOf(val)
	if valValue.Kind() != reflect.Slice {
		return fmt.Errorf("Attribute %q must be an array", path.UI())
	}

	// All values in the array must be of the same type
	attr := &Attribute{
		Type: item.Type,
		Item: item.Item,
	}

	for i := 0; i < valValue.Len(); i++ {
		v := valValue.Index(i).Interface()
		if err := ValidateAttribute(e, v, attr, path.I(i)); err != nil {
			return err
		}
	}

	return nil
}

func ValidateScalar(e *Entity, val any, attr *Attribute, path *PropPath) error {
	log.VPrintf(3, ">Enter: ValidateScalar(%s:%s)", path.UI(), ToJSON(val))
	defer log.VPrintf(3, "<Exit: ValidateScalar")

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
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return fmt.Errorf("Attribute %q is a malformed timestamp",
				path.UI())
		}
	}

	// don't "return nil" above, we may need to check enum values
	if len(attr.Enum) > 0 && attr.Strict {
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
