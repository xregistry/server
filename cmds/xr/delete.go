package main

import (
	"encoding/json"
	// "fmt"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addDeleteCmd(parent *cobra.Command) {
	deleteCmd := &cobra.Command{
		Use:     "delete XID ...",
		Short:   "Delete an entity from the registry",
		Run:     deleteFunc,
		GroupID: "Entities",
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Don't error if doesn't exist")
	deleteCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")

	parent.AddCommand(deleteCmd)
}

func deleteFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	force, _ := cmd.Flags().GetBool("force")
	data, _ := cmd.Flags().GetString("data")

	if len(args) == 0 {
		Error("Must specify the XID of an entity")
	}

	objects := map[string]json.RawMessage{}

	xidStr := args[0]
	if len(xidStr) > 0 && xidStr[0] != '/' {
		xidStr = "/" + xidStr
	}

	// For now only look at the first one
	xid, err := ParseXid(xidStr)
	Error(err)

	if len(data) > 0 {
		if len(args) > 1 {
			Error("--data can not be specified when more than one " +
				"XID is provided")
		}

		if xid.IsEntity {
			Error("--data can only be specified when referencing a collection")
		}

		if len(data) > 0 && data[0] == '@' {
			buf, xErr := xrlib.ReadFile(data[1:])
			Error(xErr)
			data = string(buf)
		}

		if err := json.Unmarshal([]byte(data), &objects); err != nil {
			Error(NewXRError("parsing_data", "error_detail="+err.Error()))
		}
	} else {
		for _, arg := range args {
			if len(arg) > 0 && arg[0] != '/' {
				arg = "/" + arg
			}
			objects[arg] = nil
		}
	}

	if !force {
		for id, _ := range objects {
			_, xErr := reg.HttpDo("GET", id, nil)
			Error(xErr)
		}
	}

	for id, _ := range objects {
		res, xErr := reg.HttpDo("DELETE", id, nil)
		if res == nil || res.Code != 404 {
			Error(xErr)
		}
		// Should we output on a 404?
		Verbose("Deleted: %s", id)
	}
}
