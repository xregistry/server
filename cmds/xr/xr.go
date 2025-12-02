package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	// "github.com/spf13/pflag"
	// "github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var GitComit string
var VerboseCount = 0

var Server = "" // Will grab DefaultServer after we add the --server flag
var DefaultServer = EnvString("XR_SERVER", "localhost:8080")
var ErrJson = false

// string, args      -> Title=sprintf(string, args...)
// xErr              -> use as is
// err               -> Title=err.Error()
// err, string, args -> Title=sprintf(string, args...)
//    if an arg is "err" then replace with err.Error()

func Error(obj any, args ...any) {
	if IsNil(obj) {
		return
	}

	var xErr *XRError

	if str, ok := obj.(string); ok {
		xErr = NewXRError("client_error", "/",
			"error_detail="+fmt.Sprintf(str, args...))
	} else if xErr, ok = obj.(*XRError); ok {
		// Use as is
		PanicIf(len(args) > 0, "Extra args to Error(xErr): %v", args)
	} else if err, ok := obj.(error); ok {
		if len(args) == 0 {
			xErr = NewXRError("client_error", "/",
				"error_detail="+err.Error())
		} else {
			for i := 1; i < len(args); i++ {
				if args[i] == "err" {
					args[i] = err.Error()
				}
			}
			str := args[0].(string)
			xErr = NewXRError("client_error", "/",
				"error_detail="+fmt.Sprintf(str, args[1:]...))
		}
	}

	PanicIf(IsNil(xErr), "xErr is nil")

	var msg string
	if ErrJson {
		msg = xErr.String()
	} else {
		msg = xErr.GetTitle()
		if xErr.Detail != "" {
			msg += ". " + xErr.Detail
		}
	}

	fmt.Fprintf(os.Stderr, "%s\n", msg)

	// ShowStack()
	os.Exit(1)
}

func Verbose(args ...any) {
	// if !VerboseFlag || len(args) == 0 || IsNil(args[0]) {
	if log.GetVerbose() == 0 || len(args) == 0 || IsNil(args[0]) {
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

		if cmd.Parent() == nil {
			BufPrintf(res, "  # Global flags:\n")
		} else {
			BufPrintf(res, "  %s\n", wrap(summary, 78, "  # "))
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
		BufPrintf(res, "%s", showAllHelp(cmd, indent)) // indent+"  "))
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
	xrCmd.PersistentFlags().BoolVarP(&ErrJson, "errjson", "", false,
		"Print errors as json")
	xrCmd.PersistentFlags().BoolP("help", "?", false, "Help for xr")

	xrCmd.AddGroup(
		&cobra.Group{"Entities", "Data Management:"},
		&cobra.Group{"Admin", "Admin:"})

	xrCmd.SetUsageTemplate(strings.ReplaceAll(xrCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))
	xrCmd.SetUsageTemplate(xrCmd.UsageTemplate() + "\nVersion: " +
		GitCommit[:min(len(GitCommit), 12)] + "\n")

	// just so 'help' is in a group and Hidden is adhered to
	xrCmd.SetHelpCommand(&cobra.Command{
		Use:     "help [command]",
		Short:   "I'm not really here",
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
	addImportCmd(xrCmd)
	addModelCmd(xrCmd)
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
