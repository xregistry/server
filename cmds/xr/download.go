package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	// "text/tabwriter"
	"os"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
	"github.com/spf13/cobra"
)

func addDownloadCmd(parent *cobra.Command) {
	downloadCmd := &cobra.Command{
		Use:     "download DIR [ XID...]",
		Short:   "Download entities from registry as individual files",
		Run:     downloadFunc,
		GroupID: "Entities",
	}

	parent.AddCommand(downloadCmd)
}

func downloadFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	if len(args) == 0 {
		Error("Missing the DIR argument")
	}

	dir := args[0]
	stat, err := os.Stat(dir)
	if os.IsNotExist(err) || !stat.IsDir() {
		Error("%q must be an existing directory", dir)
	}
	args = args[1:]

	if len(args) == 0 {
		args = []string{"/"}
	}

	downloadXIDFn := func(xid *xrlib.XID) error {
		loc := xid.String()
		if xid.Resource != "" && xid.IsEntity {
			loc += "$details"
		}

		file := dir
		switch xid.Type {
		case xrlib.ENTITY_REGISTRY:
			file += "/index.html"

		case xrlib.ENTITY_GROUP_TYPE:
			fallthrough
		case xrlib.ENTITY_RESOURCE_TYPE:
			fallthrough
		case xrlib.ENTITY_VERSION_TYPE:
			file += fmt.Sprintf("%s/index.html", xid.String())

		case xrlib.ENTITY_GROUP:
			fallthrough
		case xrlib.ENTITY_RESOURCE:
			fallthrough
		case xrlib.ENTITY_META:
			fallthrough
		case xrlib.ENTITY_VERSION:
			file += fmt.Sprintf("%s", xid.String())

		}

		DownloadToFile(reg, xid.String(), file)

		if xid.Resource != "" && xid.IsEntity {
			file += "$details"
			DownloadToFile(reg, xid.String()+"$details", file+"$details")
		}

		return nil
	}

	for _, xidStr := range args {
		xid := xrlib.ParseXID(xidStr)
		Error(traverseFromXID(reg, xid, dir, downloadXIDFn))
	}

	file := dir
	DownloadToFile(reg, "/model", file+"/model")
	DownloadToFile(reg, "/capabilities", file+"/capabilities")
}

func DownloadToFile(reg *xrlib.Registry, path string, file string) {
	Verbose("Downloading %q to %q", path, file)
	res, err := reg.HttpDo("GET", path, nil)
	Error(err)

	Error(os.MkdirAll(filepath.Dir(file), 0600))
	Error(os.WriteFile(file, res.Body, 0600))
}

type traverseFunc func(xid *xrlib.XID) error

func traverseFromXID(reg *xrlib.Registry, xid *xrlib.XID, root string, fn traverseFunc) error {
	switch xid.Type {
	case xrlib.ENTITY_REGISTRY:
		fn(xid)
		for _, gm := range reg.Model.Groups {
			nextXID := xrlib.ParseXID(xid.String() + "/" + gm.Plural)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_GROUP_TYPE:
		fallthrough
	case xrlib.ENTITY_RESOURCE_TYPE:
		fallthrough
	case xrlib.ENTITY_VERSION_TYPE:
		fn(xid)
		res, err := reg.HttpDo("GET", xid.String(), nil)
		Error(err)
		tmp := map[string]any{}
		Error(json.Unmarshal([]byte(res.Body), &tmp))
		for key, _ := range tmp {
			nextXID := xrlib.ParseXID(xid.String() + "/" + key)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_GROUP:
		fn(xid)
		gm := reg.Model.Groups[xid.Group]
		for _, rm := range gm.Resources {
			nextXID := xrlib.ParseXID(xid.String() + "/" + rm.Plural)
			traverseFromXID(reg, nextXID, root, fn)
		}

	case xrlib.ENTITY_RESOURCE:
		fn(xid)

	case xrlib.ENTITY_META:
		fn(xid)

	case xrlib.ENTITY_VERSION:
		fn(xid)

	}

	return nil
}
