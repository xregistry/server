package main

import (
	"fmt"
	"os"
	"strings"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

var GitComit string
var VerboseFlag = xrlib.EnvBool("XR_VERBOSE", false)
var DebugFlag = xrlib.EnvBool("XR_DEBUG", false)
var Server = "" // Will grab DefaultServer after we add the --server flag
var DefaultServer = xrlib.EnvString("XR_SERVER", "")

func ErrStop(err error, prefix ...any) {
	if err == nil {
		return
	}

	str := err.Error()
	if prefix != nil {
		str = fmt.Sprintf(prefix[0].(string), prefix[1:]...)
	}
	Error(str)
}

func Error(obj any, args ...any) {
	if registry.IsNil(obj) {
		return
	}
	fmtStr, ok := obj.(string)
	if !ok {
		if err, ok := obj.(error); ok {
			if err == nil {
				return
			}
			fmtStr = err.Error()

			if len(args) > 0 {
				fmtStr, ok = args[0].(string)
				if !ok {
					panic("First arg must be a string")
				}
				args = args[1:]
			}
		} else {
			panic(fmt.Sprintf("Unknown Error arg: %q(%T)", obj, obj))
		}
	}

	if fmtStr != "" {
		fmtStr = strings.TrimSpace(fmtStr) + "\n"
		fmt.Fprintf(os.Stderr, fmtStr, args...)
	}
	// registry.ShowStack()
	os.Exit(1)
}

func Verbose(args ...any) {
	if !VerboseFlag || len(args) == 0 || registry.IsNil(args[0]) {
		return
	}

	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	fmt.Fprintf(os.Stderr, fmtStr+"\n", args[1:]...)
}

func main() {
	xrCmd := &cobra.Command{
		Use:   "xr",
		Short: "xRegistry CLI",

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Just make sure Server starts with some variant of "http"
			if Server != "" && !strings.HasPrefix(Server, "http") {
				Server = "http://" + strings.TrimLeft(Server, "/")
			}

			xrlib.DebugFlag = DebugFlag
		},
	}
	xrCmd.CompletionOptions.HiddenDefaultCmd = true
	xrCmd.PersistentFlags().BoolVarP(&VerboseFlag, "verbose", "v", false,
		"Be chatty")
	xrCmd.PersistentFlags().BoolVarP(&DebugFlag, "debug", "x", false,
		"Show HTTP traffic")
	xrCmd.PersistentFlags().StringVarP(&Server, "server", "s", "",
		"Server URL")

	xrCmd.AddGroup(
		&cobra.Group{"Entities", "Data Management:"},
		&cobra.Group{"Admin", "Admin:"})
	xrCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [command]",
		Short:  "Help about any command",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Parent().Help(); err != nil {
				fmt.Println(err)
			}
		},
	})

	// Set Server after we add the --server flag so we don't show the
	// default value in the help text
	Server = DefaultServer

	addModelCmd(xrCmd)
	// addRegistryCmd(xrCmd)
	// addGroupCmd(xrCmd)
	addGetCmd(xrCmd)
	addCreateCmd(xrCmd)
	addUpsertCmd(xrCmd)
	addUpdateCmd(xrCmd)
	addDeleteCmd(xrCmd)
	addImportCmd(xrCmd)
	addDownloadCmd(xrCmd)
	addServeCmd(xrCmd)

	if err := xrCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
