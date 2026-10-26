package xrlib

import (
	"encoding/json"
	"strings"

	log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

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
