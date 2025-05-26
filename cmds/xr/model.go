package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func addModelCmd(parent *cobra.Command) {
	modelCmd := &cobra.Command{
		Use:     "model",
		Short:   "Manage a regsitry's model",
		GroupID: "Admin",
	}
	parent.AddCommand(modelCmd)

	normalizeCmd := &cobra.Command{
		Use:   "normalize [ - | FILE ]",
		Short: "Parse and resolve 'includes' in an xRegistry model document",
		Run:   modelNormalizeFunc,
	}
	// normalizeCmd.Flags().StringP("data", "d", "",
	// "Data(json), @FILE, @URL, @-(stdin)")
	modelCmd.AddCommand(normalizeCmd)

	verifyCmd := &cobra.Command{
		Use:   "verify [ - | FILE ... ]",
		Short: "Parse and verify xRegistry model documents",
		Run:   modelVerifyFunc,
	}
	// verifyCmd.Flags().StringP("data", "d", "",
	// "Data(json), @FILE, @URL, @-(stdin)")
	verifyCmd.Flags().BoolP("skip-target", "", false,
		"Skip 'target' verification for 'xid' attributes")
	modelCmd.AddCommand(verifyCmd)

	updateCmd := &cobra.Command{
		Use:   "update [ - | FILE | -d ]",
		Short: "Update the registry's model",
		Run:   modelUpdateFunc,
	}
	updateCmd.Flags().StringP("data", "d", "",
		"Data(json), @FILE, @URL, @-(stdin)")
	modelCmd.AddCommand(updateCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get the registry's model",
		Run:   modelGetFunc,
	}
	getCmd.Flags().BoolP("all", "a", false, "Show all data")
	getCmd.Flags().StringP("output", "o", "table", "output: table, json")
	modelCmd.AddCommand(getCmd)

	// "model group" commands

	modelGroupCmd := &cobra.Command{
		Use:   "group",
		Short: "Model Group operations",
	}
	modelCmd.AddCommand(modelGroupCmd)

	groupCreateCmd := &cobra.Command{
		Use:   "create PLURAL:SINGULAR...",
		Short: "Create a new Model Group type",
		Run:   modelGroupCreateFunc,
	}
	modelGroupCmd.AddCommand(groupCreateCmd)

	groupDeleteCmd := &cobra.Command{
		Use:   "delete PLURAL...",
		Short: "Delete a Model Group type",
		Run:   modelGroupDeleteFunc,
	}
	groupDeleteCmd.Flags().BoolP("force", "f", false,
		"Ignore a \"not found\" error")
	modelGroupCmd.AddCommand(groupDeleteCmd)

	// "model resource" commands
	modelResourceCmd := &cobra.Command{
		Use:   "resource",
		Short: "Model Resource operations",
	}
	modelCmd.AddCommand(modelResourceCmd)

	resourceCreateCmd := &cobra.Command{
		Use:   "create PLURAL:SINGULAR...",
		Short: "Create a new Model Resource type",
		Run:   modelResourceCreateFunc,
	}
	resourceCreateCmd.Flags().StringP("group", "g", "", "Group type name")
	modelResourceCmd.AddCommand(resourceCreateCmd)

	resourceDeleteCmd := &cobra.Command{
		Use:   "delete PLURAL...",
		Short: "Delete a Model Resource type",
		Run:   modelResourceDeleteFunc,
	}
	resourceDeleteCmd.Flags().StringP("group", "g", "", "Group type name")
	resourceDeleteCmd.Flags().BoolP("force", "f", false,
		"Ignore a \"not found\" error")
	modelResourceCmd.AddCommand(resourceDeleteCmd)

}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte

	// if len(args) > 0 && cmd.Flags().Changed("data") {
	// Error("Can't specify a FILE and the -d flag")
	// }

	if len(args) > 1 {
		Error("Only one FILE is allowed to be specified")
	}

	if len(args) == 0 {
		args = []string{"-"}
	}

	fileName := args[0]
	buf, err = xrlib.ReadFile(fileName)
	Error(err)

	buf, err = ProcessIncludes(fileName, buf, true)
	Error(err)

	tmp := map[string]any{}
	Error(Unmarshal(buf, &tmp))
	fmt.Printf("%s\n", ToJSON(tmp))
}

func modelVerifyFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var err error

	if len(args) == 0 {
		args = []string{"-"}
	}

	skipTarget, _ := cmd.Flags().GetBool("ignore-target")

	for _, fileName := range args {
		prefix := ""
		if len(args) > 1 {
			prefix = fileName + ": "
		}

		buf, err = xrlib.ReadFile(fileName)
		if err == nil {
			err = VerifyModel(fileName, buf, skipTarget)
		}
		if err != nil {
			Error(err, "%s%s", prefix, err)
		}

		Verbose("%sModel verified", prefix)
	}
}

func VerifyModel(fileName string, buf []byte, skipTarget bool) error {
	buf, err := ProcessIncludes(fileName, buf, true)
	if err != nil {
		return err
		// Error("%s%s", fileName, err)
	}

	model := &registry.Model{}

	if err := Unmarshal(buf, model); err != nil {
		return err
		//Error("%s%s", fileName, err)
	}

	if skipTarget {
		if model.Stuff == nil {
			model.Stuff = map[string]any{}
		}
		model.Stuff["skipTargetCheck"] = true
	}

	if err := model.Verify(); err != nil {
		return err
		// Error("%s%s", fileName, err)
	}
	return nil
}

func modelUpdateFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var err error

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify a FILE and the -d flag")
	}

	if len(args) > 1 {
		Error("Only one FILE is allowed to be specified")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	fileName := ""
	if len(args) > 0 {
		fileName = args[0]
	} else {
		data, _ := cmd.Flags().GetString("data")
		if len(data) > 0 {
			if data[0] == '@' {
				fileName = data[1:]
			} else {
				buf = []byte(data)
			}
		}
	}

	if len(buf) == 0 {
		buf, err = xrlib.ReadFile(fileName)
		Error(err)
	}

	buf, err = ProcessIncludes(fileName, buf, true)
	Error(err)

	if len(buf) == 0 {
		Error("Missing model data")
	}

	_, err = reg.HttpDo("PUT", "/model", []byte(buf))
	Error(err)
	Verbose("Model updated")
}

func modelGetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")

	res, err := reg.HttpDo("GET", "/model", nil)
	Error(err)

	model, err := registry.ParseModel(res.Body)
	Error(err)

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(model))
		return
	}

	fmt.Println("xRegistry Model:")
	PrintLabels(model.Labels, "  ", os.Stdout)
	PrintAttributes("", model.Attributes, "registry", "", os.Stdout, all)

	for _, gID := range SortedKeys(model.Groups) {
		g := model.Groups[gID]

		fmt.Println("")
		fmt.Printf("GROUP: %s / %s\n", g.Plural, g.Singular)

		PrintNotEmpty("  Description    ", g.Description, os.Stdout)
		PrintNotEmpty("  Model version  ", g.ModelVersion, os.Stdout)
		PrintNotEmpty("  Compatible with", g.CompatibleWith, os.Stdout)

		PrintLabels(g.Labels, "  ", os.Stdout)
		PrintAttributes("", g.Attributes, g.Singular, "  ", os.Stdout, all)

		rList := g.GetResourceList()
		sort.Strings(rList)
		for _, rName := range rList {
			r := g.FindResourceModel(rName)

			fmt.Println("")
			fmt.Printf("  RESOURCE: %s/ %s\n", r.Plural, r.Singular)

			PrintNotEmpty("    Description       ", r.Description, os.Stdout)
			PrintNotEmpty("    Max versions      ", r.MaxVersions, os.Stdout)
			PrintNotEmpty("    Set version id    ", r.SetVersionId, os.Stdout)
			PrintNotEmpty("    Set version sticky", r.SetDefaultSticky, os.Stdout)
			PrintNotEmpty("    Has document      ", r.HasDocument, os.Stdout)
			PrintNotEmpty("    Model version     ", r.ModelVersion, os.Stdout)
			PrintNotEmpty("    Compatible with   ", r.CompatibleWith, os.Stdout)

			PrintLabels(g.Labels, "    ", os.Stdout)
			PrintAttributes("", r.Attributes, r.Singular, "    ", os.Stdout, all)

			PrintAttributes("META", r.MetaAttributes, r.Singular, "    ", os.Stdout, all)
		}
	}

}

