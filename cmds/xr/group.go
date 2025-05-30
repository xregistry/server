package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

func addGroupCmd(parent *cobra.Command) {
	groupCmd := &cobra.Command{
		Use:   "group",
		Short: "group commands",
	}

	// xr group create --file/data         JSON map of GROUPS/gIDs
	// xr group create TYPE --file/data    JSON map of gIDs
	// xr group create TYPE ID | TYPE/ID --file/data/-  JSON of group
	groupCreateCmd := &cobra.Command{
		Use: `create               # Import TYPEs. Bulk: map GroupType/GroupID
  xr group create TYPE [ID...]  # Instances of TYPE. Bulk: map GroupID/Group
  xr group create TYPE/ID...    # Instances of varying TYPEs. Data is Group`,
		Short:                 "Create instances of Groups (TYPE is singular).",
		Run:                   groupCreateFunc,
		DisableFlagsInUseLine: true,
	}
	groupCreateCmd.Flags().StringP("import", "i", "", "Map of data(json)")
	groupCreateCmd.Flags().StringP("data", "d", "",
		"Data (json), @FILE, @URL, @-(stdin)")
	groupCreateCmd.Flags().StringP("file", "f", "",
		"filename for Group data (json), \"-\" for stdin")
	groupCmd.AddCommand(groupCreateCmd)

	// xr group types
	groupTypesCmd := &cobra.Command{
		Use:   "types",
		Short: "Get Group types",
		Run:   groupTypesFunc,
	}
	groupTypesCmd.Flags().StringP("output", "o", "table", "output: table,json")
	groupCmd.AddCommand(groupTypesCmd)

	// xr group get [ TYPE ]
	groupGetCmd := &cobra.Command{
		Use:   "get [ TYPE... ]",
		Short: "Get instances of Group types (TYPE is plural)",
		Run:   groupGetFunc,
	}
	groupGetCmd.Flags().StringP("output", "o", "table", "output: table,json")
	groupCmd.AddCommand(groupGetCmd)

	// xr group delete ( TYPE [ ID... ] [--all] ) | TYPE/ID...
	groupDeleteCmd := &cobra.Command{
		Use:   "delete ( TYPE [ ID... ] | [-all] ) | TYPE/ID...",
		Short: "Delete instances of a Group type (TYPE is singular)",
		Run:   groupDeleteFunc,
	}
	groupDeleteCmd.Flags().Bool("all", false, "delete all instances of TYPE")
	groupCmd.AddCommand(groupDeleteCmd)

	// parent.AddCommand(groupCmd)
}

func groupCreateFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Must specify TYPE or TYPE/ID")
	}

	data, _ := cmd.Flags().GetString("data")
	file, _ := cmd.Flags().GetString("file")

	if data != "" && file != "" {
		Error("Both --data and --file can not be used at the same time")
	}

	if len(data) > 0 && data[0] == '@' {
		buf, err := xrlib.ReadFile(data[1:])
		Error(err)
		data = string(buf)
	}

	if file != "" {
		buf, err := xrlib.ReadFile(file)
		Error(err)
		data = string(buf)
	}

	dataMap := map[string]any(nil)
	if data != "" {
		Error(Unmarshal([]byte(data), &dataMap))
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	defaultPlural := ""
	defaultSingular := ""
	type Item struct {
		Plural   string
		Singular string
		ID       string
	}
	items := []Item{}

	for _, arg := range args {
		plural := ""
		singular, gID, found := strings.Cut(arg, "/")
		if !found {
			if defaultPlural == "" {
				defaultSingular = singular
				g, err := reg.FindGroupModelBySingular(singular)
				Error(err)
				if g == nil {
					Error("Unknown group type: %s", singular)
				}
				defaultPlural = g.Plural
				continue
			}
			plural = defaultPlural
			gID = singular
			singular = defaultSingular
		} else {
			g, err := reg.FindGroupModelBySingular(singular)
			Error(err)
			if g == nil {
				Error("Unknown group type: %s", singular)
			}
			plural = g.Plural
		}
		items = append(items, Item{plural, singular, gID})
	}

	if len(items) == 0 && defaultPlural != "" {
		if dataMap == nil {
			Error("If no IDs are provided then you must provide data")
		}

		val, ok := dataMap[defaultSingular+"id"]
		if !ok {
			Error("No IDs were provided and the data doesn't have %q",
				defaultSingular+"id")
		}

		gID, err := xrlib.AnyToString(val)
		if err != nil {
			Error(fmt.Sprintf("Value of attribute %q in JSON isn't a "+
				"string: %v", defaultSingular+"id", val))
		}

		items = []Item{Item{defaultPlural, defaultSingular, gID}}
	}

	// If any exist don't create any
	for _, item := range items {
		_, err := reg.HttpDo("GET", item.Plural+"/"+item.ID, nil)
		if err == nil {
			Error("Group %q (type: %s) already exists", item.ID, item.Singular)
		}
	}

	for _, item := range items {
		_, err = reg.HttpDo("PUT", item.Plural+"/"+item.ID, []byte(data))
		if err != nil {
			tmp := err.Error()
			if len(args) > 1 {
				tmp = item.ID + ": " + tmp
			}
			Error(tmp)
		}
		Verbose("Group %s (type: %s) created", item.ID, item.Singular)
	}
}

