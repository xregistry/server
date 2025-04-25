package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/duglin/dlog"
)

var RegexpModelName = regexp.MustCompile("^[a-z_][a-z_0-9]{0,57}$")
var RegexpPropName = regexp.MustCompile("^[a-z_][a-z_0-9]{0,62}$")
var RegexpMapKey = regexp.MustCompile("^[a-z0-9][a-z0-9_.:\\-]{0,62}$")
var RegexpID = regexp.MustCompile("^[a-zA-Z0-9_][a-zA-Z0-9_.\\-~@]{0,127}$")

type ModelSerializer func(*Model, string) ([]byte, error)

var ModelSerializers = map[string]ModelSerializer{}

func IsValidModelName(name string) error {
	if RegexpModelName.MatchString(name) {
		return nil
	}
	return fmt.Errorf("Invalid model type name %q, must match: %s",
		name, RegexpModelName.String())
}

func IsValidAttributeName(name string) error {
	if RegexpPropName.MatchString(name) {
		return nil
	}
	return fmt.Errorf("Invalid attribute name %q, must match: %s",
		name, RegexpPropName.String())
}

func IsValidMapKey(key string) error {
	if RegexpMapKey.MatchString(key) {
		return nil
	}
	return fmt.Errorf("Invalid map key name %q, must match: %s",
		key, RegexpMapKey.String())
}

func IsValidID(id string) error {
	if RegexpID.MatchString(id) {
		return nil
	}
	return fmt.Errorf("Invalid ID %q, must match: %s",
		id, RegexpID.String())
}

type Model struct {
	Registry   *Registry              `json:"-"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Attributes Attributes             `json:"attributes,omitempty"`
	Groups     map[string]*GroupModel `json:"groups,omitempty"` // Plural

	attrsOrdered []*Attribute
}

type Attributes map[string]*Attribute // AttrName->Attr

// Defined a separate struct instead of just inlining these attributes so
// that we can just copy them over in one statement in SetSpecPropsFields()
// and so that if we add more we don't need to remember to update that func
type AttrInternals struct {
	types           string // show only for these eTypes, ""==all
	dontStore       bool   // don't store this prop in the DB
	alwaysSerialize bool   // even if nil
	neverSerialize  bool   // hidden attr
	httpHeader      string // custom HTTP header name, not xRegistry-xxx
	xrefrequired    bool   // required in meta even when xref is set

	getFn    func(*Entity, *RequestInfo) any // return prop's value
	checkFn  func(*Entity) error             // validate incoming prop
	updateFn func(*Entity) error             // prep prop for saving to DB

	// Used by $singular and $order
	singular     string
	orderedProps []*Attribute
}

// Do not include "omitempty" on any attribute that has a default value that
// doesn't match golang's default value for that type. E.g. bool defaults to
// 'false', but Strict needs to default to 'true'. See the custome Unmarshal
// funcs in model.go for how we set those
type Attribute struct {
	Model       *Model `json:"-"`
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Target      string `json:"target,omitempty"`
	NameCharSet string `json:"namecharset,omitempty"`
	Description string `json:"description,omitempty"`
	Enum        []any  `json:"enum,omitempty"` // just scalars though
	Strict      *bool  `json:"strict,omitempty"`
	ReadOnly    bool   `json:"readonly,omitempty"`
	Immutable   bool   `json:"immutable,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     any    `json:"default,omitempty"`

	Attributes Attributes `json:"attributes,omitempty"` // for Objs
	Item       *Item      `json:"item,omitempty"`       // for maps & arrays
	IfValues   IfValues   `json:"ifValues,omitempty"`   // Value

	// Internal fields
	// We have them here so we can have access to them in any func that
	// gets passed the model attribute.
	// If anything gets added below MAKE SURE to update SetSpecPropsFields too
	internals AttrInternals
}

type Item struct { // for maps and arrays
	Model       *Model     `json:"-"`
	Type        string     `json:"type,omitempty"`
	NameCharSet string     `json:"namecharset,omitempty"` // when 'type'=obj
	Attributes  Attributes `json:"attributes,omitempty"`  // when 'type'=obj
	Item        *Item      `json:"item,omitempty"`        // when 'type'=map,array
}

type IfValues map[string]*IfValue

type IfValue struct {
	SiblingAttributes Attributes `json:"siblingattributes,omitempty"`
}

type GroupModel struct {
	SID   string `json:"-"`
	Model *Model `json:"-"`

	Plural         string            `json:"plural"`
	Singular       string            `json:"singular"`
	Description    string            `json:"description,omitempty"`
	ModelVersion   string            `json:"modelversion,omitempty"`
	CompatibleWith string            `json:"compatiblewith,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Attributes     Attributes        `json:"attributes,omitempty"`

	Resources map[string]*ResourceModel `json:"resources,omitempty"` // Plural

	attrsOrdered []*Attribute
}

type ResourceModel struct {
	SID        string      `json:"-"`
	GroupModel *GroupModel `json:"-"`

	Plural            string            `json:"plural"`
	Singular          string            `json:"singular"`
	Description       string            `json:"description,omitempty"`
	MaxVersions       int               `json:"maxversions"`             // do not include omitempty
	SetVersionId      *bool             `json:"setversionid"`            // do not include omitempty
	SetDefaultSticky  *bool             `json:"setdefaultversionsticky"` // do not include omitempty
	HasDocument       *bool             `json:"hasdocument"`             // do not include omitempty
	SingleVersionRoot *bool             `json:"singleversionroot"`       // do not include omitempty
	TypeMap           map[string]string `json:"typemap,omitempty"`
	ModelVersion      string            `json:"modelversion,omitempty"`
	CompatibleWith    string            `json:"compatiblewith,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Attributes        Attributes        `json:"attributes,omitempty"`
	MetaAttributes    Attributes        `json:"metaattributes,omitempty"`

	attrsOrdered        []*Attribute
	metaAttrsOrdered    []*Attribute
	versionAttrsOrdered []*Attribute
}

// To be picky, let's Marshal the list of attributes with Spec defined ones
// first, and then the extensions in alphabetical order.
// This is used when serializing the model for end user consumption

func (attrs Attributes) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}
	attrsCopy := maps.Clone(attrs) // Copy so we can delete keys
	count := 0

	// Hack!
	// attribute "$singular" holds the singular name of the entity.
	// Couldn't find a better way to pass this info all the way down.
	singular := ""
	orderedProps := []*Attribute(nil)
	eType := -1
	if attr, ok := attrsCopy["$singular"]; ok {
		singular = attr.internals.singular
		delete(attrsCopy, "$singular")

		attr, _ := attrsCopy["$order"]
		orderedProps = attr.internals.orderedProps
		delete(attrsCopy, "$order")
		PanicIf(orderedProps == nil, "No, you're out of order!")
	}
	// end of hack

	buf.WriteString("{")

	// Only order (or tweak things) if we're at the top level, which
	// we know because $singular was set - not within an extension (sub-object)
	if singular != "" {
		for _, specProp := range OrderedSpecProps {
			if specProp.Name[0] == '$' {
				continue
			}

			if eType >= 0 && !specProp.InType(eType) {
				// log.Printf("Skipping: %s  L: %d", specProp.Name, eType)
				// continue
			}

			name := specProp.Name
			if name == "id" {
				if singular != "" {
					name = singular + name
				}
			}

			if attr, ok := attrsCopy[name]; ok {
				delete(attrsCopy, name)

				tmpAttr := *attr
				attr = &tmpAttr

				if attr.internals.neverSerialize {
					continue
				}

				if count > 0 {
					buf.WriteRune(',')
				}
				buf.WriteString(`"` + name + `": `)
				tmpBuf, err := json.Marshal(attr)
				if err != nil {
					return nil, err
				}
				buf.Write(tmpBuf)
				count++
			}
		}
	}

	keys := SortedKeys(attrsCopy)

	// Make sure "*" is last, it just looks nicer that way
	if len(keys) > 1 && keys[0] == "*" {
		keys = append(keys[1:], keys[0])
	}

	for _, name := range keys {
		if count > 0 {
			buf.WriteRune(',')
		}
		attr := attrsCopy[name]
		buf.WriteString(`"` + name + `": `)
		tmpBuf, err := json.Marshal(attr)
		if err != nil {
			return nil, err
		}
		buf.Write(tmpBuf)
		// delete(attrsCopy, name)
		count++
	}

	buf.WriteString("}")
	return buf.Bytes(), nil
}

func (r *ResourceModel) UnmarshalJSON(data []byte) error {
	// Set the default values
	r.MaxVersions = MAXVERSIONS
	r.SetVersionId = PtrBool(SETVERSIONID)
	r.SetDefaultSticky = PtrBool(SETDEFAULTSTICKY)
	r.HasDocument = PtrBool(HASDOCUMENT)
	r.SingleVersionRoot = PtrBool(SINGLEVERSIONROOT)

	type tmpResourceModel ResourceModel
	return Unmarshal(data, (*tmpResourceModel)(r))
}

func (m *Model) SetPointers() {
	for _, attr := range m.Attributes {
		attr.SetModel(m)
	}

	for _, gm := range m.Groups {
		gm.SetModel(m)
	}
}

func (m *Model) GetAttrsOrdered() []*Attribute {
	if m.attrsOrdered == nil {
		m.attrsOrdered = OrderedSpecProps
		/*
			m.attrsOrdered = []*Attribute{}
			for _, attr := range OrderedSpecProps {
				if attr.InType(ENTITY_REGISTRY) {
					attr = attr.Clone()

					m.attrsOrdered = append(m.attrsOrdered, attr)
					// specProp = specProp.Clone("registryid")
				}
			}
		*/
	}
	return m.attrsOrdered
}

