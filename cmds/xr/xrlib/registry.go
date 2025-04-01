package xrlib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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
	Plural     string            `json:"plural,omitempty"`
	Singular   string            `json:"singular,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Attributes Attributes        `json:"attributes,omitempty"`

	Resources map[string]*ResourceModel
}

type CollectionDefined struct {
	Plural   string
	Singular string
	URL      string
	Count    uint
}

type ResourceModel struct {
	Plural           string `json:"plural,omitempty"`
	Singular         string `json:"singular,omitempty"`
	MaxVersions      int
	SetVersionId     *bool
	SetDefaultSticky *bool
	HasDocument      *bool
	TypeMap          map[string]string
	Labels           map[string]string `json:"labels,omitempty"`
	Attributes       Attributes        `json:"attributes,omitempty"`
	MetaAttributes   Attributes
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
	res, err := reg.HttpDo("GET", "", nil)
	if err != nil {
		return err
	}

	attrs := map[string]any(nil)
	if err := registry.Unmarshal(res.Body, &attrs); err != nil {
		return err
	}

	if attrs["specversion"] != "0.5" {
		return fmt.Errorf("Not a valid xRegistry, missing 'specversion'")
	}

	// Before we process the attributes, get the model and capabilities

	if err := reg.RefreshModel(); err != nil {
		return err
	}
	if err := reg.RefreshCapabilities(); err != nil {
		return err
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

func (m *Model) FindGroupBySingular(singular string) *GroupModel {
	for _, group := range m.Groups {
		if group.Singular == singular {
			return group
		}
	}
	return nil
}

func (m *Model) FindGroupByPlural(plural string) *GroupModel {
	return m.Groups[plural]
}

func (gm *GroupModel) FindResourceByPlural(plural string) *ResourceModel {
	return gm.Resources[plural]
}

func (gm *GroupModel) FindResourceBySingular(singular string) *ResourceModel {
	for _, resource := range gm.Resources {
		if resource.Singular == singular {
			return resource
		}
	}
	return nil
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
	xid := ParseXID(xidStr)
	if xid.Resource == "" {
		return nil, nil
	}

	gm := reg.Model.Groups[xid.Group]
	if gm == nil {
		return nil, fmt.Errorf("Unknown group type: %s", xid.Group)
	}

	rm := gm.Resources[xid.Resource]
	if rm == nil {
		return nil, fmt.Errorf("Uknown resource type: %s", xid.Resource)
	}
	return rm, nil
}

func (rm *ResourceModel) HasDoc() bool {
	return rm != nil && rm.HasDocument != nil && *(rm.HasDocument) == true
}
