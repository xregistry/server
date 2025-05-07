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

	propsOrdered []*Attribute
	propsMap     map[string]*Attribute
	changed      bool
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
	internals *AttrInternals
}

type Item struct { // for maps and arrays
	Model       *Model     `json:"-"`
	Type        string     `json:"type,omitempty"`
	Target      string     `json:"target,omitempty"`
	NameCharSet string     `json:"namecharset,omitempty"` // when 'type'=obj
	Attributes  Attributes `json:"attributes,omitempty"`  // when 'type'=obj
	Item        *Item      `json:"item,omitempty"`        // when 'type'=map,array
}

type IfValues map[string]*IfValue

type IfValue struct {
	SiblingAttributes Attributes `json:"siblingattributes,omitempty"`
}

type GroupModel struct {
	SID   string `json:"sid,omitempty"`
	Model *Model `json:"-"`

	Plural         string            `json:"plural"`
	Singular       string            `json:"singular"`
	Description    string            `json:"description,omitempty"`
	ModelVersion   string            `json:"modelversion,omitempty"`
	CompatibleWith string            `json:"compatiblewith,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Attributes     Attributes        `json:"attributes,omitempty"`

	// [ GROUPS/RESOURCES * ]
	XImportResources []string                  `json:"ximportresources,omitempty"`
	Resources        map[string]*ResourceModel `json:"resources,omitempty"` // Plural

	propsOrdered []*Attribute
	propsMap     map[string]*Attribute
	imports      map[string]*ResourceModel
}

type ResourceModel struct {
	SID        string      `json:"sid,omitempty"`
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

	propsOrdered        []*Attribute
	propsMap            map[string]*Attribute
	metaPropsOrdered    []*Attribute
	metaPropsMap        map[string]*Attribute
	versionPropsOrdered []*Attribute
	versionPropsMap     map[string]*Attribute
}

func (attrs Attributes) MarshalJSON() ([]byte, error) {
	// Just alphabetize the list (which go already does) BUT put "*" last
	list := SortedKeys(attrs)
	if len(list) > 0 && list[0] == "*" {
		list = append(list[1:], list[0])
	}

	buf := bytes.Buffer{}
	buf.WriteRune('{')
	for i, k := range list {
		if i > 0 {
			buf.WriteString(`,"`)
		} else {
			buf.WriteRune('"')
		}
		buf.WriteString(k)
		buf.WriteString(`":`)
		b, _ := json.Marshal(attrs[k])
		buf.Write(b)
	}
	buf.WriteRune('}')
	return buf.Bytes(), nil
}

func (r *ResourceModel) UnmarshalJSON(data []byte) error {
	// Set the default values before we unmarshal
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

func (m *Model) SetSpecPropsFields() {
	propsOrdered, _ := m.GetPropsOrdered()

	for _, prop := range propsOrdered {
		if attr, ok := m.Attributes[prop.Name]; ok {
			attr.internals = prop.internals
		}
	}

	for plural, gm := range m.Groups {
		if attr, ok := m.Attributes[plural]; ok {
			attr.internals = CollectionsAttr.internals
		}
		if attr, ok := m.Attributes[plural+"url"]; ok {
			attr.internals = CollectionsURLAttr.internals
		}
		if attr, ok := m.Attributes[plural+"count"]; ok {
			attr.internals = CollectionsCountAttr.internals
		}

		propsOrdered, _ := gm.GetPropsOrdered()
		for _, prop := range propsOrdered {
			if attr, ok := m.Attributes[prop.Name]; ok {
				attr.internals = prop.internals
			}
		}

		for plural, rm := range gm.Resources {
			if attr, ok := gm.Attributes[plural]; ok {
				attr.internals = CollectionsAttr.internals
			}
			if attr, ok := gm.Attributes[plural+"url"]; ok {
				attr.internals = CollectionsURLAttr.internals
			}
			if attr, ok := gm.Attributes[plural+"count"]; ok {
				attr.internals = CollectionsCountAttr.internals
			}

			if attr, ok := rm.Attributes["versions"]; ok {
				attr.internals = CollectionsAttr.internals
			}
			if attr, ok := rm.Attributes["versionsurl"]; ok {
				attr.internals = CollectionsURLAttr.internals
			}
			if attr, ok := rm.Attributes["versionscount"]; ok {
				attr.internals = CollectionsCountAttr.internals
			}

			propsOrdered, _ := rm.GetPropsOrdered()
			for _, prop := range propsOrdered {
				if attr, ok := rm.Attributes[prop.Name]; ok {
					attr.internals = prop.internals
				}
			}

			propsOrdered, _ = rm.GetMetaPropsOrdered()
			for _, prop := range propsOrdered {
				if attr, ok := rm.MetaAttributes[prop.Name]; ok {
					attr.internals = prop.internals
				}
			}

			/*
				propsOrdered, _ := rm.GetVersionPropsOrdered()
				for _, prop := range propsOrdered {
					if attr, ok := rm.VersionAttributes[prop.Name]; ok {
						attr.internals = prop.internals
					}
				}
			*/
		}
	}
}

func (m *Model) ClearAllPropsOrdered() {
	m.ClearPropsOrdered()
	for _, gm := range m.Groups {
		gm.ClearPropsOrdered()
		for _, rm := range gm.Resources {
			rm.ClearPropsOrdered()
		}
	}
}

func (m *Model) ClearPropsOrdered() {
	m.propsOrdered = nil
	m.propsMap = nil
}

func (m *Model) SetChanged(val bool) {
	m.changed = val
	// ShowStack()
}

func (m *Model) GetChanged() bool {
	return m.changed
}

func (m *Model) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	if m.propsOrdered == nil {
		m.propsOrdered = []*Attribute{}
		m.propsMap = map[string]*Attribute{}

		for _, prop := range OrderedSpecProps {
			if prop.InType(ENTITY_REGISTRY) {
				if prop.Name == "id" {
					prop = prop.Clone("registryid")
					prop.ReadOnly = true
					PanicIf(prop.internals.checkFn == nil, "bad clone")
				}

				if prop.Name == "$COLLECTIONS" {
					for _, plural := range SortedKeys(m.Groups) {
						prop = CollectionsURLAttr.Clone(plural + "url")
						m.propsOrdered = append(m.propsOrdered, prop)
						m.propsMap[prop.Name] = prop

						prop = CollectionsCountAttr.Clone(plural + "count")
						m.propsOrdered = append(m.propsOrdered, prop)
						m.propsMap[prop.Name] = prop

						prop = CollectionsAttr.Clone(plural)
						m.propsOrdered = append(m.propsOrdered, prop)
						m.propsMap[prop.Name] = prop
					}
					continue
				}

				m.propsOrdered = append(m.propsOrdered, prop)
				m.propsMap[prop.Name] = prop
			}
		}
	}
	return m.propsOrdered, m.propsMap
}