// Total hack. Need a way to pass in the Singular and eType info from the
// model down into the serialization routines.
func (m *Model) SetSingular() {
	m.Attributes["$singular"] = &Attribute{
		Name:      "$singular",
		Type:      STRING,
		internals: AttrInternals{singular: "registry"},
	}
	m.Attributes["$order"] = &Attribute{
		Name:      "$order",
		Type:      STRING,
		internals: AttrInternals{orderedProps: m.GetAttrsOrdered()},
	}

	for _, gm := range m.Groups {
		gm.Attributes["$singular"] = &Attribute{
			Name:      "$singular",
			Type:      STRING,
			internals: AttrInternals{singular: gm.Singular},
		}
		gm.Attributes["$order"] = &Attribute{
			Name:      "$order",
			Type:      STRING,
			internals: AttrInternals{orderedProps: gm.GetAttrsOrdered()},
		}

		for _, rm := range gm.Resources {
			rm.Attributes["$singular"] = &Attribute{
				Name:      "$singular",
				Type:      STRING,
				internals: AttrInternals{singular: rm.Singular},
			}
			rm.Attributes["$order"] = &Attribute{
				Name:      "$order",
				Type:      STRING,
				internals: AttrInternals{orderedProps: rm.GetAttrsOrdered()},
			}

			rm.MetaAttributes["$singular"] = &Attribute{
				Name:      "$singular",
				Type:      STRING,
				internals: AttrInternals{singular: rm.Singular},
			}
			rm.MetaAttributes["$order"] = &Attribute{
				Name:      "$order",
				Type:      STRING,
				internals: AttrInternals{orderedProps: rm.GetMetaAttrsOrdered()},
			}
		}
	}
}

func (m *Model) UnsetSingular() {
	delete(m.Attributes, "$singular")
	delete(m.Attributes, "$order")
	for _, gm := range m.Groups {
		delete(gm.Attributes, "$singular")
		delete(gm.Attributes, "$order")
		for _, rm := range gm.Resources {
			delete(rm.Attributes, "$singular")
			delete(rm.Attributes, "$order")
		}
	}
}

// VerifyAndSave() should be called by automatically but there may be
// cases where someone would need to call it manually (e.g. setting an
// attribute's property - we should technically find a way to catch those
// cases so code above this shouldn't need to think about it
func (m *Model) VerifyAndSave() error {
	if err := m.Verify(); err != nil {
		// Kind of extreme, but if there's an error revert the entire
		// model to the last known good state. So, all of the changes
		// people made will be lost and any variables are bogus
		// NOTE any local variable pointing to a model entity will need to
		// be refresh/refound, the existing pointer will be bad

		// No longer needed but left around just in case
		// *m = *LoadModel(m.Registry)

		return err
	}

	return m.Save()
}

func (m *Model) Save() error {
	// Create a temporary type so that we don't use the MarshalJSON func
	// in model.go. That one will exclude "model" from the serialization and
	// we don't want to do that when we're saving it in the DB. We only want
	// to do that when we're serializing the model for the end user.
	type tmpAttributes Attributes
	buf, _ := json.Marshal((tmpAttributes)(m.Attributes))
	attrs := string(buf)

	buf, _ = json.Marshal(m.Labels)
	labels := string(buf)

	err := DoZeroTwo(m.Registry.tx, `
        INSERT INTO Models(RegistrySID, Labels, Attributes)
        VALUES(?,?,?)
        ON DUPLICATE KEY UPDATE Labels=?,Attributes=? `,

		m.Registry.DbSID, labels, attrs,
		labels, attrs)
	if err != nil {
		log.Printf("Error updating model: %s", err)
		return err
	}

	for _, gm := range m.Groups {
		if err := gm.Save(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Model) AddAttr(name, daType string) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (m *Model) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (m *Model) AddAttrObj(name string) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (m *Model) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (m *Model) AddAttrXID(name string, tgt string) (*Attribute, error) {
	return m.AddAttribute(&Attribute{Name: name, Type: XID, Target: tgt})
}

func (m *Model) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" {
		if err := IsValidAttributeName(attr.Name); err != nil {
			return nil, err
		}
	}

	if m.Attributes == nil {
		m.Attributes = Attributes{}
	}

	attr.Model = m

	oldVal := m.Attributes[attr.Name]
	m.Attributes[attr.Name] = attr

	attr.Item.SetModel(m)

	if err := m.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(m.Attributes, attr.Name, oldVal)
		return nil, err
	}

	return attr, nil
}

func (m *Model) DelAttribute(name string) error {
	if m.Attributes == nil {
		return nil
	}

	oldVal := m.Attributes[name]
	delete(m.Attributes, name)

	if err := m.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(m.Attributes, name, oldVal)
		return err
	}
	return nil
}

func (m *Model) CreateModels(gPlural, gSingular, rPlural, rSingular string) (*GroupModel, *ResourceModel, error) {
	gm, err := m.AddGroupModel(gPlural, gSingular)
	if err != nil {
		return nil, nil, err
	}
	rm, err := gm.AddResourceModelSimple(rPlural, rSingular)
	if err != nil {
		gm.Delete()
		return nil, nil, err
	}
	return gm, rm, nil
}

func (m *Model) AddGroupModel(plural string, singular string) (*GroupModel, error) {
	if plural == "" {
		return nil, fmt.Errorf("Can't add a GroupModel with an empty plural name")
	}
	if singular == "" {
		return nil, fmt.Errorf("Can't add a GroupModel with an empty singular name")
	}

	if err := IsValidModelName(plural); err != nil {
		return nil, err
	}

	if err := IsValidModelName(singular); err != nil {
		return nil, err
	}

	for _, gm := range m.Groups {
		if gm.Plural == plural {
			return nil, fmt.Errorf("GroupModel plural %q already exists",
				plural)
		}
		if gm.Singular == singular {
			return nil, fmt.Errorf("GroupModel singular %q already exists",
				singular)
		}
	}

	mSID := NewUUID()
	err := DoOne(m.Registry.tx, `
        INSERT INTO ModelEntities(
            SID, RegistrySID, ParentSID, Plural, Singular, MaxVersions)
        VALUES(?,?,?,?,?,?) `,
		mSID, m.Registry.DbSID, nil, plural, singular, 0)
	if err != nil {
		log.Printf("Error inserting groupModel(%s): %s", plural, err)
		return nil, err
	}
	gm := &GroupModel{
		SID:      mSID,
		Model:    m,
		Singular: singular,
		Plural:   plural,

		Resources: map[string]*ResourceModel{},
	}

	m.Groups[plural] = gm

	if err = m.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(m.Groups, plural, nil)
		Must(DoOne(m.Registry.tx, `
			DELETE FROM ModelEntities WHERE
			SID=? AND RegistrySID=? AND ParentSID=null AND Plural=?`,
			mSID, m.Registry.DbSID, plural))
		return nil, err
	}

	return gm, nil
}

func (m *Model) AddLabel(name string, value string) error {
	oldLabels := maps.Clone(m.Labels)

	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	m.Labels[name] = value

	if err := m.VerifyAndSave(); err != nil {
		// Undo
		m.Labels = oldLabels
		return err
	}
	return nil
}

func (m *Model) RemoveLabel(name string) error {
	if m.Labels == nil {
		return nil
	}

	oldLabels := maps.Clone(m.Labels)

	delete(m.Labels, name)
	if len(m.Labels) == 0 {
		m.Labels = nil
	}

	if err := m.VerifyAndSave(); err != nil {
		// Undo
		m.Labels = oldLabels
		return err
	}
	return nil
}

func NewItem() *Item {
	return &Item{
		// Model: m, // will be set when Item is added to attribute
	}
}
func NewItemType(daType string) *Item {
	return &Item{
		// Model: m, // will be set when Item is added to attribute
		Type: daType,
	}
}

func NewItemObject() *Item {
	return &Item{
		// Model: m,  // will be set when Item is added to attribute
		Type: OBJECT,
	}
}

func NewItemMap(item *Item) *Item {
	return &Item{
		Model: item.Model,
		Type:  MAP,
		Item:  item,
	}
}

func NewItemArray(item *Item) *Item {
	return &Item{
		Model: item.Model,
		Type:  ARRAY,
		Item:  item,
	}
}

func (i *Item) SetModel(m *Model) {
	if i == nil {
		return
	}

	i.Model = m
	i.Attributes.SetModel(m)
}

func (i *Item) SetItem(item *Item) error {
	oldVal := i.Item
	i.Item = item
	item.SetModel(i.Model)

	if i.Model != nil {
		if err := i.Model.VerifyAndSave(); err != nil {
			// Undo
			i.Item = oldVal
			return err
		}
	}
	return nil
}

func (i *Item) AddAttr(name, daType string) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (i *Item) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (i *Item) AddAttrObj(name string) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (i *Item) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return i.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (i *Item) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" {
		if i.Type == OBJECT {
			if i.NameCharSet == "extended" {
				if err := IsValidMapKey(attr.Name); err != nil {
					return nil, fmt.Errorf("Invalid attribute name %q, must "+
						"match: %s", attr.Name, RegexpMapKey.String())
				}
			} else if i.NameCharSet == "strict" || i.NameCharSet == "" {
				if err := IsValidAttributeName(attr.Name); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Invalid \"namecharset\" value: %s",
					i.NameCharSet)
			}
		} else {
			if attr.NameCharSet != "" {
				return nil, fmt.Errorf("Attribute %q must not have a "+
					"\"namecharset\" value unless its type is \"object\"",
					attr.Name)
			}
		}
	}

	if i.Attributes == nil {
		i.Attributes = Attributes{}
	}

	oldVal := i.Attributes[attr.Name]
	i.Attributes[attr.Name] = attr

	attr.Model = i.Model
	attr.Item.SetModel(i.Model)

	if i.Model != nil {
		if err := i.Model.VerifyAndSave(); err != nil {
			// Undo
			ResetMap(i.Attributes, attr.Name, oldVal)
			return nil, err
		}
	}

	return attr, nil
}

func (i *Item) DelAttribute(name string) error {
	if i.Attributes == nil {
		return nil
	}

	oldVal := i.Attributes[name]
	delete(i.Attributes, name)

	if i.Model != nil {
		if err := i.Model.VerifyAndSave(); err != nil {
			// Undo
			ResetMap(i.Attributes, name, oldVal)
			return err
		}
	}
	return nil
}

