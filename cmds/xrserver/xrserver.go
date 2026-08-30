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
)

var defPort = 8080
var defDBHost = "127.0.0.1"
var defDBPort = 3306
var defDBName = "registry"
var defDBUser = "root"
var defDBPassword = "password"
var defRegistryName = "xRegistry"

var DontCreate = false
var RecreateDB = false
var RecreateReg = false
var ConfigFileName = ".xrserver"
var XRServerConfig = NewConfig(ConfigFileName)

func init() {
	XRServerConfig.Set("defaultreg", defRegistryName)
	XRServerConfig.Set("rootapp", "ui")
	XRServerConfig.Set("ui.dir", "")
	XRServerConfig.Set("verbose", "")
	XRServerConfig.Set("http.port", fmt.Sprintf("%d", defPort))
	XRServerConfig.Set("db.name", defDBName)
	XRServerConfig.Set("db.host", defDBHost)
	XRServerConfig.Set("db.port", fmt.Sprintf("%d", defDBPort))
	XRServerConfig.Set("db.user", defDBUser)
	XRServerConfig.Set("db.password", defDBPassword)
	XRServerConfig.Set("path.ui", "ui")
	XRServerConfig.Set("path.defaultreg", "xreg")
	XRServerConfig.Set("path.regcollection", "xregs")
}

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

// runFunc uses this, true means log instead of printf. This is safe as a
// global car becaus we're only running one command at a time. But if we ever
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
	serverCmd.Flags().StringP("registry", "r", defRegistryName,
		"Default Registry name")
	serverCmd.Flag("registry").DefValue = ""
	serverCmd.Flags().IntP("port", "p", defPort,
		fmt.Sprintf("HTTP Listen port (%d*)", defPort))
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
		"Config file ($HOME/"+ConfigFileName+")")
	serverCmd.PersistentFlags().StringArray("set", nil,
		"Override configFile property: --set NAME[:VALUE]")
	serverCmd.PersistentFlags().StringP("db", "", defDBName,
		"DB name ("+defDBName+"*)")
	serverCmd.Flag("db").DefValue = "" // hide default text
	serverCmd.PersistentFlags().StringP("dbhost", "", defDBHost,
		"DB host address ("+defDBHost+"*)")
	serverCmd.Flag("dbhost").DefValue = "" // hide default text
	serverCmd.PersistentFlags().IntP("dbport", "", defDBPort,
		fmt.Sprintf("DB host port (%d*)", defDBPort))
	serverCmd.Flag("dbport").DefValue = "0" // hide default text
	serverCmd.PersistentFlags().StringP("dbuser", "", defDBUser,
		"DB user ("+defDBUser+"*)")
	serverCmd.Flag("dbuser").DefValue = "" // hide default text
	serverCmd.PersistentFlags().StringP("dbpassword", "",
		defDBPassword, "DB password ("+defDBPassword+"*)")
	serverCmd.Flag("dbpassword").DefValue = "" // hide default text
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
	runCmd.Flags().StringP("rootapp", "", "ui", "Root application (ui,xreg)")
	runCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	runCmd.Flags().IntP("port", "p", defPort,
		fmt.Sprintf("HTTP Listen port (%d*)", defPort))
	runCmd.Flag("port").DefValue = "0"
	runCmd.Flags().BoolVarP(&RecreateDB, "recreatedb", "", RecreateDB,
		"Recreate the DB")
	runCmd.Flags().BoolVarP(&RecreateReg, "recreatereg", "", RecreateReg,
		"Recreate registry")
	runCmd.Flags().BoolVarP(&DontCreate, "dontcreate", "", DontCreate,
		"Don't create DB/reg if missing")
	runCmd.Flags().StringP("registry", "r", defRegistryName,
		"Default Registry name("+defRegistryName+"*)")
	runCmd.Flag("registry").DefValue = ""

	serverCmd.AddCommand(runCmd)

	addDBCmd(serverCmd)
	addRegistryCmd(serverCmd)

	serverCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if b, _ := cmd.Flags().GetBool("version"); b {
			fmt.Printf("Version: %s\n", GitCommit[:min(len(GitCommit), 12)])
			os.Exit(0)
		}

		// load .xrserver config file - override fileName from --config
		fn, _ := cmd.Flags().GetString("config")
		ErrStop(XRServerConfig.Load(fn))

		// Override with --set flags
		sets, _ := cmd.Flags().GetStringArray("set")
		for _, set := range sets {
			name, value, ok := strings.Cut(set, ":")
			if !ok {
				// Just to be nice
				name, value, _ = strings.Cut(set, "=")
			}
			XRServerConfig.Set(name, value)
		}

		// Override with env vars
		XRServerConfig.SetFromEnv("db.name", "DBNAME")
		XRServerConfig.SetFromEnv("db.host", "DBHOST")
		XRServerConfig.SetFromEnv("db.port", "DBPORT")
		XRServerConfig.SetFromEnv("db.user", "DBUSER")
		XRServerConfig.SetFromEnv("db.password", "DBPASSWORD")

		//  Override with cmd-line params
		XRServerConfig.SetFromCmd("db.name", cmd, "db")
		XRServerConfig.SetFromCmd("db.host", cmd, "dbhost")
		XRServerConfig.SetFromCmdInt("db.port", cmd, "dbport")
		XRServerConfig.SetFromCmd("db.user", cmd, "dbuser")
		XRServerConfig.SetFromCmd("db.password", cmd, "dbpassword")

		// Set the Registry/DB flags
		registry.DBName = XRServerConfig.Get("db.name")
		registry.DBHost = XRServerConfig.Get("db.host")
		registry.DBPort = XRServerConfig.Get("db.port")
		registry.DBUser = XRServerConfig.Get("db.user")
		registry.DBPassword = XRServerConfig.Get("db.password")

		tmpV := XRServerConfig.GetAsInt("verbose")
		if cmd.Flags().Changed("verbose") {
			tmpV, _ = cmd.Flags().GetCount("verbose")
		}
		log.SetVerbose(tmpV)
	}

	return serverCmd
}