// VerifyAndSave() should be called by automatically but there may be
// cases where someone would need to call it manually (e.g. setting an
// attribute's property - we should technically find a way to catch those
// cases so code above this shouldn't need to think about it
func (m *Model) VerifyAndSave() error {
	if m.GetChanged() == false {
		return nil
	}

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
	// log.Printf("In model.Save - changed: %v", m.GetChanged())
	if m.GetChanged() == false {
		return nil
	}

	// Create a temporary type so that we don't use the MarshalJSON func
	// in model.go. That one will exclude "model" from the serialization and
	// we don't want to do that when we're saving it in the DB. We only want
	// to do that when we're serializing the model for the end user.

	buf, _ := json.Marshal(m)
	modelStr := string(buf)

	type tmpAttributes Attributes
	buf, _ = json.Marshal((tmpAttributes)(m.Attributes))
	attrs := string(buf)

	buf, _ = json.Marshal(m.Labels)
	labels := string(buf)

	// log.Printf("Saving model itself")
	err := DoZeroTwo(m.Registry.tx, `
        INSERT INTO Models(RegistrySID, Model, Labels, Attributes)
        VALUES(?,?,?,?)
        ON DUPLICATE KEY UPDATE Model=?,Labels=?,Attributes=? `,

		m.Registry.DbSID, modelStr, labels, attrs,
		modelStr, labels, attrs)
	if err != nil {
		log.Printf("Error updating model: %s", err)
		return err
	}

	existingModelEntities := map[string]string{} // Abstract->SID
	results, err := Query(m.Registry.tx,
		`SELECT SID,Abstract FROM ModelEntities WHERE RegistrySID=?`,
		m.Registry.DbSID)
	defer results.Close()
	if err != nil {
		log.Printf("Error loading model entities(%s): %s", m.Registry.UID, err)
		return nil
	}
	for {
		row := results.NextRow()
		if row == nil {
			break
		}
		sid := NotNilString(row[0])
		abs := NotNilString(row[1])
		existingModelEntities[abs] = sid
	}

	// Remove from existingModelEntities, all MEs that are going to be kept
	// around. Then we'll delete everything else before we re-add the keepers
	// to ensure there isn't any conflicts.
	// We can't just delete the entire set and re-add them because the DB
	// will erase all instances of those types automatically when the types
	// are deleted.

	inUseAbs := map[string]bool{}
	for _, gm := range m.Groups {
		for _, rName := range gm.XImportResources {
			parts := strings.Split(rName, "/")
			rAbs := "/" + gm.Plural + "/" + parts[2]
			if _, ok := existingModelEntities[rAbs]; ok {
				inUseAbs[rAbs] = true
			}
		}
		gAbs := "/" + gm.Plural
		if _, ok := existingModelEntities[gAbs]; ok {
			inUseAbs[gAbs] = true
		}
		for _, rm := range gm.Resources {
			rmAbs := gAbs + "/" + rm.Plural
			if _, ok := existingModelEntities[rmAbs]; ok {
				inUseAbs[rmAbs] = true
			}
		}
	}

	// Delete any model entities not found in the new model
	for meAbs, _ := range existingModelEntities {
		if inUseAbs[meAbs] != true {
			// TODO if we ever think this list is long, make this faster
			err = DoOne(m.Registry.tx, `
                      DELETE FROM ModelEntities
                      WHERE RegistrySID=? AND Abstract=?`,
				m.Registry.DbSID, meAbs)
			if err != nil {
				log.Printf("Error deleting modelEntity(%s): %s", meAbs, err)
				return err
			}
		}
	}

	// Now just add the new ones
	for _, gm := range m.Groups {
		gmAbs := "/" + gm.Plural
		for _, rName := range gm.XImportResources {
			// See if the ximported resource is alread there, if so skip it
			parts := strings.Split(rName, "/")
			targetRM := gm.Model.FindResourceModel(parts[1], parts[2])
			rSID := gm.SID + "-" + targetRM.SID // parts[2]
			rAbs := gmAbs + "/" + parts[2]
			if _, ok := existingModelEntities[rAbs]; !ok {
				// add the new ximported resource
				err := Do(m.Registry.tx,
					`INSERT INTO ModelEntities(
                         SID, RegistrySID, ParentSID,
                         Abstract, Plural, Singular)
                     VALUES(?,?,?,?,?,?)`,
					rSID, m.Registry.DbSID, gm.SID,
					rAbs, targetRM.Plural, targetRM.Singular)
				if err != nil {
					log.Printf("Error inserting modelEntity(%s): %s", rSID, err)
					return err
				}
			}
		}

		// If GroupModel is already in DB then skip it
		if _, ok := existingModelEntities[gmAbs]; !ok {
			// Add new GroupModel
			err := Do(m.Registry.tx,
				`INSERT INTO ModelEntities(
                     SID, RegistrySID, ParentSID,
                     Abstract, Plural, Singular)
                 VALUES(?,?,?,?,?,?)`,
				gm.SID, m.Registry.DbSID, nil,
				gmAbs, gm.Plural, gm.Singular)
			if err != nil {
				log.Printf("Error inserting modelEntity(%s): %s", gm.SID, err)
				return err
			}
		}

		for _, rm := range gm.Resources {
			rmAbs := gmAbs + "/" + rm.Plural
			// If ResourceModel is already in DB then skip it
			if _, ok := existingModelEntities[rmAbs]; !ok {
				// Add new ResourceModel
				err := Do(m.Registry.tx,
					`INSERT INTO ModelEntities(
                             SID, RegistrySID, ParentSID,
                             Abstract, Plural, Singular)
                         VALUES(?,?,?,?,?,?)`,
					rm.SID, m.Registry.DbSID, gm.SID,
					gmAbs+"/"+rm.Plural, rm.Plural, rm.Singular)
				if err != nil {
					log.Printf("Error inserting modelEntity(%s): %s",
						ToJSON(rm), err)
					return err
				}
			}
		}
	}

	m.SetChanged(false)

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
	} else if _, ok := m.Attributes[attr.Name]; ok {
		return nil, fmt.Errorf("Attribute %q already exists", attr.Name)
	}

	attr.Model = m
	m.Attributes[attr.Name] = attr
	attr.Item.SetModel(m)

	ld := &LevelData{
		Model:     m,
		AttrNames: map[string]bool{},
		Path:      NewPPP("model"),
	}
	if err := m.Attributes.Verify("strict", ld); err != nil {
		delete(m.Attributes, attr.Name)
		return nil, err
	}

	m.SetChanged(true)

	return attr, nil
}

