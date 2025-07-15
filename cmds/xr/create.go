package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

	createCmd.Long = createCmd.Short + "\n" + `
Notes:
- Order of attribute flags processing: --data, --del, --set, --add
- Using --del, --set or --add implicitly enables --details
- When setting attributes use escaped double-quotes (e.g. --set prop=\"5\") to
  force it to be a string
`

	createCmd.Flags().StringP("output", "o", "none",
		"Output format (none, json) when xReg metadata")
	createCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	createCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	createCmd.Flags().BoolP("replace", "r", false,
		"Replace entire entity (all attributes) when -f used")
	createCmd.Flags().BoolP("force", "f", false,
		"Force an 'update' if exist, skip pre-flight checks")
	createCmd.Flags().StringArray("set", nil,
		"Set an attribute: --set NAME[=(VALUE | \"STRING\")]")
	createCmd.Flags().StringArray("add", nil,
		"Add to an attribute: --add NAME[=(VALUE | \"STRING\")]")
	createCmd.Flags().StringArray("del", nil,
		"Delete an attribute: --del NAME")

	parent.AddCommand(createCmd)
}

func addUpsertCmd(parent *cobra.Command) {
	upsertCmd := &cobra.Command{
		Use:     "upsert [ XID ]",
		Short:   "UPdate, or inSERT as appropriate, an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
		// Hidden:  true,
	}

	upsertCmd.Long = upsertCmd.Short + "\n" + `
Notes:
- Order of attribute flags processing: --data, --del, --set, --add
- Using --del, --set or --add implicitly enables --details
- When setting attributes use escaped double-quotes (e.g. --set prop=\"5\") to
  force it to be a string
`

	upsertCmd.Flags().StringP("output", "o",
		"none", "Output format (none, json) when xReg metadata")
	upsertCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	upsertCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	upsertCmd.Flags().BoolP("replace ", "r", false,
		"Replace entire entity (all attributes)")
	upsertCmd.Flags().BoolP("force", "f", false,
		"Skip pre-flight checks")
	upsertCmd.Flags().StringArray("set", nil, "Set an attribute")
	upsertCmd.Flags().StringArray("add", nil, "Add to an attribute")
	upsertCmd.Flags().StringArray("del", nil, "Delete an attribute")

	parent.AddCommand(upsertCmd)
}

func addUpdateCmd(parent *cobra.Command) {
	updateCmd := &cobra.Command{
		Use:     "update [ XID ]",
		Short:   "Update an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
	}

	updateCmd.Long = updateCmd.Short + "\n" + `
Notes:
- Order of attribute flags processing: --data, --del, --set, --add
- Using --del, --set or --add implicitly enables --details
- When setting attributes use escaped double-quotes (e.g. --set prop=\"5\") to
  force it to be a string
`

	updateCmd.Flags().StringP("output", "o", "none",
		"Output format (none, json) when xReg metadata")
	updateCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	updateCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	updateCmd.Flags().BoolP("replace", "r", false,
		"Replace entire entity (all attributes)")
	updateCmd.Flags().BoolP("force", "f", false,
		"Force a 'create' if missing, skip pre-flight checks")
	updateCmd.Flags().BoolP("ignoreepoch", "", false,
		"Skip 'epoch' checks")
	updateCmd.Flags().StringArray("set", nil, "Set an attribute")
	updateCmd.Flags().StringArray("add", nil, "Add to an attribute")
	updateCmd.Flags().StringArray("del", nil, "Delete an attribute")

	parent.AddCommand(updateCmd)
}