func LoadModel(reg *Registry) *Model {
	log.VPrintf(3, ">Enter: LoadModel")
	defer log.VPrintf(3, "<Exit: LoadModel")

	PanicIf(reg == nil, "nil")
	groups := map[string]*GroupModel{} // Model SID -> *GroupModel

	model := &Model{
		Registry: reg,
		Groups:   map[string]*GroupModel{},
	}

	// Load Registry Labels, Attributes
	results, err := Query(reg.tx,
		`SELECT Labels, Attributes FROM Models WHERE RegistrySID=?`,
		reg.DbSID)
	defer results.Close()
	if err != nil {
		log.Printf("Error loading registries(%s): %s", reg.UID, err)
		return nil
	}
	row := results.NextRow()
	if row == nil {
		log.Printf("Can't find registry: %s", reg.UID)
		return nil
	}

	if row[0] != nil {
		Unmarshal([]byte(NotNilString(row[0])), &model.Labels)
	}
	if row[1] != nil {
		Unmarshal([]byte(NotNilString(row[1])), &model.Attributes)
	}
	results.Close()

	model.Attributes.SetModel(model)
	model.Attributes.SetSpecPropsFields("registry")

	// Load Groups & Resources
	results, err = Query(reg.tx, `
        SELECT
            SID, RegistrySID, ParentSID, Plural, Singular, Attributes,
			MaxVersions, SetVersionId, SetDefaultSticky, HasDocument,
			TypeMap, Labels, MetaAttributes, ModelVersion, CompatibleWith,
			Description, SingleVersionRoot
        FROM ModelEntities
        WHERE RegistrySID=?
        ORDER BY ParentSID ASC`, reg.DbSID)
	defer results.Close()

	if err != nil {
		log.Printf("Error loading model(%s): %s", reg.UID, err)
		return nil
	}

	for row := results.NextRow(); row != nil; row = results.NextRow() {
		attrs := (Attributes)(nil)
		metaAttrs := (Attributes)(nil)
		if row[5] != nil {
			Unmarshal([]byte(NotNilString(row[5])), &attrs)
		}
		typemap := map[string]string(nil)
		if row[10] != nil {
			Unmarshal([]byte(NotNilString(row[10])), &typemap)
		}
		labels := map[string]string(nil)
		if row[11] != nil {
			Unmarshal([]byte(NotNilString(row[11])), &labels)
		}
		if row[12] != nil {
			Unmarshal([]byte(NotNilString(row[12])), &metaAttrs)
		}

		if *row[2] == nil { // ParentSID nil -> new Group
			g := &GroupModel{ // Plural
				SID:   NotNilString(row[0]), // SID
				Model: model,

				Plural:         NotNilString(row[3]), // Plural
				Singular:       NotNilString(row[4]), // Singular
				Description:    NotNilString(row[15]),
				ModelVersion:   NotNilString(row[13]),
				CompatibleWith: NotNilString(row[14]),
				Labels:         labels,
				Attributes:     attrs,

				Resources: map[string]*ResourceModel{},
			}

			g.Attributes.SetSpecPropsFields(g.Singular)

			model.Groups[NotNilString(row[3])] = g
			groups[NotNilString(row[0])] = g

		} else { // New Resource
			g := groups[NotNilString(row[2])] // Parent SID

			if g != nil { // should always be true, but...
				r := &ResourceModel{
					SID:         NotNilString(row[0]),
					GroupModel:  g,
					Plural:      NotNilString(row[3]),
					Singular:    NotNilString(row[4]),
					Description: NotNilString(row[15]),
					Attributes:  attrs,

					MaxVersions:       NotNilIntDef(row[6], MAXVERSIONS),
					SetVersionId:      PtrBool(NotNilBoolDef(row[7], SETVERSIONID)),
					SetDefaultSticky:  PtrBool(NotNilBoolDef(row[8], SETDEFAULTSTICKY)),
					HasDocument:       PtrBool(NotNilBoolDef(row[9], HASDOCUMENT)),
					SingleVersionRoot: PtrBool(NotNilBoolDef(row[16], SINGLEVERSIONROOT)),
					TypeMap:           typemap,
					ModelVersion:      NotNilString(row[13]),
					CompatibleWith:    NotNilString(row[14]),
					Labels:            labels,
					MetaAttributes:    metaAttrs,
				}

				r.Attributes.SetSpecPropsFields(r.Singular)
				r.MetaAttributes.SetSpecPropsFields(r.Singular)

				g.Resources[r.Plural] = r
			}
		}
	}
	results.Close()

	reg.Model = model
	return model
}

func (m *Model) FindGroupModel(gTypePlural string) *GroupModel {
	return m.Groups[gTypePlural]
	/*
		for _, gModel := range m.Groups {
			if strings.EqualFold(gModel.Plural, gTypePlural) {
				return gModel
			}
		}
		return nil
	*/
}

/*
func (m *Model) FindGroupModelBySingular(gTypeSingular string) *GroupModel {
	for _, gModel := range m.Groups {
		if gModel.Singular == gTypeSingular {
			return gModel
		}
	}
	return nil
}
*/

func (m *Model) ApplyNewModel(newM *Model) error {
	newM.Registry = m.Registry

	if err := newM.Verify(); err != nil {
		return err
	}

	var err error
	m.Labels = newM.Labels
	m.Attributes = newM.Attributes

	// Find all old groups that need to be deleted
	for gmPlural, gm := range m.Groups {
		if newGM, ok := newM.Groups[gmPlural]; !ok {
			if err := gm.Delete(); err != nil {
				return err
			}
		} else {
			for rmPlural, rm := range gm.Resources {
				if _, ok := newGM.Resources[rmPlural]; !ok {
					if err := rm.Delete(); err != nil {
						return err
					}
				}
			}
		}
	}

	// Apply new stuff
	newM.Registry = m.Registry
	for _, newGM := range newM.Groups {
		log.VPrintf(4, "Applying Group: %s", newGM.Plural)
		newGM.Model = m
		oldGM := m.Groups[newGM.Plural]
		if oldGM == nil {
			oldGM, err = m.AddGroupModel(newGM.Plural, newGM.Singular)
			if err != nil {
				return err
			}
		} else {
			oldGM.Singular = newGM.Singular
		}
		oldGM.Description = newGM.Description
		oldGM.ModelVersion = newGM.ModelVersion
		oldGM.CompatibleWith = newGM.CompatibleWith
		oldGM.Labels = newGM.Labels
		oldGM.Attributes = newGM.Attributes

		for _, newRM := range newGM.Resources {
			log.VPrintf(4, "Applying Resource: %s", newRM.Plural)
			oldRM := oldGM.Resources[newRM.Plural]
			if oldRM == nil {
				oldRM, err = oldGM.AddResourceModelFull(&ResourceModel{
					Plural:            newRM.Plural,
					Singular:          newRM.Singular,
					Description:       newRM.Description,
					MaxVersions:       newRM.MaxVersions,
					SetVersionId:      newRM.SetVersionId,
					SetDefaultSticky:  newRM.SetDefaultSticky,
					HasDocument:       newRM.HasDocument,
					SingleVersionRoot: newRM.SingleVersionRoot,
					ModelVersion:      newRM.ModelVersion,
					CompatibleWith:    newRM.CompatibleWith,
				})
				if err != nil {
					log.VPrintf(4, "Err: %s", err)
					return err
				}

			} else {
				oldRM.Singular = newRM.Singular
				oldRM.Description = newRM.Description
				oldRM.MaxVersions = newRM.MaxVersions
				oldRM.SetVersionId = newRM.SetVersionId
				oldRM.SetDefaultSticky = newRM.SetDefaultSticky
				oldRM.HasDocument = newRM.HasDocument
				oldRM.SingleVersionRoot = newRM.SingleVersionRoot
				oldRM.ModelVersion = newRM.ModelVersion
				oldRM.CompatibleWith = newRM.CompatibleWith
			}
			oldRM.Attributes = newRM.Attributes
			oldRM.TypeMap = newRM.TypeMap
			oldRM.Labels = newRM.Labels
			oldRM.MetaAttributes = newRM.MetaAttributes
		}
	}

	if err := m.VerifyAndSave(); err != nil {
		// Too much to undo. The Verify() at the top should have caught
		// anything wrong
		return err
	}

	return nil
}

func (gm *GroupModel) GetAttrsOrdered() []*Attribute {
	if gm.attrsOrdered == nil {
		gm.attrsOrdered = []*Attribute{}
		for _, attr := range OrderedSpecProps {
			if attr.InType(ENTITY_GROUP) {
				gm.attrsOrdered = append(gm.attrsOrdered, attr)
			}
		}
	}
	return gm.attrsOrdered
}

func (gm *GroupModel) Delete() error {
	log.VPrintf(3, ">Enter: Delete.GroupModel: %s", gm.Plural)
	defer log.VPrintf(3, "<Exit: Delete.GroupModel")
	err := DoOne(gm.Model.Registry.tx, `
        DELETE FROM ModelEntities
		WHERE RegistrySID=? AND SID=?`, // SID should be enough, but ok
		gm.Model.Registry.DbSID, gm.SID)
	if err != nil {
		log.Printf("Error deleting groupModel(%s): %s", gm.Plural, err)
		return err
	}

	delete(gm.Model.Groups, gm.Plural)

	return nil
}

func (gm *GroupModel) Save() error {
	// Just updates this GroupModel, not any Resources
	// DO NOT use this to insert a new one

	buf, _ := json.Marshal(gm.Labels)
	labels := string(buf)

	buf, _ = json.Marshal(gm.Attributes)
	attrs := string(buf)

	err := DoZeroTwo(gm.Model.Registry.tx, `
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, Labels, Attributes, 
			ModelVersion, CompatibleWith, Description)
        VALUES(?,?,?,?,?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
		    ParentSID=?,Plural=?,Singular=?,Labels=?,Attributes=?,
			ModelVersion=?,CompatibleWith=?,Description=?
		`,
		gm.SID, gm.Model.Registry.DbSID,
		nil, gm.Plural, gm.Singular, labels, attrs,
		gm.ModelVersion, gm.CompatibleWith, gm.Description,
		nil, gm.Plural, gm.Singular, labels, attrs,
		gm.ModelVersion, gm.CompatibleWith, gm.Description)
	if err != nil {
		log.Printf("Error updating groupModel(%s): %s", gm.Plural, err)
	}

	for _, rm := range gm.Resources {
		if err := rm.Save(); err != nil {
			return err
		}
	}

	return err
}

/*
func (gm *GroupModel) FindResourceModelBySingular(rTypeSingular string) *ResourceModel {
	for _, rModel := range gm.Resources {
		if rModel.Singular == rTypeSingular {
			return rModel
		}
	}
	return nil
}
*/

func (gm *GroupModel) AddAttr(name, daType string) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (gm *GroupModel) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (gm *GroupModel) AddAttrObj(name string) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (gm *GroupModel) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return gm.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (gm *GroupModel) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" {
		if err := IsValidAttributeName(attr.Name); err != nil {
			return nil, err
		}
	}

	if gm.Attributes == nil {
		gm.Attributes = Attributes{}
	}

	oldVal := gm.Attributes[attr.Name]
	gm.Attributes[attr.Name] = attr

	attr.Model = gm.Model
	attr.Item.SetModel(gm.Model)

	if err := gm.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(gm.Attributes, attr.Name, oldVal)
		return nil, err
	}

	return attr, nil
}

