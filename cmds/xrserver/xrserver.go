package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/registry"
)

var GitCommit string
var DBName = "registry"
var RegistryName = "xRegistry"
var Port = 8080
var VerboseCount = 0 // to change, do it as definition of -v flag
var DontCreate = false
var RecreateDB = false
var RecreateReg = false

func ErrStop(err error, args ...any) {
	ErrStopTx(err, nil, args...)
}

func ErrStopTx(err error, tx *registry.Tx, args ...any) {
	if err == nil {
		return
	}
	if len(args) == 0 {
		args = []any{err.Error()}
	}
	StopTx(tx, args...)
}

func Stop(args ...any) {
	StopTx(nil, args...)
}

// runFunc uses this, true means log instead of printf. This is safe as a
// global car becaus we're only running one command at a time. But if we ever
// need to share it across more than one we may need to make it a param.
var UseLogging = false

func StopTx(tx *registry.Tx, args ...any) {
	if tx != nil {
		registry.Must(tx.Rollback())
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
	if VerboseCount == 0 || len(args) == 0 || registry.IsNil(args[0]) {
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
		Use:   "xrserver",
		Short: "xRegistry server",
		Run:   runFunc, // if we add this, add all of runCmd's flags
	}
	serverCmd.Flags().BoolP("verify", "", false, "Verify loading and exit")
	serverCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	serverCmd.Flags().IntVarP(&Port, "port", "p", Port, "Listen port")
	serverCmd.Flags().StringVarP(&DBName, "db", "", DBName, "DB name")
	serverCmd.Flags().BoolVarP(&RecreateDB, "recreatedb", "", RecreateDB,
		"Recreate the DB")
	serverCmd.Flags().BoolVarP(&RecreateReg, "recreatereg", "", RecreateReg,
		"Recreate registry")
	serverCmd.Flags().BoolVarP(&DontCreate, "dontcreate", "", DontCreate,
		"Don't create DB/reg if missing")
	serverCmd.Flags().StringVarP(&RegistryName, "registry", "r", RegistryName,
		"Default Registry name")

	serverCmd.CompletionOptions.HiddenDefaultCmd = true
	serverCmd.PersistentFlags().CountVarP(&VerboseCount, "verbose", "v",
		"Be chatty - can specify multiple (-v=0 to turn off)``")

	serverCmd.Flags().BoolP("help-all", "", false, "Help for all commands")

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run server (the default command)",
		Run:   runFunc,
	}
	runCmd.Flags().BoolP("verify", "", false, "Verify loading and exit")
	runCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	runCmd.Flags().IntVarP(&Port, "port", "p", Port, "Listen port")
	runCmd.Flags().StringVarP(&DBName, "db", "", DBName, "DB name")
	runCmd.Flags().BoolVarP(&RecreateDB, "recreatedb", "", RecreateDB,
		"Recreate the DB")
	runCmd.Flags().BoolVarP(&RecreateReg, "recreatereg", "", RecreateReg,
		"Recreate registry")
	runCmd.Flags().BoolVarP(&DontCreate, "dontcreate", "", DontCreate,
		"Don't create DB/reg if missing")
	runCmd.Flags().StringVarP(&RegistryName, "registry", "r", RegistryName,
		"Default Registry name")

	serverCmd.AddCommand(runCmd)

	addDBCmd(serverCmd)
	addRegistryCmd(serverCmd)

	serverCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		log.SetVerbose(VerboseCount)
		registry.DB_Name = DBName
		registry.GitCommit = GitCommit
	}

	serverCmd.PersistentFlags().BoolP("help", "?", false, "Help for commands")
	serverCmd.SetUsageTemplate(strings.ReplaceAll(serverCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))

	return serverCmd
}

func runFunc(cmd *cobra.Command, args []string) {
	helpAll, _ := cmd.Flags().GetBool("help-all")
	if helpAll {
		fmt.Printf("%s", showAllHelp(cmd, ""))
		os.Exit(0)
	}

	// Turn on timestamps for our Verbose and Error messages.
	UseLogging = true

	registry.PanicIf(GitCommit == "", "GitCommit isn't set")
	Verbose("GitCommit: %.10s", GitCommit)
	Verbose("DB server: %s:%s", registry.DBHOST, registry.DBPORT)

	if tmp := os.Getenv("XR_PORT"); tmp != "" {
		tmpInt, _ := strconv.Atoi(tmp)
		if tmpInt != 0 {
			Port = tmpInt
		}
	}

	if len(args) > 0 {
		Stop("Too many arguments on the command line")
	}

	if RegistryName == "" {
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

	if !registry.DBExists(DBName) && (!DontCreate || RecreateDB) {
		Verbose("Creating DB: %s", DBName)
		err := registry.CreateDB(DBName)
		ErrStop(err, "Error creating DB(%s): %s", DBName, err)
	}

	err := registry.OpenDB(DBName)
	ErrStop(err, "Can't connect to db(%s): %s", DBName, err)

	// Load samples before we look for the default reg because if the default
	// one points to sample, but it's not there, it might try to create it
	if val, _ := cmd.Flags().GetBool("samples"); val {
		paths := os.Getenv("XR_MODEL_PATH")
		os.Setenv("XR_MODEL_PATH", ".:"+paths+
			"http://raw.githubusercontent.com/xregistry/spec/main")

		LoadCESample(nil)
		LoadDirsSample(nil)
		LoadEndpointsSample(nil)
		LoadMessagesSample(nil)
		LoadSchemasSample(nil)
		LoadAPIGuru(nil, "APIs-guru", "openapi-directory")
		LoadDocStore(nil)

		if os.Getenv("XR_LOAD_LARGE") != "" {
			go LoadLargeSample(nil)
		}
	}

	reg, err := registry.FindRegistry(nil, RegistryName)
	ErrStop(err, "Error findng registry(%s): %s", RegistryName, err)

	if reg != nil {
		if RecreateReg {
			Verbose("Deleting xReg: %s", RegistryName)
			ErrStop(reg.Delete())
			reg = nil // force a create below
		}
	}

	if reg == nil && (!DontCreate || RecreateReg) {
		Verbose("Creating xReg: %s", RegistryName)
		reg, err = registry.NewRegistry(nil, RegistryName)
		if err == nil {
			err = reg.Commit()
		}

		ErrStop(err, "Error creating new registry(%s): %s", RegistryName, err)
	}

	Verbose("Default(/): reg-%s", reg.UID)

	if reg == nil {
		Stop("No default registry defined\n")
	}

	if val, _ := cmd.Flags().GetBool("verify"); val {
		Verbose("Done verifying, exiting")
		return
	}

	registry.DefaultRegDbSID = reg.DbSID
	registry.NewServer(Port).Serve()
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
		BufPrintf(res, showAllHelp(cmd, indent)) // indent+"  "))
	}

	return res.String()
}

func main() {
	log.SetVerbose(0)

	if tmp := os.Getenv("XR_VERBOSE"); tmp != "" {
		if tmpInt, err := strconv.Atoi(tmp); err == nil {
			VerboseCount = tmpInt
		}
	}

	serverCmd := setupCmds()

	if err := serverCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
