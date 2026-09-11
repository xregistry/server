package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	// "github.com/spf13/pflag"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var GitComit string
var VerboseCount = 0
var ShowDebug = false

var ErrJson = false

var XRConfigFileName = ".xr"
var DefaultServer = "localhost:8080"
var XRConfig = NewConfig(XRConfigFileName)

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
		msg = xErr.ToJSON()
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

	if ShowDebug {
		ShowStack()
		fmt.Printf("xErr: %s\n", xErr)
	}

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
	if log.GetLevel() == 0 || len(args) == 0 || IsNil(args[0]) {
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

func GetServer() string {
	return XRConfig.Get("server.url")
}

// GetRawJSON reports whether the "rawjson" .xr config property is set
// (e.g. in the config file, or via --cset rawjson:true). When true,
// commands that would otherwise reorder JSON output into the spec's
// canonical attribute order (see xrlib.CanonicalPrettyReorderTreeOrRaw())
// instead preserve the server's own original attribute order — but
// --min/--nodiff (cmds/xr/download.go) still fully apply either way,
// only the reordering/blank-line-insertion step is skipped. Deliberately
// NOT exposed as its own --rawjson command-line flag (unlike --server) —
// per-invocation overrides should use --cset rawjson:true instead.
func GetRawJSON() bool {
	return XRConfig.GetAsBool("rawjson")
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
			log.AddVerbose(VerboseCount)

			if b, _ := cmd.Flags().GetBool("version"); b {
				fmt.Printf("Version: %s\n", GitCommit[:min(len(GitCommit), 12)])
				os.Exit(0)
			}

			// If config FN=="" then we'll look for it in $HOME
			fn, _ := cmd.Flags().GetString("config")
			Error(XRConfig.Load(fn))

			// Override with --cset flags
			sets, _ := cmd.Flags().GetStringArray("cset")
			for _, set := range sets {
				// Just to be nice
				name, value, ok := strings.Cut(set, "=")
				if !ok {
					name, value, _ = strings.Cut(set, ":")
				}
				XRConfig.Set(name, value)
			}

			// Quick sniff test for alias names
			for key, _ := range XRConfig.Data {
				originKey := key
				if !strings.HasPrefix(key, "server.alias.") {
					continue
				}
				key = strings.TrimSpace(key[13:])
				if len(key) == 0 {
					Error("Bad config name: %q", originKey)
				}
				name, _, _ := strings.Cut(key, ".")
				ok, _ := regexp.MatchString("^[a-zA-Z0-9]+$", name)
				if !ok {
					Error("Bad server.alias: %q", originKey)
				}
			}

			// Load the HTTP headers, if specified
			xrlib.HTTPHeaders = XRConfig.GetHeaders()

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

			// Clean
			server = strings.TrimSpace(server)

			// Look to see if 'server' is an alias
			ok, _ := regexp.MatchString("^[a-zA-Z0-9]+$", server)
			if ok {
				if data, ok := XRConfig.Data["server.alias."+server]; ok {
					prefix := "server.alias." + server + ".header."
					data := strings.TrimSpace(data)
					if len(data) == 0 {
						Error("Missing URL for server.alias.%s in config "+
							"file(%s)", server, XRConfig.FileName)
					}
					server = data

					for key, data := range XRConfig.Data {
						// Doesn't start with prefix
						if !strings.HasPrefix(key, prefix) {
							continue
						}
						key = key[len(prefix):]
						if len(key) == 0 {
							Error("%s is missing a NAME in config file(%s)",
								prefix, XRConfig.FileName)
						}
						if xrlib.HTTPHeaders == nil {
							xrlib.HTTPHeaders = map[string]string{}
						}
						xrlib.HTTPHeaders[key] = data
					}
				}
			}

			// Clean & make sure 'server' starts with some variant of "http"
			if server != "" && !strings.HasPrefix(server, "http") {
				server = "http://" + strings.TrimLeft(server, "/")
			}

			XRConfig.Set("server.url", server)
		},
	}

	// Put "usage" annotation after "Usage" but before "Flags".
	// Don't use Command.Long property.
	cmdTemplate := xrCmd.UsageTemplate()
	i := strings.Index(cmdTemplate, `{{if gt (len .Aliases) 0}}`)
	if i > 0 {
		newText := "{{if index .Annotations \"usage\"}}\n" +
			"{{index .Annotations \"usage\"}}{{end}}"
		cmdTemplate = cmdTemplate[:i] + newText + cmdTemplate[i:]
	}
	xrCmd.SetUsageTemplate(cmdTemplate)

	xrCmd.CompletionOptions.HiddenDefaultCmd = true
	xrCmd.PersistentFlags().StringP("config", "", "",
		"Config file ($HOME/"+XRConfigFileName+")")
	xrCmd.PersistentFlags().StringArray("cset", nil,
		"Override configFile property: --cset NAME[:VALUE]")
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
		Short:   "Use [command] --help instead",
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

	xrCmd.PersistentFlags().BoolVarP(&ShowDebug, "debug", "", false,
		"Show debug info")
	xrCmd.PersistentFlags().MarkHidden("debug")

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