func (m *Model) DelAttribute(name string) error {
	if m.Attributes == nil {
		return nil
	}

	delete(m.Attributes, name)

	m.SetChanged(true)

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
	gm := &GroupModel{
		SID:      mSID,
		Model:    m,
		Singular: singular,
		Plural:   plural,

		Resources: map[string]*ResourceModel{},
	}

	m.Groups[plural] = gm
	m.SetChanged(true)

	return gm, nil
}

func (m *Model) AddLabel(name string, value string) error {
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	m.Labels[name] = value

	m.SetChanged(true)

	return nil
}

func (m *Model) RemoveLabel(name string) error {
	if m.Labels == nil {
		return nil
	}

	delete(m.Labels, name)
	if len(m.Labels) == 0 {
		m.Labels = nil
	}

	m.SetChanged(true)

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
	i.Item = item
	item.SetModel(i.Model)

	i.Model.SetChanged(true)

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
	} else if _, ok := i.Attributes[attr.Name]; ok {
		return nil, fmt.Errorf("Attribute %q already exists", attr.Name)
	}

	i.Attributes[attr.Name] = attr

	attr.Model = i.Model
	attr.Item.SetModel(i.Model)

	if i.Model != nil {
		i.Model.SetChanged(true)
	}

	return attr, nil
}

func (i *Item) DelAttribute(name string) error {
	if i.Attributes != nil {
		delete(i.Attributes, name)
	}

	i.Model.SetChanged(true)

	return nil
}

