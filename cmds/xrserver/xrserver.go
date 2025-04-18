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
var VerboseCount = 2

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
		// Run:   runFunc,  // if we add this, add all of runCmd's flags

	}

	serverCmd.CompletionOptions.HiddenDefaultCmd = true
	serverCmd.PersistentFlags().CountVarP(&VerboseCount, "verbose", "v",
		"Be chatty``")

	runCmd := &cobra.Command{
		Use:   "run [default-registry-name]",
		Short: "Run server",
		Run:   runFunc,
	}
	runCmd.Flags().BoolP("verify", "", false, "Verify loading and exit")
	runCmd.Flags().BoolP("samples", "", false, "Load sample registries")
	runCmd.Flags().IntVarP(&Port, "port", "p", Port, "Listen port")
	runCmd.Flags().StringVarP(&DBName, "db", "", DBName, "DB name")
	runCmd.Flags().BoolP("recreatedb", "", false, "Recreate the DB")
	runCmd.Flags().BoolP("createreg", "", false, "Create registry if missing")

	serverCmd.AddCommand(runCmd)
	serverCmd.AddCommand(createDBCmd())
	serverCmd.AddCommand(createRegistryCmd())

	serverCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		log.SetVerbose(VerboseCount)
		registry.DB_Name = DBName
		registry.GitCommit = GitCommit
	}

	serverCmd.PersistentFlags().BoolP("help", "h", false, "Help for xrserver")
	serverCmd.SetUsageTemplate(strings.ReplaceAll(serverCmd.UsageTemplate(),
		"\"help\"", "\"hide-me\""))

	return serverCmd
}

func runFunc(cmd *cobra.Command, args []string) {
	// Turn on timestamps for our Verbose and Error messages.
	UseLogging = true

	registry.PanicIf(GitCommit == "", "GitCommit isn't set")
	Verbose("GitCommit: %.10s", GitCommit)
	Verbose("DB server: %s:%s", registry.DBHOST, registry.DBPORT)

	if tmp := os.Getenv("PORT"); tmp != "" {
		tmpInt, _ := strconv.Atoi(tmp)
		if tmpInt != 0 {
			Port = tmpInt
		}
	}

	if len(args) > 0 {
		if len(args) > 1 {
			Stop("Too many arguments on the command line")
		}
		RegistryName = args[0]
	}

	if val, _ := cmd.Flags().GetBool("recreatedb"); val {
		if registry.DBExists(DBName) {
			Verbose("Deleting DB: %s", DBName)
			err := registry.DeleteDB(DBName)
			ErrStop(err, "Error deleting DB(%s): %s", DBName, err)
		}

		// Force us to create the default registry, otherwise we'll die
		cmd.Flags().Set("createreg", "true")
	}

	if !registry.DBExists(DBName) {
		Verbose("Creating DB: %s", DBName)
		err := registry.CreateDB(DBName)
		ErrStop(err, "Error creating DB(%s): %s", DBName, err)
	}

	err := registry.OpenDB(DBName)
	ErrStop(err, "Can't connect to db(%s): %s", DBName, err)

	// Load samples before we look for the default reg because if the default
	// on points to sample, but it's not there, it might try to create it
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

	if reg == nil {
		if val, _ := cmd.Flags().GetBool("createreg"); !val {
			Stop("Can't find registry: %s", RegistryName)
		}

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
