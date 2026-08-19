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
			"TestTDUtils":   TestTDUtils,
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
	td.Run(TestTDDepDepFail)
	td.Run(TestTDLevel2DepF)
	td.Run(TestTDLevel23Skip)
	td.Run(TestTDDepCacheFail)
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

func TestTDLevel2DepF(td *TD) { td.Run(TestTDLevel3DepF) }
func TestTDLevel3DepF(td *TD) {
	td.DependsOn(TestTDInitFail)
	td.Pass("Level3")
}

func TestTDLevel3Fail(td *TD) { td.Fail("Level3Fail") }

func TestTDLevel23Skip(td *TD) {
	td.DependsOn(TestTDSimpleSkip)
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

func TestTDDepDepFail(td *TD) { td.DependsOn(TestTDLevel3DepF) }

func TestTDDepCacheFail(td *TD) {
	td.DependsOn(TestTDSimpleFail)
	td.Pass("Should never see this")
}

func TestTDUtils(td *TD) {
	reg := td.GetRegistry()
	res, _ := reg.HttpDo(VerboseCount > 2, "GET", "/", nil)

	td.Log("==== PASSing tests ====")
	td.HTTPStatusMustEqual(res, 200, "GET /")
	td.HTTPBodyMustJSON(res, "GET /")

	td.ObjCheck(res.JSON, "specversion", TD_REQUIRED, TD_MUST, TD_EQ, SPECVERSION)
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_NE, "")
	td.ObjCheck(res.JSON, "self", TD_REQUIRED, TD_MUST, TD_NE, "")
	td.ObjCheck(res.JSON, "xid", TD_REQUIRED, TD_MUST, TD_EQ, "/")
	td.ObjCheck(res.JSON, "epoch", TD_REQUIRED, TD_MUST, TD_GE, 0)
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "modifiedat", TD_REQUIRED, TD_MUST, TD_EQ, "ts")

	if reg.Capabilities == nil {
		td.Skip("\"shortself\" capability found")
	} else {
		if reg.Capabilities.ShortSelf {
			td.ObjCheck(res.JSON, "shortself", TD_REQUIRED, TD_MUST, TD_NE, "")
		} else {
			td.ObjCheck(res.JSON, "shortself", TD_MUST, TD_NOT_EXIST)
		}
	}

	// String checks
	// //////////////////////////////////////////////////
	// testing: EQ & must/should/may
	td.Log("==== String tests ====")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_SHOULD, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MAY, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_EQ, "hi") // def=MAY

	// testing: NE
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_NE, "")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_NE, "hi")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_NE, "TestXRConformBasic")

	// testing: LT
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LT, "")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LT, "ZZ")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LT, "TestXRConformBasic")

	// testing: LE
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LE, "")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LE, "ZZ")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_LE, "TestXRConformBasic")

	// testing: GT
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GT, "")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GT, "ZZ")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GT, "TestXRConformBasic")

	// testing: GE
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GE, "")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GE, "ZZ")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_GE, "TestXRConformBasic")

	// testing: EXIST
	td.ObjCheck(res.JSON, "registryid", TD_MUST, TD_EXIST)
	td.ObjCheck(res.JSON, "name", TD_MUST, TD_EXIST)

	// testing: OPTIONAL
	td.ObjCheck(res.JSON, "registryid", TD_OPTIONAL, TD_MUST, TD_NE, "")
	td.ObjCheck(res.JSON, "name", TD_OPTIONAL, TD_MUST, TD_NE, "")

	// testing: REQUIRED
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_NE, "")
	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MUST, TD_NE, "")

	// Timestamp checks
	// //////////////////////////////////////////////////
	td.Log("==== Timestamp tests ====")
	// testing: EQ
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_SHOULD, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MAY, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_EQ, "ts") // def=MAY
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "YYYY-MM-DD")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_SHOULD, TD_EQ, "YYYY-MM-DD")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MAY, TD_EQ, "YYYY-MM-DD")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_EQ, "YYYY-MM-DD")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_SHOULD, TD_EQ, "")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MAY, TD_EQ, "")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_EQ, "")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "2026-01-01T12:00:00Z")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_SHOULD, TD_EQ, "2026-01-01T12:00:00Z")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MAY, TD_EQ, "2026-01-01T12:00:00Z")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_EQ, "2026-01-01T12:00:00Z")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "2026/01/01T12:00:00")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "description", TD_REQUIRED, TD_MUST, TD_EQ, "ts")

	// testing: NE

	// testing: LT

	// testing: LE

	// testing: GT

	// testing: GE

	// testing: EXIST

	// testing: OPTIONAL

	// testing: REQUIRED

	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_SHOULD, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MAY, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_EQ, "hi")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_MUST, TD_EQ, "2026/01/01T12:00:00Z")

	td.ObjCheck(res.JSON, "epoch", TD_REQUIRED, TD_MUST, TD_EQ, 3)

	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MUST, TD_EQ, "")
	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MUST, TD_EXIST)
	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MUST, TD_NOT_EXIST)
	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MAY, TD_EXIST)
	td.ObjCheck(res.JSON, "name", TD_REQUIRED, TD_MAY, TD_NOT_EXIST)
	td.ObjCheck(res.JSON, "name", TD_OPTIONAL, TD_MUST, TD_EXIST)
	td.ObjCheck(res.JSON, "name", TD_OPTIONAL, TD_MUST, TD_NOT_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_REQUIRED, TD_MUST, TD_EQ, "")
	td.ObjCheck(res.JSON, "label.f", TD_REQUIRED, TD_MUST, TD_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_REQUIRED, TD_MUST, TD_NOT_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_REQUIRED, TD_MAY, TD_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_REQUIRED, TD_MAY, TD_NOT_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_OPTIONAL, TD_MUST, TD_EXIST)
	td.ObjCheck(res.JSON, "label.f", TD_OPTIONAL, TD_MUST, TD_NOT_EXIST)

	td.ObjCheck(res.JSON, "createdat", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "modifiedat", TD_REQUIRED, TD_MUST, TD_EQ, "ts")
	td.ObjCheck(res.JSON, "modifiedat", TD_REQUIRED, TD_MUST, TD_EQ, "YYYY-MM-DD")
	td.ObjCheck(res.JSON, "modifiedat", TD_REQUIRED, TD_MUST, TD_NE, "YYYY-MM-DD")
}