func (gm *GroupModel) DelAttribute(name string) error {
	if gm.Attributes == nil {
		return nil
	}

	oldVal := gm.Attributes[name]
	delete(gm.Attributes, name)

	if err := gm.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(gm.Attributes, name, oldVal)
	}
	return nil
}

func (gm *GroupModel) AddResourceModelSimple(plural, singular string) (*ResourceModel, error) {
	return gm.AddResourceModelFull(&ResourceModel{
		Plural:            plural,
		Singular:          singular,
		MaxVersions:       MAXVERSIONS,
		SetVersionId:      PtrBool(SETVERSIONID),
		SetDefaultSticky:  PtrBool(SETDEFAULTSTICKY),
		HasDocument:       PtrBool(HASDOCUMENT),
		SingleVersionRoot: PtrBool(SINGLEVERSIONROOT),
	})
}

func (gm *GroupModel) AddResourceModel(plural string, singular string, maxVersions int, setVerId bool, setDefaultSticky bool, hasDocument bool) (*ResourceModel, error) {
	return gm.AddResourceModelFull(&ResourceModel{
		Plural:           plural,
		Singular:         singular,
		MaxVersions:      maxVersions,
		SetVersionId:     PtrBool(setVerId),
		SetDefaultSticky: PtrBool(setDefaultSticky),
		HasDocument:      PtrBool(hasDocument),
	})
}

