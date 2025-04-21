package main

import (
	"fmt"
	"io"
	"os"
	// "text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
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
		Use:   "normalize [ - | FILE | -d ]",
		Short: "Parse and resolve 'includes' in an xRegistry model document",
		Run:   modelNormalizeFunc,
	}
	normalizeCmd.Flags().StringP("data", "d", "", "Data(json), @FILE, @URL, @-")
	modelCmd.AddCommand(normalizeCmd)

	verifyCmd := &cobra.Command{
		Use:   "verify [ - | FILE | -d ]",
		Short: "Parse and verify xRegistry model document",
		Run:   modelVerifyFunc,
	}
	verifyCmd.Flags().StringP("data", "d", "", "Data(json), @FILE, @URL, @-")
	modelCmd.AddCommand(verifyCmd)

	updateCmd := &cobra.Command{
		Use:   "update [ - | FILE | -d ]",
		Short: "Update the registry's model",
		Run:   modelUpdateFunc,
	}
	updateCmd.Flags().StringP("data", "d", "", "Data(json), @FILE, @URL, @-")
	modelCmd.AddCommand(updateCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get the registry's model",
		Run:   modelGetFunc,
	}
	getCmd.Flags().BoolP("all", "a", false, "Show all data")
	getCmd.Flags().StringP("output", "o", "table", "output: table, json")
	modelCmd.AddCommand(getCmd)
}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify a FILE and the -d flag")
	}

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

	buf, err = registry.ProcessIncludes(fileName, buf, true)
	Error(err)

	tmp := map[string]any{}
	Error(registry.Unmarshal(buf, &tmp))
	fmt.Printf("%s\n", registry.ToJSON(tmp))
}

func modelVerifyFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var err error

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify a FILE and the -d flag")
	}

	if len(args) > 1 {
		Error("Only one FILE is allowed to be specified")
	}

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

	buf, err = registry.ProcessIncludes(fileName, buf, true)
	Error(err)

	VerifyModel("", buf)
	Verbose("Model verified")
}

func VerifyModel(fileName string, buf []byte) {
	buf, err := registry.ProcessIncludes(fileName, buf, true)
	if err != nil {
		Error("%s%s", fileName, err)
	}

	model := &registry.Model{}

	if err := registry.Unmarshal(buf, model); err != nil {
		Error("%s%s", fileName, err)
	}

	if err := model.Verify(); err != nil {
		Error("%s%s", fileName, err)
	}
}

func modelUpdateFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var err error

	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify a FILE and the -d flag")
	}

	if len(args) > 1 {
		Error("Only one FILE is allowed to be specified")
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

	buf, err = registry.ProcessIncludes(fileName, buf, true)
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

	model := (*registry.Model)(nil)
	Error(registry.Unmarshal(res.Body, &model))

	if output == "json" {
		fmt.Printf("%s\n", registry.ToJSON(model))
		return
	}

	fmt.Println("xRegistry Model:")
	PrintLabels(model.Labels, "  ", os.Stdout)
	PrintAttributes("", model.Attributes, "registry", "", os.Stdout, all)

	for _, gID := range registry.SortedKeys(model.Groups) {
		g := model.Groups[gID]

		fmt.Println("")
		fmt.Printf("GROUP: %s / %s\n", g.Plural, g.Singular)

		PrintNotEmpty("  Description    ", g.Description, os.Stdout)
		PrintNotEmpty("  Model version  ", g.ModelVersion, os.Stdout)
		PrintNotEmpty("  Compatible with", g.CompatibleWith, os.Stdout)

		PrintLabels(g.Labels, "  ", os.Stdout)
		PrintAttributes("", g.Attributes, g.Singular, "  ", os.Stdout, all)

		for _, rID := range registry.SortedKeys(g.Resources) {
			r := g.Resources[rID]

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
		for i, k := range registry.SortedKeys(labels) {
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
	list := registry.SortedKeys(attrs)
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
		if typ == registry.MAP {
			typ = fmt.Sprintf("%s(%s)", typ, attr.Item.Type)
		}
		req := xrlib.YesDash(attr.Required)
		ro := xrlib.YesDash(attr.ReadOnly)
		immut := xrlib.YesDash(!attr.Immutable)
		def := ""
		if !registry.IsNil(attr.Default) {
			if attr.Type == registry.STRING {
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
