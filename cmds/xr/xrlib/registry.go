package xrlib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/registry"
)

type Registry struct {
	Entity
	Capabilities *Capabilities     `json:"capabilities,omitempty"`
	Model        *Model            `json:"model,omitempty"`
	Groups       map[string]*Group `json:"groups,omitempty"`

	isNew  bool
	server string
}

type RegistryDefined struct {
	SpecVersion   string            `json:"specversion,omitempty"`
	RegistryID    string            `json:"registryid,omitempty"`
	Self          string            `json:"self,omitempty"`
	ShortSelf     string            `json:"shortself,omitempty"`
	XID           string            `json:"xid,omitempty"`
	Epoch         uint              `json:"self,omitempty"`
	Name          string            `json:"name,omitempty"`
	Description   string            `json:"description,omitempty"`
	Documentation string            `json:"documentation,omitempty"`
	Labels        map[string]string `json:"labels,omitemty"`
	CreatedAt     string            `json:"createdat,omitempty"`
	ModifiedAt    string            `json:"modifiedat,omitempty"`

	Extensions  map[string]any       `json:"-"`
	Collections []*CollectionDefined `json:"-"`
}

type Capabilities map[string]any

type Model struct {
	Registry   *Registry              `json:"-"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Attributes Attributes             `json:"attributes,omitempty"`
	Groups     map[string]*GroupModel `json:"groups,omitempty"`
}

type Attributes map[string]*Attribute

type Attribute struct {
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Target       string `json:"target,omitempty"`
	RelaxedNames bool   `json:"relaxednames,omitempty"`
	Description  string `json:"description,omitempty"`
	Enum         []any  `json:"enum,omitempty"`
	Strict       bool   `json:"strict,omitempty"`
	ReadOnly     bool   `json:"readonly,omitempty"`
	Immutable    bool   `json:"immutable,omitempty"`
	Required     bool   `json:"required,omitempty"`
	Default      any    `json:"default,omitempty"`

	Attributes Attributes `json:"attributes,omitempty"`
	Item       *Item      `json:"item,omitempty"`
	IfValues   IfValues   `json:"ifvalues,omitempty"`
}

type Item struct {
	Type         string     `json:"type,omitempty"`
	RelaxedNames bool       `json:"relaxednames,omitempty"`
	Attribute    Attributes `json:"item,omitempty"`
	Item         *Item      `json:"item,omitempty"`
}

type IfValues map[string]*IfValue

type IfValue struct {
	SiblingAttributes Attributes `json:"siblingattributes,omitempty"`
}

type GroupModel struct {
	Model      *Model            `json:"-"`
	Plural     string            `json:"plural,omitempty"`
	Singular   string            `json:"singular,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Attributes Attributes        `json:"attributes,omitempty"`

	XImportResources []string                  `json:"ximportresources,omitempty"`
	Resources        map[string]*ResourceModel `json:"resources,omitempty"`

	imports map[string]*ResourceModel
}

type CollectionDefined struct {
	Plural   string
	Singular string
	URL      string
	Count    uint
}

type ResourceModel struct {
	Plural           string            `json:"plural,omitempty"`
	Singular         string            `json:"singular,omitempty"`
	MaxVersions      int               `json:"maxversions,omitempty"`
	SetVersionId     *bool             `json:"setversionid,omitempty"`
	SetDefaultSticky *bool             `json:"setdefaultversionsticky,omitempty"`
	HasDocument      *bool             `json:"hasdocument,omitempty"`
	TypeMap          map[string]string `json:"typemap,omitempty"`
	Labels           map[string]string `json:"labels,omitempty"`
	Attributes       Attributes        `json:"attributes,omitempty"`
	MetaAttributes   Attributes        `json:"metaattributes,omitempty"`
}

type Group struct {
	Entity
	registry  *Registry
	resources map[string]*Resource
}

type Resource struct {
	Entity
	group    *Group
	meta     *Meta
	versions map[string]*Version
}

type Meta struct {
	Entity
	resource *Resource
}

type Version struct {
	Entity
	resource *Resource
}

type Entity struct {
	registry   *Registry
	uid        string
	attributes map[string]any

	daType   int
	path     string
	abstract string
}

func GetRegistry(url string) (*Registry, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("No Server address provided")
	}

	if !strings.HasPrefix(url, "http") {
		url = "http://" + strings.TrimLeft(url, "/")
	}

	reg := &Registry{
		Entity: Entity{
			daType:   registry.ENTITY_REGISTRY,
			path:     "", // [GROUPS/gID[/RESOURCES/rID[/versions/vID]]]
			abstract: "", // [GROUPS[/RESOURCES[/versions]]]
		},
		server: url,
	}
	reg.Entity.registry = reg

	return reg, reg.Refresh()
}