func (gm *GroupModel) AddResourceModelFull(rm *ResourceModel) (*ResourceModel, error) {
	if rm.Plural == "" {
		return nil, fmt.Errorf("Can't add a group with an empty plural name")
	}
	if rm.Singular == "" {
		return nil, fmt.Errorf("Can't add a group with an empty singular name")
	}

	if rm.MaxVersions < 0 {
		return nil, fmt.Errorf(`"maxversions"(%d) must be >= 0`,
			rm.MaxVersions)
	}

	if rm.MaxVersions == 1 && rm.GetSetDefaultSticky() != false {
		return nil, fmt.Errorf("'setdefaultversionsticky' must be 'false' " +
			"since 'maxversions' is '1'")
	}

	if err := IsValidModelName(rm.Plural); err != nil {
		return nil, err
	}
	if err := IsValidModelName(rm.Singular); err != nil {
		return nil, err
	}

	for _, r := range gm.Resources {
		if r.Plural == rm.Plural {
			return nil, fmt.Errorf("Resource model plural %q already "+
				"exists for group %q", rm.Plural, gm.Plural)
		}
		if r.Singular == rm.Singular {
			return nil,
				fmt.Errorf("Resource model singular %q already "+
					"exists for group %q", rm.Singular, gm.Plural)
		}
	}

	rm.SID = NewUUID()
	rm.GroupModel = gm

	buf, _ := json.Marshal(rm.TypeMap)
	typemap := string(buf)

	buf, _ = json.Marshal(rm.Labels)
	labels := string(buf)

	err := DoOne(gm.Model.Registry.tx, `
		INSERT INTO ModelEntities(
			SID, RegistrySID, ParentSID, Plural, Singular, MaxVersions,
			SetVersionId, SetDefaultSticky, HasDocument, TypeMap, Labels,
			ModelVersion, CompatibleWith, Description, SingleVersionRoot)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rm.SID, gm.Model.Registry.DbSID, gm.SID, rm.Plural, rm.Singular, rm.MaxVersions,
		rm.GetSetVersionId(), rm.GetSetDefaultSticky(), rm.GetHasDocument(), typemap, labels,
		rm.ModelVersion, rm.CompatibleWith, rm.Description, rm.GetSingleVersionRoot())
	if err != nil {
		log.Printf("Error inserting resourceModel(%s): %s", rm.Plural, err)
		return nil, err
	}

	oldVal := gm.Resources[rm.Plural]
	gm.Resources[rm.Plural] = rm

	if err = gm.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(gm.Resources, rm.Plural, oldVal)
		return nil, err
	}

	return rm, nil
}

func (gm *GroupModel) AddLabel(name string, value string) error {
	oldLabels := maps.Clone(gm.Labels)

	if gm.Labels == nil {
		gm.Labels = map[string]string{}
	}
	gm.Labels[name] = value

	if err := gm.Model.VerifyAndSave(); err != nil {
		// Undo
		gm.Labels = oldLabels
		return err
	}
	return nil
}

func (gm *GroupModel) RemoveLabel(name string) error {
	if gm.Labels == nil {
		return nil
	}

	oldLabels := maps.Clone(gm.Labels)

	delete(gm.Labels, name)
	if len(gm.Labels) == 0 {
		gm.Labels = nil
	}

	if err := gm.Model.VerifyAndSave(); err != nil {
		// Undo
		gm.Labels = oldLabels
		return err
	}
	return nil
}

func (rm *ResourceModel) GetAttrsOrdered() []*Attribute {
	if rm.attrsOrdered == nil {
		rm.attrsOrdered = []*Attribute{}
		for _, attr := range OrderedSpecProps {
			if attr.InType(ENTITY_RESOURCE) {
				rm.attrsOrdered = append(rm.attrsOrdered, attr)
			}
		}
	}
	return rm.attrsOrdered
}

func (rm *ResourceModel) GetMetaAttrsOrdered() []*Attribute {
	if rm.metaAttrsOrdered == nil {
		rm.metaAttrsOrdered = []*Attribute{}
		for _, attr := range OrderedSpecProps {
			if attr.InType(ENTITY_RESOURCE) {
				rm.metaAttrsOrdered = append(rm.metaAttrsOrdered, attr)
			}
		}
	}
	return rm.metaAttrsOrdered
}

func (rm *ResourceModel) GetVersionsAttrsOrdered() []*Attribute {
	if rm.versionAttrsOrdered == nil {
		rm.versionAttrsOrdered = []*Attribute{}
		for _, attr := range OrderedSpecProps {
			if attr.InType(ENTITY_RESOURCE) {
				rm.versionAttrsOrdered = append(rm.versionAttrsOrdered, attr)
			}
		}
	}
	return rm.versionAttrsOrdered
}

func (rm *ResourceModel) GetSetVersionId() bool {
	if rm.SetVersionId == nil {
		return SETVERSIONID
	}
	return *rm.SetVersionId
}

func (rm *ResourceModel) GetSetDefaultSticky() bool {
	if rm.SetDefaultSticky == nil {
		return SETDEFAULTSTICKY
	}
	return *rm.SetDefaultSticky
}

func (rm *ResourceModel) GetHasDocument() bool {
	if rm.HasDocument == nil {
		return SINGLEVERSIONROOT
	}
	return *rm.HasDocument
}

func (rm *ResourceModel) GetSingleVersionRoot() bool {
	if rm.SingleVersionRoot == nil {
		return SINGLEVERSIONROOT
	}
	return *rm.SingleVersionRoot
}

func (rm *ResourceModel) Delete() error {
	log.VPrintf(3, ">Enter: Delete.ResourceModel: %s", rm.Plural)
	defer log.VPrintf(3, "<Exit: Delete.ResourceModel")
	err := DoOne(rm.GroupModel.Model.Registry.tx, `
        DELETE FROM ModelEntities
		WHERE RegistrySID=? AND SID=?`, // SID should be enough, but ok
		rm.GroupModel.Model.Registry.DbSID, rm.SID)
	if err != nil {
		log.Printf("Error deleting resourceModel(%s): %s", rm.Plural, err)
		return err
	}

	delete(rm.GroupModel.Resources, rm.Plural)

	return nil
}

func (rm *ResourceModel) Save() error {
	// Just updates this GroupModel, not any Resources
	// DO NOT use this to insert a new one

	buf, _ := json.Marshal(rm.TypeMap)
	typemap := string(buf)
	buf, _ = json.Marshal(rm.Labels)
	labels := string(buf)
	buf, _ = json.Marshal(rm.Attributes)
	attrs := string(buf)
	buf, _ = json.Marshal(rm.MetaAttributes)
	metaAttrs := string(buf)

	err := DoZeroTwo(rm.GroupModel.Model.Registry.tx, `
        INSERT INTO ModelEntities(
            SID, RegistrySID,
			ParentSID, Plural, Singular, MaxVersions,
			Attributes,
			SetVersionId, SetDefaultSticky, HasDocument, TypeMap,
			Labels, MetaAttributes,ModelVersion,CompatibleWith, Description,
			SingleVersionRoot)
        VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
        ON DUPLICATE KEY UPDATE
            ParentSID=?, Plural=?, Singular=?,
			Attributes=?,
            MaxVersions=?, SetVersionId=?, SetDefaultSticky=?, HasDocument=?, TypeMap=?, Labels=?,
			MetaAttributes=?, ModelVersion=?, CompatibleWith=?, Description=?,
			SingleVersionRoot=?`,
		rm.SID, rm.GroupModel.Model.Registry.DbSID,
		rm.GroupModel.SID, rm.Plural, rm.Singular, rm.MaxVersions,
		attrs,
		rm.GetSetVersionId(), rm.GetSetDefaultSticky(), rm.GetHasDocument(), typemap, labels,
		metaAttrs, rm.ModelVersion, rm.CompatibleWith, rm.Description,
		rm.GetSingleVersionRoot(),

		rm.GroupModel.SID, rm.Plural, rm.Singular,
		attrs,
		rm.MaxVersions, rm.GetSetVersionId(), rm.GetSetDefaultSticky(), rm.GetHasDocument(), typemap, labels,
		metaAttrs, rm.ModelVersion, rm.CompatibleWith, rm.Description,
		rm.GetSingleVersionRoot())
	if err != nil {
		log.Printf("Error updating resourceModel(%s): %s", rm.Plural, err)
		return err
	}
	return err
}

func (rm *ResourceModel) AddMetaAttr(name, daType string) (*Attribute, error) {
	return rm.AddMetaAttribute(&Attribute{Name: name, Type: daType})
}

func (rm *ResourceModel) AddMetaAttrMap(name string, item *Item) (*Attribute, error) {
	return rm.AddMetaAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (rm *ResourceModel) AddMetaAttrObj(name string) (*Attribute, error) {
	return rm.AddMetaAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (rm *ResourceModel) AddMetaAttrArray(name string, item *Item) (*Attribute, error) {
	return rm.AddMetaAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (rm *ResourceModel) AddMetaAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" {
		if err := IsValidAttributeName(attr.Name); err != nil {
			return nil, err
		}
	}

	if rm.MetaAttributes == nil {
		rm.MetaAttributes = Attributes{}
	}

	oldVal := rm.MetaAttributes[attr.Name]
	rm.MetaAttributes[attr.Name] = attr

	attr.Model = rm.GroupModel.Model
	attr.Item.SetModel(rm.GroupModel.Model)

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(rm.MetaAttributes, attr.Name, oldVal)
		return nil, err
	}

	return attr, nil
}

func (rm *ResourceModel) AddAttr(name, daType string) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: daType})
}

func (rm *ResourceModel) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (rm *ResourceModel) AddAttrObj(name string) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (rm *ResourceModel) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return rm.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}

func (rm *ResourceModel) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr == nil {
		return nil, nil
	}

	if attr.Name != "*" {
		if err := IsValidAttributeName(attr.Name); err != nil {
			return nil, err
		}
	}

	if rm.GetHasDocument() == true {
		invalidNames := []string{
			rm.Singular,
			rm.Singular + "url",
			rm.Singular + "base64",
			rm.Singular + "proxyurl",
		}

		for _, name := range invalidNames {
			if attr.Name == name {
				return nil, fmt.Errorf("Attribute name is reserved: %s", name)
			}
		}
	}

	if rm.Attributes == nil {
		rm.Attributes = Attributes{}
	}

	oldVal := rm.Attributes[attr.Name]
	rm.Attributes[attr.Name] = attr

	attr.Model = rm.GroupModel.Model
	attr.Item.SetModel(rm.GroupModel.Model)

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(rm.Attributes, attr.Name, oldVal)
		return nil, err
	}

	return attr, nil
}

func (rm *ResourceModel) DelMetaAttribute(name string) error {
	if rm.MetaAttributes == nil {
		return nil
	}

	oldVal := rm.MetaAttributes[name]
	delete(rm.MetaAttributes, name)

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(rm.MetaAttributes, name, oldVal)
		return err
	}
	return nil
}

func (rm *ResourceModel) DelAttribute(name string) error {
	if rm.Attributes == nil {
		return nil
	}

	oldVal := rm.Attributes[name]
	delete(rm.Attributes, name)

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(rm.Attributes, name, oldVal)
		return err
	}
	return nil
}

func (attrs Attributes) SetModel(m *Model) {
	if attrs == nil {
		return
	}

	for _, attr := range attrs {
		attr.Model = m
		attr.Item.SetModel(m)
		attr.IfValues.SetModel(m)
	}
}

// This just does the top-level attributes with the assumption that we'll
// do the lower-level ones later on in Entity.ValidateObject
func (attrs Attributes) AddIfValuesAttributes(obj map[string]any) {
	attrNames := Keys(attrs)
	for i := 0; i < len(attrNames); i++ { // since attrs changes
		attr := attrs[attrNames[i]]
		if len(attr.IfValues) == 0 || attr.Name == "*" {
			continue
		}

		val, ok := obj[attr.Name]
		if !ok {
			continue
		}

		valStr := fmt.Sprintf("%v", val)
		for ifValStr, ifValueData := range attr.IfValues {
			if ifValStr != valStr {
				continue
			}

			for _, newAttr := range ifValueData.SiblingAttributes {
				if _, ok := attrs[newAttr.Name]; ok {
					Panicf(`Attribute %q has an ifvalue(%s) that `+
						`defines a conflicting siblingattribute: %s`,
						attr.Name, ifValStr, newAttr.Name)
				}
				attrs[newAttr.Name] = newAttr
				// Add new attr name to the list so we can check its ifValues
				attrNames = append(attrNames, newAttr.Name)
			}
		}
	}
}

func KindIsScalar(k reflect.Kind) bool {
	// SOOOO risky :-)
	return k < reflect.Array || k == reflect.String
}

func IsScalar(daType string) bool {
	return daType == BOOLEAN || daType == DECIMAL || daType == INTEGER ||
		daType == STRING || daType == TIMESTAMP || daType == UINTEGER ||
		daType == URI || daType == URI_REFERENCE || daType == URI_TEMPLATE ||
		daType == URL ||
		daType == XID
}

// Is some string variant
func IsString(daType string) bool {
	return daType == STRING || daType == TIMESTAMP || daType == XID ||
		daType == URI || daType == URI_REFERENCE || daType == URI_TEMPLATE ||
		daType == URL
}

func (a *Attribute) GetStrict() bool {
	return a.Strict == nil || *a.Strict == true
}

func (a *Attribute) InType(eType int) bool {
	return a.internals.types == "" ||
		strings.ContainsRune(a.internals.types, rune('0'+byte(eType)))
}

func (a *Attribute) IsScalar() bool {
	return IsScalar(a.Type)
}

func (a *Attribute) SetModel(m *Model) {
	if a == nil {
		return
	}

	a.Model = m
	a.Item.SetModel(m)
	a.IfValues.SetModel(m)
}

func (a *Attribute) AddAttr(name, daType string) (*Attribute, error) {
	return a.AddAttribute(&Attribute{
		Model: a.Model,
		Name:  name,
		Type:  daType,
	})
}

func (a *Attribute) AddAttrMap(name string, item *Item) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: MAP, Item: item})
}

func (a *Attribute) AddAttrObj(name string) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: OBJECT})
}

func (a *Attribute) AddAttrArray(name string, item *Item) (*Attribute, error) {
	return a.AddAttribute(&Attribute{Name: name, Type: ARRAY, Item: item})
}
func (a *Attribute) AddAttribute(attr *Attribute) (*Attribute, error) {
	if attr.Name != "*" {
		if a.Type == OBJECT {
			if a.NameCharSet == "extended" {
				if err := IsValidMapKey(attr.Name); err != nil {
					return nil, fmt.Errorf("Invalid attribute name %q, must "+
						"match: %s", attr.Name, RegexpMapKey.String())
				}
			} else if a.NameCharSet == "strict" || a.NameCharSet == "" {
				if err := IsValidAttributeName(attr.Name); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("Invalid \"namecharset\" value: %s",
					a.NameCharSet)
			}
		} else {
			if a.NameCharSet != "" {
				return nil, fmt.Errorf("Attribute %q must not have a "+
					"\"namecharset\" value unless it is of type \"object\"",
					a.Name)
			}
		}
	}

	if a.Attributes == nil {
		a.Attributes = Attributes{}
	}

	oldVal := a.Attributes[attr.Name]
	a.Attributes[attr.Name] = attr
	attr.SetModel(a.Model)

	if err := a.Model.VerifyAndSave(); err != nil {
		// Undo
		ResetMap(a.Attributes, attr.Name, oldVal)
		return nil, err
	}
	return attr, nil
}

// Make sure that the attribute doesn't deviate too much from the
// spec defined version of it. There's only so much that we allow the
// user to customize
func EnsureAttrOK(userAttr *Attribute, specAttr *Attribute) error {
	if userAttr.Name == "" {
		userAttr.Name = specAttr.Name
	}

	// Just blindly ignore any updates made to "model"
	if userAttr.Name == "model" || userAttr.Name == "capabilities" {
		*userAttr = *specAttr
		return nil
	}

	if specAttr.Required {
		if userAttr.Required == false {
			return fmt.Errorf(`"model.%s" must have its "required" `+
				`attribute set to "true"`, specAttr.Name)
		}
		if specAttr.ReadOnly && !userAttr.ReadOnly {
			return fmt.Errorf(`"model.%s" must have its "readonly" `+
				`attribute set to "true"`, specAttr.Name)
		}
	}

	if specAttr.Type != userAttr.Type {
		return fmt.Errorf(`"model.%s" must have a "type" of %q`,
			userAttr.Name, specAttr.Type)
	}

	return nil
}

func (m *Model) Verify() error {
	// TODO: Verify that the Registry data is model compliant

	// Check Groups first so that if the Group name isn't valid we'll
	// flag that instead of an invalid GROUPScount attribute name
	for gmName, gm := range m.Groups {
		if gm == nil {
			return fmt.Errorf("GroupModel %q can't be empty", gmName)
		}
		gm.Model = m
		// PanicIf(m.Registry.Model == nil, "nil")
		if err := gm.Verify(gmName); err != nil {
			return err
		}
	}

	// First, make sure we have the xRegistry core/spec defined attributes
	// in the list and they're not changed in an inappropriate way.
	// This just checks the Registry.Attributes. Groups and Resources will
	// be done in their own Verify funcs

	if m.Attributes == nil {
		m.Attributes = Attributes{}

	}

	for _, specProp := range OrderedSpecProps {
		// If it's not a Registry level attribute, then skip it
		if !specProp.InType(ENTITY_REGISTRY) {
			continue
		}

		/*
			if specProp.Name == "$COLLECTIONS" {
				for plural, _ := range m.Groups {
					m.Attributes[plural+"url"] = &Attribute{
						Type:      URL,
						ReadOnly:  true,
						Immutable: true,
						// Required:  true,
						internals: AttrInternals{dontStore: true},
					}
					m.Attributes[plural+"count"] = &Attribute{
						Type:     UINTEGER,
						ReadOnly: true,
						// Required:  true,
						internals: AttrInternals{dontStore: true},
					}
					m.Attributes[plural] = &Attribute{
						Type:      MAP,
						Item:      &Item{Type: OBJECT},
						internals: AttrInternals{dontStore: true},
					}
				}
			}
		*/

		if specProp.Name[0] == '$' {
			continue
		}

		if specProp.Name == "id" {
			specProp = specProp.Clone("registryid")
			specProp.ReadOnly = true // Just registryid is "readonly"
		} else {
			specProp = specProp.Clone("")
		}

		modelAttr, ok := m.Attributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			m.Attributes[specProp.Name] = specProp
			continue
		}

		// It's there but make sure it's not changed in a bad way
		if err := EnsureAttrOK(modelAttr, specProp); err != nil {
			return err
		}
	}

	// Now check Registry attributes for correctness
	ld := &LevelData{
		Model:     m,
		AttrNames: map[string]bool{},
		Path:      NewPPP("model"),
	}
	if err := m.Attributes.Verify("strict", ld); err != nil {
		return err
	}

	return nil
}

func (m *Model) GetBaseAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, m.Attributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	for _, specProp := range OrderedSpecProps {
		if specProp.Name == "id" {
			// Skip "id"
			continue
		}

		if specProp.InType(ENTITY_REGISTRY) && IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		}
	}

	return attrs
}

func (gm *GroupModel) Verify(gmName string) error {
	if err := IsValidModelName(gmName); err != nil {
		return err
	}

	if gm.Plural == "" {
		// Allow auto-populate
		gm.Plural = gmName
	} else if gm.Plural != gmName {
		return fmt.Errorf("Group %q must have a `plural` value of %q, not %q",
			gmName, gmName, gm.Plural)
	}

	if gm.Singular == "" {
		return fmt.Errorf(`Group %q is missing a "singular" value`, gmName)
	}

	if gm.Singular == gm.Plural {
		return fmt.Errorf(`Group %q has same value for "plural" `+
			`and "singular"`, gmName)
	}

	if gm.Model.Groups[gm.Singular] != nil {
		return fmt.Errorf(`Group %q has a "singular" value (%s) that `+
			`matches another Group's "plural" value`, gmName, gm.Singular)
	}

	if err := IsValidModelName(gm.Singular); err != nil {
		return err
	}

	// TODO: verify the Groups data are model compliant

	// Verify the Resources to catch invalid Resource names early
	for rmName, rm := range gm.Resources {
		if rm == nil {
			return fmt.Errorf("Resource %q can't be empty", rmName)
		}
		rm.GroupModel = gm
		if err := rm.Verify(rmName); err != nil {
			return err
		}
	}

	// Make sure we have the xRegistry core/spec defined attributes
	// in the list and they're not changed in an inappropriate way.
	// This just checks the Group level Attributes
	if gm.Attributes == nil {
		gm.Attributes = Attributes{}

	}

	for _, specProp := range OrderedSpecProps {
		/*
			if specProp.Name == "$COLLECTIONS" {
				for plural, _ := range gm.Resources {
					gm.Attributes[plural+"url"] = &Attribute{
						Type:      URL,
						ReadOnly:  true,
						Immutable: true,
						// Required:  true,
						internals: AttrInternals{dontStore: true},
					}
					gm.Attributes[plural+"count"] = &Attribute{
						Type:     UINTEGER,
						ReadOnly: true,
						// Required:  true,
						internals: AttrInternals{dontStore: true},
					}
					gm.Attributes[plural] = &Attribute{
						Type:      MAP,
						Item:      &Item{Type: OBJECT},
						internals: AttrInternals{dontStore: true},
					}
				}
			}
		*/

		if specProp.Name[0] == '$' {
			continue
		}

		// If it's not a Group level attribute, then skip it
		if !specProp.InType(ENTITY_GROUP) {
			continue
		}

		if specProp.Name == "id" {
			specProp = specProp.Clone(gm.Singular + "id")
		} else {
			specProp = specProp.Clone("")
		}

		modelAttr, ok := gm.Attributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			gm.Attributes[specProp.Name] = specProp
			continue
		}

		// It's there but make sure it's not changed in a bad way
		if err := EnsureAttrOK(modelAttr, specProp); err != nil {
			return err
		}
	}

	ld := &LevelData{
		Model:     gm.Model,
		AttrNames: map[string]bool{},
		Path:      NewPPP("groups").P(gm.Plural),
	}
	if err := gm.Attributes.Verify("strict", ld); err != nil {
		return err
	}

	return nil
}

