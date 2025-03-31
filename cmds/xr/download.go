package main

import (
	// "encoding/json"
	// "fmt"
	// "text/tabwriter"
	"os"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
	"github.com/spf13/cobra"
)

func addDownloadCmd(parent *cobra.Command) {
	downloadCmd := &cobra.Command{
		Use:   "download DIR [ XID...]",
		Short: "Download entities from registry as individual files",
		Run:   downloadFunc,
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

	for _, xidStr := range args {
		suffix := ""
		xid := xrlib.ParseXID(xidStr)
		if xid.Type == xrlib.ENTITY_RESOURCE || xid.Type == xrlib.ENTITY_VERSION {
			suffix = "$details"
		}
		suffix += "?inline"

		res, err := reg.HttpDo("GET", xid.String()+suffix, nil)
		Error(err)

		Error(writeToDisk(reg, xid, dir, string(res.Body)))
	}

}

func writeToDisk(reg *xrlib.Registry, xid *xrlib.XID, dir string, data string) error {
	return nil
}
