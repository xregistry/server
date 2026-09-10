package main

import (
	"encoding/json"
	"fmt"
	"strings"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addGetCmd(parent *cobra.Command) {
	getCmd := &cobra.Command{
		Use:     "get [XID]",
		Short:   "Retrieve entities from the registry",
		Run:     getFunc,
		GroupID: "Entities",
	}
	getCmd.Flags().StringArrayP("filter", "f", nil, "Filter: expr[,expr]")
	getCmd.Flags().StringArrayP("inline", "i", nil, "Inline entities: *, ...")
	getCmd.Flags().Bool("min", false,
		"Minimize the data (e.g. no *url attributes)")
	getCmd.Flags().Bool("doc", false, "Retieve document view of entities")
	getCmd.Flags().StringP("output", "o", "json", "Output format: json*, table")
	getCmd.Flag("output").DefValue = "" // hide default text
	getCmd.Flags().BoolP("details", "m", false, "Show resource metadata")

	parent.AddCommand(getCmd)
}

func getFunc(cmd *cobra.Command, args []string) {
	if GetServer() == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, xErr := xrlib.GetRegistry(GetServer())
	Error(xErr)

	filters, _ := cmd.Flags().GetStringArray("filter")
	inlines, _ := cmd.Flags().GetStringArray("inline")
	docView, _ := cmd.Flags().GetBool("doc")
	minimum, _ := cmd.Flags().GetBool("min")
	output, _ := cmd.Flags().GetString("output")
	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of: json, table")
	}

	if len(args) == 0 {
		args = []string{"/"}
	}

	if len(args) > 1 {
		Error("Only one XID is allowed to be specified")
	}

	xidStr := args[0]
	xidStr, queryParams, _ := strings.Cut(xidStr, "?")

	if len(xidStr) > 0 && xidStr[0] != '/' {
		xidStr = "/" + xidStr
	}
	object := any(nil)
	xid, err := ParseXid(xidStr)
	Error(err)
	resIsJSON := true
	suffix := ""

	rm, xErr := xrlib.GetResourceModelFrom(xid, reg)
	Error(xErr)

	hasDetails, _ := cmd.Flags().GetBool("details")

	// If we have doc + ../rID or ../vID (but not .../versions) then...
	if xid.ResourceID != "" && rm.HasDoc() && xid.IsEntity {
		if hasDetails == false {
			resIsJSON = false
		} else {
			suffix = "$details"
		}
	}

	path := xid.String() + suffix

	path = AddQuery(path, queryParams)

	if docView {
		path = AddQuery(path, "doc")
	}
	if len(filters) > 0 {
		path = AddQuery(path, "filter="+strings.Join(filters, ","))
	}

	if cmd.Flags().Changed("inline") && len(inlines) == 0 {
		path = AddQuery(path, "inline")
	} else if len(inlines) > 0 {
		path = AddQuery(path, "inline="+strings.Join(inlines, ","))
	}

	res, xErr := reg.HttpDo(VerboseCount > 1, "GET", path, nil)
	Error(xErr)

	path = strings.TrimRight(GetServer(), "/") + "/" +
		strings.TrimLeft(path, "/")

	if !resIsJSON {
		fmt.Printf("%s", string(res.Body))
		// Don't add a \n since that could mess people up if they're sending
		// the output on to another cmd or file (don't mess with their data)
		/*
			if len(res.Body) > 0 && res.Body[len(res.Body)-1] != '\n' {
				fmt.Print("\n")
			}
		*/
		return
	}

	if output == "json" {
		minimize := (func(objAny any) *XRError)(nil)
		minimize = func(objAny any) *XRError {
			if IsNil(objAny) {
				return nil
			}
			obj, ok := objAny.(map[string]any)
			if !ok {
				return nil
			}

			xidStr, ok := obj["xid"].(string)
			if !ok {
				// Not an entity, must be a collection, just iterate
				for _, nextObj := range obj {
					Error(minimize(nextObj))
				}
				return nil
			}
			xid, err := ParseXid(xidStr)
			Error(err)

			delete(obj, "xid")
			delete(obj, "self")
			model, xErr := reg.GetModel()
			Error(xErr)

			switch xid.Type {
			case ENTITY_REGISTRY:
				delete(obj, "specversion")
				delete(obj, "registryid")

				for _, gm := range model.Groups {
					delete(obj, gm.Plural+"url")
					delete(obj, gm.Plural+"count")
					Error(minimize(obj[gm.Plural]))
				}
			case ENTITY_GROUP:
				gm, _ := model.Groups[xid.Group]

				delete(obj, gm.Singular+"id")

				for _, rm := range gm.Resources {
					delete(obj, rm.Plural+"url")
					delete(obj, rm.Plural+"count")
					Error(minimize(obj[rm.Plural]))

					// If collection is empty, delete it
					if nextAny, ok := obj[rm.Plural]; ok {
						if nextObj, ok := nextAny.(map[string]any); ok {
							if len(nextObj) == 0 {
								delete(obj, rm.Plural)
							}
						}
					}
				}

			case ENTITY_RESOURCE:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				if obj["versionscount"] == 1.0 {
					delete(obj, "ancestorid")
				}

				if rm.GetMaxVersions() == 1 {
					delete(obj, rm.Singular+"id")
					delete(obj, "versionid")
					delete(obj, "isdefault")
					delete(obj, "versionscount")
					delete(obj, "versionsurl")
					delete(obj, "versions")
				} else {
					for attr, _ := range obj {
						if attr != "meta" && attr != "versions" {
							delete(obj, attr)
						}
					}
					Error(minimize(obj["versions"]))
				}

				delete(obj, "metaurl")
				Error(minimize(obj["meta"]))

			case ENTITY_META:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				delete(obj, rm.Singular+"id")
				delete(obj, "defaultversionurl")
				if obj["defaultversionsticky"] == false {
					delete(obj, "defaultversion")
					delete(obj, "defaultversionsticky")
				}
				if obj["readonly"] == false {
					delete(obj, "readonly")
				}

			case ENTITY_VERSION:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				delete(obj, rm.Singular+"id")
				delete(obj, "versionid")
				if obj["isdefault"] == false {
					delete(obj, "isdefault")
				}

			default:
				panic(xidStr)
			}

			return nil
		}

		if minimum {
			Error(minimize(res.JSON))
			res.Body, err = json.MarshalIndent(res.JSON, "", "  ")
			Error(err)
		}

		// buf, err := PrettyPrintJSON(res.Body, "", "  ")
		buf, err := xrlib.CanonicalPrettyPrintJSON(res.Body)
		Error(err, NewXRError("parsing_response", path,
			"error_detail="+Err2String(err)).
			SetDetail("Response: "+string(res.Body)+"."))

		fmt.Printf("%s\n", string(buf))
		return
	}

	if output == "table" {
		err = json.Unmarshal(res.Body, &object)
		Error(err, NewXRError("parsing_response", path,
			"error_detail="+Err2String(err)).
			SetDetail("Response: "+string(res.Body)+"."))
		fmt.Printf("%s\n", xrlib.Tablize(xid.String(), object))
		return
	}

	Error("Unknown output format: %s", output)
}