func (gm *GroupModel) SetModel(m *Model) {
	if gm == nil {
		return
	}

	gm.Model = m
	if gm.Attributes == nil {
		gm.Attributes = map[string]*Attribute{}
	}

	gm.Attributes.SetModel(m)

	for _, rm := range gm.Resources {
		// rm.GroupModel = gm
		rm.SetModel(m)
	}
}

func (gm *GroupModel) GetBaseAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, gm.Attributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	for _, specProp := range OrderedSpecProps {
		if specProp.Name == "id" {
			// Skip "id"
			continue
		}

		if specProp.InType(ENTITY_GROUP) && IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		}
	}

	return attrs
}

func (rm *ResourceModel) Verify(rmName string) error {
	if err := IsValidModelName(rmName); err != nil {
		return err
	}

	if rm.Plural == "" {
		// Allow auto-populate
		rm.Plural = rmName
	} else if rm.Plural != rmName {
		return fmt.Errorf("Resource %q must have a 'plural' value of %q, "+
			"not %q", rmName, rmName, rm.Plural)
	}

	if rm.Singular == "" {
		return fmt.Errorf(`Resource %q is missing a "singular" value`, rmName)
	}

	if rm.Singular == rm.Plural {
		return fmt.Errorf(`Resource %q has same value for "plural" `+
			`and "singular"`, rmName)
	}

	if rm.GroupModel.Resources[rm.Singular] != nil {
		return fmt.Errorf(`Resource %q has a "singular" value (%s) that `+
			`matches another Resource's "plural" value`, rmName, rm.Singular)
	}

	if err := IsValidModelName(rm.Singular); err != nil {
		return err
	}

	if rm.MaxVersions < 0 {
		return fmt.Errorf("Resource %q must have a 'maxversions' value >= 0",
			rmName)
	}

	// Make sure we have the xRegistry core/spec defined attributes
	// in the list and they're not changed in an inappropriate way.
	// This just checks the Group level Attributes
	if rm.Attributes == nil {
		rm.Attributes = Attributes{}
	}

	if rm.MetaAttributes == nil {
		rm.MetaAttributes = Attributes{}
	}

	for _, specProp := range OrderedSpecProps {
		name := specProp.Name

		/* TODO ADD BACK IN?
		if rm.GetHasDocument() == true {
			if strings.HasPrefix(name, "$RESOURCE") {
				name = rm.Singular + name[9:]
				specProp = specProp.Clone(name)
			}
		}
		*/
		// TODO ADD BACK IN - END

		/*
			if name == "$COLLECTIONS" {
				// Now add the COLLECTIONS attributes("versions")
				rm.Attributes["versionsurl"] = &Attribute{
					Type:      URL,
					ReadOnly:  true,
					Immutable: true,
					// Required:  true,
					internals: AttrInternals{dontStore: true},
				}
				rm.Attributes["versionscount"] = &Attribute{
					Type:     UINTEGER,
					ReadOnly: true,
					// Required:  true,
					internals: AttrInternals{dontStore: true},
				}
				rm.Attributes["versions"] = &Attribute{
					Type:      MAP,
					Item:      &Item{Type: OBJECT},
					internals: AttrInternals{dontStore: true},
				}
			}
		*/

		if name[0] == '$' {
			continue
		}

		if name == "id" {
			name = rm.Singular + "id"
			specProp = specProp.Clone(name)
		}

		if specProp.InType(ENTITY_VERSION) { // || specProp.InType(ENTITY_RESOURCE) {
			modelAttr, ok := rm.Attributes[name]
			if !ok {
				// Missing in model, so add it
				rm.Attributes[name] = specProp
			} else {
				// It's there but make sure it's not changed in a bad way
				if err := EnsureAttrOK(modelAttr, specProp); err != nil {
					return err
				}
			}
		}

		if specProp.InType(ENTITY_META) {
			modelAttr, ok := rm.MetaAttributes[name]
			if !ok {
				// Missing in model, so add it
				rm.MetaAttributes[name] = specProp
			} else {
				// It's there but make sure it's not changed in a bad way
				if err := EnsureAttrOK(modelAttr, specProp); err != nil {
					return err
				}
			}
		}
	}

	ld := &LevelData{
		Model:     rm.GroupModel.Model,
		AttrNames: map[string]bool{},
		Path:      NewPPP("resources").P(rm.Plural),
	}

	// TODO REMOVE START
	// Make a copy so we can add the RESOURCExxx attributes, if it has a doc
	attrs := maps.Clone(rm.Attributes)
	// attrs := rm.Attributes
	if rm.GetHasDocument() == true {
		// DUG TODO see if there's a better way to do this
		attrs[rm.Singular] = &Attribute{Name: rm.Singular, Type: ANY}
		attrs[rm.Singular+"url"] = &Attribute{Type: URL}
		attrs[rm.Singular+"proxyurl"] = &Attribute{Type: URL}
		// attrs[rm.Singular+"base64"] = &Attribute{Type: STRING}
	}

	// Now add the COLLECTIONS attributes("versions")
	/*
		attrs["versionsurl"] = &Attribute{Type: URL}
		attrs["versionscount"] = &Attribute{Type: UINTEGER}
		attrs["versions"] = &Attribute{Type: MAP, Item: &Item{Type: OBJECT}}
	*/

	if err := attrs.Verify("strict", ld); err != nil {
		// TODO REMOVE END
		// if err := rm.Attributes.Verify("strict", ld); err != nil {
		return err
	}

	if err := rm.MetaAttributes.Verify("strict", ld); err != nil {
		return err
	}

	// TODO: verify the Resources data are model compliant
	// Only do this if we have a Registry. It assumes that if we have
	// no Registry then we're not connected to a backend and there's no data
	// to verify
	if rm.GroupModel.Model.Registry != nil {
		if err := rm.VerifyData(); err != nil {
			return err
		}
	}

	// Make sure the typemap's values are just certain strings
	for _, v := range rm.TypeMap {
		if v != "string" && v != "json" && v != "binary" {
			return fmt.Errorf("Resource %q has an invalid 'typemap' value "+
				"(%s). Must be one of 'string', 'json' or 'binary'", rmName, v)
		}
	}

	return nil
}

func (rm *ResourceModel) VerifyData() error {
	reg := rm.GroupModel.Model.Registry

	// Query to find all Groups/Resources of the proper type.
	// The resulting list MUST be Group followed by it's Resources, repeat...
	gAbs := NewPPP(rm.GroupModel.Plural).Abstract()
	rAbs := NewPPP(rm.GroupModel.Plural).P(rm.Plural).Abstract()
	entities, err := RawEntitiesFromQuery(reg.tx, reg.DbSID,
		`Abstract=? OR Abstract=?`, gAbs, rAbs)
	if err != nil {
		return err
	}

	// First, let's make sure each Resource doesn't have too many Versions
	// or has too many root versions

	group := (*Group)(nil)
	resource := (*Resource)(nil)
	for _, e := range entities {
		if e.Type == ENTITY_GROUP {
			group = &Group{Entity: *e, Registry: reg}
			group.Self = group
		} else {
			PanicIf(group == nil, "Group can't be nil")
			resource = &Resource{Entity: *e, Group: group}
			resource.Self = resource

			if err = resource.EnsureSingleVersionRoot(); err != nil {
				return err
			}

			if err = resource.EnsureMaxVersions(); err != nil {
				return err
			}

			resource.tx.AddResource(resource)
		}
	}

	return nil
}

