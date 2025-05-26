package main

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addImportCmd(parent *cobra.Command) {
	importCmd := &cobra.Command{
		Use:     "import [ XID ]",
		Short:   "Import entities into the registry",
		Run:     importFunc,
		GroupID: "Entities",
	}
	importCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")

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
	xid, err := ParseXid(xidStr)
	Error(err)
	suffix := ""

	if xid.Type == ENTITY_RESOURCE {
		Error("Using 'import' on a Resource isn't allowed. Try 'update', " +
			"'version' or importing on its 'versions' collection")
	}
	if xid.Type == ENTITY_VERSION {
		Error("Using 'import' on a Version isn't allowed. Try 'create' instead")
	}

	rm, err := xrlib.GetResourceModelFrom(xid, reg)
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
	subObj := (map[string]json.RawMessage)(nil)

	switch xid.Type {
	case ENTITY_REGISTRY:
		for _, gType := range SortedKeys(obj) {
			group := obj[gType]
			Error(json.Unmarshal(group, &subObj))
			Verbose("Imported: %d %s", len(subObj), gType)
		}
	case ENTITY_GROUP:
		for _, rType := range SortedKeys(obj) {
			resource := obj[rType]
			Error(json.Unmarshal(resource, &subObj))
			Verbose("Imported: %d %s", len(subObj), rType)
		}
	case ENTITY_RESOURCE:
		Verbose("Imported: %d versions", len(obj))
	case ENTITY_META:
		Verbose("Should have errored")
	case ENTITY_VERSION:
		Verbose("Should have errored")
	case ENTITY_MODEL:
		Verbose("Should have errored")

	case ENTITY_GROUP_TYPE:
		Verbose("Imported: %d %s", len(obj), xid.Group)
	case ENTITY_RESOURCE_TYPE:
		Verbose("Imported: %d %s", len(obj), xid.Resource)
	case ENTITY_VERSION_TYPE:
		Verbose("Imported: %d versions", len(obj))
	}

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
}
