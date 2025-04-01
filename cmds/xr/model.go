package main

import (
	"fmt"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

func addModelCmd(parent *cobra.Command) {
	modelCmd := &cobra.Command{
		Use:     "model",
		Short:   "model commands",
		GroupID: "Admin",
	}

	modelNormalizeCmd := &cobra.Command{
		Use:   "normalize [ - | FILE | -d ]",
		Short: "Parse and resolve includes in an xRegistry model document",
		Run:   modelNormalizeFunc,
	}
	modelNormalizeCmd.Flags().StringP("data", "d", "",
		"Data (json),@FILE,@URL,-")
	modelCmd.AddCommand(modelNormalizeCmd)

	modelVerifyCmd := &cobra.Command{
		Use:   "verify [ - | FILE | -d ]",
		Short: "Parse and verify xRegistry model document",
		Run:   modelVerifyFunc,
	}
	modelVerifyCmd.Flags().StringP("data", "d", "", "Data (json),@FILE,@URL,-")
	modelCmd.AddCommand(modelVerifyCmd)

	modelUpdateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update the registry's model",
		Run:   modelUpdateFunc,
	}
	modelUpdateCmd.Flags().StringP("data", "d", "", "Data (json),@FILE,@URL,-")
	modelCmd.AddCommand(modelUpdateCmd)

	modelGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Get the registry's model",
		Run:   modelGetFunc,
	}
	modelCmd.AddCommand(modelGetCmd)

	parent.AddCommand(modelCmd)
}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte

	if len(args) > 0 && cmd.Flags().Changed("data") {
		Error("Can't specify an arg and the -d flag")
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
		Error("Can't specify an arg and the -d flag")
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

	res, err := reg.HttpDo("GET", "/model", nil)
	Error(err)

	tmp := map[string]any{}
	Error(registry.Unmarshal(res.Body, &tmp))
	fmt.Printf("%s\n", registry.ToJSON(tmp))
}
