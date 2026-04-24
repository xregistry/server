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
	getCmd.Flags().StringP("output", "o", "table", "Output format: table*, json")
	getCmd.Flag("output").DefValue = "" // hide default text
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
		"Output format: table*, json")
	groupListCmd.Flag("output").DefValue = "" // hide default text
	modelGroupCmd.AddCommand(groupListCmd)

	groupGetCmd := &cobra.Command{
		Use:   "get PLURAL",
		Short: "Retrieve details about a Model Group type",
		Run:   modelGroupGetFunc,
	}
	groupGetCmd.Flags().BoolP("all", "a", false, "Include default attributes")
	groupGetCmd.Flags().StringP("output", "o", "table",
		"Output format: table*, json")
	groupGetCmd.Flag("output").DefValue = "" // hide default text
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
		"Output format: none*, table, json")
	groupCreateCmd.Flag("output").DefValue = "" // hide default text
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
		"Output format: table*, json")
	resourceListCmd.Flag("output").DefValue = "" // hide default text
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
		"Output format: table*, json")
	resourceGetCmd.Flag("output").DefValue = "" // hide default text
	modelResourceCmd.AddCommand(resourceGetCmd)

	resourceCreateCmd := &cobra.Command{
		Use:   "create PLURAL:SINGULAR...",
		Short: "Create a new Model Resource type",
		Run:   modelResourceCreateFunc,
	}
	AddResourceFlags(resourceCreateCmd)
	modelResourceCmd.AddCommand(resourceCreateCmd)

	resourceUpdateCmd := &cobra.Command{
		Use:   "update PLURAL...",
		Short: "Update a Model Resource type",
		Run:   modelResourceCreateFunc,
	}
	AddResourceFlags(resourceUpdateCmd)
	modelResourceCmd.AddCommand(resourceUpdateCmd)

	resourceUpsertCmd := &cobra.Command{
		Use:   "upsert PLURAL:SINGULAR...",
		Short: "UPdate, or inSERT as appropriate, a Model Resource type",
		Run:   modelResourceCreateFunc,
	}
	AddResourceFlags(resourceUpsertCmd)
	modelResourceCmd.AddCommand(resourceUpsertCmd)

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

