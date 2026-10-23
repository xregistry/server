package xrlib

import (
	"encoding/json"
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

	if _, xErr := (*registry.Registry)(cliReg).LoadCapabilities(); xErr != nil {
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

	_, xErr := reg.GetModel()
	if xErr != nil {
		return nil, xErr
	}

	if reg.Model == nil {
		xErr := cliReg.RefreshModel()
		if xErr != nil {
			return nil, xErr
		}
	}

	if reg.Model.Source == nil {
		if xErr := cliReg.RefreshModelSource(); xErr != nil {
			return nil, xErr
		}
	}

	modelSource := registry.Model{}
	if len(reg.Model.Source) > 0 {
		tmpAny := map[string]any{}
		err := Unmarshal(reg.Model.Source, &tmpAny)
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

	reg.Model.Source = nil

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
		reg.Model.Source = res.Body
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

func (cliReg *CLIReg) DownloadObject(path string) (map[string]any, *XRError) {
	// reg := (*registry.Registry)(cliReg)
	urlPath := cliReg.WithPath(path)

	return DownloadObject(log.GetLevel() > 2, urlPath)
}
