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

func addCreateCmd(parent *cobra.Command) {
	createCmd := &cobra.Command{
		Use:     "create [ XID ]",
		Short:   "Create a new entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
	}
	createCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	createCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")
	createCmd.Flags().BoolP("patch", "p", false,
		"Only 'update' specified attributes when -f is applied")
	createCmd.Flags().BoolP("force", "f", false,
		"Force an 'update' if already exist, skip pre-flight checks")

	parent.AddCommand(createCmd)
}

func addUpsertCmd(parent *cobra.Command) {
	upsertCmd := &cobra.Command{
		Use:     "upsert [ XID ]",
		Short:   "Update, or insert(create), an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
		Hidden:  true,
	}
	upsertCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	upsertCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")
	upsertCmd.Flags().BoolP("patch", "p", false,
		"Only update specified attributes")
	upsertCmd.Flags().BoolP("force", "f", false,
		"Skip pre-flight checks")

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
	updateCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")
	updateCmd.Flags().BoolP("patch", "p", false,
		"Only update specified attributes")
	updateCmd.Flags().BoolP("force", "f", false,
		"Force a 'create' if doesnt exist, skip pre-flight checks")
	updateCmd.Flags().BoolP("noepoch", "", false,
		"Skip 'epoch' checks")

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
	xid, err := ParseXid(xidStr)
	Error(err)
	suffix := ""

	// rm, err := xid.GetResourceModelFrom(reg)
	// Error(err)

	isMetadata, _ := cmd.Flags().GetBool("details")
	patch, _ := cmd.Flags().GetBool("patch")
	force, _ := cmd.Flags().GetBool("force")
	noEpoch, _ := cmd.Flags().GetBool("noepoch")

	data, _ := cmd.Flags().GetString("data")
	if len(data) > 0 && data[0] == '@' {
		buf, err := xrlib.ReadFile(data[1:])
		Error(err)
		data = string(buf)
	}

	// If we have doc + ../rID or ../vID then...
	if xid.ResourceID != "" && isMetadata {
		suffix = "$details"

		// If not uploading a domain doc then make sure data has something
		if len(data) == 0 {
			data = `{}`
		}
	}

	if xid.Type == ENTITY_GROUP && len(data) == 0 {
		data = `{}`
	}

	if !force {
		Error(xrlib.ValidateTypes(xid, reg, false))
	}

	objects := (map[string]json.RawMessage)(nil)
	IDs := ""

	// Make sure none of the top-level entities already exist
	if xid.IsEntity {
		if !force && action != "upsert" {
			_, err = reg.HttpDo("GET", xid.String(), nil)
			if action == "create" && err == nil {
				Error("%q already exists", xid.String())
			}
			if action == "update" && err != nil {
				Error("%q doesn't exists", xid.String())
			}
		}
		IDs = xid.String()
	} else {
		if len(data) == 0 {
			Error("Missing data")
		}

		Error(json.Unmarshal([]byte(data), &objects))

		for i, id := range SortedKeys(objects) {
			if i != 0 {
				IDs += ", "
			}
			IDs += id
			if !force && action != "upsert" {
				id = xid.String() + "/" + id
				_, err = reg.HttpDo("GET", id, nil)
				if action == "create" && err == nil {
					Error("%q already exists", id)
				}
				if action == "update" && err != nil {
					Error("%q doesn't exists", id)
				}
			}
		}
	}

	method := "PUT"
	if !xid.IsEntity {
		method = "POST"
	}
	if patch {
		method = "PATCH"
	}

	path := xid.String()
	if xid.IsEntity {
		path += suffix
	}

	if noEpoch {
		path = AddQuery(path, "noepoch")
	}

	res, err := reg.HttpDo(method, path, []byte(data))

	Error(err)
	// Verbose("Processed: %s", IDs)
	if res.Code == 201 || (action == "create" && !xid.IsEntity) {
		Verbose("Created: %s", IDs)
	} else {
		if patch {
			Verbose("Patched: %s", IDs)
		} else {
			Verbose("Updated: %s", IDs)
		}
	}

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
}
