package main

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"strings"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
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
		Short: "Retrieve details about the registry's model",
		Run:   modelGetFunc,
	}
	getCmd.Flags().BoolP("all", "a", false, "Include default attributes")
	getCmd.Flags().StringP("output", "o", "table", "Output format: table, json")
	modelCmd.AddCommand(getCmd)

	// "model group" commands
	// ////////////////////////////////////////////////////////////////////

	modelGroupCmd := &cobra.Command{
		Use:   "group",
		Short: "Model Group operations",
	}
	modelCmd.AddCommand(modelGroupCmd)

	groupListCmd := &cobra.Command{
		Use:   "list",
		Short: "List the Group types defined in the model",
		Run:   modelGroupListFunc,
	}
	groupListCmd.Flags().StringP("output", "o", "table",
		"Output format: table, json")
	modelGroupCmd.AddCommand(groupListCmd)

	groupGetCmd := &cobra.Command{
		Use:   "get PLURAL",
		Short: "Retrieve details about a Model Group type",
		Run:   modelGroupGetFunc,
	}
	groupGetCmd.Flags().BoolP("all", "a", false, "Include default attributes")
	groupGetCmd.Flags().StringP("output", "o", "table",
		"Output format: table, json")
	modelGroupCmd.AddCommand(groupGetCmd)

	groupCreateCmd := &cobra.Command{
		Use:   "create PLURAL:SINGULAR...",
		Short: "Create a new Model Group type",
		Run:   modelGroupCreateFunc,
	}
	groupCreateCmd.Flags().BoolP("resources", "r", false,
		"Show Resource types in output")
	groupCreateCmd.Flags().BoolP("all", "a", false,
		"Include default attributes in output")
	groupCreateCmd.Flags().StringP("output", "o", "none",
		"Output format: none, table, json")
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
	// ////////////////////////////////////////////////////////////////////

	modelResourceCmd := &cobra.Command{
		Use:   "resource",
		Short: "Model Resource operations",
	}
	modelCmd.AddCommand(modelResourceCmd)

	resourceListCmd := &cobra.Command{
		Use:   "list",
		Short: "List the Resource types in a Group type",
		Run:   modelResourceListFunc,
	}
	resourceListCmd.Flags().StringP("group", "g", "", "Group type plural name")
	resourceListCmd.Flags().StringP("output", "o", "table",
		"Output format: table, json")
	modelResourceCmd.AddCommand(resourceListCmd)

	resourceGetCmd := &cobra.Command{
		Use:   "get PLURAL",
		Short: "Retrieve details about a Model Resource type",
		Run:   modelResourceGetFunc,
	}
	resourceGetCmd.Flags().StringP("group", "g", "", "Group type plural name")
	resourceGetCmd.Flags().BoolP("all", "a", false,
		"Include default attributes")
	resourceGetCmd.Flags().StringP("output", "o", "table",
		"Output format: table, json")
	modelResourceCmd.AddCommand(resourceGetCmd)

	resourceCreateCmd := &cobra.Command{
		Use:   "create PLURAL:SINGULAR...",
		Short: "Create a new Model Resource type",
		Run:   modelResourceCreateFunc,
	}
	resourceCreateCmd.Flags().StringP("group", "g", "",
		"Group type plural name (add \":SINGULAR\" to create)")
	resourceCreateCmd.Flags().IntP("max-versions", "m", 0,
		"Max versions allowed (default 0 - no limit)")
	resourceCreateCmd.Flags().BoolP("no-doc", "n", false,
		"Don't allow for domain docs")
	resourceCreateCmd.Flags().BoolP("no-set-versionid", "i", false,
		"Don't allow for setting of versionid")
	resourceCreateCmd.Flags().BoolP("single-root", "r", false,
		"Only allow one root version")
	resourceCreateCmd.Flags().BoolP("all", "a", false,
		"Include default attributes in output")
	resourceCreateCmd.Flags().StringP("output", "o", "none",
		"Output format: none, table, json")
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

	// All this just to remove $schema
	tmp := map[string]any{}
	err = Unmarshal(buf, &tmp)
	if err != nil {
		return err
	}
	delete(tmp, "$schema")
	buf, _ = json.Marshal(tmp)

	model := &xrlib.Model{}

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

	_, err = reg.HttpDo("PUT", "/modelsource", []byte(buf))
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

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
	}

	model, err := reg.GetModel()
	Error(err)

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(model))
		return
	}

	fmt.Println("xRegistry Model:")
	PrintLabels(model.Labels, "  ", os.Stdout)
	PrintAttributes(ENTITY_REGISTRY, "", model.Attributes, "registry", "",
		os.Stdout, all)

	fmt.Println("")
	PrintGroupModelsByName(reg.Model, SortedKeys(model.Groups), output, "", true, all)
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

