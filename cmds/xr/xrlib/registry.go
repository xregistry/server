package xrlib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
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

var Registries = map[string]*registry.Registry{}

func GetRegistry(url string, xrConfig *Config) (*registry.Registry, *XRError) {
	if url == "" {
		url = xrConfig.GetAsString("server.url")
	}

	if url == "" {
		return nil, NewXRError("client_error", "/",
			"error_detail="+
				"No server address provided. "+
				"Try either -s or XR_SERVER env var")
	}

	reg := Registries[url]
	if reg != nil {
		return reg, nil
	}

	reg, xErr := registry.DefineRegistry(url, xrConfig)
	if xErr != nil {
		return nil, xErr
	}

	/*
		if xErr := reg.LoadRemoteModel(true); xErr != nil {
			return nil, xErr
		}

		if xErr := reg.LoadRemoteCapabilities(true); xErr != nil {
			return nil, xErr
		}
	*/

	Registries[url] = reg

	return reg, nil
}

type CLIReg registry.Registry

func (r *CLIReg) GetServerURL() string {
	reg := (*registry.Registry)(r)
	return reg.GetStuffAsString("server")
}

func (cliReg *CLIReg) Refresh() *XRError {
	if xErr := cliReg.RefreshModel(); xErr != nil {
		return xErr
	}

	if xErr := cliReg.RefreshCapabilities(); xErr != nil {
		return xErr
	}

	return nil
}

func (cliReg *CLIReg) RefreshModel() *XRError {
	reg := (*registry.Registry)(cliReg)
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/model"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})
	if xErr != nil {
		return xErr
	}

	if err := json.Unmarshal(res.Body, &reg.Model); err != nil {
		return NewXRError("parsing_data", "/model",
			"error_detail="+err.Error()).
			SetDetailf("Response: %s.", string(res.Body))
	}
	reg.Model.ApplyDefaults(reg)
	reg.Model.SetPointers()
	return nil
}

func (cliReg *CLIReg) RefreshCapabilities() *XRError {
	reg := (*registry.Registry)(cliReg)
	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/capabilities"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})
	if xErr != nil {
		return xErr
	}

	if err := json.Unmarshal(res.Body, &reg.Capabilities); err != nil {
		return NewXRError("parsing_data", "/capabilities",
			"error_detail="+err.Error()).
			SetDetailf("Response: %s.", string(res.Body))
	}
	return nil
}

