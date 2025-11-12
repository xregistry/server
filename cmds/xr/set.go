package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addSetCmd(parent *cobra.Command) {
	setCmd := &cobra.Command{
		Use:     "set XID [+]NAME[=(VALUE | \"STRING\")]",
		Short:   "Update an entity's xRegistry metadata",
		Run:     setFunc,
		GroupID: "Entities",
	}
	setCmd.Long = setCmd.Short + "\n" + `
- Use "+NAME" to add to existing complex attribute rather than replace it
- Use "null" VALUE to delete the attribute
- Use escaped double-quotes (e.g. \"5\") to force it to be a string
`

	setCmd.Flags().StringP("output", "o", "json", "Output format: json, table")
	setCmd.Flags().BoolP("details", "m", false, "Show resource metadata")
	// Note that -m is ignored because we'll automatically add $details (or not)
	// for them, but we include the flag for consistency. Meaning, some folks
	// might want to always use it when pointing at Resources/Versions.

	parent.AddCommand(setCmd)
}

func setFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	if len(args) < 2 {
		Error("Must specify an XID and one or more NAME[=VALUE] expressions")
	}

	output, _ := cmd.Flags().GetString("output")
	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of: json, table")
	}

	xidStr := args[0]
	object := any(nil)
	xid, err := ParseXid(xidStr)
	Error(err)

	if !xid.IsEntity {
		Error("XID (%s) must reference a single entity, not a collection", xid)
	}

	resIsJSON := true

	rm, xErr := xrlib.GetResourceModelFrom(xid, reg)
	Error(xErr)

	// If we have doc + ../rID or ../vID (but not .../versions) then...
	if xid.ResourceID != "" && rm.HasDoc() && xid.IsEntity {
		xid.HasDetails = true // set even if already set via $details
	}

	oldData := map[string]any(nil)

	dataMap := map[string]any{}

	// set foo           set to null  (erase attr)
	// set foo=          set to ""
	// set foo=null      set to null  (erase attr)
	// set foo=abc       set to "abc"
	// set foo="foo bar" set to "foo bar"
	// set foo=true      set to true (bool)
	// set foo=5         set to 5 (int)
	// set foo='"5"'     set to "5"
	// set foo='"null"'  set to "null"
	// set foo='""'      set to ""
	// set +label.foo=bar   adds foo=bar to existing labels
	for _, arg := range args[1:] {
		var val any
		name, valStr, ok := strings.Cut(arg, "=")
		if name == "" {
			Error("Missing a NAME on %q", arg)
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
			ok, err := regexp.MatchString(`^(\.[0-9]+()|([0-9]+(\.[0-9]+)?))$`,
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

		if name[0] == '+' {
			name = name[1:]
			if name == "" {
				Error("Bad name in: %s", arg)
			}
			if oldData == nil {
				oldData, xErr = reg.DownloadObject(xid.String())
				Error(xErr)
			}
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
		// the attribute with a value of "nil" which is what we nede for our
		// PATCH
		if pp.Len() == 1 && val == nil {
			dataMap[pp.Top()] = nil
		}
	}

	data, err := json.Marshal(dataMap)
	Error(err)

	Verbose("Updating %q", xid)
	res, xErr := reg.HttpDo("PATCH", xid.String(), data)
	Error(xErr)

	if !resIsJSON {
		fmt.Printf("%s", string(res.Body))
		if len(res.Body) > 0 && res.Body[len(res.Body)-1] != '\n' {
			fmt.Print("\n")
		}
		return
	}

	Error(json.Unmarshal(res.Body, &object))

	if output == "json" {
		fmt.Printf("%s\n", xrlib.PrettyPrint(object, "", "  "))
		return
	}

	if output == "table" {
		fmt.Printf("%s\n", xrlib.Tablize(xid.String(), object))
		return
	}

	Error("Unknown output format: %s", output)
}