func LoadModel(reg *Registry) *Model {
	log.VPrintf(3, ">Enter: LoadModel")
	defer log.VPrintf(3, "<Exit: LoadModel")

	PanicIf(reg == nil, "nil")

	// Load Registry Labels, Attributes
	results, err := Query(reg.tx,
		`SELECT Model, Labels, Attributes FROM Models WHERE RegistrySID=?`,
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

	modelBuf := []byte(nil)
	if row[0] != nil {
		modelBuf = []byte(NotNilString(row[0]))
	}
	results.Close()

	model, err := ParseModel(modelBuf)
	if err != nil {
		return nil
	}
	model.Registry = reg

	reg.Model = model
	return model
}

func ParseModel(buf []byte) (*Model, error) {
	model := Model{}
	if err := Unmarshal(buf, &model); err != nil {
		return nil, err
	}
	model.SetPointers()
	model.SetSpecPropsFields()
	return &model, nil
}

func (m *Model) FindGroupModel(gType string) *GroupModel {
	if m.Groups == nil {
		return nil
	}
	return m.Groups[gType]
}

func (m *Model) FindResourceModel(gType string, rType string) *ResourceModel {
	gm := m.FindGroupModel(gType)
	if gm == nil {
		return nil
	}

	return gm.FindResourceModel(rType)
}

func (m *Model) ApplyNewModel(newM *Model) error {
	newM.Registry = m.Registry
	// log.Printf("ApplyNewModel:\n%s\n", ToJSON(newM))

	/*
		if err := newM.Verify(); err != nil {
			return err
		}
	*/

	m.Registry.Model = newM
	m = newM
	m.SetChanged(true)

	if err := m.VerifyAndSave(); err != nil {
		// Too much to undo. The Verify() at the top should have caught
		// anything wrong
		return err
	}

	return nil
}

func (gm *GroupModel) ClearPropsOrdered() {
	gm.propsOrdered = nil
	gm.propsMap = nil
}

func (gm *GroupModel) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	if gm.propsOrdered == nil {
		gm.propsOrdered = []*Attribute{}
		gm.propsMap = map[string]*Attribute{}

		for _, prop := range OrderedSpecProps {
			if prop.InType(ENTITY_GROUP) {
				if prop.Name == "id" {
					prop = prop.Clone(gm.Singular + "id")
					PanicIf(prop.internals.checkFn == nil, "bad clone")
				}

				if prop.Name == "$COLLECTIONS" {
					for _, plural := range SortedKeys(gm.Resources) {
						prop = CollectionsURLAttr.Clone(plural + "url")
						gm.propsOrdered = append(gm.propsOrdered, prop)
						gm.propsMap[prop.Name] = prop

						prop = CollectionsCountAttr.Clone(plural + "count")
						gm.propsOrdered = append(gm.propsOrdered, prop)
						gm.propsMap[prop.Name] = prop

						prop = CollectionsAttr.Clone(plural)
						gm.propsOrdered = append(gm.propsOrdered, prop)
						gm.propsMap[prop.Name] = prop
					}
					continue
				}

				gm.propsOrdered = append(gm.propsOrdered, prop)
				gm.propsMap[prop.Name] = prop
			}
		}
	}
	return gm.propsOrdered, gm.propsMap
}

func (gm *GroupModel) Delete() error {
	log.VPrintf(3, ">Enter: Delete.GroupModel: %s", gm.Plural)
	defer log.VPrintf(3, "<Exit: Delete.GroupModel")

	gm.Model.RemoveConditionalProps()
	delete(gm.Model.Groups, gm.Plural)

	gm.Model.SetChanged(true)

	return nil
}

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
	} else if _, ok := gm.Attributes[attr.Name]; ok {
		return nil, fmt.Errorf("Attribute %q already exists", attr.Name)
	}

	gm.Attributes[attr.Name] = attr

	attr.Model = gm.Model
	attr.Item.SetModel(gm.Model)

	ld := &LevelData{
		Model:     gm.Model,
		AttrNames: map[string]bool{},
		Path:      NewPPP("groups").P(gm.Plural),
	}
	if err := gm.Attributes.Verify("strict", ld); err != nil {
		delete(gm.Attributes, attr.Name)
		return nil, err
	}

	attr.Model.SetChanged(true)

	return attr, nil
}

func (gm *GroupModel) DelAttribute(name string) error {
	if gm.Attributes != nil {
		delete(gm.Attributes, name)
	}

	gm.Model.SetChanged(true)

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

	gm.Resources[rm.Plural] = rm

	rm.GroupModel.Model.SetChanged(true)

	return rm, nil
}

func (gm *GroupModel) AddLabel(name string, value string) error {
	if gm.Labels == nil {
		gm.Labels = map[string]string{}
	}
	gm.Labels[name] = value

	gm.Model.SetChanged(true)

	return nil
}

func (gm *GroupModel) RemoveLabel(name string) error {
	if gm.Labels == nil {
		return nil
	}

	delete(gm.Labels, name)
	if len(gm.Labels) == 0 {
		gm.Labels = nil
	}

	gm.Model.SetChanged(true)

	return nil
}

