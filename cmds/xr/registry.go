package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

func addRegistryCmd(parent *cobra.Command) {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "registry commands",
	}

	// registry get
	registryGetCmd := &cobra.Command{
		Use:    "get [ [PATH][?QUERY] ]",
		Short:  "Export data from the Registry",
		Run:    registryGetFunc,
		Hidden: true,
	}
	registryGetCmd.Flags().BoolP("model", "m", false, "Show model")
	registryGetCmd.Flags().BoolP("capabilities", "c", false, "Show capabilities")
	registryGetCmd.Flags().StringArrayP("inline", "i", nil, "Inline value")
	registryGetCmd.Flags().StringArrayP("filter", "f", nil, "Filter value")
	registryCmd.AddCommand(registryGetCmd)

	// registry export (alias for 'get')
	registryExportCmd := &cobra.Command{
		Use:   "export [ [PATH][?QUERY] ]",
		Short: "Retrieve the Registry",
		Run:   registryGetFunc,
	}
	registryExportCmd.Flags().BoolP("model", "m", false, "Show model")
	registryExportCmd.Flags().BoolP("capabilities", "c", false, "Show capabilities")
	registryExportCmd.Flags().StringArrayP("inline", "i", nil, "Inline value")
	registryExportCmd.Flags().StringArrayP("filter", "f", nil, "Filter value")
	registryCmd.AddCommand(registryExportCmd)

	// registry set
	registrySetCmd := &cobra.Command{
		Use:   "set attributePath[=value | -]",
		Short: "Modify an attribute on the Registry entity",
		Run:   registrySetFunc,
	}
	registryCmd.AddCommand(registrySetCmd)

	// registry put
	registryPutCmd := &cobra.Command{
		Use:    "put [ [PATH]?[QUERY] ] [ - | FILE... ]",
		Short:  "import data into the Registry",
		Run:    registryPutFunc,
		Hidden: true,
	}
	registryPutCmd.Flags().BoolP("model", "m", false, "Show model")
	registryPutCmd.Flags().BoolP("capabilities", "c", false, "Show capabilities")
	registryPutCmd.Flags().StringArrayP("inline", "i", nil, "Inline value")
	registryPutCmd.Flags().StringArrayP("filter", "f", nil, "Filter value")
	registryCmd.AddCommand(registryPutCmd)

	// registry import (alias for put)
	registryImportCmd := &cobra.Command{
		Use:   "import [ [PATH]?[QUERY] ] [ - | FILE... ]",
		Short: "Upload data into the Registry",
		Run:   registryPutFunc,
	}
	registryImportCmd.Flags().BoolP("model", "m", false, "Show model")
	registryImportCmd.Flags().BoolP("capabilities", "c", false, "Show capabilities")
	registryImportCmd.Flags().StringArrayP("inline", "i", nil, "Inline value")
	registryImportCmd.Flags().StringArrayP("filter", "f", nil, "Filter value")
	registryCmd.AddCommand(registryImportCmd)

	// parent.AddCommand(registryCmd)

	// Put some of these commands on the 'xr' cmd itself as short-cuts
	// parent.AddCommand(registryGetCmd)
	// parent.AddCommand(registryExportCmd)
	// parent.AddCommand(registryPutCmd)
	// parent.AddCommand(registryImportCmd)

}

func registryGetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	url := Server
	if len(args) == 1 {
		url += "/" + strings.TrimLeft(args[0], "/")
	} else if len(args) > 1 {
		Error("Too many arguments - just PATH[?QUERY] allowed")
	}

	next := "?"
	if strings.Contains(url, "?") {
		next = "&"
	}

	model, _ := cmd.Flags().GetBool("model")
	if model {
		url += next + "model"
		next = "&"
	}

	inlines, _ := cmd.Flags().GetStringArray("inline")
	for _, inline := range inlines {
		url += next + "inline=" + inline
		next = "&"
	}

	filters, _ := cmd.Flags().GetStringArray("filter")
	for _, filter := range filters {
		url += next + "filter=" + filter
		next = "&"
	}

	res, err := http.Get(url)
	Error(err, "Error talking to server (%s): %s", Server, err)

	body, err := io.ReadAll(res.Body)
	Error(err, "Error reading server response: %s", err)
	fmt.Printf("%s", string(body))
}

func registrySetFunc(cmd *cobra.Command, args []string) {
	if Server == "" {
		Error("No Server address provided. Try either -s or XR_SERVER env var")
	}

	if len(args) == 0 {
		Error("Need at least one name=value pair")
	}

	values := map[string]*string{}

	for _, arg := range args {
		// Note: foo= and foo are equivalent
		// Note: foo- means delete it
		path, value, found := strings.Cut(arg, "=")
		if len(path) == 0 {
			Error("Missing an attribute path on %q", arg)
		}
		valPtr := &value

		del := false
		if path, del = strings.CutSuffix(path, "-"); del {
			if found {
				Error("Using both \"-\" and \"=\" on %q isn't allowed", arg)
			}
			valPtr = nil
		}

		values[path] = valPtr
	}

	fmt.Printf("Values:\n%v\n", values)
}

func registryPutFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte

	url := Server
	if len(args) > 0 && args[0] != "-" {
		// If args[0] is NOT a valid local file name then assume args[0]
		// is meant to be the PATH of the URL
		stat, err := os.Stat(args[0])
		if os.IsNotExist(err) || stat.IsDir() {
			// File doesn't exist, or if it's a dir then it's a URL Path
			url += "/" + strings.TrimLeft(args[0], "/")
			args = args[1:]
		}
	}

	next := "?"
	if strings.Contains(url, "?") {
		next = "&"
	}

	model, _ := cmd.Flags().GetBool("model")
	if model {
		url += next + "model"
		next = "&"
	}

	inlines, _ := cmd.Flags().GetStringArray("inline")
	for _, inline := range inlines {
		url += next + "inline=" + inline
		next = "&"
	}

	filters, _ := cmd.Flags().GetStringArray("filter")
	for _, filter := range filters {
		url += next + "filter=" + filter
		next = "&"
	}

	if len(args) == 0 {
		args = []string{"-"}
	}

	for _, fileName := range args {
		Verbose("%s:\n", fileName)

		if fileName == "" || fileName == "-" {
			buf, err = io.ReadAll(os.Stdin)
			if err != nil {
				Error("Error reading from stdin: %s", err)
			}
		} else if strings.HasPrefix(fileName, "http") {
			res, err := http.Get(fileName)
			if err == nil {
				buf, err = io.ReadAll(res.Body)
				res.Body.Close()

				if res.StatusCode/100 != 2 {
					err = fmt.Errorf("Error getting data: %s\n%s",
						res.Status, string(buf))
				}
			}
		} else {
			buf, err = os.ReadFile(fileName)
		}

		if err != nil {
			Error("Error reading %q: %s", fileName, err)
		}

		// Make sure it's value JSON
		tmp := map[string]any{}
		Error(registry.Unmarshal(buf, &tmp))

		// xr registry put [PATH?QUERY] [ - | FILE... ]

		Verbose("PUT %s\n%s\n", url, string(buf))

		_, err := xrlib.HttpDo("PUT", url, buf)

		if VerboseFlag || err != nil {
			Verbose(err)
		}

		if err != nil {
			Error("")
		}
	}
}
