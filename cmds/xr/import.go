package main

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

func addImportCmd(parent *cobra.Command) {
	importCmd := &cobra.Command{
		Use:     "import [ XID ]",
		Short:   "Import entities into the registry",
		Run:     importFunc,
		GroupID: "Entities",
	}
	importCmd.Flags().StringP("data", "d", "", "Data(json), @FILE, @URL, @-")

	parent.AddCommand(importCmd)
}

func importFunc(cmd *cobra.Command, args []string) {
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

	xidStr := args[0]
	xid, err := xrlib.ParseXID(xidStr)
	Error(err)
	suffix := ""

	if xid.Type == xrlib.ENTITY_RESOURCE {
		Error("Using 'import' on a Resource isn't allowed. Try 'update', " +
			"'version' or importing on its 'versions' collection")
	}
	if xid.Type == xrlib.ENTITY_VERSION {
		Error("Using 'import' on a Version isn't allowed. Try 'create' instead")
	}

	rm, err := xid.GetResourceModelFrom(reg)
	Error(err)

	// If we have doc + ../rID or ../vID (but not .../versions) then...
	if xid.ResourceID != "" && rm.HasDoc() && xid.IsEntity {
		suffix = "$details"
	}

	data, _ := cmd.Flags().GetString("data")
	if len(data) > 0 && data[0] == '@' {
		buf, err := xrlib.ReadFile(data[1:])
		Error(err)
		data = string(buf)
	}

	if len(data) == 0 {
		Error("Missing data")
	}

	obj := map[string]json.RawMessage{}
	Error(json.Unmarshal([]byte(data), &obj))

	res, err := reg.HttpDo("POST", xid.String()+suffix, []byte(data))
	Error(err)

	Error(json.Unmarshal(res.Body, &obj))

	keys := registry.SortedKeys(obj)
	for _, key := range keys {
		tmpObj := map[string]any{}

		if xid.Type == xrlib.ENTITY_REGISTRY {
			Error(json.Unmarshal(obj[key], &tmpObj))
			tmpKeys := registry.SortedKeys(tmpObj)
			for _, tmpKey := range tmpKeys {
				Verbose("Imported: %s/%s", key, tmpKey)
			}
		} else {
			Verbose("Imported: %s", key)
		}
	}

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
}