func (gm *GroupModel) GetImports() map[string]*ResourceModel {
	if gm.imports == nil && len(gm.XImportResources) > 0 {
		gm.imports = map[string]*ResourceModel{}
		for _, grName := range gm.XImportResources {
			parts := strings.Split(grName, "/")
			r := gm.Model.FindResourceModel(parts[1], parts[2])
			PanicIf(r == nil, "Can't find %q", grName)
			gm.imports[parts[2]] = r
		}
	}
	return gm.imports
}

func (gm *GroupModel) FindResourceModel(rType string) *ResourceModel {
	if gm == nil {
		return nil
	}
	if rm := gm.Resources[rType]; rm != nil {
		return rm
	}

	imps := gm.GetImports()
	if imps != nil {
		return imps[rType]
	}
	return nil
}

func (gm *GroupModel) GetResourceList() []string {
	list := make([]string, len(gm.Resources)+len(gm.XImportResources))
	i := 0
	for plural, _ := range gm.Resources {
		list[i] = plural
		i++
	}

	imps := gm.GetImports()
	for k, _ := range imps {
		list[i] = k
		i++
	}
	return list
}

func (rm *ResourceModel) ClearPropsOrdered() {
	rm.propsOrdered = nil
	rm.propsMap = nil
	rm.metaPropsOrdered = nil
	rm.metaPropsOrdered = nil
	rm.versionPropsOrdered = nil
	rm.versionPropsOrdered = nil
}

func (rm *ResourceModel) GetPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	if rm.propsOrdered == nil {
		rm.propsOrdered = []*Attribute{}
		rm.propsMap = map[string]*Attribute{}
		rm.metaPropsOrdered = []*Attribute{}
		rm.metaPropsMap = map[string]*Attribute{}
		rm.versionPropsOrdered = []*Attribute{}
		rm.versionPropsMap = map[string]*Attribute{}

		for _, prop := range OrderedSpecProps {
			if prop.Name == "id" { // singular=Resource's for all 3
				prop = prop.Clone(rm.Singular + "id")
				PanicIf(prop.internals.checkFn == nil, "bad clone")
			}

			// this will clone all $RESOURCE attribute
			if strings.HasPrefix(prop.Name, "$RESOURCE") {
				if rm.GetHasDocument() {
					name := rm.Singular + prop.Name[len("$RESOURCE"):]
					prop = prop.Clone(name)
				} else {
					continue // no hasDoc so skip it
				}
			}

			if prop.Name == "$COLLECTIONS" {
				if prop.InType(ENTITY_RESOURCE) || prop.InType(ENTITY_VERSION) {
					prop = CollectionsURLAttr.Clone("versionsurl")
					rm.propsOrdered = append(rm.propsOrdered, prop)
					rm.propsMap[prop.Name] = prop
					if prop.InType(ENTITY_VERSION) {
						rm.versionPropsOrdered = append(rm.versionPropsOrdered, prop)
						rm.versionPropsMap[prop.Name] = prop
					}

					prop = CollectionsCountAttr.Clone("versionscount")
					rm.propsOrdered = append(rm.propsOrdered, prop)
					rm.propsMap[prop.Name] = prop
					if prop.InType(ENTITY_VERSION) {
						rm.versionPropsOrdered = append(rm.versionPropsOrdered, prop)
						rm.versionPropsMap[prop.Name] = prop
					}

					prop = CollectionsAttr.Clone("versions")
					rm.propsOrdered = append(rm.propsOrdered, prop)
					rm.propsMap[prop.Name] = prop
					if prop.InType(ENTITY_VERSION) {
						rm.versionPropsOrdered = append(rm.versionPropsOrdered, prop)
						rm.versionPropsMap[prop.Name] = prop
					}
				}

				continue
			}

			if prop.InType(ENTITY_RESOURCE) || prop.InType(ENTITY_VERSION) {
				rm.propsOrdered = append(rm.propsOrdered, prop)
				rm.propsMap[prop.Name] = prop
			}
			if prop.InType(ENTITY_VERSION) {
				rm.versionPropsOrdered = append(rm.versionPropsOrdered, prop)
				rm.versionPropsMap[prop.Name] = prop
			}
			if prop.InType(ENTITY_META) {
				rm.metaPropsOrdered = append(rm.metaPropsOrdered, prop)
				rm.metaPropsMap[prop.Name] = prop
			}
		}
	}
	return rm.propsOrdered, rm.propsMap
}

func (rm *ResourceModel) Refresh() *ResourceModel {
	// Weird we need to assume the current rm's info (except Reg) is bad
	gm := rm.GroupModel
	return gm.Model.Registry.Model.FindResourceModel(gm.Plural, rm.Plural)
}

func (rm *ResourceModel) GetMetaPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	if rm.metaPropsOrdered == nil {
		rm.GetPropsOrdered()
	}

	return rm.metaPropsOrdered, rm.metaPropsMap
}

func (rm *ResourceModel) GetVersionPropsOrdered() ([]*Attribute, map[string]*Attribute) {
	if rm.versionPropsOrdered == nil {
		rm.GetPropsOrdered()
	}

	return rm.versionPropsOrdered, rm.versionPropsMap
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
		return HASDOCUMENT
	}
	return *rm.HasDocument
}