func (rm *ResourceModel) SetModel(m *Model) {
	if rm == nil {
		return
	}

	if rm.Attributes == nil {
		rm.Attributes = map[string]*Attribute{}
	}

	rm.Attributes.SetModel(m)
}

func (rm *ResourceModel) SetMaxVersions(maxV int) error {
	rm.MaxVersions = maxV
	return rm.VerifyAndSave()
}

func (rm *ResourceModel) SetSetDefaultSticky(val bool) error {
	rm.SetDefaultSticky = PtrBool(val)
	return rm.VerifyAndSave()
}

func (rm *ResourceModel) VerifyAndSave() error {
	if err := rm.Verify(rm.Plural); err != nil {
		return err
	}
	return rm.Save()
}

func (rm *ResourceModel) GetBaseMetaAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, rm.MetaAttributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	for _, specProp := range OrderedSpecProps {
		if specProp.Name == "id" {
			// Skip "id"
			continue
		}

		if specProp.InType(ENTITY_META) && IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		}
	}

	return attrs
}

func EnsureJustOneRESOURCE(obj map[string]any, singular string) error {
	count := 0
	list := []string{"", "url", "base64", "proxyurl"}
	for i, suffix := range list {
		list[i] = singular + suffix
		if v, ok := obj[list[i]]; ok && !IsNil(v) {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("Only one of %s can be present at a time",
			strings.Join(list, ",")) // include proxyurl
	}
	return nil
}

func RESOURCEcheckFn(e *Entity) error {
	_, rm := e.GetModels()
	return EnsureJustOneRESOURCE(e.NewObject, rm.Singular)
}

func (rm *ResourceModel) GetBaseAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, rm.Attributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	/*
	   for _, specProp := range OrderedSpecProps {
	       if specProp.InType(eType) {
	           attrs[specProp.Name] = specProp
	       }
	   }
	*/

	// Find all Resource level attributes (not Meta) so we can show them
	// mixed in with the Default Version attributes - e.g. metaurl
	for _, specProp := range OrderedSpecProps {
		if specProp.Name == "id" {
			// Skip "id"
			continue
		}

		if specProp.InType(ENTITY_RESOURCE) && IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		}
	}

	// Resource has hasDoc=true, then add $RESOURCE attrs
	if rm.GetHasDocument() {
		attrs[rm.Singular] =
			SpecProps["$RESOURCE"].Clone(rm.Singular)

		attrs[rm.Singular+"url"] =
			SpecProps["$RESOURCEurl"].Clone(rm.Singular + "url")

		attrs[rm.Singular+"proxyurl"] =
			SpecProps["$RESOURCEproxyurl"].Clone(rm.Singular + "proxyurl")
	}

	// Either way, delete the template ones since they're not used
	delete(attrs, "$RESOURCE")
	delete(attrs, "$RESOURCEurl")
	delete(attrs, "$RESOURCEproxyurl")

	return attrs
}

func (rm *ResourceModel) AddTypeMap(ct string, format string) error {
	oldMap := maps.Clone(rm.TypeMap)

	if format != "binary" && format != "json" && format != "string" {
		return fmt.Errorf("Invalid typemap format: %q", format)
	}
	if rm.TypeMap == nil {
		rm.TypeMap = map[string]string{}
	}
	rm.TypeMap[ct] = format

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		rm.TypeMap = oldMap
		return err
	}
	return nil
}

func (rm *ResourceModel) RemoveTypeMap(ct string) error {
	oldMap := maps.Clone(rm.TypeMap)

	if rm.TypeMap == nil {
		return nil
	}
	delete(rm.TypeMap, ct)
	if len(rm.TypeMap) == 0 {
		rm.TypeMap = nil
	}

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		rm.TypeMap = oldMap
		return err
	}
	return nil
}

func (rm *ResourceModel) AddLabel(name string, value string) error {
	oldLabels := maps.Clone(rm.Labels)

	if rm.Labels == nil {
		rm.Labels = map[string]string{}
	}
	rm.Labels[name] = value

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		rm.Labels = oldLabels
		return err
	}
	return nil
}

func (rm *ResourceModel) RemoveLabel(name string) error {
	if rm.Labels == nil {
		return nil
	}

	oldLabels := maps.Clone(rm.Labels)

	delete(rm.Labels, name)
	if len(rm.Labels) == 0 {
		rm.Labels = nil
	}

	if err := rm.GroupModel.Model.VerifyAndSave(); err != nil {
		// Undo
		rm.Labels = oldLabels
		return err
	}
	return nil
}

// Map incoming "contentType" (ct) to its typemap value.
// If there is no match (or more than one match with a different type)
// then default to "binary"
func (rm *ResourceModel) MapContentType(ct string) string {
	result := ""

	// Strip all parameters
	ct, _ = strings.CutSuffix(ct, ";")
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return "binary"
	}

	for k, v := range rm.TypeMap {
		k = strings.ToLower(k)
		if Match(k, ct) {
			// We got another match but it's a different value, so "binary"
			if result != "" && result != v {
				return "binary"
			}
			// Save result so we can check to see if there's another match
			result = v
		}
	}
	// If we have at least one match, with the same value, return the value
	if result != "" {
		return result
	}

	// Check our implied/default typemaps before we give up
	if Match("application/json", ct) || Match("*+json", ct) {
		return "json"
	}
	if Match("text/plain", ct) {
		return "string"
	}

	return "binary"
}

type LevelData struct {
	Model *Model
	// AttrNames is the list of known attribute names for a certain eType
	// an entity (basically the Attributes list + ifValues). We use this to know
	// if an IfValue SiblingAttribute would conflict if another attribute's name
	AttrNames map[string]bool
	Path      *PropPath
}

func (attrs Attributes) ConvertStrings(obj Object) {
	for key, val := range obj {
		attr := attrs[key]
		if attr == nil {
			attr = attrs["*"]
			if attr == nil {
				// Can't find it, so it must be an error.
				// Assume we'll catch it during the normal verification checks
				continue
			}
		}

		// We'll only try to convert strings and one-level-scalar maps
		valValue := reflect.ValueOf(val)
		if valValue.Kind() != reflect.String && valValue.Kind() != reflect.Map {
			continue
		}
		valStr := fmt.Sprintf("%v", val)

		// If not one of these, just skip it
		switch attr.Type {
		case BOOLEAN, DECIMAL, INTEGER, UINTEGER:
			if newVal, ok := ConvertString(valStr, attr.Type); ok {
				// Replace the string with the non-string value
				obj[key] = newVal
			}
		case MAP:
			if valValue.Kind() == reflect.Map {
				valMap := val.(map[string]any)
				for k, v := range valMap {
					vStr := fmt.Sprintf("%v", v)
					// Only saved the converted string if we did a conversion
					if nV, ok := ConvertString(vStr, attr.Item.Type); ok {
						valMap[k] = nV
					}
				}
			}
		}
	}
}

func ConvertString(val string, toType string) (any, bool) {
	switch toType {
	case BOOLEAN:
		if val == "true" {
			return true, true
		} else if val == "false" {
			return false, true
		}
	case DECIMAL:
		tmpFloat, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return tmpFloat, true
		}
	case INTEGER, UINTEGER:
		tmpInt, err := strconv.Atoi(val)
		if err == nil {
			return tmpInt, true
		}
	}
	return nil, false
}

// 0=complete 1=GROUPS 2=RESOURCES|"" 3=versions|""  4=[/versions]|""
// nil, or [0]="" means error
var targetREstr = `^(?:/([^/]+)(?:/([^[/]+)(?:(?:/(versions)|(\[(?:/versions)]))?))?)?$`
var targetRE = regexp.MustCompile(targetREstr)

