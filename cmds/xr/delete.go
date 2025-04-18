package main

import (
	"encoding/json"
	// "fmt"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
	"github.com/spf13/cobra"
)

func addDeleteCmd(parent *cobra.Command) {
	deleteCmd := &cobra.Command{
		Use:     "delete [ XID ... ]",
		Short:   "Delete an entity from the registry",
		Run:     deleteFunc,
		GroupID: "Entities",
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Don't error if doesn't exist")
	deleteCmd.Flags().StringP("data", "d", "", "Data(json), @FILE, @URL, @-")

	parent.AddCommand(deleteCmd)
}

func deleteFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	force, _ := cmd.Flags().GetBool("force")
	data, _ := cmd.Flags().GetString("data")

	if len(args) == 0 {
		args = []string{"/"}
	}

	objects := map[string]json.RawMessage{}

	// For now only look at the first one
	XID, err := xrlib.ParseXID(args[0])
	Error(err)

	if len(data) > 0 {
		if len(args) > 1 {
			Error("--data can not be specified when more than one XID is provided")
		}

		if XID.IsEntity {
			Error("--data can only be specified when referencing a collection")
		}

		if len(data) > 0 && data[0] == '@' {
			buf, err := xrlib.ReadFile(data[1:])
			Error(err)
			data = string(buf)
		}

		Error(json.Unmarshal([]byte(data), &objects))
	} else {
		for _, arg := range args {
			objects[arg] = nil
		}
	}

	if !force {
		for id, _ := range objects {
			if _, err := reg.HttpDo("GET", id, nil); err != nil {
				Error("%q does not exist", id)
			}
		}
	}

	for id, _ := range objects {
		res, err := reg.HttpDo("DELETE", id, nil)
		if res == nil || res.Code != 404 {
			Error(err)
		}
		// Should we output on a 404?
		Verbose("Deleted: %s", id)
	}
}