func (rm *ResourceModel) SetHasDocument(val bool) {
	if rm.GetHasDocument() != val {
		rm.HasDocument = PtrBool(val)
		rm.GroupModel.Model.SetChanged(true)
	}
}

func (rm *ResourceModel) Dump(indents ...string) {
	indent := strings.Join(indents, "")
	log.Printf("%sRM: %s/%s", indent, rm.Plural, rm.Singular)
	log.Printf("%s  - HasDoc: %v", indent, rm.GetHasDocument())
	log.Printf("%s  - Attributes:", indent)
	for _, attr := range rm.Attributes {
		attr.Dump(indent + "  ")
	}
}

func (attr *Attribute) Dump(indents ...string) {
	indent := strings.Join(indents, "")
	log.Printf("%s%s:", indent, attr.Name)
	log.Printf("%s  - type: ", indent, attr.Type)
	log.Printf("%s  - internal: %v", indent, attr.internals != nil)
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

	rm.GroupModel.RemoveConditionalProps()
	delete(rm.GroupModel.Resources, rm.Plural)

	rm.GroupModel.Model.SetChanged(true)

	return nil
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

	rm.MetaAttributes[attr.Name] = attr

	attr.Model = rm.GroupModel.Model
	attr.Item.SetModel(rm.GroupModel.Model)

	rm.GroupModel.Model.SetChanged(true)

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
	} else if _, ok := rm.Attributes[attr.Name]; ok {
		return nil, fmt.Errorf("Attribute %q already exists", attr.Name)
	}

	rm.Attributes[attr.Name] = attr

	attr.Model = rm.GroupModel.Model
	attr.Item.SetModel(rm.GroupModel.Model)

	ld := &LevelData{
		Model:     rm.GroupModel.Model,
		AttrNames: map[string]bool{},
		Path:      NewPPP("resources").P(rm.Plural),
	}
	if err := rm.Attributes.Verify("strict", ld); err != nil {
		delete(rm.Attributes, attr.Name)
		return nil, err
	}

	rm.GroupModel.Model.SetChanged(true)

	return attr, nil
}

func (rm *ResourceModel) DelMetaAttribute(name string) error {
	if rm.MetaAttributes != nil {
		delete(rm.MetaAttributes, name)
	}

	rm.GroupModel.Model.SetChanged(true)

	return nil
}

func (rm *ResourceModel) DelAttribute(name string) error {
	if rm.Attributes != nil {
		delete(rm.Attributes, name)
	}

	rm.GroupModel.Model.SetChanged(true)

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
		// Don't set changed since we don't want to save the ifvalue attrs
		// attr.Model.SetChanged(true)
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
	PanicIf(a.internals == nil, "nil")
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

// TODO add more setters
func (a *Attribute) SetStrict(val bool) {
	a.Strict = PtrBool(val)
	a.Model.SetChanged(true)
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
	} else if _, ok := a.Attributes[attr.Name]; ok {
		return nil, fmt.Errorf("Attribute %q already exists", attr.Name)
	}

	a.Attributes[attr.Name] = attr
	attr.SetModel(a.Model)

	attr.Model.SetChanged(true)

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

func RemoveCollectionProps(plural string, attrs Attributes,
	propsOrdered []*Attribute, propsMap map[string]*Attribute) []*Attribute {

	if attrs != nil {
		if attr, ok := attrs[plural]; ok && attr.Model != nil {
			attr.Model.SetChanged(true)
		}
		delete(attrs, plural)
		delete(attrs, plural+"count")
		delete(attrs, plural+"url")
	}

	if propsMap != nil {
		delete(propsMap, plural)
		delete(propsMap, plural+"count")
		delete(propsMap, plural+"url")
	}

	if propsOrdered != nil {
		for i := 0; i < len(propsOrdered); i++ {
			// Starts with "plural"
			prop := propsOrdered[i]
			if strings.HasPrefix(prop.Name, plural) {
				// Grab rest of string
				rest := prop.Name[len(plural):]
				if rest == "" || rest == "count" || rest == "url" {
					// Match, one of them so remove from array
					propsOrdered = append(propsOrdered[:i],
						propsOrdered[i+1:]...)
					i--
				}
			}
		}
	}

	return propsOrdered
}

func (m *Model) RemoveConditionalProps() {
	for _, gm := range m.Groups {
		m.propsOrdered = RemoveCollectionProps(gm.Plural, m.Attributes,
			m.propsOrdered, m.propsMap)
	}
}

func (m *Model) Verify() error {
	// TODO: Verify that the Registry data is model compliant

	m.ClearPropsOrdered()

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

	propsOrdered, _ := m.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if specProp.Name[0] == '$' {
			continue
		}

		modelAttr, ok := m.Attributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			m.Attributes[specProp.Name] = specProp
			m.SetChanged(true)
		} else {
			// It's there but make sure it's not changed in a bad way
			if err := EnsureAttrOK(modelAttr, specProp); err != nil {
				return err
			}
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
	propsOrdered, _ := m.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		} else {
			attrs[specProp.Name].internals = specProp.internals
		}
	}

	return attrs
}

