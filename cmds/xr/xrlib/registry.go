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

func GetRegistry(url string) (*Registry, *XRError) {
	reg := Registries[url]
	if reg != nil {
		return reg, nil
	}

	url = strings.TrimSpace(url)
	if url == "" {
		return nil, NewXRError("bad_request", "/", "No Server address provided")
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

func (reg *Registry) Refresh() *XRError {
	if xErr := reg.RefreshModel(); xErr != nil {
		return xErr
	}

	if xErr := reg.RefreshCapabilities(); xErr != nil {
		return xErr
	}

	return nil
}

func (reg *Registry) RefreshModel() *XRError {
	res, xErr := reg.HttpDo("GET", "/model", nil)
	if xErr != nil {
		return xErr
	}

	if err := json.Unmarshal(res.Body, &reg.Model); err != nil {
		return NewXRError("bad_request", "/",
			fmt.Sprintf("Unable to parse registry model: %s\n%s",
				err, string(res.Body)))
	}
	reg.Model.ApplyDefaults()
	reg.Model.SetPointers()
	return nil
}

func (reg *Registry) RefreshCapabilities() *XRError {
	res, xErr := reg.HttpDo("GET", "/capabilities", nil)
	if xErr != nil {
		return xErr
	}

	if err := json.Unmarshal(res.Body, &reg.Capabilities); err != nil {
		return NewXRError("bad_request", "/",
			fmt.Sprintf("Unable to parse registry capabilities: %s\n%s",
				err, string(res.Body)))
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

func (reg *Registry) GetModel() (*Model, *XRError) {
	if reg.Model == nil {
		xErr := reg.RefreshModel()
		if xErr != nil {
			return nil, xErr
		}
	}
	return reg.Model, nil
}

func (reg *Registry) GetModelSource() (*Model, *XRError) {
	if reg.Model == nil {
		xErr := reg.RefreshModel()
		if xErr != nil {
			return nil, xErr
		}
	}

	if reg.Model.Source == "" {
		if xErr := reg.RefreshModelSource(); xErr != nil {
			return nil, xErr
		}
	}

	tmpModel := Model{}
	if reg.Model.Source != "" {
		err := Unmarshal([]byte(reg.Model.Source), &tmpModel)
		if err != nil {
			return nil, NewXRError("bad_request", "/", err.Error())
		}
	}
	tmpModel.SetPointers()

	return &tmpModel, nil
}

func (reg *Registry) RefreshModelSource() *XRError {
	if reg.Model == nil {
		if xErr := reg.RefreshModel(); xErr != nil {
			return xErr
		}
	}

	res, xErr := reg.HttpDo("GET", "/modelsource", nil)

	reg.Model.Source = ""

	if res.Code != 404 {
		// We silently ignore 404 for modelsource
		if xErr != nil {
			return xErr
		}

		srcModel := Model{}

		if err := json.Unmarshal(res.Body, &srcModel); err != nil {
			return NewXRError("bad_request", "/",
				fmt.Sprintf("Unable to parse registry modelsource: %s\n%s",
					err, string(res.Body)))
		}
		reg.Model.Source = string(res.Body)
	}

	return nil
}

func (reg *Registry) GetCapabilities() (*Capabilities, *XRError) {
	if reg.Capabilities == nil {
		xErr := reg.RefreshCapabilities()
		if xErr != nil {
			return nil, xErr
		}
	}
	return reg.Capabilities, nil
}

func (reg *Registry) FindGroupModel(gPlural string) (*GroupModel, *XRError) {
	model, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}
	return model.FindGroupModel(gPlural), nil
}

func (reg *Registry) FindGroupModelBySingular(gSingular string) (*GroupModel, *XRError) {
	model, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}
	return model.FindGroupBySingular(gSingular), nil
}

func (reg *Registry) ListGroupModels() ([]string, *XRError) {
	model, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}

	res := []string(nil)
	for _, gm := range model.Groups {
		res = append(res, gm.Plural)
	}

	return res, nil
}

func (reg *Registry) FindResourceModel(gPlural, rPlural string) (*ResourceModel, *XRError) {
	model, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}
	return model.FindResourceModel(gPlural, rPlural), nil
}

func (reg *Registry) HttpDo(verb, path string, body []byte) (*HttpResponse, *XRError) {
	u, xErr := reg.URLWithPath(path)
	if xErr != nil {
		return nil, xErr
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

func (reg *Registry) URLWithPath(path string) (*url.URL, *XRError) {
	server := reg.GetServerURL()
	PanicIf(server == "", "stuff.server isn't set")

	if !strings.HasPrefix(server, "http") {
		reg.SetStuff("server", "http://"+strings.TrimLeft(server, "/"))
	}

	path = strings.TrimRight(server, "/") + "/" +
		strings.TrimLeft(path, "/")

	u, err := url.Parse(path)
	if err != nil {
		return nil, NewXRError("bad_request", "/", err.Error())
	}

	/*

		if u.Scheme == "" {
			u.Scheme = "http"
		}
		u.Path += "/" + strings.TrimLeft(path, "/")
	*/

	return u, nil
}

func (reg *Registry) GetResourceModelFromXID(xidStr string) (*ResourceModel, *XRError) {
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("bad_request", "/", err.Error())
	}
	if xid.Resource == "" {
		return nil, nil
	}

	gm, xErr := reg.FindGroupModel(xid.Group)
	if xErr != nil {
		return nil, xErr
	}
	if gm == nil {
		return nil, NewXRError("not_found", xid.Group, xid.Group).
			SetDetailf("Unknown Group type: %s", xid.Group)
	}

	rm := gm.FindResourceModel(xid.Resource)
	if rm == nil {
		return nil, NewXRError("not_found", xid.Resource, xid.Resource).
			SetDetailf("Unknown Resource type: %s", xid.Group)
	}
	return rm, nil
}

func (reg *Registry) DownloadObject(path string) (map[string]any, *XRError) {
	urlPath, xErr := reg.URLWithPath(path)
	if xErr != nil {
		return nil, xErr
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

func (reg *Registry) LoadConfigFromFile(file string) *XRError {
	buf, xErr := ReadFile(file)
	if xErr != nil {
		return xErr
	}

	return reg.LoadConfigFromString(string(buf))
}

// Buffer syntax:
// prop: value
// # comment
func (reg *Registry) LoadConfigFromString(buffer string) *XRError {
	lines := strings.Split(buffer, "/n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		if name == "" {
			return NewXRError("bad_request", "/",
				fmt.Sprintf("Error in config data - no name: %q", line))
		}
		reg.SetConfig(name, value)
	}
	return nil
}

var PropsFuncs = []*Attribute{}

func (rm *ResourceModel) VerifyData() *XRError {
	return nil
}
