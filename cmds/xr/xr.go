package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	// "github.com/spf13/pflag"
	"github.com/xregistry/server/cmds/xr/xrlib"
	"github.com/xregistry/server/registry"
)

var GitComit string
var VerboseCount = 0

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
	// if !VerboseFlag || len(args) == 0 || registry.IsNil(args[0]) {
	if log.GetVerbose() == 0 || len(args) == 0 || registry.IsNil(args[0]) {
		return
	}

	fmtStr := ""
	ok := false

	if fmtStr, ok = args[0].(string); ok {
		// fmtStr already set
	} else {
		fmtStr = fmt.Sprintf("%v", args[0])
	}

	if fmtStr != "" {
		fmtStr = strings.TrimSpace(fmtStr) + "\n"
	}

	fmt.Fprintf(os.Stderr, fmtStr, args[1:]...)
}

func mainFunc(cmd *cobra.Command, args []string) {
	helpAll, _ := cmd.Flags().GetBool("help-all")
	if helpAll == false {
		cmd.Help()
	} else {
		fmt.Printf("%s", showAllHelp(cmd, ""))
	}
}

func BufPrintf(buf *strings.Builder, fmtStr string, args ...any) {
	str := fmt.Sprintf(fmtStr, args...)
	buf.WriteString(str)
}

func wrap(str string, col int, indent string) string {
	res := ""

	for chop := col; chop > 0; chop-- {
		if chop >= len(str) || str[chop] == ' ' || chop == 1 {
			if chop >= len(str) {
				chop = len(str)
			} else if str[chop] != ' ' {
				chop = col
			}
			if res != "" {
				res += "\n" + indent
			}
			res += strings.TrimRight(str[:chop], " ")
			str = strings.TrimLeft(str[chop:], " ")
			if len(str) == 0 {
				break
			}
			chop = col + 1 - len(indent)
		}
	}
	return res
}

func showAllHelp(cmd *cobra.Command, indent string) string {
	res := &strings.Builder{}

	childCmdStr := ""
	if len(cmd.Commands()) > 0 {
		childCmdStr = " [command]"
	}
	summary := cmd.Short
	if summary != "" {
		summary = "# " + summary
	}

	parents := ""
	for p := cmd.Parent(); p != nil; p = p.Parent() {
		parents = p.Name() + " " + parents
	}

	usages := cmd.Flags().FlagUsagesWrapped(80 - len(indent))

	// only show this command it if has flags or is runnable
	if len(usages) != 0 || cmd.Runnable() {
		line := fmt.Sprintf("%s%s%s", parents, cmd.Use, childCmdStr)
		if cmd.Parent() != nil {
			BufPrintf(res, "\n")
		}
		BufPrintf(res, "%s\n", line)
		BufPrintf(res, "  %s\n", wrap(summary, 78, "  # "))

		if cmd.Parent() == nil {
			BufPrintf(res, "  # Global flags:\n")
		}
	}

	if len(usages) > 0 {
		for _, line := range strings.Split(usages, "\n") {
			if len(line) == 0 {
				continue
			}
			BufPrintf(res, "%s%s\n", indent, line)
		}
	}

	for _, cmd := range cmd.Commands() {
		if cmd.Hidden {
			continue
		}
		BufPrintf(res, showAllHelp(cmd, indent)) // indent+"  "))
	}

	return res.String()
}

func main() {
	xrCmd := &cobra.Command{
		Use:   "xr",
		Short: "xRegistry CLI",
		Run:   mainFunc,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Just make sure Server starts with some variant of "http"
			if Server != "" && !strings.HasPrefix(Server, "http") {
				Server = "http://" + strings.TrimLeft(Server, "/")
			}

			log.SetVerbose(VerboseCount)
		},
	}
	xrCmd.CompletionOptions.HiddenDefaultCmd = true
	xrCmd.PersistentFlags().CountVarP(&VerboseCount, "verbose", "v",
		"Be chatty``")
	xrCmd.PersistentFlags().StringVarP(&Server, "server", "s", "",
		"xRegistry server URL")
	xrCmd.PersistentFlags().BoolP("help", "?", false, "Help for xr")

	xrCmd.AddGroup(
		&cobra.Group{"Entities", "Data Management:"},
		&cobra.Group{"Admin", "Admin:"})

	xrCmd.SetUsageTemplate(strings.ReplaceAll(xrCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))

	// just so 'help' is in a group and Hidden is adhered to
	xrCmd.SetHelpCommand(&cobra.Command{
		Use:     "help [command]",
		Short:   "Help about any command",
		Hidden:  true,
		GroupID: "Admin",
		/*
			Run: func(cmd *cobra.Command, args []string) {
				if err := cmd.Parent().Help(); err != nil {
					fmt.Println(err)
				}
			},
		*/
	})

	xrCmd.Flags().BoolP("help-all", "", false, "Help for all commands")

	// Set Server after we add the --server flag so we don't show the
	// default value in the help text
	Server = DefaultServer

	addCreateCmd(xrCmd)
	addDeleteCmd(xrCmd)
	addGetCmd(xrCmd)
	// addGroupCmd(xrCmd)
	addImportCmd(xrCmd)
	addModelCmd(xrCmd)
	// addRegistryCmd(xrCmd)
	addSetCmd(xrCmd)
	addUpdateCmd(xrCmd)
	addUpsertCmd(xrCmd)

	addDownloadCmd(xrCmd)
	addServeCmd(xrCmd)
	addConformCmd(xrCmd)

	if err := xrCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
