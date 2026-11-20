package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
	_ "github.com/xregistry/server/registry/formats"
	xrhttp "github.com/xregistry/server/registry/http"
	_ "github.com/xregistry/server/registry/memory"
	_ "github.com/xregistry/server/registry/mysql"
)

var DontCreate = false
var RecreateDB = false
var RecreateReg = false
var ConfigFileName = ".xrserver"
var XRSConfig = registry.NewXRServerConfig(ConfigFileName)

func ErrStop(errAny any, args ...any) {
	ErrStopTx(errAny, nil, args...)
}

func ErrStopTx(errAny any, tx *registry.Tx, args ...any) {
	if IsNil(errAny) {
		return
	}
	if len(args) == 0 {
		args = []any{fmt.Sprintf("%s", errAny)}
	}
	StopTx(tx, args...)
}

func Stop(args ...any) {
	StopTx(nil, args...)
}

// runFunc uses this, 'true' means log instead of printf. This is safe as a
// global var becaus we're only running one command at a time. But if we ever
// need to share it across more than one we may need to make it a param.
var UseLogging = true

func StopTx(tx *registry.Tx, args ...any) {
	if tx != nil {
		Must(tx.Rollback())
	}
	if len(args) > 0 {
		fmtStr := args[0].(string)
		if len(fmtStr) > 0 && fmtStr[len(fmtStr)-1] != '\n' {
			fmtStr += "\n"
		}
		if UseLogging {
			log.Printf(fmtStr, args[1:]...)
		} else {
			fmt.Fprintf(os.Stderr, fmtStr, args[1:]...)
		}
	}
	os.Exit(1)
}

func Verbose(args ...any) {
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

	if len(fmtStr) > 0 && fmtStr[len(fmtStr)-1] != '\n' {
		fmtStr += "\n"
	}

	if UseLogging {
		log.Printf(fmtStr, args[1:]...)
	} else {
		fmt.Fprintf(os.Stderr, fmtStr, args[1:]...)
	}
}

func setupCmds() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:          "xrserver",
		Short:        "xRegistry server",
		Run:          runFunc, // if we add this, add all of runCmd's flags
		SilenceUsage: true,
	}

	// xrserver & xrserver run flags
	serverCmd.Flags().StringP("registry", "r", registry.DefRegistryName,
		"Default Registry name")
	serverCmd.Flag("registry").DefValue = ""
	serverCmd.Flags().StringP("addr", "", registry.DefAddr,
		fmt.Sprintf("HTTP listen address (%q*)", registry.DefAddr))
	serverCmd.Flags().IntP("port", "p", registry.DefPort,
		fmt.Sprintf("HTTP Listen port (%d*)", registry.DefPort))
	serverCmd.Flag("port").DefValue = "0"
	serverCmd.Flags().StringP("rootapp", "", "ui", "Root application (ui,xreg)")

	serverCmd.Flags().BoolP("verify", "", false, "Verify loading and exit")
	serverCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	serverCmd.Flags().BoolVarP(&RecreateDB, "recreatedb", "", RecreateDB,
		"Recreate the DB")
	serverCmd.Flags().BoolVarP(&RecreateReg, "recreatereg", "", RecreateReg,
		"Recreate registry")
	serverCmd.Flags().BoolVarP(&DontCreate, "dontcreate", "", DontCreate,
		"Don't create DB/reg if missing")
	serverCmd.Flags().StringP("ui-dir", "", "",
		"Serve new UI from this directory (dev mode)")
	serverCmd.Flags().BoolP("help-all", "", false, "Help for all commands")

	// global flags
	serverCmd.CompletionOptions.HiddenDefaultCmd = true
	serverCmd.PersistentFlags().StringP("config", "", "",
		"Config file ($HOME/"+ConfigFileName+"*)")
	serverCmd.PersistentFlags().StringArray("cset", nil,
		"Override configFile property: --cset NAME[:VALUE]")
	serverCmd.PersistentFlags().StringArray("backend", nil,
		"Backend and its options: [NAME,][OPT=VALUE,]...")
	serverCmd.PersistentFlags().CountP("verbose", "v",
		"Be chatty``")
	serverCmd.PersistentFlags().BoolP("version", "", false,
		"Print command version string")

	serverCmd.PersistentFlags().BoolP("help", "?", false, "Help for commands")
	serverCmd.SetUsageTemplate(strings.ReplaceAll(serverCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))
	// serverCmd.SetUsageTemplate(serverCmd.UsageTemplate() + "\nVersion: " +
	// GitCommit[:min(len(GitCommit), 12)] + "\n")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run server (the default command)",
		Run:   runFunc,
	}

	runCmd.Flags().BoolP("verify", "", false, "Verify loading and exit")
	runCmd.Flags().StringP("rootapp", "", "ui", "Root application (ui*,xreg)")

	runCmd.Flag("rootapp").DefValue = "" // hide default text

	runCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	runCmd.Flags().StringP("addr", "", registry.DefAddr,
		fmt.Sprintf("HTTP Listen address (%q*)", registry.DefAddr))
	runCmd.Flags().IntP("port", "p", registry.DefPort,
		fmt.Sprintf("HTTP Listen port (%d*)", registry.DefPort))
	runCmd.Flag("port").DefValue = "0"
	runCmd.Flags().BoolVarP(&RecreateDB, "recreatedb", "", RecreateDB,
		"Recreate the DB")
	runCmd.Flags().BoolVarP(&RecreateReg, "recreatereg", "", RecreateReg,
		"Recreate registry")
	runCmd.Flags().BoolVarP(&DontCreate, "dontcreate", "", DontCreate,
		"Don't create DB/reg if missing")
	runCmd.Flags().StringP("registry", "r", registry.DefRegistryName,
		"Default Registry name("+registry.DefRegistryName+"*)")
	runCmd.Flag("registry").DefValue = ""

	serverCmd.AddCommand(runCmd)

	addDBCmd(serverCmd)
	addBackendCmd(serverCmd)
	addRegistryCmd(serverCmd)

	serverCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if b, _ := cmd.Flags().GetBool("version"); b {
			fmt.Printf("Version: %s\n", GitCommit[:min(len(GitCommit), 12)])
			os.Exit(0)
		}

		// load .xrserver config file - override fileName from --config
		fn, _ := cmd.Flags().GetString("config")
		ErrStop(XRSConfig.Load(fn))

		// Override with --cset flags
		sets, _ := cmd.Flags().GetStringArray("cset")
		for _, set := range sets {
			name, value, ok := strings.Cut(set, ":")
			if !ok {
				// Just to be nice
				name, value, _ = strings.Cut(set, "=")
			}
			XRSConfig.Set(name, value)
		}

		// Override with env vars
		_, xErr := registry.SetXRServerConfigFromEnvVars(XRSConfig)
		if xErr != nil {
			Stop("Error processing XR_BACKEND: %s", xErr.Detail)
		}

		//  Override with cmd-line params
		specs, _ := cmd.Flags().GetStringArray("backend")
		for _, spec := range specs {
			xErr := registry.ParseBackendSpec(XRSConfig, spec)
			if xErr != nil {
				Stop("Error processing --backend: %s", xErr.Detail)
			}
		}

		// "backend list" is allowed to run even if the config is bad
		if cmd.Parent() == nil || cmd.Parent().Name() != "backend" {
			xErr = registry.ValidateBackendConfig(XRSConfig)
			if xErr != nil {
				Stop("Backend config error: %s", xErr.Detail)
			}
		}

		tmpV := XRSConfig.GetAsInt("verbose")
		if cmd.Flags().Changed("verbose") {
			tmpV, _ = cmd.Flags().GetCount("verbose")
		}
		log.AddVerbose(tmpV)
	}

	return serverCmd
}

