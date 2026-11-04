package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func addCreateCmd(parent *cobra.Command) {
	createCmd := &cobra.Command{
		Use:     "create XID",
		Short:   "Create a new entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",

		Annotations: map[string]string{
			"usage": `
Notes:
  - First --data is processed, then --set/--del based on command line order
  - When setting attributes use escaped double-quotes (e.g. --set prop=\"5\")
    to force it to be a string`,
		},
	}

	createCmd.Flags().StringP("output", "o", "none",
		"Output format (none*, json) when xReg metadata")
	createCmd.Flag("output").DefValue = "" // hide default text
	createCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	createCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	createCmd.Flags().BoolP("replace", "r", false,
		"Replace entire entity (all attributes)")
	createCmd.Flags().BoolP("force", "f", false,
		"Force an 'update' if exist, no pre-flight checks")
	createCmd.Flags().StringArray("ignore", nil, "Skip certain checks")
	createCmd.Flags().StringArray("set", nil,
		"Set an attribute: --set NAME[=(VALUE | \"STRING\")]")
	createCmd.Flags().StringArray("del", nil,
		"Delete an attribute: --del NAME")

	parent.AddCommand(createCmd)
}

func addUpsertCmd(parent *cobra.Command) {
	upsertCmd := &cobra.Command{
		Use:     "upsert XID",
		Short:   "UPdate, or inSERT as appropriate, an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",
		// Hidden:  true,

		Annotations: map[string]string{
			"usage": `
Notes:
  - First --data is processed, then --set/..del based on command line order
  - When setting attributes use escaped double-quotes (e.g. --set prop=\"5\")
    to force it to be a string`,
		},
	}

	upsertCmd.Flags().StringP("output", "o",
		"none", "Output format (none*, json) when xReg metadata")
	upsertCmd.Flag("output").DefValue = "" // hide default text
	upsertCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	upsertCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	upsertCmd.Flags().BoolP("replace", "r", false,
		"Replace entire entity (all attributes)")
	upsertCmd.Flags().BoolP("force", "f", false,
		"Skip pre-flight checks")
	upsertCmd.Flags().StringArray("ignore", nil, "Skip certain checks")
	upsertCmd.Flags().StringArray("set", nil,
		"Set an attribute: --set NAME[=(VALUE | \"STRING\")]")
	upsertCmd.Flags().StringArray("del", nil, "Delete an attribute")

	parent.AddCommand(upsertCmd)
}

func addUpdateCmd(parent *cobra.Command) {
	updateCmd := &cobra.Command{
		Use:     "update XID",
		Short:   "Update an entity in the registry",
		Run:     createFunc,
		GroupID: "Entities",

		Annotations: map[string]string{
			"usage": `
Notes:
  - First --data is processed, then --set/--del based on command line order
  - When setting attributes use escaped double-quotes (e.g. --set prop=\"5\")
    to force it to be a string`,
		},
	}

	updateCmd.Flags().StringP("output", "o", "none",
		"Output format (none*, json) when xReg metadata")
	updateCmd.Flag("output").DefValue = "" // hide default text
	updateCmd.Flags().BoolP("details", "m", false, "Data is resource metadata")
	updateCmd.Flags().StringP("data", "d", "",
		"Data, @FILE, @URL, @-(stdin)")
	updateCmd.Flags().BoolP("replace", "r", false,
		"Replace entire entity (all attributes)")
	updateCmd.Flags().BoolP("force", "f", false,
		"Force a 'create' if missing, no pre-flight checks")
	updateCmd.Flags().StringArray("ignore", nil, "Skip certain checks")
	updateCmd.Flags().StringArray("set", nil,
		"Set an attribute: --set NAME[=(VALUE | \"STRING\")]")
	updateCmd.Flags().StringArray("del", nil, "Delete an attribute")

	parent.AddCommand(updateCmd)
}

