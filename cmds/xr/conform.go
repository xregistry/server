package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var depth = 0
var ConfigFile = EnvString("XR_CONFORM_CONFIG", "")
var ShowLogs = EnvBool("XR_SHOWLOGS", false)

func conformFunc(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		Error("No arguments allowed for this command")
	}

	reg, xErr := xrlib.GetRegistry(GetServer())
	Error(xErr)

	if ConfigFile != "" {
		Error(reg.LoadConfigFromFile(ConfigFile))
	}

	td := NewTD(GetServer())
	td.SetRegistry(reg)

	FailFast, _ = cmd.Flags().GetBool("failfast")

	runFunc, _ := cmd.Flags().GetString("run")
	if runFunc == "" {
		td.Run(TestRegistry)
	} else {
		funcs := map[string]TestFn{
			"TestTDAllPass": TestTDAllPass,
			"TestTDDepFail": TestTDDepFail,
			"TestTDMixture": TestTDMixture,
		}
		fn := funcs[runFunc]
		if fn == nil {
			panic(fmt.Sprintf("No function by name: %s", runFunc))
		}
		td.Run(fn)
	}

	// td.Dump("")
	if depth <= 0 {
		// Can't actually do zero, so zero = -1 (all)
		depth = 9999999
	}
	td.Print(os.Stdout, "", ShowLogs, depth-1)

	if td.ExitCode() != 0 {
		os.Exit(td.ExitCode())
	}
}

func addConformCmd(parent *cobra.Command) {
	conformCmd := &cobra.Command{
		Use:     "conform",
		Short:   "xRegistry Conformance Tester",
		Run:     conformFunc,
		GroupID: "Admin",
	}
	conformCmd.Flags().BoolVarP(&ShowLogs, "logs", "l", ShowLogs,
		"Show logs even on success")
	conformCmd.Flags().IntVarP(&depth, "depth", "d", depth, "Console depth")
	conformCmd.Flags().BoolVarP(&tdDebug, "tdDebug", "t", tdDebug, "td debug")
	conformCmd.Flags().Bool("failfast", false, "stop on first failure")
	conformCmd.Flags().StringP("run", "r", "", "run function")

	conformCmd.Flags().MarkHidden("run")
	conformCmd.Flags().MarkHidden("tdDebug")

	parent.AddCommand(conformCmd)
}

func TestTDAllPass(td *TD) {
	td.DependsOn(TestTDInit)
	td.Run(TestTDSimple1)
	td.Pass("Local passing test")
	td.Run(TestTDLevel2)
	td.Run(TestTDLevel3) // dup, should be called in level2
	td.Run(TestTDInit)
	td.Run(TestTDLevel2a)
}

func TestTDDepFail(td *TD) {
	td.DependsOn(TestTDInitFail)
	td.Run(TestTDSimple1)
}

func TestTDMixture(td *TD) {
	td.DependsOn(TestTDInit)
	td.Run(TestTDSimple1)
	td.Fail("Local fail test")
	td.Run(TestTDSimpleFail)
	td.Run(TestTDSimpleSkip)
	td.Run(TestTDSimpleWarn)
	td.Run(TestTDLevel2Fail)
	td.Skip("Top-level-skip")
	td.Run(TestTDLevel23Skip)
	td.Run(TestTDLevel23Fail)
}

func TestTDInit(td *TD)       { td.Pass("Init") }
func TestTDInitFail(td *TD)   { td.Fail("Init") }
func TestTDSimple1(td *TD)    { td.Pass("Simple1") }
func TestTDSimpleFail(td *TD) { td.Fail("SimpleFail") }
func TestTDSimpleSkip(td *TD) { td.Skip("SimpleSkip") }
func TestTDSimpleWarn(td *TD) { td.Warn("SimpleWarn") }

func TestTDLevel2(td *TD) { td.Run(TestTDLevel3) }
func TestTDLevel3(td *TD) {
	td.DependsOn(TestTDInit)
	td.Pass("Level3")
}

func TestTDLevel3Fail(td *TD) { td.Fail("Level3Fail") }

func TestTDLevel23Skip(td *TD) {
	td.DependsOn(TestTDLevel3Skip)
	td.Pass("Level23Skip-2PASS")
}

func TestTDLevel3Skip(td *TD) { td.Skip("Level3skip") }

func TestTDLevel2Fail(td *TD) {
	td.Run(TestTDLevel3)
	td.Pass("Level2Fail")
}

func TestTDLevel23Fail(td *TD) {
	td.Run(TestTDLevel3Fail)
	td.Pass("Level2Pass")
}

func TestTDLevel2a(td *TD) { td.Pass("Level2a") }