func runFunc(cmd *cobra.Command, args []string) {
	helpAll, _ := cmd.Flags().GetBool("help-all")
	if helpAll {
		fmt.Printf("%s", showAllHelp(cmd, ""))
		os.Exit(0)
	}

	// Override with non-global env vars
	XRServerConfig.SetFromEnv("http.port", "XR_PORT")

	// Override with cmd-line params
	XRServerConfig.SetFromCmd("defaultreg", cmd, "registry")
	XRServerConfig.SetFromCmdInt("http.port", cmd, "port")
	XRServerConfig.SetFromCmd("rootapp", cmd, "rootapp")
	XRServerConfig.SetFromCmd("ui.dir", cmd, "ui-dir")

	// Set the Registry/DB flags
	registry.RootApp = XRServerConfig.Get("rootapp")
	registry.UISegment = XRServerConfig.Get("path.ui")
	registry.DefaultRegSegment = XRServerConfig.Get("path.defaultreg")
	registry.RegCollectionSegment = XRServerConfig.Get("path.regcollection")
	registry.UIDir = XRServerConfig.Get("ui.dir")
	registry.XRUIJSON = XRServerConfig.Get("ui.xrui.json")

	DBName := XRServerConfig.Get("db.name")

	// Turn on timestamps for our Verbose and Error messages.
	// UseLogging = true

	if XRServerConfig.FileName != "" {
		Verbose("Config: %s", XRServerConfig.FileName)
	}

	PanicIf(GitCommit == "" || GitCommit == "<n/a>", "GitCommit isn't set")
	Verbose("GitCommit: %.12s", GitCommit)
	Verbose("DB: %s@%s:%s", DBName, registry.DBHost, registry.DBPort)

	if len(args) > 0 {
		Stop("Too many arguments on the command line")
	}

	if registry.UIDir != "" {
		if _, err := os.Stat(registry.UIDir); err != nil {
			Stop("Error locating UIDir(%s): %s", registry.UIDir,
				errors.Unwrap(err))
		}
		if _, err := os.Stat(registry.UIDir + "/index.html"); err != nil {
			Stop("Error locating UIDir(%s)/index.html: %s", registry.UIDir,
				errors.Unwrap(err))
		}
	}

	regName := XRServerConfig.Get("defaultreg")
	if regName == "" {
		Stop("Default Registry name missing, try: -r NAME")
	}

	if RecreateDB {
		if registry.DBExists(DBName) {
			Verbose("Deleting DB: %s", DBName)
			err := registry.DeleteDB(DBName)
			ErrStop(err, "Error deleting DB(%s): %s", DBName, err)
		}

		// Force us to create the default registry, otherwise we'll die
		// cmd.Flags().Set("createreg", "true")
	}

	if !registry.DBExists(DBName) {
		if !DontCreate || RecreateDB {
			Verbose("Creating DB: %s", DBName)
			err := registry.CreateDB(DBName)
			ErrStop(err, "Error creating DB(%s): %s", DBName, err)
		} else {
			Stop("DB %q does not exist", DBName)
		}
	}

	err := registry.OpenDB(DBName)
	ErrStop(err, "Can't connect to db(%s): %s", DBName, err)

	// Load samples before we look for the default reg because if the default
	// one points to sample, but it's not there, it might try to create it
	if val, _ := cmd.Flags().GetBool("samples"); val {
		// log.Printf("Loading samples")
		paths := os.Getenv("XR_MODEL_PATH")
		os.Setenv("XR_MODEL_PATH", ".:"+paths+
			":http://raw.githubusercontent.com/xregistry/spec/main")

		saveV := log.GetVerbose()
		log.SetVerbose(1) // Hide the HTTP PUTs, etc.

		LoadCESample(nil)
		LoadDirsSample(nil)
		LoadEndpointsSample(nil)
		LoadMessagesSample(nil)
		LoadSchemasSample(nil)
		LoadAPIGuru(nil, "APIs-guru", "openapi-directory")
		LoadDocStore(nil)

		log.SetVerbose(saveV)

		if os.Getenv("XR_LOAD_LARGE") != "" {
			go LoadLargeSample(nil)
		}
	}

	reg, xErr := registry.FindRegistry(nil, regName, registry.FOR_READ)
	ErrStop(xErr, "Error finding registry(%s): %s", regName, xErr)

	if reg != nil {
		if RecreateReg {
			Verbose("Deleting xReg: %s", regName)
			ErrStop(reg.Delete())
			ErrStop(reg.Commit())
			reg = nil // force a create below
		}
	}

	if reg == nil && (!DontCreate || RecreateReg) {
		Verbose("Creating: %s/%s",
			XRServerConfig.Get("path.regcollection"), regName)
		reg, xErr = registry.NewRegistry(nil, regName)
		if IsNil(xErr) {
			xErr = reg.Commit()
		}

		ErrStop(xErr, "Error creating new registry(%s): %s", regName, xErr)
	}

	if reg == nil {
		if regName != "" {
			Stop("Registry %q does not exist", regName)
		}
		Stop("No default registry defined")
	}

	if registry.UIDir != "" {
		Verbose("UI Dir: %s", registry.UIDir)
	}

	if registry.XRUIJSON != "" {
		Verbose("UI xrui.json: %s", registry.XRUIJSON)
	}

	Verbose("Path: /%s -> UI", XRServerConfig.Get("path.ui"))

	Verbose("Path: /%s -> %s/%s",
		XRServerConfig.Get("path.defaultreg"),
		XRServerConfig.Get("path.regcollection"), reg.UID)

	if registry.RootApp != "ui" && registry.RootApp != "xreg" {
		Stop("--root must be either \"ui\" or \"xreg\"")
	}

	if registry.RootApp == "xreg" {
		Verbose("Path: / -> %s/%s",
			XRServerConfig.Get("path.regcollection"), reg.UID)
	} else {
		Verbose("Path: / -> %s", XRServerConfig.Get("path.ui"))
	}

	if val, _ := cmd.Flags().GetBool("verify"); val {
		Verbose("Done verifying, exiting")
		return
	}

	registry.DefaultRegDbSID = reg.DbSID
	registry.NewServer(XRServerConfig.GetAsInt("http.port")).Serve()
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
	log.SetVerbose(0)

	if tmp := os.Getenv("XR_VERBOSE"); tmp != "" {
		log.AddVerboseString(tmp)
	}

	serverCmd := setupCmds()
	ValidateCmd(serverCmd)

	if err := serverCmd.Execute(); err != nil {
		// fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