func (r *CLIReg) ToString() string {
	reg := (*registry.Registry)(r)
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

func (cliReg *CLIReg) GetModelSource() (*registry.Model, *XRError) {
	reg := (*registry.Registry)(cliReg)
	if reg.Model == nil {
		xErr := cliReg.RefreshModel()
		if xErr != nil {
			return nil, xErr
		}
	}

	if reg.Model.Source == "" {
		if xErr := cliReg.RefreshModelSource(); xErr != nil {
			return nil, xErr
		}
	}

	modelSource := registry.Model{}
	if reg.Model.Source != "" {
		tmpAny := map[string]any{}
		err := Unmarshal([]byte(reg.Model.Source), &tmpAny)
		if err != nil {
			return nil, NewXRError("parsing_data", "/modelsource",
				"error_detail="+err.Error()).
				SetDetailf("Response: %s.", string(reg.Model.Source))
		}

		delete(tmpAny, "$schema")
		src := ToJSON(tmpAny)

		err = Unmarshal([]byte(src), &modelSource)
		if err != nil {
			return nil, NewXRError("parsing_data", "/modelsource",
				"error_detail="+err.Error()).
				SetDetailf("Response: %s.", string(src))
		}
	}

	// If "modelsource" appears to be empty, then use "model" instead.
	// a bit of a hack but better than nothing
	if ToJSON(modelSource) == "{}" {
		tmp := ToJSON(reg.Model)
		err := Unmarshal([]byte(tmp), &modelSource)
		if err != nil {
			return nil, NewXRError("parsing_data", "/modelsource",
				"error_detail="+err.Error()).
				SetDetailf("Response: %s.", string(reg.Model.Source))
		}
	}

	modelSource.SetPointers()

	return &modelSource, nil
}

func (cliReg *CLIReg) RefreshModelSource() *XRError {
	reg := (*registry.Registry)(cliReg)
	if reg.Model == nil {
		if xErr := cliReg.RefreshModel(); xErr != nil {
			return xErr
		}
	}

	res, xErr := HttpDo(&HttpRequest{
		Verb:    "GET",
		URL:     cliReg.WithPath("/modelsource"),
		Headers: HTTPHeaders,
		Debug:   log.GetLevel() > 2,
	})

	reg.Model.Source = ""

	if res.Code != 404 {
		// We silently ignore 404 for modelsource
		if xErr != nil {
			return xErr
		}

		srcModel := registry.Model{}

		if err := json.Unmarshal(res.Body, &srcModel); err != nil {
			return NewXRError("parsing_data", "/modelsource",
				"error_detail="+err.Error()).
				SetDetailf("Response: %s.", string(res.Body))
		}
		reg.Model.Source = string(res.Body)
	}

	return nil
}

func (cliReg *CLIReg) GetCapabilities() (*Capabilities, *XRError) {
	reg := (*registry.Registry)(cliReg)
	if reg.Capabilities == nil {
		xErr := cliReg.RefreshCapabilities()
		if xErr != nil {
			return nil, xErr
		}
	}
	return reg.Capabilities, nil
}

func (r *CLIReg) ListGroupModels() ([]string, *XRError) {
	reg := (*registry.Registry)(r)
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

func (r *CLIReg) FindResourceModel(gPlural, rPlural string) (*registry.ResourceModel, *XRError) {
	reg := (*registry.Registry)(r)
	model, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}
	return model.FindResourceModel(gPlural, rPlural), nil
}

type cliModel registry.Model

func (cliModel *cliModel) FindGroupBySingular(singular string) *registry.GroupModel {
	m := (*registry.Model)(cliModel)
	for _, group := range m.Groups {
		if group.Singular == singular {
			return group
		}
	}
	return nil
}

func (cliReg *CLIReg) WithPath(path string) string {
	server := cliReg.GetServerURL()
	if !strings.HasPrefix(server, "http") {
		server = "http://" + strings.TrimLeft(server, "/")
	}
	return strings.TrimRight(server, "/") + "/" + strings.TrimLeft(path, "/")
}

func (cliReg *CLIReg) URLWithPath(path string) (*url.URL, *XRError) {
	reg := (*registry.Registry)(cliReg)
	server := cliReg.GetServerURL()
	PanicIf(server == "", "stuff.server isn't set")

	if !strings.HasPrefix(server, "http") {
		reg.SetStuff("server", "http://"+strings.TrimLeft(server, "/"))
	}

	path = strings.TrimRight(server, "/") + "/" +
		strings.TrimLeft(path, "/")

	u, err := url.Parse(path)
	if err != nil {
		return nil, NewXRError("client_error", path,
			"error_detail="+
				fmt.Sprintf("Error parsing url (%s): %s", path, err.Error()))
	}

	/*

		if u.Scheme == "" {
			u.Scheme = "http"
		}
		u.Path += "/" + strings.TrimLeft(path, "/")
	*/

	return u, nil
}

func (r *CLIReg) GetResourceModelFromXID(xidStr string) (*registry.ResourceModel, *XRError) {
	reg := (*registry.Registry)(r)
	xid, err := ParseXid(xidStr)
	if err != nil {
		return nil, NewXRError("malformed_xid", "",
			"xid="+xidStr,
			"error_detail="+err.Error())
	}
	if xid.Resource == "" {
		return nil, nil
	}

	gm, xErr := reg.FindGroupModel(xid.Group)
	if xErr != nil {
		return nil, xErr
	}
	if gm == nil {
		return nil, NewXRError("not_found", xid.Group).
			SetDetailf("Unknown Group type: %s.", xid.Group)
	}

	rm := gm.Resources[xid.Resource]
	if rm == nil {
		return nil, NewXRError("not_found", xid.Resource).
			SetDetailf("Unknown Resource type: %s.", xid.Group)
	}
	return rm, nil
}

func (cliReg *CLIReg) DownloadObject(path string) (map[string]any, *XRError) {
	// reg := (*registry.Registry)(cliReg)
	urlPath, xErr := cliReg.URLWithPath(path)
	if xErr != nil {
		return nil, xErr
	}

	return DownloadObject(log.GetLevel() > 2, urlPath.String())
}