func AddResourceFlags(cmd *cobra.Command) {
	if strings.HasPrefix(cmd.Use, "create") {
		cmd.Flags().BoolP("force", "f", false,
			"Force an 'update' if exist")
	}
	if strings.HasPrefix(cmd.Use, "update") {
		cmd.Flags().BoolP("force", "f", false,
			"Force a 'create' if missing")
	}

	cmd.Flags().StringP("group", "g", "",
		"Group plural name (create with \":SINGULAR\")")
	cmd.Flags().BoolP("all", "a", false,
		"Include default attributes in output")
	cmd.Flags().StringP("output", "o", "none",
		"Output format: none*, table, json")
	cmd.Flag("output").DefValue = "" // hide default text

	cmd.Flags().String("description", "", "Description text")
	cmd.Flags().String("docs", "", "Documenations URL")
	cmd.Flags().String("icon", "", "Icon URL")
	cmd.Flags().StringArray("label", nil, "NAME[=VALUE)]")
	cmd.Flags().StringArray("type-map", nil, "NAME[=VALUE)]")
	cmd.Flags().String("model-version", "", "Model version string")
	cmd.Flags().String("model-compat-with", "", "URI of model")

	PanicIf(MAXVERSIONS != 0, "fix me")
	cmd.Flags().Int("max-versions", MAXVERSIONS,
		"Max versions allowed (0=unlimited*)")

	PanicIf(SETVERSIONID != true, "fix me")
	cmd.Flags().Bool("no-set-version-id", false,
		"VersionID is not settable")
	cmd.Flags().Bool("set-version-id", true,
		"Version ID is settable (true*)")
	cmd.Flag("set-version-id").DefValue = ""

	PanicIf(SETDEFAULTSTICKY != true, "fix me")
	cmd.Flags().Bool("no-set-default-sticky", false,
		"Can't set sticky version")
	cmd.Flags().Bool("set-default-sticky", SETDEFAULTSTICKY,
		"Can set sticky version (true*)")
	cmd.Flag("set-default-sticky").DefValue = ""

	PanicIf(HASDOCUMENT != true, "fix me")
	cmd.Flags().Bool("no-has-doc", false,
		"Doesn't support domain doc")
	cmd.Flags().Bool("has-doc", true,
		"Supports domain doc (true*)")
	cmd.Flag("has-doc").DefValue = ""

	cmd.Flags().String("version-mode", "", "Versioning algorithm")

	PanicIf(SINGLEVERSIONROOT != false, "fix me")
	cmd.Flags().Bool("no-single-version-root", true,
		"Allow multiple verson roots (true*)")
	cmd.Flags().Bool("single-version-root", false,
		"Restrict to single root")
	cmd.Flag("no-single-version-root").DefValue = ""

	PanicIf(VALIDATEFORMAT != false, "fix me")
	cmd.Flags().Bool("no-validate-format", true,
		"Disable format validation (true*)")
	cmd.Flags().Bool("validate-format", false,
		"Enable format validation")
	cmd.Flag("no-validate-format").DefValue = ""

	PanicIf(VALIDATECOMPATIBILITY != false, "fix me")
	cmd.Flags().Bool("no-validate-compat", true,
		"Disable compatibility validation (true*)")
	cmd.Flags().Bool("validate-compat", false,
		"Enable compatibility validation")
	cmd.Flag("no-validate-compat").DefValue = ""

	PanicIf(STRICTVALIDATION != false, "fix me")
	cmd.Flags().Bool("no-strict-validation", true,
		"Disable strict validation (true*)")
	cmd.Flags().Bool("strict-validation", false,
		"Enforce strict validation")
	cmd.Flag("no-strict-validation").DefValue = ""

	PanicIf(CONSISTENTFORMAT != false, "fix me")
	cmd.Flags().Bool("no-consistent-format", true,
		"Allow varying format values (true*)")
	cmd.Flags().Bool("consistent-format", false,
		"Enforce same format values")
	cmd.Flag("no-consistent-format").DefValue = ""
}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var xErr *XRError
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
	buf, xErr = xrlib.ReadFile(fileName)
	Error(xErr)

	buf, xErr = ProcessIncludes(fileName, buf, true)
	Error(xErr)

	tmp := map[string]any{}
	if err := Unmarshal(buf, &tmp); err != nil {
		Error(NewXRError("parsing_data", "",
			"error_detail="+err.Error()))
	}
	fmt.Printf("%s\n", ToJSON(tmp))
}

func modelVerifyFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var xErr *XRError

	if len(args) == 0 {
		args = []string{"-"}
	}

	skipTarget, _ := cmd.Flags().GetBool("skip-target")

	for _, fileName := range args {
		prefix := ""
		if len(args) > 1 {
			prefix = fileName + ": "
		}

		buf, xErr = xrlib.ReadFile(fileName)
		if xErr == nil {
			xErr = VerifyModel(fileName, buf, skipTarget)
		}
		if xErr != nil {
			if len(args) > 1 {
				xErr.SetDetailf("Found at: %s.", fileName)
			}
			Error(xErr)
			// Error(xErr, "%s%s", prefix, xErr)
		}

		Verbose("%sModel verified", prefix)
	}
}

func VerifyModel(fileName string, buf []byte, skipTarget bool) *XRError {
	buf, xErr := ProcessIncludes(fileName, buf, true)
	if xErr != nil {
		return xErr
		// Error("%s%s", fileName, err)
	}

	// All this just to remove $schema
	tmp := map[string]any{}
	err := Unmarshal(buf, &tmp)
	if err != nil {
		return NewXRError("parsing_data", fileName,
			"error_detail="+err.Error())
	}
	delete(tmp, "$schema")
	buf, _ = json.Marshal(tmp)

	model := &xrlib.Model{}

	if err := Unmarshal(buf, model); err != nil {
		return NewXRError("parsing_data", fileName,
			"error_detail="+err.Error())
		//Error("%s%s", fileName, err)
	}

	if skipTarget {
		if model.Stuff == nil {
			model.Stuff = map[string]any{}
		}
		model.Stuff["skipTargetCheck"] = true
	}

	if xErr := model.Verify(); xErr != nil {
		return xErr
		// Error("%s%s", fileName, err)
	}
	return nil
}

func modelUpdateFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var xErr *XRError

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify a FILE and the -d flag")
	}

	if len(args) > 1 {
		Error("Only one FILE is allowed to be specified")
	}

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

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
		buf, xErr = xrlib.ReadFile(fileName)
		Error(xErr)
	}

	buf, xErr = ProcessIncludes(fileName, buf, true)
	Error(xErr)

	if len(buf) == 0 {
		Error("Missing model data")
	}

	_, xErr = reg.HttpDo(VerboseCount > 1, "PUT", "/modelsource", []byte(buf))
	Error(xErr)
	Verbose("Model updated")
}

func modelGetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")

	if !ArrayContains([]string{"table", "json"}, output) {
		Error("--output must be one of 'json', 'table'")
	}

	model, xErr := reg.GetModel()
	Error(xErr)

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(model))
		return
	}

	fmt.Println("xRegistry Model:")
	PrintMap(model.Labels, "Labels", "  ", os.Stdout)
	PrintAttributes(ENTITY_REGISTRY, "", model.Attributes, "registry", "",
		os.Stdout, all)

	if len(model.Groups) > 0 {
		fmt.Println("")
		PrintGroupModelsByName(reg.Model, SortedKeys(model.Groups), output,
			"", true, all)
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

func PrintGroupModelsByName(model *xrlib.Model, gmNames []string, format, indent string, resources, all bool) {
	if format == "none" {
		return
	}
	if format == "json" && len(gmNames) > 1 {
		fmt.Print(indent + "[\n")
		indent += "  "
		fmt.Print(indent)
	}
	for i, gmName := range gmNames {
		gm := model.FindGroupModel(gmName)
		if gm == nil {
			Error("Unknown Group type: %s", gmName)
		}

		PrintGroupModel(gm, format, indent, resources, all)
		if i+1 < len(gmNames) {
			if format == "table" {
				fmt.Print("\n")
			} else {
				fmt.Print(",\n" + indent)
			}
		}
	}
	if format == "json" {
		if len(gmNames) > 1 {
			indent = indent[:len(indent)-2]
			fmt.Print("\n" + indent + "]")
		}
		fmt.Print("\n")
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
	PrintNotEmpty(indent+"  Model Compatible with", gm.ModelCompatibleWith, os.Stdout)

	PrintMap(gm.Labels, "Labels", indent+"  ", os.Stdout)
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
	fmt.Printf(indent+"RESOURCE: %s / %s\n", rm.Plural, rm.Singular)

	PrintNotEmpty(indent+"  Description         ", rm.Description, os.Stdout)
	PrintNotEmpty(indent+"  Documentation       ", rm.Documentation, os.Stdout)
	PrintNotEmpty(indent+"  Max versions        ", rm.GetMaxVersions(), os.Stdout)
	PrintNotEmpty(indent+"  Set version id      ", rm.SetVersionId, os.Stdout)
	PrintNotEmpty(indent+"  Set version sticky  ", rm.SetDefaultSticky, os.Stdout)
	PrintNotEmpty(indent+"  Has document        ", rm.HasDocument, os.Stdout)
	PrintNotEmpty(indent+"  Version mode        ", rm.VersionMode, os.Stdout)
	PrintNotEmpty(indent+"  Single version root ", rm.SingleVersionRoot, os.Stdout)
	PrintNotEmpty(indent+"  Validate format     ", rm.ValidateFormat, os.Stdout)
	PrintNotEmpty(indent+"  Validate compat     ", rm.ValidateCompatibility, os.Stdout)
	PrintNotEmpty(indent+"  Strict valiation    ", rm.StrictValidation, os.Stdout)
	PrintNotEmpty(indent+"  Consistent format   ", rm.ConsistentFormat, os.Stdout)
	PrintNotEmpty(indent+"  Icon URL            ", rm.Icon, os.Stdout)
	PrintNotEmpty(indent+"  Model version       ", rm.ModelVersion, os.Stdout)
	PrintNotEmpty(indent+"  Model Compat with   ", rm.ModelCompatibleWith, os.Stdout)

	PrintMap(rm.Labels, "Labels", indent+"  ", os.Stdout)
	PrintMap(rm.TypeMap, "Type map", indent+"  ", os.Stdout)
	PrintAttributes(ENTITY_VERSION, "", rm.VersionAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)

	PrintAttributes(ENTITY_RESOURCE, "RESOURCE", rm.ResourceAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)
	PrintAttributes(ENTITY_META, "META", rm.MetaAttributes,
		rm.Singular, indent+"  ", os.Stdout, all)
}

func PrintMap(daMap map[string]string, title string, indent string, w io.Writer) {
	if len(daMap) > 0 {
		for i, k := range SortedKeys(daMap) {
			v := daMap[k]
			if i == 0 {
				fmt.Fprintf(w, "%s%-19s : %s=%s\n", indent, title, k, v)
			} else {
				fmt.Fprintf(w, "%s%-19s   %s=%s\n", indent, "", k, v)
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

var useTree = false

func PrintAttributes(level int, attrPrefix string, attrs xrlib.Attributes,
	singular string, indent string, w io.Writer, all bool) {

	if !all {
		attrs = RemoveSystemAttributes(attrs, level, singular)
	}

	ntw := NewTabWriter(w, []byte(indent), 0, 1, 3, ' ', 0)

	// Make sure list if alphabetical, but put "*" last
	list := SortedKeys(attrs)
	if len(list) > 0 && list[0] == "*" {
		list = append(list[1:], list[0])
	}

	// Prefix for "ATTRIBUTES" column title
	if attrPrefix != "" {
		attrPrefix += " "
	}

	if len(list) != 0 {
		fmt.Println("")
		fmt.Fprintln(ntw, attrPrefix+"ATTRIBUTES:\tTYPE\tREQ\tRO\tMUT\tDEFAULT")
	}

	// https://www.ee.ucl.ac.uk/mflanaga/java/HTMLandASCIItableC1.html
	/*
	   ─   9472 2500 &#9472; &#x2500; &boxh;  box drawings horizontal
	   │   9474 2502 &#9474; &#x2502; &boxv;  box drawings vertical
	   ┌   9484 250C &#9484; &#x250C; &boxdr; box drawings down and right
	   ┐   9488 2510 &#9488; &#x2510; &boxdl; box drawings down and left
	   └   9492 2514 &#9492; &#x2514; &boxur; box drawings up and right
	   ┘   9496 2518 &#9496; &#x2518; &boxul; box drawings up and left
	   ├   9500 251C &#9500; &#x251C; &boxvr; box drawings vertical and right
	   ┤   9508 2524 &#9508; &#x2524; &boxvl; box drawings vertical and left
	   ┬   9516 252C &#9516; &#x252C; &boxhd; box drawings down and horizontal
	   ┴   9524 2534 &#9524; &#x2534; &boxhu; box drawings up and horizontal
	   ┼   9532 253C &#9532; &#x253C; &boxvh; box drawings vertical and horizontal
	*/
	lastItem := '\u2514'   // rune('└')
	middleItem := '\u251c' // rune('├')
	passThru := '\u2502'   // rune('│')

	processList := func(list xrlib.Attributes, counts *[]int, atDepth int) {}
	showAttr := func(attr *xrlib.Attribute, counts *[]int, atDepth int) {}

	processList = func(list xrlib.Attributes, counts *[]int, atDepth int) {
		// Count the number of attributes that will appear at this level.
		// Start with the current list, then we'll add ones from nested ifVals
		count := len(list)
		for _, attr := range list {
			for _, ifVal := range attr.IfValues {
				count += len(ifVal.SiblingAttributes)
			}
		}

		// If this is an attr of an IfVal, we've already counted this as
		// part of its grandparent, so don't count it here.
		if useTree && atDepth != len(*counts) {
			count = 0
		}

		// Push this number/count of attributes to our stack/counts
		*counts = append(*counts, count)

		// Now print each attribute, making sure "*" is last
		keys := SortedKeys(list)
		if len(keys) > 0 && keys[0] == "*" {
			keys = append(keys[1:], keys[0])
		}
		for _, key := range keys {
			attr := list[key]
			showAttr(attr, counts, atDepth)
		}

		// pop stack
		*counts = (*counts)[:len(*counts)-1]
	}

	showAttr = func(attr *xrlib.Attribute, counts *[]int, atDepth int) {
		// First get the values for all columsn except attr name (1st col)
		nestedAttrs := attr.Attributes
		typeStr := ""
		tmpType := attr.Type
		tmpItem := attr.Item
		for {
			if typeStr != "" {
				typeStr += "/"
			}
			typeStr += tmpType
			if tmpItem == nil {
				break
			}
			nestedAttrs = tmpItem.Attributes
			tmpType = tmpItem.Type
			tmpItem = tmpItem.Item
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
		if attr.Type == "(if value)" {
			// Make these columns empty
			if useTree {
				req, ro, immut = "", "", ""
			} else {
				typeStr, req, ro, immut = "", "", "", ""
			}
		}

		// Now generate the name + indent tree stuff
		indent := ""
		for depth, count := range *counts {
			if depth == atDepth && (useTree || attr.Type != "(if value)") {
				if count == 1 {
					indent += string(lastItem) + " "
				} else if count > 1 {
					indent += string(middleItem) + " "
				} else {
					indent += "  "
				}
				(*counts)[depth] = count - 1
			} else {
				if useTree {
					if count == 0 {
						indent += "  "
					} else if atDepth != depth-2 {
						indent += string(passThru) + " "
					} else {
						indent += "  "
					}
				} else {
					if count == 0 {
						indent += "  "
					} else {
						indent += string(passThru) + " "
					}
				}
			}
		}

		if attr.Type == "(if value)" {
			fmt.Fprintf(ntw, "%s%s\n", indent, attr.Name)
		} else {
			fmt.Fprintf(ntw, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
				indent, attr.Name, typeStr, req, ro, immut, def)
		}

		if useTree {
			if len(attr.IfValues) > 0 {
				*counts = append(*counts, len(attr.IfValues))
				keys := SortedKeys(attr.IfValues)
				for _, valStr := range keys {
					ifVal := attr.IfValues[valStr]
					showAttr(&xrlib.Attribute{
						Name:       fmt.Sprintf("%q", valStr),
						Type:       "(if value)",
						Attributes: ifVal.SiblingAttributes,
					}, counts, atDepth+1)
				}
				*counts = (*counts)[:len(*counts)-1]
			}
		} else {
			if len(attr.IfValues) > 0 {
				keys := SortedKeys(attr.IfValues)
				for _, valStr := range keys {
					ifVal := attr.IfValues[valStr]
					showAttr(&xrlib.Attribute{
						Name:       fmt.Sprintf(">> if %s=%q", attr.Name, valStr),
						Type:       "(if value)",
						Attributes: ifVal.SiblingAttributes,
					}, counts, atDepth)
					showAttr(&xrlib.Attribute{
						Name: fmt.Sprintf("<< endif"),
						Type: "(if value)",
					}, counts, atDepth)
				}
			}
		}

		// Now process any nested child attributes
		if len(nestedAttrs) > 0 {
			if attr.Type == "(if value)" {
				if useTree {
					processList(nestedAttrs, counts, atDepth-1)
				} else {
					keys := SortedKeys(nestedAttrs)
					for _, key := range keys {
						showAttr(nestedAttrs[key], counts, atDepth)
					}
				}
			} else {
				if useTree && atDepth != len(*counts)-1 {
					processList(nestedAttrs, counts, atDepth+3)
				} else {
					processList(nestedAttrs, counts, atDepth+1)
				}
			}
		}
	}

	processList(attrs, &[]int{}, 0)

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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModel()
	Error(xErr)

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(model.Groups))
		return
	}

	itw := NewTabWriter(os.Stdout, nil, 0, 1, 3, ' ', 0)
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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModel()
	Error(xErr)

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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModelSource()
	Error(xErr)

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
	if err != nil {
		Error(err)
	}
	_, xErr = reg.HttpDo(VerboseCount > 1, "PUT", "/modelsource", buf)
	Error(xErr)
	Verbose(verMsg)

	if output == "none" {
		return
	}

	Error(reg.RefreshModel())

	PrintGroupModelsByName(reg.Model, gmNames, output, "", resources, all)
}

func ValidateNewGroup(model *xrlib.Model, plural, singular string) *XRError {
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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModelSource()
	Error(xErr)
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
	_, xErr = reg.HttpDo(VerboseCount > 1, "PUT", "/modelsource", buf)
	Error(xErr)
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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModel()
	Error(xErr)

	gm := model.FindGroupModel(group)
	if gm == nil {
		Error("Unknown Group type: %s", group)
	}

	if output == "json" {
		fmt.Printf("%s\n", ToJSON(gm.Resources))
		return
	}

	itw := NewTabWriter(os.Stdout, nil, 0, 1, 3, ' ', 0)
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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModel()
	Error(xErr)

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

	action, _, _ := strings.Cut(cmd.Use, " ")

	ExclusiveFlags(cmd, "set-version-id", "no-set-version-id")
	ExclusiveFlags(cmd, "set-default-version-id", "no-set-default-version-id")
	ExclusiveFlags(cmd, "has-doc", "no-has-doc")
	ExclusiveFlags(cmd, "single-version-root", "no-single-version-root")
	ExclusiveFlags(cmd, "validate-format", "no-validate-format")
	ExclusiveFlags(cmd, "validate-compat", "no-validate-compat")
	ExclusiveFlags(cmd, "strict-validation", "no-strict-validation")
	ExclusiveFlags(cmd, "consistent-format", "no-consitent-format")

	output, _ := cmd.Flags().GetString("output")
	all, _ := cmd.Flags().GetBool("all")
	force, _ := cmd.Flags().GetBool("force")

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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	groupPlural, groupSingular, _ := strings.Cut(group, ":")

	modelSrc, xErr := reg.GetModelSource()
	Error(xErr)
	gm := modelSrc.FindGroupModel(groupPlural)
	verMsg := ""

	if gm == nil {
		if groupSingular == "" {
			// Doesn't doesn't exist and they didn't ask us to create it by
			// adding :SINGULAR to the --group value, so just error
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

		verMsg += fmt.Sprintf("Created Group type: %s:%s\n",
			groupPlural, groupSingular)
		gm = modelSrc.FindGroupModel(groupPlural)
	} else {
		// Just sanity check the :SINGULAR part of the group name provided
		if groupSingular != "" && groupSingular != gm.Singular {
			Error("Group type %q already exists with a different "+
				"singular name: %s", gm.Singular)
		}
	}

	// Now we can move on to processing the Resources

	resourceNames := []string{}
	for _, arg := range args {
		resourcePlural, resourceSingular, _ := strings.Cut(arg, ":")
		if resourcePlural == "" || strings.Contains(resourceSingular, ":") {
			Error("Resource type name must be of the form: PLURAL:SINGULAR")
		}

		if resourcePlural == resourceSingular {
			Error("Resource PLURAL and SINGULAR names must be different")
		}

		// First see if one exists already with the given plural name
		rm := gm.FindResourceModel(resourcePlural)

		isNew := false

		if rm == nil {
			if action == "update" && !force {
				Error("Resource type %q doesn't exists", resourcePlural)
			}

			if resourceSingular == "" {
				Error("Resource type name must be of the form: PLURAL:SINGULAR")
			}

			// Error check the names before we try to create it
			for _, tmpR := range gm.Resources {
				if resourcePlural == tmpR.Singular {
					Error("PLURAL value (%s) conflicts with an existing "+
						"Resource type's SINGULAR name", resourcePlural)
				}
				if resourceSingular == tmpR.Plural {
					Error("SINGULAR value (%s) conflicts with an existing "+
						"Resource type's PLURAL name", resourceSingular)
				}
				if resourceSingular == tmpR.Singular {
					Error("SINGULAR value (%s) conflicts with an existing "+
						" Resource SINGULAR name", resourceSingular)
				}
			}

			if gm.Resources == nil {
				gm.Resources = map[string]*xrlib.ResourceModel{}
			}

			rm = &xrlib.ResourceModel{
				GroupModel: gm,
				Plural:     resourcePlural,
				Singular:   resourceSingular,
			}
			gm.Resources[resourcePlural] = rm
			isNew = true

		} else {
			if action == "create" && !force {
				Error("Resource type %q already exists", resourcePlural)
			}

			if resourceSingular == "" {
				resourceSingular = rm.Singular
			} else if resourceSingular != rm.Singular {
				Error("Resource type %q singular name must be: %s",
					resourcePlural, rm.Singular)
			}
		}

		// Reglardless of whether 'rm' is new or existing, set its properties
		// based on what the user gave us

		resourceNames = append(resourceNames, resourcePlural)

		if cmd.Flags().Changed("description") {
			str, _ := cmd.Flags().GetString("description")
			rm.SetDescription(str)
		}

		if cmd.Flags().Changed("docs") {
			str, _ := cmd.Flags().GetString("docs")
			rm.SetDocumentation(str)
		}

		if cmd.Flags().Changed("icon") {
			str, _ := cmd.Flags().GetString("icon")
			rm.SetIcon(str)
		}

		strs, _ := cmd.Flags().GetStringArray("label")
		for _, value := range strs {
			name, val, _ := strings.Cut(value, "=")
			rm.AddLabel(name, val)
		}

		strs, _ = cmd.Flags().GetStringArray("type-map")
		for _, value := range strs {
			name, val, _ := strings.Cut(value, "=")
			Error(rm.AddTypeMap(name, val))
		}

		if cmd.Flags().Changed("model-version") {
			str, _ := cmd.Flags().GetString("model-version")
			rm.SetModelVersion(str)
		}

		if cmd.Flags().Changed("model-compat-with") {
			str, _ := cmd.Flags().GetString("model-compat-with")
			rm.SetModelCompatibleWith(str)
		}

		if cmd.Flags().Changed("max-versions") {
			i, _ := cmd.Flags().GetInt("max-versions")
			rm.SetMaxVersions(i)
		}

		if b, setIt := SetBoolFlag(cmd, "set-version-id"); setIt {
			rm.SetSetVersionId(b)
		}

		if b, setIt := SetBoolFlag(cmd, "set-default-sticky"); setIt {
			rm.SetSetDefaultSticky(b)
		}

		if b, setIt := SetBoolFlag(cmd, "has-doc"); setIt {
			rm.SetHasDocument(b)
		}

		if cmd.Flags().Changed("version-mode") {
			str, _ := cmd.Flags().GetString("version-mode")
			rm.SetVersionMode(str)
		}

		if b, setIt := SetBoolFlag(cmd, "single-version-root"); setIt {
			rm.SetSingleVersionRoot(b)
		}

		if b, setIt := SetBoolFlag(cmd, "validate-format"); setIt {
			rm.SetValidateFormat(b)
		}

		if b, setIt := SetBoolFlag(cmd, "validate-compat"); setIt {
			rm.SetValidateCompatibility(b)
		}

		if b, setIt := SetBoolFlag(cmd, "strict-validation"); setIt {
			rm.SetStrictValidation(b)
		}

		if b, setIt := SetBoolFlag(cmd, "consistent-format"); setIt {
			rm.SetConsistentFormat(b)
		}

		actionStr := "Created"
		if !isNew {
			actionStr = "Updated"
		}

		verMsg += fmt.Sprintf("%s Resource type: %s\n",
			actionStr, resourcePlural)
	}

	buf, err := json.MarshalIndent(modelSrc, "", "  ")
	Error(err)
	_, xErr = reg.HttpDo(VerboseCount > 1, "PUT", "/modelsource", buf)
	Error(xErr)
	Verbose(verMsg)

	if output == "none" {
		return
	}

	Error(reg.RefreshModel())
	gm = reg.Model.FindGroupModel(gm.Plural)
	if gm == nil {
		Error("Unknown Group type: %s", gm.Plural)
	}

	jsonOutput := map[string]any{}

	for i, rmName := range resourceNames {
		rm := gm.FindResourceModel(rmName)
		if rm == nil {
			Error("Unknown Resource type: %s", rmName)
		}

		if output == "json" {
			jsonOutput[rm.Plural] = rm
			continue
		}

		if i != 0 {
			fmt.Print("")
		}
		PrintResourceModel(rm, output, "", all)
	}

	if len(jsonOutput) > 0 {
		fmt.Printf("%s\n", ToJSON(jsonOutput))
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

	reg, xErr := xrlib.GetRegistry(Server)
	Error(xErr)

	model, xErr := reg.GetModelSource()
	Error(xErr)
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
	_, xErr = reg.HttpDo(VerboseCount > 1, "PUT", "/modelsource", buf)
	Error(xErr)
	Verbose(verMsg)
}
