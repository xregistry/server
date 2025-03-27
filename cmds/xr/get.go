package main

import (
	"encoding/json"
	"fmt"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/xregistry/server/cmds/xr/xrlib"
	// "github.com/xregistry/server/registry"
	"github.com/spf13/cobra"
)

func addGetCmd(parent *cobra.Command) {
	getCmd := &cobra.Command{
		Use:   "get [ XID ]",
		Short: "Retrieve data from the registry",
		Run:   getFunc,
	}
	getCmd.Flags().StringP("output", "o", "json", "Output format(json,human)")
	getCmd.Flags().BoolP("details", "m", false, "Show resource metadata")

	parent.AddCommand(getCmd)
}

func getFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	output, _ := cmd.Flags().GetString("output")
	if !xrlib.ArrayContains([]string{"human", "json"}, output) {
		Error("--output must be one of 'json', 'human'")
	}

	if len(args) == 0 {
		args = []string{"/"}
	}

	if len(args) > 1 {
		Error("Only one XID is allowed to be specified")
	}

	xid := args[0]
	object := any(nil)
	XID := xrlib.ParseXID(xid)
	resIsJSON := true
	suffix := ""

	rm, err := XID.GetResourceModelFrom(reg)
	Error(err)

	hasDetails, _ := cmd.Flags().GetBool("details")

	// If we have doc + ../rID or ../vID (but not .../versions) then...
	if XID.ResourceID != "" && rm.HasDoc() && XID.IsEntity {
		if hasDetails == false {
			resIsJSON = false
		} else {
			suffix = "$details"
		}
	}

	res, err := reg.HttpDo("GET", xid+suffix, nil)
	Error(err)

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

	if output == "human" {
		fmt.Printf("%s\n", xrlib.Humanize(xid, object))
		return
	}

	Error("Unknown output format: %s", output)
}