func PrintGroupModelsByName(model *xrlib.Model, gmNames []string, format, indent string, resources, all bool) {
	if format == "none" {
		return
	}
	if format == "json" && len(gmNames) > 1 {
		fmt.Printf(indent + "[\n")
		indent += "  "
		fmt.Printf(indent)
	}
	for i, gmName := range gmNames {
		gm := model.FindGroupModel(gmName)
		if gm == nil {
			Error("Unknown Group type: %s", gmName)
		}

		PrintGroupModel(gm, format, indent, resources, all)
		if i+1 < len(gmNames) {
			if format == "table" {
				fmt.Printf("\n")
			} else {
				fmt.Printf(",\n" + indent)
			}
		}
	}
	if format == "json" {
		if len(gmNames) > 1 {
			indent = indent[:len(indent)-2]
			fmt.Printf("\n" + indent + "]")
		}
		fmt.Printf("\n")
	}
}

func PrintGroupModel(gm *xrlib.GroupModel, format, indent string, showResources bool, all bool) {
	if format == "none" {
		// Should probably never see this, but just in case
		return
	}

	if format == "json" {
		tmpGM := *gm // copy the struct, not just the pointer
		if !showResources {
			tmpGM.Resources = nil
		}
		buf, _ := json.MarshalIndent(tmpGM, indent, "  ")
		fmt.Printf("%s", string(buf))
		return
	}

	if format != "table" {
		Error("--output must be one of 'json', 'table'")
	}

	fmt.Printf(indent+"GROUP: %s / %s\n", gm.Plural, gm.Singular)

	PrintNotEmpty(indent+"  Description    ", gm.Description, os.Stdout)
	PrintNotEmpty(indent+"  Model version  ", gm.ModelVersion, os.Stdout)
	PrintNotEmpty(indent+"  Compatible with", gm.CompatibleWith, os.Stdout)

	PrintLabels(gm.Labels, indent+"  ", os.Stdout)
	PrintAttributes(ENTITY_GROUP, "", gm.Attributes, gm.Singular, indent+"  ",
		os.Stdout, all)

	if showResources == false {
		return
	}

	rList := gm.GetResourceList()
	sort.Strings(rList)
	for _, rName := range rList {
		rm := gm.FindResourceModel(rName)

		fmt.Println("")
		PrintResourceModel(rm, format, indent+"  ", all)
	}
}

func PrintResourceModel(rm *xrlib.ResourceModel, format string, indent string, all bool) {
	fmt.Printf(indent+"RESOURCE: %s/ %s\n", rm.Plural, rm.Singular)

	PrintNotEmpty(indent+"  Description       ", rm.Description, os.Stdout)
	PrintNotEmpty(indent+"  Max versions      ", rm.GetMaxVersions(), os.Stdout)
	PrintNotEmpty(indent+"  Set version id    ", rm.SetVersionId, os.Stdout)
	PrintNotEmpty(indent+"  Set version sticky", rm.SetDefaultSticky, os.Stdout)
	PrintNotEmpty(indent+"  Has document      ", rm.HasDocument, os.Stdout)
	PrintNotEmpty(indent+"  Model version     ", rm.ModelVersion, os.Stdout)
	PrintNotEmpty(indent+"  Compatible with   ", rm.CompatibleWith, os.Stdout)

	PrintLabels(rm.Labels, indent+"  ", os.Stdout)
	PrintAttributes(ENTITY_VERSION, "", rm.VersionAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)

	PrintAttributes(ENTITY_RESOURCE, "RESOURCE", rm.ResourceAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)
	PrintAttributes(ENTITY_META, "META", rm.MetaAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)
}

func PrintLabels(labels map[string]string, indent string, w io.Writer) {
	if len(labels) > 0 {
		for i, k := range SortedKeys(labels) {
			v := labels[k]
			if i == 0 {
				fmt.Fprintf(w, indent+"Labels         : %s=%s\n", k, v)
			} else {
				fmt.Fprintf(w, indent+"                 %s=%s\n", k, v)
			}
		}
	}
}

