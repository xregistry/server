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
		// minimize walks an already-canonically-reordered *OrderedMap
		// tree (see xrlib.CanonicalPrettyReorderTree(), called below)
		// and deletes the same set of keys as before — but now via
		// OrderedMap.Delete() (order-preserving), and recursing into
		// nested *OrderedMap/[]interface{} structures instead of a
		// generic map[string]any. Because the tree is already in
		// canonical order (with real xid/model info intact) BEFORE any
		// deletion happens, there's no need to re-derive that
		// classifying information afterward — unlike the old pipeline,
		// which deleted first and only then tried (and often failed) to
		// reorder what minimize() had left behind.
		minimize := (func(objAny any) *XRError)(nil)
		minimize = func(objAny any) *XRError {
			if IsNil(objAny) {
				return nil
			}
			obj, ok := objAny.(*OrderedMap)
			if !ok {
				// Not an entity/object (e.g. an array), nothing to do.
				return nil
			}

			xidStr, ok := obj.Get("xid").(string)
			if !ok {
				// Not an entity, must be a collection map, just iterate
				for _, k := range obj.Keys {
					Error(minimize(obj.Values[k]))
				}
				return nil
			}
			xid, err := ParseXid(xidStr)
			Error(err)

			obj.Delete("xid")
			obj.Delete("self")
			model, xErr := reg.GetModel()
			Error(xErr)

			switch xid.Type {
			case ENTITY_REGISTRY:
				obj.Delete("specversion")
				obj.Delete("registryid")

				for _, gm := range model.Groups {
					obj.Delete(gm.Plural + "url")
					obj.Delete(gm.Plural + "count")
					Error(minimize(obj.Get(gm.Plural)))
				}
			case ENTITY_GROUP:
				gm, _ := model.Groups[xid.Group]

				obj.Delete(gm.Singular + "id")

				for _, rm := range gm.Resources {
					obj.Delete(rm.Plural + "url")
					obj.Delete(rm.Plural + "count")
					Error(minimize(obj.Get(rm.Plural)))

					// If collection is empty, delete it
					if nextAny := obj.Get(rm.Plural); !IsNil(nextAny) {
						if nextObj, ok := nextAny.(*OrderedMap); ok {
							if len(nextObj.Keys) == 0 {
								obj.Delete(rm.Plural)
							}
						}
					}
				}

			case ENTITY_RESOURCE:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				if vc, ok := obj.Get("versionscount").(json.Number); ok && vc.String() == "1" {
					obj.Delete("ancestorid")
				}

				if rm.GetMaxVersions() == 1 {
					obj.Delete(rm.Singular + "id")
					obj.Delete("versionid")
					obj.Delete("isdefault")
					obj.Delete("versionscount")
					obj.Delete("versionsurl")
					obj.Delete("versions")
				} else {
					for _, attr := range append([]string{}, obj.Keys...) {
						if attr != "meta" && attr != "versions" {
							obj.Delete(attr)
						}
					}
					Error(minimize(obj.Get("versions")))
				}

				obj.Delete("metaurl")
				Error(minimize(obj.Get("meta")))

			case ENTITY_META:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				obj.Delete(rm.Singular + "id")
				obj.Delete("defaultversionurl")
				if obj.Get("defaultversionsticky") == false {
					obj.Delete("defaultversion")
					obj.Delete("defaultversionsticky")
				}
				if obj.Get("readonly") == false {
					obj.Delete("readonly")
				}

			case ENTITY_VERSION:
				gm, _ := model.Groups[xid.Group]
				rm, _ := gm.Resources[xid.Resource]

				obj.Delete(rm.Singular + "id")
				obj.Delete("versionid")
				if obj.Get("isdefault") == false {
					obj.Delete("isdefault")
				}

			default:
				panic(xidStr)
			}

			return nil
		}

		// rawjson mode: if we don't need to examine/modify the data
		// (i.e. no -m/--min), skip parsing entirely and just echo the
		// server's own bytes verbatim - no reorder, no re-stringify.
		if GetRawJSON() && !minimum {
			fmt.Printf("%s", string(res.Body))
			if len(res.Body) > 0 && res.Body[len(res.Body)-1] != '\n' {
				fmt.Print("\n")
			}
			return
		}

		tree, xErr := xrlib.CanonicalPrettyReorderTreeOrRaw(res.Body, GetRawJSON())
		Error(xErr, NewXRError("parsing_response", path,
			"error_detail="+Err2String(xErr)).
			SetDetail("Response: "+string(res.Body)+"."))

		if minimum {
			Error(minimize(tree))
		}

		buf, err := StringifyCanonicalTree(tree)
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