func (attrs Attributes) Verify(namecharset string, ld *LevelData) error {
	ld = &LevelData{
		Model:     ld.Model,
		AttrNames: maps.Clone(ld.AttrNames),
		Path:      ld.Path.Clone(),
	}
	if ld.AttrNames == nil {
		ld.AttrNames = map[string]bool{}
	}

	// First add the new attribute names, while checking the attr
	for name, attr := range attrs {
		if attr == nil {
			return fmt.Errorf("Error processing %q: "+
				"attribute %q can't be empty", ld.Path.UI(), name)
		}
		if name == "" { // attribute key empty?
			return fmt.Errorf("Error processing %q: "+
				"it has an empty attribute key", ld.Path.UI())
		}
		if ld.AttrNames[name] == true { // Dup attr name?
			return fmt.Errorf("Duplicate attribute name (%s) at: %s", name,
				ld.Path.UI())
		}
		// Not sure why we look at SpecProp, I suspect it's because at one
		// point in time we had non-conforming (special) names in there and
		// we wanted to let those pass.
		// Technically we should convert the XXXid into id but any XXXid needs
		// to be a valid name/string so we should be ok
		if name != "*" && SpecProps[name] == nil {
			if namecharset == "extended" {
				if err := IsValidMapKey(name); err != nil {
					return fmt.Errorf("Error processing %q: %s", ld.Path.UI(), err)
				}
			} else if namecharset == "strict" || namecharset == "" {
				if err := IsValidAttributeName(name); err != nil {
					return fmt.Errorf("Error processing %q: %s", ld.Path.UI(), err)
				}
			} else {
				return fmt.Errorf("Invalid \"namecharset\" value: %s",
					namecharset)
			}
		}
		path := ld.Path.P(name)
		if attr.Name == "" {
			// auto-populate
			attr.Name = name
		}
		if name != attr.Name { // missing Name: field?
			return fmt.Errorf("%q must have a \"name\" set to %q", path.UI(),
				name)
		}
		if attr.Type == "" {
			return fmt.Errorf("%q is missing a \"type\"", path.UI())
		}
		if DefinedTypes[attr.Type] != true { // valie Type: field?
			return fmt.Errorf("%q has an invalid type: %s", path.UI(),
				attr.Type)
		}

		if attr.Type == XID {
			/* no longer required
			if attr.Target == "" {
				return fmt.Errorf("%q must have a \"target\" value "+
					"since \"type\" is \"xid\"", path.UI())
			}
			*/
			if attr.Target != "" {
				target := strings.TrimSpace(attr.Target)
				parts := targetRE.FindStringSubmatch(target)
				// 0=all  1=GROUPS  2=RESOURCES  3=versions|""  4=[/versions]|""
				if len(parts) == 0 || parts[0] == "" {
					return fmt.Errorf("%q \"target\" must be of the form: "+
						"/GROUPS[/RESOURCES[/versions | \\[/versions\\] ]]",
						path.UI())
				}

				gm := ld.Model.FindGroupModel(parts[1])
				if gm == nil {
					return fmt.Errorf("%q has an unknown Group type: %q",
						path.UI(), parts[1])
				}
				if parts[2] != "" {
					if rm := gm.Resources[parts[2]]; rm == nil {
						return fmt.Errorf("%q has an unknown Resource type: %q",
							path.UI(), parts[2])
					}
				}
			}
		}

		if attr.Target != "" && attr.Type != XID {
			return fmt.Errorf("%q must not have a \"target\" value "+
				"since \"type\" is not \"xid\"", path.UI())
		}

		// Is it ok for strict=true and enum=[] ? Require no value???
		// if attr.Strict == true && len(attr.Enum) == 0 {
		// }

		// check enum values
		if attr.Enum != nil && len(attr.Enum) == 0 {
			return fmt.Errorf("%q specifies an \"enum\" but it is empty",
				path.UI())
		}
		if len(attr.Enum) > 0 {
			if IsScalar(attr.Type) != true {
				return fmt.Errorf("%q is not a scalar, so \"enum\" is not "+
					"allowed", path.UI())
			}

			for _, val := range attr.Enum {
				if !IsOfType(val, attr.Type) {
					return fmt.Errorf("%q enum value \"%v\" must be of type %q",
						path.UI(), val, attr.Type)
				}
			}
		}

		if !IsNil(attr.Default) {
			if IsScalar(attr.Type) != true {
				return fmt.Errorf("%q is not a scalar, so \"default\" is not "+
					"allowed", path.UI())
			}

			val := attr.Default
			if !IsOfType(val, attr.Type) {
				return fmt.Errorf("%q \"default\" value must be of type %q",
					path.UI(), attr.Type)
			}

			if attr.Required == false {
				return fmt.Errorf("%q must have \"require\" set to "+
					"\"true\" since a default value is defined", path.UI())
			}
		}

		// Object doesn't need an Item, but maps and arrays do
		if attr.Type == MAP || attr.Type == ARRAY {
			if attr.Item == nil {
				return fmt.Errorf("%q must have an \"item\" section", path.UI())
			}
		}

		if attr.Type == OBJECT {
			if attr.NameCharSet != "" && attr.NameCharSet != "strict" && attr.NameCharSet != "extended" {
				return fmt.Errorf("%q has an invalid \"namecharset\" value: "+
					attr.NameCharSet, path.UI())
			}

			if attr.Item != nil {
				return fmt.Errorf("%q must not have an \"item\" section", path.UI())
			}
			if err := attr.Attributes.Verify(attr.NameCharSet, &LevelData{ld.Model, nil, path}); err != nil {
				return err
			}
		} else {
			if attr.NameCharSet != "" {
				return fmt.Errorf("Attribute %q must not have a "+
					"\"namecharset\" value unless its type is \"object\"",
					attr.Name)
			}
		}

		if attr.Item != nil {
			if err := attr.Item.Verify(path); err != nil {
				return err
			}
		}

		ld.AttrNames[attr.Name] = true
	}

	// Now that we have all of the attribute names for this level, go ahead
	// and check the IfValues, not just for validatity but to also make sure
	// they don't define duplicate attribute names
	for _, attr := range attrs {
		for valStr, ifValue := range attr.IfValues {
			if valStr == "" {
				return fmt.Errorf("%q has an empty ifvalues key", ld.Path.UI())
			}

			if valStr[0] == '^' {
				return fmt.Errorf("%q has an ifvalues key that starts "+
					"with \"^\"", ld.Path.UI())
			}

			nextLD := &LevelData{
				ld.Model,
				ld.AttrNames,
				ld.Path.P(attr.Name).P("ifvalues").P(valStr)}

			// Recursive
			if err := ifValue.SiblingAttributes.Verify(namecharset, nextLD); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copy the internal data for spec defined properties so we can access
// that info directly from these Attributes instead of having to go back
// to the SpecProps stuff
func (attrs Attributes) SetSpecPropsFields(singular string) {
	for k, attr := range attrs {
		if k == singular+"id" {
			k = "id"
		}
		if specProp := SpecProps[k]; specProp != nil {
			attr.internals = specProp.internals
		}
	}
}

func (ifvalues IfValues) SetModel(m *Model) {
	if ifvalues == nil {
		return
	}

	for _, ifvalue := range ifvalues {
		ifvalue.SiblingAttributes.SetModel(m)
	}
}

func (item *Item) Verify(path *PropPath) error {
	p := path.P("item")

	if item.Type == "" {
		return fmt.Errorf("%q must have a \"type\" defined", p.UI())
	}

	if DefinedTypes[item.Type] != true {
		return fmt.Errorf("%q has an invalid \"type\": %s", p.UI(),
			item.Type)
	}

	if item.Type != OBJECT && item.Attributes != nil {
		return fmt.Errorf("%q must not have \"attributes\"", p.UI())
	}

	if item.Type == MAP || item.Type == ARRAY {
		if item.Item == nil {
			return fmt.Errorf("%q must have an \"item\" section", p.UI())
		}
	}

	if item.Type == OBJECT {
		if item.NameCharSet != "" && item.NameCharSet != "strict" && item.NameCharSet != "extended" {
			return fmt.Errorf("Invalid \"namecharset\" value: %s",
				item.NameCharSet)
		}
	} else {
		if item.NameCharSet != "" {
			return fmt.Errorf("%q must not have a \"namecharset\" value "+
				"since it is not of type \"object\"", p.UI())
		}
	}

	if item.Attributes != nil {
		if err := item.Attributes.Verify(item.NameCharSet, &LevelData{item.Model, nil, p}); err != nil {
			return err
		}
	}

	if item.Item != nil {
		return item.Item.Verify(p)
	}
	return nil
}

var DefinedTypes = map[string]bool{
	ANY:     true,
	BOOLEAN: true,
	DECIMAL: true, INTEGER: true, UINTEGER: true,
	ARRAY:     true,
	MAP:       true,
	OBJECT:    true,
	XID:       true,
	STRING:    true,
	TIMESTAMP: true,
	URI:       true, URI_REFERENCE: true, URI_TEMPLATE: true, URL: true}

// attr.Type must be a scalar
// Used to check JSON type vs our types
func IsOfType(val any, attrType string) bool {
	switch reflect.ValueOf(val).Kind() {
	case reflect.Bool:
		return attrType == BOOLEAN

	case reflect.String:
		if attrType == TIMESTAMP {
			str := val.(string)
			_, err := time.Parse(time.RFC3339, str)
			return err == nil
		}

		return IsString(attrType)

	case reflect.Float64: // JSON ints show up as floats
		if attrType == DECIMAL {
			return true
		}
		if attrType == INTEGER || attrType == UINTEGER {
			valInt := int(val.(float64))
			if float64(valInt) != val.(float64) {
				return false
			}
			return attrType == INTEGER || valInt >= 0
		}
		return false

	case reflect.Int:
		if attrType == DECIMAL {
			return true
		}
		if attrType == INTEGER || attrType == UINTEGER {
			valInt := val.(int)
			return attrType == INTEGER || valInt >= 0
		}
		return false

	default:
		return false
	}
}

/*
func AbstractToSingular(reg *Registry, abs string) string {
	absParts := strings.Split(abs, string(DB_IN))

	if len(absParts) == 0 {
		panic("help")
	}
	gm := reg.Model.Groups[absParts[0]]
	PanicIf(gm == nil, "no gm")

	if len(absParts) == 1 {
		return gm.Singular
	}

	rm := gm.Resources[absParts[1]]
	PanicIf(rm == nil, "no rm")
	return rm.Singular
}
*/

// The model serializer we use for the "xRegistry" schema format
func Model2xRegistryJson(m *Model, format string) ([]byte, error) {
	m.SetSingular()

	buf, err := json.MarshalIndent(m, "", "  ")
	m.UnsetSingular()

	return buf, err
}

func GetModelSerializer(format string) ModelSerializer {
	format = strings.ToLower(format)
	searchParts := strings.SplitN(format, "/", 2)
	if searchParts[0] == "" {
		return nil
	}
	if len(searchParts) == 1 {
		searchParts = append(searchParts, "")
	}

	result := ModelSerializer(nil)
	resultVersion := ""

	for format, sm := range ModelSerializers {
		format = strings.ToLower(format)
		parts := strings.SplitN(format, "/", 2)
		if searchParts[0] != parts[0] {
			continue
		}
		if len(parts) == 1 {
			parts = append(parts, "")
		}

		if searchParts[1] != "" {
			if searchParts[1] == parts[1] {
				// Exact match - stop immediately
				result = sm
				break
			}
			// Looking for an exact match - not it so skip it
			continue
		}

		if resultVersion == "" || strings.Compare(parts[1], resultVersion) > 0 {
			result = sm
			resultVersion = parts[1]
		}
	}

	return result
}

func RegisterModelSerializer(name string, sm ModelSerializer) {
	ModelSerializers[name] = sm
}

func init() {
	RegisterModelSerializer(XREGSCHEMA+"/"+SPECVERSION, Model2xRegistryJson)
}

func AbstractToModels(reg *Registry, abs string) (*GroupModel, *ResourceModel) {
	parts := strings.Split(abs, string(DB_IN))
	if len(parts) == 0 || parts[0] == "" {
		return nil, nil
	}
	gm := reg.Model.Groups[parts[0]]
	PanicIf(gm == nil, "Can't find Group %q", parts[0])

	rm := (*ResourceModel)(nil)
	if len(parts) > 1 {
		rm = gm.Resources[parts[1]]
		PanicIf(rm == nil, "Cant' find Resource \"%s/%s\"", parts[0], parts[1])
	}
	// *GroupModel, *ResourceModel, isVersion
	return gm, rm
}

func (attr *Attribute) Clone(newName string) *Attribute {
	newAttr := *attr
	if newName != "" {
		newAttr.Name = newName
	}

	return &newAttr
}
