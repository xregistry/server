package main

import (
	"fmt"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	// "github.com/spf13/pflag"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var GitComit string
var VerboseCount = 0

var ErrJson = false

var DefaultServer = "localhost:8080"
var UserConfig = map[string]string{}
var ConfigFileName = ".xrconfig"

// Error():
// string, args      -> Title=sprintf(string, args...)
// xErr              -> use as is
// err               -> Title=err.Error()
// err, string, args -> Title=sprintf(string, args...)
//    if an arg is "err" then replace with err.Error()

func Error(obj any, args ...any) {
	if !ShowError(obj, args...) {
		return
	}

	os.Exit(1)
}

func ShowError(obj any, args ...any) bool {
	if IsNil(obj) {
		return false // no error
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
			if xErr, ok = (args[0]).(*XRError); ok {
				// Use it
			} else {
				str := args[0].(string)
				xErr = NewXRError("client_error", "/",
					"error_detail="+fmt.Sprintf(str, args[1:]...))
			}
		}
	}

	PanicIf(IsNil(xErr), "xErr is nil")

	var msg string
	if ErrJson {
		msg = xErr.String()
	} else {
		msg = xErr.GetTitle()
		if xErr.Detail != "" {
			if !strings.HasSuffix(msg, ".") {
				msg += "."
			}
			msg += " " + xErr.Detail
		}
	}

	fmt.Fprintf(os.Stderr, "%s\n", msg)

	// ShowStack()
	// fmt.Printf("xErr: %s\n", xErr)

	return true // yes we printed something
}

// Same as Error() but will print the cmd's usage text afterwards
func ErrorUsage(cmd *cobra.Command, obj any, args ...any) {
	if !ShowError(obj, args...) {
		return
	}

	cmd.Usage()

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

// File syntax:
// prop: value
// # comment
func LoadConfigFromFile(fn string) *XRError {
	if fn == "" {
		if _, err := os.Stat("./" + ConfigFileName); err == nil {
			fn = "./" + ConfigFileName
		} else {
			path, _ := os.UserHomeDir()
			if path != "" {
				path = path + "/" + ConfigFileName
				if _, err := os.Stat(path); err == nil {
					fn = path
				} else {
					// No config file, just return
					return nil
				}
			}
		}
	}

	buf, err := os.ReadFile(fn)
	if err != nil {
		return NewXRError("client_error", "",
			"error_detail="+
				fmt.Sprintf("Error loading config file (%s): %s",
					fn, err.Error()))
	}

	return LoadConfigFromBuffer(string(buf))
}

// Buffer syntax:
// prop: value
// # comment
func LoadConfigFromBuffer(buffer string) *XRError {
	lines := strings.Split(buffer, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, _ := strings.Cut(line, ":")
		name = strings.TrimSpace(name)

		if name == "" {
			return NewXRError("client_error", "",
				"error_detail="+
					fmt.Sprintf("Error in config data - no name: %q", line))
		}

		value = strings.TrimSpace(value)
		SetConfig(name, value)
	}

	xrlib.HTTPHeaders = GetHeaders()

	return nil
}

func GetConfig(name string) string {
	if UserConfig == nil {
		return ""
	}
	return UserConfig[name]
}

func GetServer() string {
	return GetConfig("server.url")
}

func GetHeaders() map[string]string {
	headers := map[string]string(nil)

	for key, value := range UserConfig {
		if !strings.HasPrefix(key, "header.") {
			continue
		}
		key = strings.TrimSpace(key[7:])
		if key != "" {
			if headers == nil {
				headers = map[string]string{}
			}
			headers[key] = value
		}
	}
	return headers
}

func SetConfig(name string, value string) *XRError {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	if name == "" {
		return NewXRError("client_error", "",
			"error_detail="+
				fmt.Sprintf("Config name can't be blank"))
	}
	if value == "" {
		delete(UserConfig, name)
	} else {
		if UserConfig == nil {
			UserConfig = map[string]string{}
		}
		UserConfig[name] = value
	}
	return nil
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

func HelpDefBool(b bool) string {
	if b {
		return fmt.Sprintf(" (true*)")
	}
	return ""
}

func ExclusiveFlags(cmd *cobra.Command, flags ...string) {
	full := map[string]bool{}
	used := map[string]bool{}
	for _, flag := range flags {
		full["--"+flag] = true
		if cmd.Flags().Changed(flag) {
			used[flag] = true
		}
	}
	if len(used) <= 1 {
		return
	}
	Error("Only one of '%s' may be specified at a time",
		strings.Join(SortedKeys(full), ","))
}

// Look for --xxx and --no-xxx flags and return the net result.
// If neither are used then return "setIt" as false so we don't do anything
// Return: value, setIt?
func SetBoolFlag(cmd *cobra.Command, flagName string) (bool, bool) {
	if cmd.Flags().Changed("no-" + flagName) {
		// Using --no-xxx=false means the same as --xxx=true
		// So return the opposite of whatever this flag's value is
		value, _ := cmd.Flags().GetBool("no-" + flagName)
		return !value, true
	}

	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetBool(flagName)
		return value, true
	}

	// Neither are set so don't set any underlying attribute at all
	return false, false
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
		Use:          "xr",
		Short:        "xRegistry CLI",
		Run:          mainFunc,
		SilenceUsage: true,

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.SetVerbose(VerboseCount)

			if b, _ := cmd.Flags().GetBool("version"); b {
				fmt.Printf("Version: %s\n", GitCommit[:min(len(GitCommit), 12)])
				os.Exit(0)
			}

			// If config FN=="" then we'll look for it in $HOME
			fn, _ := cmd.Flags().GetString("config")
			Error(LoadConfigFromFile(fn))

			// Calc Server: cmdline->env->configFile->default
			server, _ := cmd.Flags().GetString("server")
			if server == "" {
				server = os.Getenv("XR_SERVER")

				if server == "" {
					server = GetServer()

					if server == "" {
						server = DefaultServer
					}
				}
			}

			// Clean & make sure 'server' starts with some variant of "http"
			server = strings.TrimSpace(server)
			if server != "" && !strings.HasPrefix(server, "http") {
				server = "http://" + strings.TrimLeft(server, "/")
			}

			SetConfig("server.url", server)
		},
	}

	xrCmd.CompletionOptions.HiddenDefaultCmd = true
	xrCmd.PersistentFlags().StringP("config", "", "",
		"Config file ($HOME/.xrconfig)")
	xrCmd.PersistentFlags().StringP("server", "s", "",
		"xRegistry server URL")
	xrCmd.PersistentFlags().BoolVarP(&ErrJson, "errjson", "", false,
		"Print errors as json")
	xrCmd.PersistentFlags().BoolP("help", "?", false, "Help for xr")
	xrCmd.PersistentFlags().CountVarP(&VerboseCount, "verbose", "v",
		"Be chatty``")
	xrCmd.PersistentFlags().BoolP("version", "", false,
		"Print command version string")

	xrCmd.AddGroup(
		&cobra.Group{"Entities", "Data Management:"},
		&cobra.Group{"Admin", "Admin:"})

	xrCmd.SetUsageTemplate(strings.ReplaceAll(xrCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))
	// xrCmd.SetUsageTemplate(xrCmd.UsageTemplate() + "\nVersion: " +
	// GitCommit[:min(len(GitCommit), 12)] + "\n")

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

	ValidateCmd(xrCmd)

	if err := xrCmd.Execute(); err != nil {
		// fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
