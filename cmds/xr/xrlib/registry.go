package xrlib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

/*
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

type CollectionDefined struct {
	Plural   string
	Singular string
	URL      string
	Count    uint
}
*/

type EntityExtensions struct {
}

var Registries = map[string]*Registry{}

func GetRegistry(url string) (*Registry, error) {
	reg := Registries[url]
	if reg != nil {
		return reg, nil
	}

	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("No Server address provided")
	}

	if !strings.HasPrefix(url, "http") {
		url = "http://" + strings.TrimLeft(url, "/")
	}

	reg = &Registry{
		Entity: Entity{
			Type:     ENTITY_REGISTRY,
			Path:     "", // [GROUPS/gID[/RESOURCES/rID[/versions/vID]]]
			Abstract: "", // [GROUPS[/RESOURCES[/versions]]]
		},
		// server: url,
		// config: map[string]any{},
	}
	reg.Entity.Registry = reg
	reg.SetStuff("server", url)

	Registries[url] = reg

	return reg, nil
}

func (reg *Registry) GetServerURL() string {
	return reg.GetStuffAsString("server")
}

func (reg *Registry) Refresh() error {
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
	reg.Model.SetPointers()

	res, err = reg.HttpDo("GET", "/modelsource", nil)

	if res.Code != 404 {
		// We silently ignore 404 for modelsource
		if err != nil {
			return err
		}

		srcModel := Model{}

		if err := json.Unmarshal(res.Body, &srcModel); err != nil {
			return fmt.Errorf("Unable to parse registry modelsource: %s\n%s",
				err, string(res.Body))
		}
		reg.Model.Source = string(res.Body)
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
		tmp["model"], _ = reg.GetModel()
		tmp["capabilities"], _ = reg.GetCapabilities()
	*/

	buf, _ := json.MarshalIndent(reg, "", "  ")
	return string(buf)
}

func (reg *Registry) GetModel() (*Model, error) {
	if reg.Model == nil {
		err := reg.RefreshModel()
		if err != nil {
			return nil, err
		}
	}
	return reg.Model, nil
}

func (reg *Registry) GetModelSource() (*Model, error) {
	if reg.Model == nil {
		err := reg.RefreshModel()
		if err != nil {
			return nil, err
		}
	}
	tmpModel := Model{}
	if reg.Model.Source != "" {
		err := Unmarshal([]byte(reg.Model.Source), &tmpModel)
		if err != nil {
			return nil, err
		}
	}
	return &tmpModel, nil
}

func (reg *Registry) GetCapabilities() (*Capabilities, error) {
	if reg.Capabilities == nil {
		err := reg.RefreshCapabilities()
		if err != nil {
			return nil, err
		}
	}
	return reg.Capabilities, nil
}

func (reg *Registry) FindGroupModel(gPlural string) (*GroupModel, error) {
	model, err := reg.GetModel()
	if err != nil {
		return nil, err
	}
	return model.FindGroupModel(gPlural), nil
}

func (reg *Registry) FindGroupModelBySingular(gSingular string) (*GroupModel, error) {
	model, err := reg.GetModel()
	if err != nil {
		return nil, err
	}
	return model.FindGroupBySingular(gSingular), nil
}

func (reg *Registry) ListGroupModels() ([]string, error) {
	model, err := reg.GetModel()
	if err != nil {
		return nil, err
	}

	res := []string(nil)
	for _, gm := range model.Groups {
		res = append(res, gm.Plural)
	}

	return res, nil
}

func (reg *Registry) FindResourceModel(gPlural, rPlural string) (*ResourceModel, error) {
	model, err := reg.GetModel()
	if err != nil {
		return nil, err
	}
	return model.FindResourceModel(gPlural, rPlural), nil
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

func (reg *Registry) URLWithPath(path string) (*url.URL, error) {
	server := reg.GetServerURL()
	PanicIf(server == "", "stuff.server isn't set")

	if !strings.HasPrefix(server, "http") {
		reg.SetStuff("server", "http://"+strings.TrimLeft(server, "/"))
	}

	path = strings.TrimRight(server, "/") + "/" +
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
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, err
	}
	if xid.Resource == "" {
		return nil, nil
	}

	gm, err := reg.FindGroupModel(xid.Group)
	if err != nil {
		return nil, err
	}
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

func (reg *Registry) SetConfig(name string, value any) {
	val, _ := reg.GetStuff("config")
	config := map[string]any(nil)
	if IsNil(val) {
		config = map[string]any{}
	} else {
		config = val.(map[string]any)
	}

	name = strings.TrimSpace(name)
	if value == nil {
		delete(config, name)
	} else {
		config[name] = value
	}

	reg.SetStuff("config", config)
}

func (reg *Registry) GetConfig(name string) any {
	val, ok := reg.GetStuff("config")
	if !ok {
		return nil
	}
	config := map[string]any(nil)
	if IsNil(val) {
		config = map[string]any{}
	} else {
		config = val.(map[string]any)
	}
	return config[name]
}

func (reg *Registry) GetConfigAsString(name string) string {
	val := reg.GetConfig(name)

	if IsNil(val) {
		return ""
	}

	if str, ok := val.(string); ok {
		return str
	}

	return ""
}

func (reg *Registry) LoadConfigFromFile(file string) error {
	buf, err := ReadFile(file)
	if err != nil {
		return err
	}

	return reg.LoadConfigFromString(string(buf))
}

// Buffer syntax:
// prop: value
// # comment
func (reg *Registry) LoadConfigFromString(buffer string) error {
	lines := strings.Split(buffer, "/n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		if name == "" {
			return fmt.Errorf("Error in config data - no name: %q", line)
		}
		reg.SetConfig(name, value)
	}
	return nil
}

var PropsFuncs = []*Attribute{}

func (rm *ResourceModel) VerifyData() error {
	return nil
}