func (reg *Registry) Refresh() error {
	// GET root and verify it's an xRegistry
	res, err := reg.HttpDo("GET", "?inline=model,capabilities", nil)
	if err != nil {
		return err
	}

	attrs := map[string]json.RawMessage(nil)
	if err := registry.Unmarshal(res.Body, &attrs); err != nil {
		return fmt.Errorf("Not an xRegistry(%s), invalid response: %s",
			reg.server, err)
	}

	specVersion := ""
	err = json.Unmarshal(attrs["specversion"], &specVersion)
	if err != nil || specVersion != registry.SPECVERSION {
		return fmt.Errorf("Not an xRegistry(%s), missing 'specversion'",
			reg.server)
	}

	// Before we process the attributes, get the model and capabilities
	if !registry.IsNil(attrs["model"]) {
		if err := json.Unmarshal(attrs["model"], &reg.Model); err != nil {
			return fmt.Errorf("Unable to parse registry model: %s\n%s",
				err, string(attrs["model"]))
		}
		reg.Model.SetPointers()
	} else {
		if err := reg.RefreshModel(); err != nil {
			return err
		}
	}

	if !registry.IsNil(attrs["capabilities"]) {
		err := json.Unmarshal(attrs["capabilities"], &reg.Capabilities)
		if err != nil {
			return fmt.Errorf("Unable to parse registry capabilities: %s\n%s",
				err, string(attrs["capabilities"]))
		}
	} else {
		if err := reg.RefreshCapabilities(); err != nil {
			return err
		}
	}

	return nil
}

func (reg *Registry) RefreshModel() error {
	res, err := reg.HttpDo("GET", "/model", nil)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(res.Body, &reg.Model); err != nil {
		return fmt.Errorf("Unable to parse registry model: %s\n%s",
			err, string(res.Body))
	}
	reg.Model.SetPointers()
	return nil
}

func (reg *Registry) RefreshCapabilities() error {
	res, err := reg.HttpDo("GET", "/capabilities", nil)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(res.Body, &reg.Capabilities); err != nil {
		return fmt.Errorf("Unable to parse registry capabilities: %s\n%s",
			err, string(res.Body))
	}
	return nil
}

func (reg *Registry) ToString() string {
	/*
		tmp := map[string]any{}
		for k, v := range reg.attributes {
			tmp[k] = v
		}
		tmp["model"] = reg.Model
		tmp["capabilities"] = reg.Capabilities
	*/

	buf, _ := json.MarshalIndent(reg, "", "  ")
	return string(buf)
}

func (reg *Registry) HttpDo(verb, path string, body []byte) (*HttpResponse, error) {
	u, err := reg.URLWithPath(path)
	if err != nil {
		return nil, err
	}
	return HttpDo(verb, u.String(), body)
}

func (m *Model) SetPointers() {
	for _, gm := range m.Groups {
		gm.SetModel(m)
	}
}

func (m *Model) FindGroupBySingular(singular string) *GroupModel {
	for _, group := range m.Groups {
		if group.Singular == singular {
			return group
		}
	}
	return nil
}

func (m *Model) FindGroupModel(plural string) *GroupModel {
	return m.Groups[plural]
}

func (m *Model) FindResourceModel(gType, rType string) *ResourceModel {
	gm := m.FindGroupModel(gType)
	if gm == nil {
		return nil
	}
	return gm.FindResourceModel(rType)
}

func (gm *GroupModel) SetModel(m *Model) {
	gm.Model = m
}

func (gm *GroupModel) FindResourceBySingular(singular string) *ResourceModel {
	for _, resource := range gm.Resources {
		if resource.Singular == singular {
			return resource
		}
	}
	return nil
}

func (gm *GroupModel) GetImports() map[string]*ResourceModel {
	if gm.imports == nil && len(gm.XImportResources) > 0 {
		gm.imports = map[string]*ResourceModel{}
		for _, grName := range gm.XImportResources {
			parts := strings.Split(grName, "/")
			r := gm.Model.FindResourceModel(parts[1], parts[2])
			// PanicIf(r == nil, "Can't find %q", grName)
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

func (reg *Registry) URLWithPath(path string) (*url.URL, error) {
	if !strings.HasPrefix(reg.server, "http") {
		reg.server = "http://" + strings.TrimLeft(reg.server, "/")
	}

	path = strings.TrimRight(reg.server, "/") + "/" +
		strings.TrimLeft(path, "/")

	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	/*

		if u.Scheme == "" {
			u.Scheme = "http"
		}
		u.Path += "/" + strings.TrimLeft(path, "/")
	*/

	return u, nil
}

func (reg *Registry) GetResourceModelFromXID(xidStr string) (*ResourceModel, error) {
	xid, err := ParseXID(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.Resource == "" {
		return nil, nil
	}

	gm := reg.Model.FindGroupModel(xid.Group)
	if gm == nil {
		return nil, fmt.Errorf("Unknown group type: %s", xid.Group)
	}

	rm := gm.FindResourceModel(xid.Resource)
	if rm == nil {
		return nil, fmt.Errorf("Unknown resource type: %s", xid.Resource)
	}
	return rm, nil
}

func (reg *Registry) DownloadObject(path string) (map[string]any, error) {
	urlPath, err := reg.URLWithPath(path)
	if err != nil {
		return nil, err
	}

	return DownloadObject(urlPath.String())
}

func (rm *ResourceModel) HasDoc() bool {
	return rm != nil && rm.HasDocument != nil && *(rm.HasDocument) == true
}