func createFunc(cmd *cobra.Command, args []string) {
	action, _, _ := strings.Cut(cmd.Use, " ")

	reg, xErr := xrlib.GetRegistry("", XRConfig)
	Error(xErr)

	cliReg := (*xrlib.CLIReg)(reg)

	if len(args) == 0 {
		Error("Must specify the XID of an entity")
	}

	if len(args) > 1 {
		Error("Only one XID is allowed to be specified")
	}

	if action == "create" && args[0] == "/" {
		Error("To create a registry use the 'xrserver registry create' command")
	}

	xidStr, queryParams, _ := strings.Cut(args[0], "?")
	if len(xidStr) > 0 && xidStr[0] != '/' {
		xidStr = "/" + xidStr
	}

	xid, err := ParseXid(xidStr)
	Error(err)

	isMetadata, _ := cmd.Flags().GetBool("details")
	replace, _ := cmd.Flags().GetBool("replace")
	force, _ := cmd.Flags().GetBool("force")
	ignores, _ := cmd.Flags().GetStringArray("ignore")
	output, _ := cmd.Flags().GetString("output")

	if !force {
		// Skipping error checking, unless we really need it down below
		Error(reg.ValidateXID(xid))
	}

	if xid.HasDetails {
		isMetadata = true
	}

	data := []byte(nil)
	dataJSON := map[string]any(nil)

	dataStr, _ := cmd.Flags().GetString("data")
	if len(dataStr) > 0 {
		if dataStr[0] == '@' {
			data, xErr = xrlib.ReadFile(dataStr[1:])
			Error(xErr)
		} else {
			data = []byte(dataStr)
		}
	}

	if xid.Type == ENTITY_RESOURCE || xid.Type == ENTITY_VERSION {
		xid.HasDetails = true
	}
	ObjectPath := xid.String()

	// Existing entity's data if we ever download it
	oldData := map[string]any(nil)

	ops := GetArgOrder(cmd, os.Args)
	doSetDel := cmd.Flags().Changed("set") || cmd.Flags().Changed("del")

	// Convert any cmd line "--data" into "--set RESOURCEbase64=..." if needed
	if data != nil && !isMetadata {
		if xid.Type == ENTITY_RESOURCE || xid.Type == ENTITY_VERSION {
			// Need to validate despite --force to make sure hasDoc is set
			Error(reg.ValidateXID(xid))

			rm, xErr := reg.FindResourceModel(xid.Group, xid.Resource)
			Error(xErr)

			if rm.GetHasDocument() {
				ops = append([]Operation{{
					Action: "set",
					Value: rm.Singular +
						"base64=" +
						base64.StdEncoding.EncodeToString(data),
				}}, ops...)
				data = nil
				doSetDel = true
			}
		}
	}

	if data != nil {
		if err := Unmarshal(data, &dataJSON); err != nil {
			Error(NewXRError("parsing_data", "/",
				"error_detail="+err.Error()))
		}
	}

	getRes := (*HttpResponse)(nil)

	if doSetDel {
		if oldData == nil && !replace {
			getRes, xErr = HttpDo(&HttpRequest{
				Verb:    "GET",
				URL:     cliReg.WithPath(ObjectPath),
				Headers: xrlib.HTTPHeaders,
				Debug:   VerboseCount > 2,
			})
			if !xErr.IsType("not_found") {
				Error(xErr)
			}
			oldData = getRes.JSON
		}

		// --del attr             # attr
		// --del obj              # del entire obj (maps are same as obj)
		// --del array            # delete entire array

		// --del obj.key          # del just the one key
		// --del array[2]         # del just the index 2, 0 for first
		// --del array[-1]        # del last

		// --set attr             # set to null (erase attr)
		// --set attr=            # set to ""
		// --set attr=null        # set to null (erase attr)
		// --set attr=abc         # set to 'abc"
		// --set attr="foo bar"   # set to "foo bar"
		// --set attr=true        # set to true (bool)
		// --set attr=5           # set to 5 (int)
		// --set attr='"5"'       # set to "5" (string)
		// --set attr='"null"'    # set to "null" (string)
		// --set attr='""'        # set to ""
		// --set labels.foo=bar   # add foo=bar to existing labels
		// --del attr             # same as --set attr  or  (--set attr= ??)

		// --set obj={}           # create empty obj/map, clear it present
		// --set array=[]         # create empty array, clear if present

		// --set attr=...         # set scalar attribute
		// --set obj.key=...      # set key
		// --set array[2]=...     # set index 2
		// --set array[2:]=...    # insert before index 2 (0 = add to start)
		// --set array[-1]=...    # add to end of array

		// NOT SUPPORTED YET:
		//   array[2:]
		//   array[-1]

		for _, op := range ops {
			action := op.Action
			arg := op.Value

			// Only process know ops - everything else is an arg we don't
			// care about for the purposes of set/del...
			if action != "set" && action != "del" {
				continue
			}

			val := any(nil) // assume we're deleting attr fornow

			name, valStr, ok := strings.Cut(arg, "=")
			if name == "" {
				Error("Missing a NAME on %q", arg)
			}

			if action == "del" && ok {
				Error(`"=" isn't allowed on --del %q`, arg)
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
				// Look for a floating point number: .NUM+ or NUM+(.NUM+)?
				ok, err :=
					regexp.MatchString(`^(\.[0-9]+()|([0-9]+(\.[0-9]+)?))$`,
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

			pp, err := PropPathFromUI(name)
			Error(err)

			rootName := pp.Top()

			if dataJSON == nil {
				dataJSON = map[string]any{}
			}

			oldVal, ok := dataJSON[rootName]
			if !ok && oldData != nil {
				oldVal = oldData[rootName]
				if !IsNil(oldVal) {
					dataJSON[rootName] = oldVal
				}
			}

			/*
				if action == "del" || action == "add" {
					// Grab top-level attribute's value from oldData so we can
					// add to it, only if we don't already have data in dataMap
					if IsNil(dataJSON[rootName]) && !IsNil(oldData[rootName]) {
						dataJSON[rootName] = oldData[rootName]
					}
				}
			*/

			err = ObjectSetProp(dataJSON, pp, val, "SHRINK_ARRAYS")
			Error(err)

			// Not sure if this is smart or a hack but until something breaks
			// do it. We need this because ObjectSetProp doesn't apply a patch,
			// it will erase an attribute being set to null rather than storing
			// the attribute with a value of "nil" which is what we need for
			// our PATCH call to the server. Notice this just applies for the
			// root attr processing.
			if pp.Len() == 1 && val == nil {
				dataJSON[rootName] = nil
			}
		}

		data, err = json.Marshal(dataJSON)
		Error(err)
	}

	if xid.IsEntity && len(data) == 0 {
		data = []byte("{}")
	}

	IDs := ""

	// Make sure none of the top-level entities already exist
	if xid.IsEntity {
		if !force && action != "upsert" {
			if getRes == nil {
				getRes, xErr = HttpDo(&HttpRequest{
					Verb:    "GET",
					URL:     cliReg.WithPath(ObjectPath),
					Headers: xrlib.HTTPHeaders,
					Debug:   VerboseCount > 2,
				})
				if xErr != nil && getRes == nil {
					Error(xErr)
				}
				oldData = getRes.JSON
			}

			if action == "create" && xErr == nil {
				Error(NewXRError("exists", xidStr))
			}
			if action == "update" && getRes != nil && getRes.Code == 404 {
				Error(NewXRError("not_found", xidStr))
			}
		}
		IDs = xidStr // not xid.String() to use the original (not $details)
	} else {
		if len(data) == 0 {
			Error("Missing 'data'")
		}

		for i, id := range SortedKeys(dataJSON) {
			if i != 0 {
				IDs += ", "
			}
			IDs += id
			if !force && action != "upsert" {
				id = xid.String() + "/" + id
				_, xErr := HttpDo(&HttpRequest{
					Verb:    "GET",
					URL:     cliReg.WithPath(id),
					Headers: xrlib.HTTPHeaders,
					Debug:   VerboseCount > 2,
				})
				Error(xErr)

				if action == "update" && xErr != nil {
					Error(NewXRError("not_found", id))
				}
			}
		}
	}

	method := "PUT"
	if !xid.IsEntity {
		method = "POST"
	}
	if !replace {
		method = "PATCH"
	}

	path := AddQuery(ObjectPath, queryParams)

	if cmd.Flags().Changed("ignore") {
		path = AddQuery(path, "ignore="+strings.Join(ignores, ","))
	}

	res, xErr := HttpDo(&HttpRequest{
		Verb:    method,
		URL:     cliReg.WithPath(path),
		Headers: xrlib.HTTPHeaders,
		Body:    data,
		Debug:   VerboseCount > 2,
	})
	Error(xErr)

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

	// Response MUST always be JSON since we never send domain doc in body
	Error(res.JSONError)

	// TODO allow for GET output to be shown via -o and inline/doc/filter...
	if output == "json" {
		if GetRawJSON() {
			fmt.Printf("%s", string(res.Body))
			if len(res.Body) > 0 && res.Body[len(res.Body)-1] != '\n' {
				fmt.Print("\n")
			}
			return
		}

		tree, xErr := registry.CanonicalPrettyReorderTree(res.Body)
		Error(xErr, NewXRError("parsing_response", path,
			"error_detail="+Err2String(xErr)))
		buf, err := StringifyCanonicalTree(tree)
		Error(err)
		fmt.Printf("%s\n", string(buf))
		return
	}

	/*
		if output == "table" {
			fmt.Printf("%s\n", xrlib.Tablize(xidStr, objects))
			return
		}
	*/
}