func (gm *GroupModel) RemoveConditionalProps() {
	rList := gm.GetResourceList()
	for _, rName := range rList {
		rm := gm.FindResourceModel(rName)
		PanicIf(rm == nil, "Not found: %s", rName)
		gm.propsOrdered = RemoveCollectionProps(rm.Plural, gm.Attributes,
			gm.propsOrdered, gm.propsMap)
	}
}

func (gm *GroupModel) Verify(gmName string) error {
	gm.ClearPropsOrdered()

	if err := IsValidModelName(gmName); err != nil {
		return err
	}

	if gm.Plural == "" {
		// Allow auto-populate
		gm.Plural = gmName
		gm.Model.SetChanged(true)
	} else if gm.Plural != gmName {
		return fmt.Errorf("Group %q must have a `plural` value of %q, not %q",
			gmName, gmName, gm.Plural)
	}

	if gm.SID == "" {
		gm.SID = NewUUID()
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

	// Verify the "ximportresources" list, same names for later checking
	resList := map[string]bool{}
	for _, grName := range gm.XImportResources {
		parts := strings.Split(grName, "/")
		if len(parts) != 3 {
			return fmt.Errorf("Group %q has an invalid ximportresources value "+
				"(%s), must be of the form \"/GroupType/ResourceType\"",
				gm.Plural, grName)
		}
		if parts[0] != "" {
			return fmt.Errorf("Group %q has an invalid ximportresources value "+
				"(%s), must start with \"/\" and be of the form "+
				"\"/GroupType/ResourceType\"", gm.Plural, grName)
		}
		if parts[1] == gm.Plural {
			return fmt.Errorf("Group %q has a bad ximportresources value "+
				"(%s), it can't reference itself", gm.Plural, grName)
		}

		g := gm.Model.FindGroupModel(parts[1])
		if g == nil {
			return fmt.Errorf("Group %q references a non-existing Group in: "+
				"%s", gm.Plural, grName)
		}

		r := g.FindResourceModel(parts[2])
		if r == nil {
			return fmt.Errorf("Group %q references a non-existing Resource "+
				"in: "+"%s", gm.Plural, grName)
		}
		resList[r.Plural] = true
	}

	// Verify the Resources to catch invalid Resource names early
	for rmName, rm := range gm.Resources {
		if rm == nil {
			return fmt.Errorf("Resource %q can't be empty", rmName)
		}

		if resList[rmName] == true {
			return fmt.Errorf("Resource %q is a duplicate name from the "+
				"\"ximportresources\" list", rmName)
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

	propsOrdered, _ := gm.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if specProp.Name[0] == '$' {
			continue
		}

		modelAttr, ok := gm.Attributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			gm.Attributes[specProp.Name] = specProp
			gm.Model.SetChanged(true)
		} else {
			// It's there but make sure it's not changed in a bad way
			if err := EnsureAttrOK(modelAttr, specProp); err != nil {
				return err
			}
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
		rm.GroupModel = gm
		rm.SetModel(m)
	}
}

func (gm *GroupModel) GetBaseAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, gm.Attributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	propsOrdered, _ := gm.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		} else {
			attrs[specProp.Name].internals = specProp.internals
		}
	}

	return attrs
}

func (rm *ResourceModel) RemoveConditionalProps() {
	rm.propsOrdered = RemoveCollectionProps("versions", rm.Attributes,
		rm.propsOrdered, rm.propsMap)
	rm.versionPropsOrdered = RemoveCollectionProps("versions", nil,
		rm.versionPropsOrdered, rm.versionPropsMap)

	// Only delete them if they're system added
	if attr, ok := rm.Attributes[rm.Singular]; ok {
		if attr.internals != nil {
			delete(rm.Attributes, rm.Singular)
			rm.GroupModel.Model.SetChanged(true)
		}
	}
	if attr, ok := rm.Attributes[rm.Singular+"url"]; ok {
		if attr.internals != nil {
			delete(rm.Attributes, rm.Singular+"url")
			rm.GroupModel.Model.SetChanged(true)
		}
	}
	if attr, ok := rm.Attributes[rm.Singular+"proxyurl"]; ok {
		if attr.internals != nil {
			delete(rm.Attributes, rm.Singular+"proxyurl")
			rm.GroupModel.Model.SetChanged(true)
		}
	}
}