func PrintNotEmpty(title, val any, w io.Writer) {
	str, ok := val.(string)
	if !ok {
		i, ok := val.(int)
		if ok {
			str = fmt.Sprintf("%d", i)
		} else {
			p, ok := val.(*bool)
			if ok {
				if p == nil || !*p {
					str = "false"
				} else {
					str = "true"
				}
			} else {
				panic("dunno")
			}
		}
	}

	if val != "" {
		fmt.Fprintf(w, "%s: %s\n", title, str)
	}
}

func PrintLabels(labels map[string]string, indent string, w io.Writer) {
	if len(labels) > 0 {
		for i, k := range SortedKeys(labels) {
			v := labels[k]
			if i == 0 {
				fmt.Fprintf(w, "  Labels         : %s=%s\n", k, v)
			} else {
				fmt.Fprintf(w, "                   %s=%s\n", k, v)
			}
		}
	}
}

func PrintAttributes(prefix string, attrs registry.Attributes,
	singular string, indent string, w io.Writer, all bool) {

	ntw := xrlib.NewIndentTabWriter(indent, w, 0, 1, 1, ' ', 0)

	// Make sure list if alphabetical, but put "*" last
	list := SortedKeys(attrs)
	if len(list) > 0 && list[0] == "*" {
		list = append(list[1:], list[0])
	}

	if prefix != "" {
		prefix += " "
	}

	count := 0
	for _, aName := range list {
		attr, _ := attrs[aName]

		if !all {
			if singular != "" && aName == singular+"id" {
				continue
			}
			if registry.SpecProps[aName] != nil {
				continue
			}
		}

		if count == 0 {
			fmt.Fprintln(ntw, "")
			fmt.Fprintln(ntw, prefix+"ATTRIBUTES:\tTYPE\tREQ\tRO\tMUT\tDEFAULT")
		}
		count++
		typ := attr.Type
		if typ == MAP {
			typ = fmt.Sprintf("%s(%s)", typ, attr.Item.Type)
		}
		req := xrlib.YesDash(attr.Required)
		ro := xrlib.YesDash(attr.ReadOnly)
		immut := xrlib.YesDash(!attr.Immutable)
		def := ""
		if !IsNil(attr.Default) {
			if attr.Type == STRING {
				def = fmt.Sprintf("%q", attr.Default)
			} else {
				def = fmt.Sprintf("%v", attr.Default)
			}
		}
		fmt.Fprintf(ntw, "%s\t%s\t%s\t%s\t%s\t%s\n", aName, typ,
			req, ro, immut, def)
	}

	ntw.Flush()
}

func modelGroupCreateFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Group type name must be specified")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)
	verMsg := ""
	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			Error("Group type name must be of the form: PLURAL:SINGULAR")
		}

		if parts[0] == parts[1] {
			Error("Group PLURAL and SINGULAR names must be different")
		}

		for _, gm := range model.Groups {
			if parts[0] == gm.Plural {
				Error("PLURAL value (%s) conflicts with an existing Group "+
					"PLURAL name", parts[0])
			}
			if parts[0] == gm.Singular {
				Error("PLURAL value (%s) conflicts with an existing Group "+
					"SINGULAR name", parts[0])
			}
			if parts[1] == gm.Plural {
				Error("SINGULAR value (%s) conflicts with an existing Group "+
					"PLURAL name", parts[1])
			}
			if parts[1] == gm.Singular {
				Error("SINGULAR value (%s) conflicts with an existing Group "+
					"SINGULAR name", parts[1])
			}
		}

		if model.Groups == nil {
			model.Groups = map[string]*xrlib.GroupModel{}
		}

		model.Groups[parts[0]] = &xrlib.GroupModel{
			Model:    model,
			Plural:   parts[0],
			Singular: parts[1],
		}

		verMsg += fmt.Sprintf("Created Group type: %s:%s\n",
			parts[0], parts[1])

	}

	buf, err := json.MarshalIndent(model, "", "  ")
	Error(err)
	_, err = reg.HttpDo("PUT", "/model", buf)
	Error(err)
	Verbose(verMsg)
}

func modelGroupDeleteFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Group type name must be specified")
	}

	force, _ := cmd.Flags().GetBool("force")

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)
	verMsg := ""
	for _, arg := range args {
		gm := model.FindGroupModel(arg)
		if gm == nil {
			msg := fmt.Sprintf("Group type %q does not exist", arg)
			if !force {
				Error(msg)
			}
			Verbose(msg + ", ignored")
			continue
		}

		delete(model.Groups, arg)

		// Remove the GROUPSxxx COLLECTION attributes
		delete(model.Attributes, arg)
		delete(model.Attributes, arg+"count")
		delete(model.Attributes, arg+"url")

		verMsg += fmt.Sprintf("Deleted Group type: %s\n", arg)
	}

	buf, err := json.MarshalIndent(model, "", "  ")
	Error(err)
	_, err = reg.HttpDo("PUT", "/model", buf)
	Error(err)
	Verbose(verMsg)
}

func modelResourceCreateFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Resource type name must be specified")
	}

	group, _ := cmd.Flags().GetString("group")
	if group == "" {
		Error("A Group type name must be provided via the --group flag")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)
	gm := model.FindGroupModel(group)
	if gm == nil {
		Error("Group type %q does not exist", group)
	}

	verMsg := ""
	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			Error("Resource type name must be of the form: PLURAL:SINGULAR")
		}

		if parts[0] == parts[1] {
			Error("Resource PLURAL and SINGULAR names must be different")
		}

		for _, rm := range gm.Resources {
			if parts[0] == rm.Plural {
				Error("PLURAL value (%s) conflicts with an existing Resource "+
					"PLURAL name", parts[0])
			}
			if parts[0] == rm.Singular {
				Error("PLURAL value (%s) conflicts with an existing Resource "+
					"SINGULAR name", parts[0])
			}
			if parts[1] == rm.Plural {
				Error("SINGULAR value (%s) conflicts with an existing "+
					"Resource PLURAL name", parts[1])
			}
			if parts[1] == rm.Singular {
				Error("SINGULAR value (%s) conflicts with an existing "+
					" Resource SINGULAR name", parts[1])
			}
		}

		if gm.Resources == nil {
			gm.Resources = map[string]*xrlib.ResourceModel{}
		}

		gm.Resources[parts[0]] = &xrlib.ResourceModel{
			Plural:   parts[0],
			Singular: parts[1],
		}

		verMsg += fmt.Sprintf("Created Resource type: %s:%s\n",
			parts[0], parts[1])

	}

	buf, err := json.MarshalIndent(model, "", "  ")
	Error(err)
	_, err = reg.HttpDo("PUT", "/model", buf)
	Error(err)
	Verbose(verMsg)
}

func modelResourceDeleteFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Resource type name must be specified")
	}

	group, _ := cmd.Flags().GetString("group")
	if group == "" {
		Error("A Group type name must be provided via the --group flag")
	}

	force, _ := cmd.Flags().GetBool("force")

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)
	gm := model.FindGroupModel(group)
	if gm == nil {
		Error("Group type %q does not exist", group)
	}

	verMsg := ""
	for _, arg := range args {
		rm := gm.FindResourceModel(arg)
		if rm == nil {
			msg := fmt.Sprintf("Resource type %q does not exist", arg)
			if !force {
				Error(msg)
			}
			Verbose(msg + ", ignored")
			continue
		}

		delete(gm.Resources, arg)

		// Remove the RESOURCESxxx COLLECTION attributes
		delete(gm.Attributes, arg)
		delete(gm.Attributes, arg+"count")
		delete(gm.Attributes, arg+"url")

		verMsg += fmt.Sprintf("Deleted Resource type: %s\n", arg)
	}

	buf, err := json.MarshalIndent(model, "", "  ")
	Error(err)
	_, err = reg.HttpDo("PUT", "/model", buf)
	Error(err)
	Verbose(verMsg)
}
