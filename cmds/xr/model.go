package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	// log "github.com/duglin/dlog"
	// "github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/registry"
)

func addModelCmd(parent *cobra.Command) {
	modelCmd := &cobra.Command{
		Use:   "model",
		Short: "model commands",
	}

	modelNormalizeCmd := &cobra.Command{
		Use:   "normalize [ - | FILE... ]",
		Short: "Parse and resolve includes in an xRegistry model document",
		Run:   modelNormalizeFunc,
	}
	modelCmd.AddCommand(modelNormalizeCmd)

	modelVerifyCmd := &cobra.Command{
		Use:   "verify [ - | FILE... ]",
		Short: "Parse and verify xRegistry model document",
		Run:   modelVerifyFunc,
	}
	modelCmd.AddCommand(modelVerifyCmd)

	parent.AddCommand(modelCmd)
}

func modelNormalizeFunc(cmd *cobra.Command, args []string) {
	var err error
	var buf []byte

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
					err = fmt.Errorf("Error getting model: %s\n%s",
						res.Status, string(buf))
				}
			}
		} else {
			buf, err = os.ReadFile(fileName)
		}

		if err != nil {
			Error("Error reading %q: %s", fileName, err)
		}

		buf, err = registry.ProcessIncludes(fileName, buf, true)
		Error(err)

		tmp := map[string]any{}
		Error(registry.Unmarshal(buf, &tmp))
		fmt.Printf("%s\n", registry.ToJSON(tmp))
	}
}

func modelVerifyFunc(cmd *cobra.Command, args []string) {
	var buf []byte
	var err error

	if len(args) == 0 {
		buf, err = io.ReadAll(os.Stdin)
		if err != nil {
			Error("Error reading from stdin: %s", err)
		}
		VerifyModel("", buf)
	}

	for _, fileName := range args {
		Verbose("%s:\n", fileName)
		if strings.HasPrefix(fileName, "http") {
			res, err := http.Get(fileName)
			if err == nil {
				buf, err = io.ReadAll(res.Body)
				res.Body.Close()

				if res.StatusCode/100 != 2 {
					err = fmt.Errorf("Error getting model: %s\n%s",
						res.Status, string(buf))
				}
			}
		} else {
			buf, err = os.ReadFile(fileName)
		}
		if err != nil {
			Error("Error reading %q: %s", fileName, err)
		}
		VerifyModel(fileName, buf)
	}

}

func VerifyModel(fileName string, buf []byte) {
	var err error

	if len(os.Args) > 2 && fileName != "" {
		fileName += ": "
	} else {
		fileName = ""
	}

	buf, err = registry.ProcessIncludes(fileName, buf, true)
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