func (rm *ResourceModel) Verify(rmName string) error {
	rm.ClearPropsOrdered()

	if err := IsValidModelName(rmName); err != nil {
		return err
	}

	if rm.Plural == "" {
		// Allow auto-populate
		rm.Plural = rmName
		rm.GroupModel.Model.SetChanged(true)
	} else if rm.Plural != rmName {
		return fmt.Errorf("Resource %q must have a 'plural' value of %q, "+
			"not %q", rmName, rmName, rm.Plural)
	}

	if rm.SID == "" {
		rm.SID = NewUUID()
	}

	if rm.Singular == "" {
		return fmt.Errorf(`Resource %q is missing a "singular" value`, rmName)
	}

	if rm.Singular == rm.Plural {
		return fmt.Errorf(`Resource %q has same value for "plural" `+
			`and "singular"`, rmName)
	}

	if rm.GroupModel.FindResourceModel(rm.Singular) != nil {
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

	// If the hasDoc was changed to false, remove the $RESOURCE* attrs
	rm.RemoveConditionalProps()

	propsOrdered, _ := rm.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if specProp.Name[0] == '$' {
			continue
		}

		modelAttr, ok := rm.Attributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			rm.Attributes[specProp.Name] = specProp
			rm.GroupModel.Model.SetChanged(true)
		} else {
			// It's there but make sure it's not changed in a bad way
			if err := EnsureAttrOK(modelAttr, specProp); err != nil {
				return err
			}
		}
	}

	propsOrdered, _ = rm.GetMetaPropsOrdered()
	for _, specProp := range propsOrdered {
		if specProp.Name[0] == '$' {
			continue
		}
		modelAttr, ok := rm.MetaAttributes[specProp.Name]
		if !ok {
			// Missing in model, so add it
			rm.MetaAttributes[specProp.Name] = specProp
			rm.GroupModel.Model.SetChanged(true)
		} else {
			// It's there but make sure it's not changed in a bad way
			if err := EnsureAttrOK(modelAttr, specProp); err != nil {
				return err
			}
		}
	}

	ld := &LevelData{
		Model:     rm.GroupModel.Model,
		AttrNames: map[string]bool{},
		Path:      NewPPP("resources").P(rm.Plural),
	}

	if err := rm.Attributes.Verify("strict", ld); err != nil {
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
	if rm.MaxVersions != maxV {
		rm.MaxVersions = maxV
		rm.GroupModel.Model.SetChanged(true)
	}

	return nil
}

func (rm *ResourceModel) SetSetDefaultSticky(val bool) error {
	if rm.GetSetDefaultSticky() != val {
		rm.SetDefaultSticky = PtrBool(val)
		rm.GroupModel.Model.SetChanged(true)
	}

	return nil
}

func (rm *ResourceModel) SetSingleVersionRoot(val bool) error {
	if rm.GetSingleVersionRoot() != val {
		rm.SingleVersionRoot = PtrBool(val)
		rm.GroupModel.Model.SetChanged(true)
	}

	return nil
}

func (rm *ResourceModel) GetBaseMetaAttributes() Attributes {
	attrs := Attributes{}
	maps.Copy(attrs, rm.MetaAttributes)

	// Add xReg defined attributes
	// TODO Check for conflicts
	propsOrdered, _ := rm.GetMetaPropsOrdered()
	for _, specProp := range propsOrdered {
		if IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		} else {
			attrs[specProp.Name].internals = specProp.internals
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

	// Find all Resource level attributes (not Meta) so we can show them
	// mixed in with the Default Version attributes - e.g. metaurl
	propsOrdered, _ := rm.GetPropsOrdered()
	for _, specProp := range propsOrdered {
		if IsNil(attrs[specProp.Name]) {
			attrs[specProp.Name] = specProp
		} else {
			attrs[specProp.Name].internals = specProp.internals
		}
	}

	return attrs
}

func (rm *ResourceModel) AddTypeMap(ct string, format string) error {
	if format != "binary" && format != "json" && format != "string" {
		return fmt.Errorf("Invalid typemap format: %q", format)
	}
	if rm.TypeMap == nil {
		rm.TypeMap = map[string]string{}
	}
	rm.TypeMap[ct] = format

	rm.GroupModel.Model.SetChanged(true)

	return nil
}

func (rm *ResourceModel) RemoveTypeMap(ct string) error {
	if rm.TypeMap == nil {
		return nil
	}
	delete(rm.TypeMap, ct)
	if len(rm.TypeMap) == 0 {
		rm.TypeMap = nil
	}

	rm.GroupModel.Model.SetChanged(true)

	return nil
}

func (rm *ResourceModel) AddLabel(name string, value string) error {
	if rm.Labels == nil {
		rm.Labels = map[string]string{}
	}
	rm.Labels[name] = value

	rm.GroupModel.Model.SetChanged(true)

	return nil
}

func (rm *ResourceModel) RemoveLabel(name string) error {
	if rm.Labels == nil {
		return nil
	}

	delete(rm.Labels, name)
	if len(rm.Labels) == 0 {
		rm.Labels = nil
	}

	rm.GroupModel.Model.SetChanged(true)

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
		if name[0] == '$' {
			continue
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
			if attr.Model != nil {
				attr.Model.SetChanged(true)
			}
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
					if rm := gm.FindResourceModel(parts[2]); rm == nil {
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

    rm := gm.FindResourceModel(absParts[1])
	PanicIf(rm == nil, "no rm")
	return rm.Singular
}
*/

// The model serializer we use for the "xRegistry" schema format
func Model2xRegistryJson(m *Model, format string) ([]byte, error) {
	return json.MarshalIndent((*UserModel)(m), "", "  ")
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
		rm = gm.FindResourceModel(parts[1])
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