func runFunc(cmd *cobra.Command, args []string) {
	helpAll, _ := cmd.Flags().GetBool("help-all")
	if helpAll {
		fmt.Printf("%s", showAllHelp(cmd, ""))
		os.Exit(0)
	}

	// Override with cmd-line params
	XRSConfig.SetFromCmd("defaultreg", cmd, "registry")
	XRSConfig.SetFromCmd("http.addr", cmd, "addr")
	XRSConfig.SetFromCmdInt("http.port", cmd, "port")
	XRSConfig.SetFromCmd("rootapp", cmd, "rootapp")
	XRSConfig.SetFromCmd("ui.dir", cmd, "ui-dir")

	// Turn on timestamps for our Verbose and Error messages.
	// UseLogging = true

	if XRSConfig.FileName != "" {
		Verbose("Config: %s", XRSConfig.FileName)
	}

	DBName := registry.BackendOpt(XRSConfig, "dbname") // "" if no DBs
	ConnStr, xErr := registry.DBConnectionString(XRSConfig)
	ErrStop(xErr, "Error talking to DB %q: %s", DBName, xErr.String())
	if ConnStr != "" {
		ConnStr = "(" + ConnStr + ")"
	}

	PanicIf(GitCommit == "" || GitCommit == "<n/a>", "GitCommit isn't set")
	Verbose("GitCommit: %.12s", GitCommit)
	Verbose("Backend: %s%s", registry.BackendName(XRSConfig), ConnStr)

	if len(args) > 0 {
		Stop("Too many arguments on the command line")
	}

	uiDir := XRSConfig.GetAsString("ui.dir")
	if uiDir != "" {
		if _, err := os.Stat(uiDir); err != nil {
			Stop("Error locating UIDir(%s): %s", uiDir, errors.Unwrap(err))
		}
		if _, err := os.Stat(uiDir + "/index.html"); err != nil {
			Stop("Error locating UIDir(%s)/index.html: %s", uiDir,
				errors.Unwrap(err))
		}
	}

	regName := XRSConfig.GetAsString("defaultreg")
	if regName == "" {
		Stop("Default Registry name missing, try: -r NAME")
	}

	// Backends w/o DBs silently ignore --recreatedb and --dontcreate (for DBs)
	supportsDBs := registry.BackendSupportsDBs(XRSConfig)

	// If --verify is there then we want to see the error and stop
	if val, _ := cmd.Flags().GetBool("verify"); !val {
		xErr := registry.WaitForBackend(XRSConfig)
		ErrStop(xErr, "Error waiting for backend: %s", xErr)
	}

	if supportsDBs {
		if RecreateDB {
			exists, xErr := registry.DBExists(XRSConfig, DBName)
			ErrStop(xErr, "Error looking for DB %q: %s", DBName, xErr)

			if exists {
				Verbose("Deleting DB: %s", DBName)
				err := registry.DBDelete(XRSConfig, DBName)
				ErrStop(err, "Error deleting DB %q: %s", DBName, err)
			}

			// Force us to create the default registry, otherwise we'll die
			// cmd.Flags().Set("createreg", "true")
		}

		exists, xErr := registry.DBExists(XRSConfig, DBName)
		ErrStop(xErr, "Error looking for DB %q: %s", DBName, xErr)

		if !exists {
			if !DontCreate || RecreateDB {
				Verbose("Creating DB: %s", DBName)
				err := registry.DBCreate(XRSConfig, DBName)
				ErrStop(err, "Error creating DB %q: %s", DBName, err)
			} else {
				Stop("DB %q does not exist", DBName)
			}
		}
	}

	// Load samples before we look for the default reg because if the default
	// one points to sample, but it's not there, it might try to create it
	if val, _ := cmd.Flags().GetBool("samples"); val {
		// log.Printf("Loading samples")
		paths := os.Getenv("XR_MODEL_PATH")
		os.Setenv("XR_MODEL_PATH", ".:"+paths+
			":http://raw.githubusercontent.com/xregistry/spec/main")

		saveV := log.Clone()
		log.AddVerbose(1) // Hide the HTTP PUTs, etc.

		LoadCESample(nil)
		LoadDirsSample(nil)
		LoadEndpointsSample(nil)
		LoadMessagesSample(nil)
		LoadSchemasSample(nil)
		LoadAPIGuru(nil, "APIs-guru", "openapi-directory")
		LoadDocStore(nil)

		log.Reset(saveV)

		if os.Getenv("XR_LOAD_LARGE") != "" {
			go LoadLargeSample(nil)
		}
	}

	reg, xErr := registry.FindRegistryByUID(nil, XRSConfig, regName, FOR_READ)
	ErrStop(xErr, "Error finding registry %q: %s", regName, xErr)

	if reg != nil {
		if RecreateReg {
			Verbose("Deleting xReg: %s", regName)
			ErrStop(reg.Delete())
			ErrStop(reg.Commit())
			reg = nil // force a create below
		}
	}

	if reg == nil && (!DontCreate || RecreateReg) {
		Verbose("Creating: %s/%s", XRSConfig.Get("path.regcollection"), regName)

		reg, xErr = registry.NewRegistry(nil, XRSConfig, regName)
		if IsNil(xErr) {
			xErr = reg.Commit()
		}

		ErrStop(xErr, "Error creating new registry %q: %s", regName, xErr)
	}

	if reg == nil {
		if regName != "" {
			Stop("Registry %q does not exist", regName)
		}
		Stop("No default registry defined")
	}

	if uiDir != "" {
		Verbose("UI Dir: %s", uiDir)
	}

	xrUIJSON := XRSConfig.GetAsString("ui.xrui.json")
	if xrUIJSON != "" {
		Verbose("UI xrui.json: %s", xrUIJSON)
	}

	Verbose("Path: /%s -> UI", XRSConfig.Get("path.ui"))

	Verbose("Path: /%s -> %s/%s",
		XRSConfig.Get("path.defaultreg"),
		XRSConfig.Get("path.regcollection"), reg.UID)

	rootApp := XRSConfig.GetAsString("rootapp")
	if rootApp != "ui" && rootApp != "xreg" {
		Stop("--root must be either \"ui\" or \"xreg\"")
	}

	if rootApp == "xreg" {
		Verbose("Path: / -> %s/%s",
			XRSConfig.Get("path.regcollection"), reg.UID)
	} else {
		Verbose("Path: / -> %s", XRSConfig.Get("path.ui"))
	}

	if val, _ := cmd.Flags().GetBool("verify"); val {
		Verbose("Done verifying, exiting")
		return
	}

	XRSConfig.Set("DefaultRegistrySID", reg.SID)
	xrhttp.NewServer(XRSConfig).Serve()
}

func BufPrintf(buf *strings.Builder, fmtStr string, args ...any) {
	str := fmt.Sprintf(fmtStr, args...)
	buf.WriteString(str)
}

func BufPrint(buf *strings.Builder, fmtStr string) {
	str := fmt.Sprint(fmtStr)
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

		BufPrint(res, showAllHelp(cmd, indent)) // indent+"  "))
	}

	return res.String()
}

func main() {
	if tmp := os.Getenv("XR_VERBOSE"); tmp != "" {
		log.AddVerbose(tmp)
	}

	serverCmd := setupCmds()
	ValidateCmd(serverCmd)

	if err := serverCmd.Execute(); err != nil {
		// fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
