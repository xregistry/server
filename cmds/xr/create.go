package main

import (
	"encoding/json"
	// "fmt"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
)

func addCreateCmd(parent *cobra.Command) {
	createCmd := &cobra.Command{
		Use:     "create [ XID ]",
		Short:   "Create a new entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
	}
	createCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	createCmd.Flags().StringP("data", "d", "", "Data (json),@FILE,@URL,-")

	parent.AddCommand(createCmd)
}

func addUpsertCmd(parent *cobra.Command) {
	upsertCmd := &cobra.Command{
		Use:     "upsert [ XID ]",
		Short:   "Update, or insert(create), an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
	}
	upsertCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	upsertCmd.Flags().StringP("data", "d", "", "Data (json),@FILE,@URL,-")

	parent.AddCommand(upsertCmd)
}

func addUpdateCmd(parent *cobra.Command) {
	updateCmd := &cobra.Command{
		Use:     "update [ XID ]",
		Short:   "Update an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
	}
	updateCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	updateCmd.Flags().StringP("data", "d", "", "Data (json),@FILE,@URL,-")
	updateCmd.Flags().BoolP("patch", "p", false, "Only update specified attributes")

	parent.AddCommand(updateCmd)
}

func createFunc(cmd *cobra.Command, args []string) {
	action := cmd.Use[:6]

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	if len(args) == 0 {
		args = []string{"/"}
	}

	if len(args) > 1 {
		Error("Only one XID is allowed to be specified")
	}

	if action == "create" && args[0] == "/" {
		Error("To create a registry use the 'xr registry create' command")
	}

	xidStr := args[0]
	// object := any(nil)
	xid := xrlib.ParseXID(xidStr)
	dataIsMeta := true
	suffix := ""

	rm, err := xid.GetResourceModelFrom(reg)
	Error(err)

	isMetadata, _ := cmd.Flags().GetBool("details")
	patch, _ := cmd.Flags().GetBool("patch")

	// If we have doc + ../rID or ../vID (but not .../versions) then...
	if xid.ResourceID != "" && rm.HasDoc() && xid.IsEntity {
		if isMetadata == false {
			dataIsMeta = false
		} else {
			suffix = "$details"
		}
	}

	data, _ := cmd.Flags().GetString("data")
	if len(data) > 0 && data[0] == '@' {
		buf, err := xrlib.ReadFile(data[1:])
		Error(err)
		data = string(buf)
	}

	objects := map[string]json.RawMessage{}

	Error(xid.ValidateTypes(reg, false))

	if xid.IsEntity {
		// If not uploading a domain doc then make sure data has something
		if dataIsMeta && len(data) == 0 {
			data = `{}`
		}
		objects[xid.String()] = []byte(data)
	} else {
		if len(data) == 0 {
			Error("Missing data")
		}
		Error(json.Unmarshal([]byte(data), &objects))
	}

	switch action {
	case "create":
		// Make sure none of the top-level entities already exist
		for id, _ := range objects {
			if !xid.IsEntity {
				id = xid.String() + "/" + id
			}
			if _, err = reg.HttpDo("GET", id, nil); err == nil {
				Error("%q already exists", id)
			}
		}
	case "update":
		// Make sure all of the top-level entities already exist
		for id, _ := range objects {
			if !xid.IsEntity {
				id = xid.String() + "/" + id
			}
			if _, err = reg.HttpDo("GET", id, nil); err != nil {
				Error("%q doesn't exists", id)
			}
		}
	case "upsert":
		// Nothing
	}

	method := "PUT"
	if patch {
		method = "PATCH"
	}

	for id, content := range objects {
		if !xid.IsEntity {
			id = xid.String() + "/" + id
		}
		res, err := reg.HttpDo(method, id+suffix, content)
		Error(err)
		if res.Code == 201 {
			Verbose("Created: %s", id)
		} else {
			if patch {
				Verbose("Patched: %s", id)
			} else {
				Verbose("Updated: %s", id)
			}
		}
	}

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
}