func RemoveSystemAttributes(attrs xrlib.Attributes, level int, singular string) xrlib.Attributes {
	newAttrs := maps.Clone(attrs)

	for aName, _ := range attrs {
		if singular != "" && aName == singular+"id" {
			delete(newAttrs, aName)
			continue
		}

		if xrlib.SpecProps[aName] != nil {
			delete(newAttrs, aName)
			continue
		}

		if level == ENTITY_RESOURCE {
			excludes := []string{"versions", "versionscount", "versionsurl",
				singular, singular + "url",
				singular + "base64", singular + "proxyurl"}
			if ArrayContains(excludes, aName) {
				delete(newAttrs, aName)
			}
		}

		if level == ENTITY_VERSION {
			excludes := []string{singular, singular + "url",
				singular + "base64", singular + "proxyurl"}
			if ArrayContains(excludes, aName) {
				delete(newAttrs, aName)
			}
		}
	}

	return newAttrs
}

func PrintAttributes(level int, prefix string, attrs xrlib.Attributes,
	singular string, indent string, w io.Writer, all bool) {

	if !all {
		attrs = RemoveSystemAttributes(attrs, level, singular)
	}

	ntw := xrlib.NewIndentTabWriter(indent, w, 0, 1, 3, ' ', 0)

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

		if count == 0 {
			fmt.Println("")
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

func modelGroupListFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		Error("No arguments allowed")
	}

	output, _ := cmd.Flags().GetString("output")

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(model.Groups))
		return
	}

	itw := xrlib.NewIndentTabWriter("", os.Stdout, 0, 1, 3, ' ', 0)
	fmt.Fprintln(itw, "GROUP\tRESOURCES\tDESCRIPTION")
	for _, gmName := range SortedKeys(model.Groups) {
		gm := model.Groups[gmName]
		fmt.Fprintf(itw, "%s / %s\t%d\t%s\n", gm.Plural, gm.Singular,
			len(gm.Resources), gm.Description)
	}
	itw.Flush()
}

func modelGroupGetFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Group type name must be specified")
	}

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModel()
	Error(err)

	PrintGroupModelsByName(model, args, output, "", true, all)
}

func modelGroupCreateFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Group type name must be specified")
	}

	resources, _ := cmd.Flags().GetBool("resources")
	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")

	if !ArrayContains([]string{"none", "table", "json"}, output) {
		Error("--output must be one of 'json', 'none', 'table'")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, err := xrlib.GetRegistry(Server)
	Error(err)

	model, err := reg.GetModelSource()
	Error(err)

	verMsg := ""
	gmNames := []string{}

	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			Error("Group type name must be of the form: PLURAL:SINGULAR")
		}

		Error(ValidateNewGroup(model, parts[0], parts[1]))

		if model.Groups == nil {
			model.Groups = map[string]*xrlib.GroupModel{}
		}

		gmNames = append(gmNames, parts[0])

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
	_, err = reg.HttpDo("PUT", "/modelsource", buf)
	Error(err)
	Verbose(verMsg)

	if output == "none" {
		return
	}

	Error(reg.RefreshModel())

	PrintGroupModelsByName(reg.Model, gmNames, output, "", resources, all)
}

func ValidateNewGroup(model *xrlib.Model, plural, singular string) error {
	if plural == singular {
		Error("Group PLURAL and SINGULAR names must be different")
	}

	for _, gm := range model.Groups {
		if plural == gm.Plural {
			Error("PLURAL value (%s) conflicts with an existing Group "+
				"PLURAL name", plural)
		}
		if plural == gm.Singular {
			Error("PLURAL value (%s) conflicts with an existing Group "+
				"SINGULAR name", plural)
		}
		if singular == gm.Plural {
			Error("SINGULAR value (%s) conflicts with an existing Group "+
				"PLURAL name", singular)
		}
		if singular == gm.Singular {
			Error("SINGULAR value (%s) conflicts with an existing Group "+
				"SINGULAR name", singular)
		}
	}
	return nil
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

	model, err := reg.GetModelSource()
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
	_, err = reg.HttpDo("PUT", "/modelsource", buf)
	Error(err)
	Verbose(verMsg)
}

func modelResourceListFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		Error("No arguments allowed")
	}

	output, _ := cmd.Flags().GetString("output")

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
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
		Error("Unknown Group type: %s", group)
	}

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(gm.Resources))
		return
	}

	itw := xrlib.NewIndentTabWriter("", os.Stdout, 0, 1, 3, ' ', 0)
	fmt.Fprintln(itw, "RESOURCE\tHAS DOC\tEXT ATTRS\tDESCRIPTION")
	for _, rmName := range SortedKeys(gm.Resources) {
		rm := gm.Resources[rmName]
		attrs := rm.VersionAttributes
		attrs = RemoveSystemAttributes(attrs, ENTITY_RESOURCE, rm.Singular)
		fmt.Fprintf(itw, "%s / %s\t%v\t%d\t%s\n", rm.Plural, rm.Singular,
			rm.GetHasDocument(), len(attrs), rm.Description)
	}
	itw.Flush()
}

func modelResourceGetFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Resource type name must be specified")
	}

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
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
		Error("Unknown Group type: %s", group)
	}

	for _, rmName := range args {
		rm := gm.FindResourceModel(rmName)
		if rm == nil {
			Error("Unknown Resource type: %s", rmName)
		}

		PrintResourceModel(rm, output, "", all)
	}
}

func modelResourceCreateFunc(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		Error("At least one Resource type name must be specified")
	}

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")
	maxVersions, _ := cmd.Flags().GetInt("max-versions")
	noDoc, _ := cmd.Flags().GetBool("no-doc")
	noSetVersionId, _ := cmd.Flags().GetBool("no-set-versionid")
	singleRoot, _ := cmd.Flags().GetBool("single-root")

	if !ArrayContains([]string{"none", "table", "json"}, output) {
		Error("--output must be one of 'json', 'none', 'table'")
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

	groupPlural, groupSingular, _ := strings.Cut(group, ":")

	modelSrc, err := reg.GetModelSource()
	Error(err)
	gm := modelSrc.FindGroupModel(groupPlural)
	if gm == nil {
		if groupSingular == "" {
			Error("Group type %q does not exist", group)
		}

		// Now create the group
		Error(ValidateNewGroup(modelSrc, groupPlural, groupSingular))

		if modelSrc.Groups == nil {
			modelSrc.Groups = map[string]*xrlib.GroupModel{}
		}

		modelSrc.Groups[groupPlural] = &xrlib.GroupModel{
			Model:    modelSrc,
			Plural:   groupPlural,
			Singular: groupSingular,
		}

		buf, err := json.MarshalIndent(modelSrc, "", "  ")
		Error(err)
		_, err = reg.HttpDo("PUT", "/modelsource", buf)
		Error(err)
		Verbose("Created Group type: %s:%s\n", groupPlural, groupSingular)
		gm = modelSrc.FindGroupModel(groupPlural)
	} else {
		if groupSingular != "" && groupSingular != gm.Singular {
			Error("Group type %q already exists with a different "+
				"singular name: %s", gm.Singular)
		}
	}

	verMsg := ""
	resourceNames := []string{}
	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			Error("Resource type name must be of the form: PLURAL:SINGULAR")
		}

		if parts[0] == parts[1] {
			Error("Resource PLURAL and SINGULAR names must be different")
		}

		resourceNames = append(resourceNames, parts[0])

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

		rm := &xrlib.ResourceModel{
			Plural:   parts[0],
			Singular: parts[1],
		}
		gm.Resources[parts[0]] = rm

		if cmd.Flags().Changed("max-versions") {
			rm.SetMaxVersions(maxVersions)
		}
		if cmd.Flags().Changed("no-doc") {
			rm.HasDocument = PtrBool(!noDoc)
		}
		if cmd.Flags().Changed("no-set-versionid") {
			rm.SetVersionId = PtrBool(!noSetVersionId)
		}
		if cmd.Flags().Changed("single-root") {
			rm.SingleVersionRoot = &singleRoot
		}

		verMsg += fmt.Sprintf("Created Resource type: %s:%s\n",
			parts[0], parts[1])

	}

	buf, err := json.MarshalIndent(modelSrc, "", "  ")
	Error(err)
	_, err = reg.HttpDo("PUT", "/modelsource", buf)
	Error(err)
	Verbose(verMsg)

	if output == "none" {
		return
	}

	Error(reg.RefreshModel())
	gm = reg.Model.FindGroupModel(gm.Plural)
	if gm == nil {
		Error("Unknown Group type: %s", gm.Plural)
	}

	for i, rmName := range resourceNames {
		rm := gm.FindResourceModel(rmName)
		if rm == nil {
			Error("Unknown Resource type: %s", rmName)
		}
		if i != 0 {
			fmt.Printf("")
		}
		PrintResourceModel(rm, output, "", all)
	}
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

	model, err := reg.GetModelSource()
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
	_, err = reg.HttpDo("PUT", "/modelsource", buf)
	Error(err)
	Verbose(verMsg)
}