// xr group types
func groupTypesFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	keys, err := reg.ListGroupModels()
	Error(err)
	sort.Strings(keys)

	output, _ := cmd.Flags().GetString("output")
	switch output {
	case "table":
		tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
		fmt.Fprintln(tw, "PLURAL\tSINGULAR\tURL")
		for _, key := range keys {
			g, err := reg.FindGroupModel(key)
			Error(err)
			url, err := reg.URLWithPath(g.Plural)
			Error(err)
			fmt.Fprintf(tw, "%s\t%s\t%s\n", g.Plural, g.Singular, url.String())
		}
		tw.Flush()
	case "json":
		type out struct {
			Plural   string
			Singular string
			URL      string
		}
		res := []out{}
		for _, key := range keys {
			g, err := reg.FindGroupModel(key)
			Error(err)
			url, err := reg.URLWithPath(g.Plural)
			Error(err)
			res = append(res, out{g.Plural, g.Singular, url.String()})
		}
		buf, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("%s\n", string(buf))
	default:
		Error("--output must be one of 'table', 'json'")
	}
}

// xr group get [ TYPE [ ID ] ... | TYPE/ID ... ]
func groupGetFunc(cmd *cobra.Command, args []string) {
	output, _ := cmd.Flags().GetString("output")

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	if len(args) == 0 {
		keys, err := reg.ListGroupModels()
		Error(err)
		sort.Strings(keys)
		args = append(args, keys...)
	}

	// GroupType / GroupID / GroupAttrName / AttrValue(any)
	res := map[string]map[string]map[string]any{}

	for _, plural := range args {
		g, err := reg.FindGroupModel(plural)
		Error(err)
		if g == nil {
			Error("Uknown Group type: %s", plural)
		}
		resHttp, err := reg.HttpDo("GET", plural, nil)
		Error(err)
		resMap := map[string]map[string]any{}
		Error(json.Unmarshal(resHttp.Body, &resMap))
		res[plural] = resMap
	}

	switch output {
	case "table":
		tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
		fmt.Fprintln(tw, "TYPE\tNAME\tRESOURCES\tPATH")
		groupKeys := SortedKeys(res)
		for _, groupKey := range groupKeys {
			gMap := res[groupKey]
			gMapKeys := SortedKeys(gMap)
			for _, gMapKey := range gMapKeys {
				group := gMap[gMapKey]

				gm, err := reg.FindGroupModel(groupKey)
				Error(err)
				children := 0
				rList := gm.GetResourceList()
				for _, rName := range rList {
					// for _, rm := range gm.Resources {
					// if cntAny, ok := group[rm.Plural+"count"]; ok {
					if cntAny, ok := group[rName+"count"]; ok {
						if cnt, ok := cntAny.(float64); ok {
							children += int(cnt)
						}
					}
				}

				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
					groupKey, gMapKey, children, group["xid"])
			}
		}
		tw.Flush()
	case "json":
		fmt.Printf("%s\n", ToJSON(res))
	default:
		Error("--output must be one of 'table', 'json'")
	}
}

// xr group delete ( TYPE [ ID... ] [--all] ) | TYPE/ID... | -
func groupDeleteFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Must specify TYPE or TYPE/ID")
	}

	/*
					all, _ := cmd.Flags().GetBool("all")

					reg, err := xrlib.GetRegistry(Server)
					Error(err)

					defaultPlural := ""
					defaultSingular := ""
					type Item struct {
						Plural   string
						Singular string
						ID       string
					}
					items := []Item{}

					for _, arg := range args {
						plural := ""
						singular, gID, found := strings.Cut(arg, "/")
						if !found {
							if defaultPlural == "" {
								defaultSingular = singular
								g, err := reg.FindGroupModelBySingular(singular)
			                    Error(err)
								if g == nil {
									Error("Unknown group type: %s", singular)
								}
								defaultPlural = g.Plural
								continue
							}
							plural = defaultPlural
							gID = singular
							singular = defaultSingular
						} else {
							g, err := reg.FindGroupModelBySingular(singular)
		                    Error(err)
							if g == nil {
								Error("Unknown group type: %s", singular)
							}
							plural = g.Plural
						}
						items = append(items, Item{plural, singular, gID})
					}

					if len(items) == 0 && defaultPlural != "" {
						if dataMap == nil {
							Error("If no IDs are provided then you must provide data")
						}

						val, ok := dataMap[defaultSingular+"id"]
						if !ok {
							Error("No IDs were provided and the data doesn't have %q",
								defaultSingular+"id")
						}

						gID, err := xrlib.AnyToString(val)
						if err != nil {
							Error(fmt.Sprintf("Value of attribute %q in JSON isn't a "+
								"string: %v", defaultSingular+"id", val))
						}

						items = []Item{Item{defaultPlural, defaultSingular, gID}}
					}

					// If any exist don't create any
					for _, item := range items {
						_, err := reg.HttpDo("GET", item.Plural+"/"+item.ID, nil)
						if err == nil {
							Error("Group %q (type: %s) already exists", item.ID, item.Singular)
						}
					}

					for _, item := range items {
						_, err = reg.HttpDo("PUT", item.Plural+"/"+item.ID, []byte(data))
						if err != nil {
							tmp := err.Error()
							if len(args) > 1 {
								tmp = item.ID + ": " + tmp
							}
							Error(tmp)
						}
						Verbose("Group %s (type: %s) created", item.ID, item.Singular)
					}
	*/
}