func createFunc(cmd *cobra.Command, args []string) {
	action, _, _ := strings.Cut(cmd.Use, " ")

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
		Error("To create a registry use the 'xrserver registry create' command")
	}

	xidStr := args[0]
	// object := any(nil)
	xid, err := ParseXid(xidStr)
	Error(err)
	suffix := ""

	// rm, err := xid.GetResourceModelFrom(reg)
	// Error(err)

	isMetadata, _ := cmd.Flags().GetBool("details")
	replace, _ := cmd.Flags().GetBool("replace")
	force, _ := cmd.Flags().GetBool("force")
	ignoreEpoch, _ := cmd.Flags().GetBool("ignoreepoch")
	output, _ := cmd.Flags().GetString("output")
	isDomainDoc := false

	data, _ := cmd.Flags().GetString("data")
	if len(data) > 0 && data[0] == '@' {
		buf, err := xrlib.ReadFile(data[1:])
		Error(err)
		data = string(buf)
	}

	sets, _ := cmd.Flags().GetStringArray("set")
	adds, _ := cmd.Flags().GetStringArray("add")
	dels, _ := cmd.Flags().GetStringArray("del")

	if len(sets)+len(adds)+len(dels) > 0 {
		isMetadata = true

		oldData := map[string]any(nil)
		dataMap := map[string]any{}

		if len(dels) > 0 || len(adds) > 0 {
			oldData, err = reg.DownloadObject(xid.String())
			Error(err)
		}

		if len(data) > 0 {
			Error(json.Unmarshal([]byte(data), &dataMap))
		}

		setsIndex := len(dels)
		sets = append(dels, sets...)
		addsIndex := len(sets)
		sets = append(sets, adds...)
		// --set foo           set to null  (erase attr)
		// --set foo=          set to ""
		// --set foo=null      set to null  (erase attr)
		// --set foo=abc       set to "abc"
		// --set foo="foo bar" set to "foo bar"
		// --set foo=true      set to true (bool)
		// --set foo=5         set to 5 (int)
		// --set foo='"5"'     set to "5"
		// --set foo='"null"'  set to "null"
		// --set foo='""'      set to ""
		// --add label.foo=bar   adds foo=bar to existing labels
		// --del foo           same as: --set foo  or  --set foo=
		for i, arg := range sets {
			doDel := (i < setsIndex)
			doAdd := (i >= addsIndex)

			var val any
			name, valStr, ok := strings.Cut(arg, "=")
			if name == "" {
				Error("Missing a NAME on %q", arg)
			}

			if doDel && ok {
				Error("\"=\" isn't allowed on \"--del %q\"", arg)
			}

			if !ok || valStr == "null" {
				val = nil
			} else if valStr == "" {
				val = ""
			} else if valStr == "true" {
				val = true
			} else if valStr == "false" {
				val = false
			} else if valStr == "{}" { // Not sure if this is the best way
				val = struct{}{}
			} else if valStr == "[]" { // Not sure if this is the best way
				val = []any{}
			} else if valStr[0] == '"' {
				if valStr[len(valStr)-1] != '"' {
					Error("Missing closing \" on: %s", arg)
				}
				val = valStr[1 : len(valStr)-1]
			} else {
				ok, err := regexp.MatchString(`^(\.[0-9]+()|([0-9]+(\.[0
-9]+)?))$`,
					valStr)
				Error(err)
				if ok {
					fl, err := strconv.ParseFloat(valStr, 64)
					Error(err)
					val = fl
				} else {
					// use as is
					val = valStr
				}
			}

			if doDel || doAdd {
				tmpName, _, _ := strings.Cut(name, ".")
				if tmpName != "" && IsNil(dataMap[tmpName]) &&
					!IsNil(oldData[tmpName]) {

					dataMap[tmpName] = oldData[tmpName]
				}
			}

			pp, err := PropPathFromUI(name)
			Error(err)
			err = ObjectSetProp(dataMap, pp, val)
			Error(err)

			// Not sure if this is smart or a hack but until something breaks
			// do it. We need this because ObjectSetProp doesn't apply a patch
			// it will erase an attribute being set to null rather than storing
			// the attribute with a value of "nil" which is what we need for
			// our PATCH
			if pp.Len() == 1 && val == nil {
				dataMap[pp.Top()] = nil
			}
		}

		dataBuf, err := json.Marshal(dataMap)
		Error(err)
		data = string(dataBuf)
	}

	// If we have doc + ../rID or ../vID then...
	if xid.ResourceID != "" {
		if isMetadata {
			suffix = "$details"

			// If not uploading a domain doc then make sure data has something
			if len(data) == 0 {
				data = `{}`
			}
		} else {
			isDomainDoc = true
		}
	}

	if len(data) == 0 {
		if xid.Type == ENTITY_REGISTRY || xid.Type == ENTITY_GROUP {
			data = `{}`
		}
	}

	if !force {
		Error(xrlib.ValidateTypes(xid, reg, false))
	}

	objects := (map[string]json.RawMessage)(nil)
	IDs := ""

	// Make sure none of the top-level entities already exist
	if xid.IsEntity {
		if !force && action != "upsert" {
			res, err := reg.HttpDo("GET", xid.String(), nil)
			if err != nil && res == nil {
				Error("Can't connect to server: %s", reg.GetServerURL())
			}
			if action == "create" && err == nil {
				Error("%q already exists", xid.String())
			}
			if action == "update" && res != nil && res.Code == 404 {
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
	if !replace && !isDomainDoc {
		method = "PATCH"
	}

	path := xid.String()
	if xid.IsEntity {
		path += suffix
	}

	if ignoreEpoch {
		path = AddQuery(path, "ignoreepoch")
	}

	res, err := reg.HttpDo(method, path, []byte(data))
	Error(err)

	// Verbose("Processed: %s", IDs)
	if res.Code == 201 || (action == "create" && !xid.IsEntity) {
		Verbose("Created: %s", IDs)
	} else {
		if replace {
			Verbose("Replaced: %s", IDs)
		} else {
			Verbose("Updated: %s", IDs)
		}
	}

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
	if xid.ResourceID == "" || isMetadata {
		Error(json.Unmarshal(res.Body, &objects))

		if output == "json" {
			fmt.Printf("%s\n", xrlib.PrettyPrint(objects, "", "  "))
			return
		}

		/*
			if output == "table" {
				fmt.Printf("%s\n", xrlib.Tablize(xid.String(), objects))
				return
			}
		*/
	}
}
