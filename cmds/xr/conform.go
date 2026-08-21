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

	td := NewTD(nil, GetServer())
	td.SetRegistry(reg)

	FailFast, _ = cmd.Flags().GetBool("failfast")
	NoWrap, _ := cmd.Flags().GetBool("nowrap")
	if NoWrap {
		WrapAt = 0
	}

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
	conformCmd.Flags().Bool("failfast", false, "Stop on first failure")
	conformCmd.Flags().StringP("run", "r", "", "Run function")
	conformCmd.Flags().BoolP("nowrap", "", false, "Don't wrap output")

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

	td.ObjReqMustEq(res.JSON, "specversion", SPECVERSION)
	td.ObjReqMustNe(res.JSON, "registryid", "")
	td.ObjReqMustNe(res.JSON, "self", "")
	td.ObjReqMustEq(res.JSON, "xid", "/")
	td.ObjReqMustGe(res.JSON, "epoch", 0)

	td.ObjReqMustEq(res.JSON, "createdat", "ts")
	td.ObjReqMustEq(res.JSON, "modifiedat", "ts")

	if reg.Capabilities == nil {
		td.Skip(`"shortself" capability not enabled`)
	} else {
		if reg.Capabilities.ShortSelf {
			td.ObjReqMustNe(res.JSON, "shortself", "")
		} else {
			td.ObjMustNotExist(res.JSON, "shortself")
		}
	}

	// Bad type checks - always fail, not warn
	// //////////////////////////////////////////////////
	td.Log("==== Bad type tests ====")
	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "registryid", 5) // Fail: bad type
	td.Expect(FAIL)
	td.ObjReqShouldEq(res.JSON, "registryid", 5) // Fail: bad type
	td.Expect(FAIL)
	td.ObjReqMayEq(res.JSON, "registryid", 5) // Fail: bad type

	// String checks
	// //////////////////////////////////////////////////
	td.Log("==== String tests ====")
	// testing: RE
	td.ObjReqMustRe(res.JSON, "registryid", ".*")
	td.Expect(FAIL)
	td.ObjReqMustRe(res.JSON, "registryid", ".*zBOGUSz*")
	td.ObjReqShouldRe(res.JSON, "registryid", ".*")
	td.Expect(WARN)
	td.ObjReqShouldRe(res.JSON, "registryid", ".*zBOGUSz*")
	td.ObjReqMayRe(res.JSON, "registryid", ".*")
	td.ObjReqMayRe(res.JSON, "registryid", ".*zBOGUSz*")

	// testing: EQ & must/should/may
	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "registryid", "hi") // Fail
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "registryid", "hi") // Warn
	td.ObjReqMayEq(res.JSON, "registryid", "hi")
	td.Expect(FAIL)
	td.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_EQ, "hi") // def=MUST

	// testing: NE
	td.ObjReqMustNe(res.JSON, "registryid", "")
	td.ObjReqMustNe(res.JSON, "registryid", "hi")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: LT
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "registryid", "") // Fail
	td.ObjReqMustLt(res.JSON, "registryid", "ZZ")
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: LE
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "registryid", "") // Fail
	td.ObjReqMustLe(res.JSON, "registryid", "ZZ")
	td.ObjReqMustLe(res.JSON, "registryid", "TestXRConformBasic")

	// testing: GT
	td.ObjReqMustGt(res.JSON, "registryid", "")
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "registryid", "ZZ") // Fail
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: GE
	td.ObjReqMustGe(res.JSON, "registryid", "")
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "registryid", "ZZ") // Fail
	td.ObjReqMustGe(res.JSON, "registryid", "TestXRConformBasic")

	// testing: EXIST
	td.ObjCheck(res.JSON, "registryid", TD_MUST, TD_EXIST)
	td.Expect(FAIL)
	td.ObjCheck(res.JSON, "name", TD_MUST, TD_EXIST) // Fail: must exist
	td.Expect(FAIL)
	td.ObjCheck(res.JSON, "registryid", TD_EXIST, 5) // Fail: bad type

	// testing: OPTIONAL
	td.ObjOptMustEq(res.JSON, "registryid", "TestXRConformBasic")
	td.Expect(FAIL)
	td.ObjOptMustEq(res.JSON, "registryid", "")
	td.ObjOptMustNe(res.JSON, "registryid", "")
	td.Expect(FAIL)
	td.ObjOptMustNe(res.JSON, "registryid", "TestXRConformBasic")
	td.ObjOptMustNe(res.JSON, "name", "")

	// testing: REQUIRED
	td.ObjReqMustNe(res.JSON, "registryid", "")
	td.ObjReqMustEq(res.JSON, "registryid", "TestXRConformBasic")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "name", "") // Fail: missing

	// Timestamp checks
	// //////////////////////////////////////////////////
	td.Log("==== Timestamp tests ====")
	// testing: EQ
	td.ObjReqMustEq(res.JSON, "createdat", "ts")
	td.ObjReqShouldEq(res.JSON, "createdat", "ts")
	td.ObjReqMayEq(res.JSON, "createdat", "ts")

	td.ObjReqMustEq(res.JSON, "createdat", "YYYY-MM-DD")
	td.ObjReqShouldEq(res.JSON, "createdat", "YYYY-MM-DD")
	td.ObjReqMayEq(res.JSON, "createdat", "YYYY-MM-DD")

	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "createdat", "")
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "createdat", "")
	td.ObjReqMayEq(res.JSON, "createdat", "")

	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.ObjReqMayEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")

	td.Expect(FAIL) // exp is a string, not a ts
	td.ObjReqMustEq(res.JSON, "createdat", "2026/01/01T12:00:00")
	td.Expect(FAIL) // registryid isn't a TS
	td.ObjReqMustEq(res.JSON, "registryid", "ts")
	td.Expect(FAIL) // optional description isn't a TS
	td.ObjReqMustEq(res.JSON, "description", "ts")

	// testing: NE
	td.ObjReqMustNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "createdat", "ts")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "createdat", "YYYY-MM-DD")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "foo", "YYYY-MM-DD")

	td.ObjReqShouldNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "createdat", "ts")
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "createdat", "YYYY-MM-DD")
	td.Expect(FAIL)
	td.ObjReqShouldNe(res.JSON, "foo", "YYYY-MM-DD")

	td.ObjReqMayNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.ObjReqMayNe(res.JSON, "createdat", res.JSON["createdat"])
	td.ObjReqMayNe(res.JSON, "createdat", "ts")
	td.ObjReqMayNe(res.JSON, "createdat", "YYYY-MM-DD")
	td.Expect(FAIL)
	td.ObjReqMayNe(res.JSON, "foo", "YYYY-MM-DD")

	// testing: LT
	td.ObjReqMustLt(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "createdat", "ts")
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: LE
	td.ObjReqMustLe(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.ObjReqMustLe(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "createdat", "ts")
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: GT
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	td.ObjReqMustGt(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "createdat", "ts")
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: GE
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	td.ObjReqMustGe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.ObjReqMustGe(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "createdat", "ts")
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: OPTIONAL
	td.ObjOptMustEq(res.JSON, "createdat", "ts")
	td.ObjOptMustEq(res.JSON, "createdat", res.JSON["createdat"])
	td.Expect(FAIL)
	td.ObjOptMustEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	td.Expect(FAIL)
	td.ObjOptMustNe(res.JSON, "createdat", res.JSON["createdat"])
	td.ObjOptMustNe(res.JSON, "foo", "ts")

	// Int checks
	// //////////////////////////////////////////////////
	td.Log("==== Int tests ====")
	// testing: EQ
	td.ObjReqMustEq(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqShouldEq(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMayEq(res.JSON, "epoch", res.JSON["epoch"])

	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "epoch", 0)
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "epoch", 0)
	td.ObjReqMayEq(res.JSON, "epoch", 0)

	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "epoch", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "epoch", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "xid", 1)

	// testing: NE
	td.ObjReqMustNe(res.JSON, "epoch", 0)
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "epoch", res.JSON["epoch"])

	td.ObjReqShouldNe(res.JSON, "epoch", 0)
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "epoch", res.JSON["epoch"])

	td.ObjReqMayNe(res.JSON, "epoch", 0)
	td.ObjReqMayNe(res.JSON, "epoch", res.JSON["epoch"])

	// testing: LT
	td.ObjReqMustLt(res.JSON, "epoch", 100000)
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "epoch", 0)

	td.ObjReqShouldLt(res.JSON, "epoch", 100000)
	td.Expect(WARN)
	td.ObjReqShouldLt(res.JSON, "epoch", 0)

	td.ObjReqMayLt(res.JSON, "epoch", 100000)
	td.ObjReqMayLt(res.JSON, "epoch", 0)

	// testing: LE
	td.ObjReqMustLe(res.JSON, "epoch", 100000)
	td.ObjReqMustLe(res.JSON, "epoch", res.JSON["epoch"])
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "epoch", 0)

	td.ObjReqShouldLe(res.JSON, "epoch", 100000)
	td.ObjReqShouldLe(res.JSON, "epoch", res.JSON["epoch"])
	td.Expect(WARN)
	td.ObjReqShouldLe(res.JSON, "epoch", 0)

	td.ObjReqMayLe(res.JSON, "epoch", 100000)
	td.ObjReqMayLe(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMayLe(res.JSON, "epoch", 0)

	// testing: GT
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "epoch", 10000)
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMustGt(res.JSON, "epoch", 0)

	td.Expect(WARN)
	td.ObjReqShouldGt(res.JSON, "epoch", 100000)
	td.Expect(WARN)
	td.ObjReqShouldGt(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqShouldGt(res.JSON, "epoch", 0)

	td.ObjReqMayGt(res.JSON, "epoch", 100000)
	td.ObjReqMayGt(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMayGt(res.JSON, "epoch", 0)

	// testing: GE
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "epoch", 10000)
	td.ObjReqMustGe(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMustGe(res.JSON, "epoch", 0)

	td.Expect(WARN)
	td.ObjReqShouldGe(res.JSON, "epoch", 10000)
	td.ObjReqShouldGe(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqShouldGe(res.JSON, "epoch", 0)

	td.ObjReqMayGe(res.JSON, "epoch", 10000)
	td.ObjReqMayGe(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjReqMayGe(res.JSON, "epoch", 0)

	// testing: OPTIONAL
	td.Expect(FAIL)
	td.ObjOptMustEq(res.JSON, "epoch", 10000)
	td.ObjOptMustEq(res.JSON, "epoch", res.JSON["epoch"])
	td.Expect(FAIL)
	td.ObjOptMustNe(res.JSON, "epoch", res.JSON["epoch"])
	td.ObjOptMustNe(res.JSON, "foo", 1)

	// Float checks
	// //////////////////////////////////////////////////
	td.Log("==== Float/decimal tests ====")
	// testing: EQ
	td.ObjReqMustEq(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqShouldEq(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMayEq(res.JSON, "dec", res.JSON["dec"])

	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "dec", 0.0)
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "dec", 0.0)
	td.ObjReqMayEq(res.JSON, "dec", 0.0)

	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "dec", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "dec", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "xid", 0.0)

	// testing: NE
	td.ObjReqMustNe(res.JSON, "dec", 0.0)
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "dec", res.JSON["dec"])

	td.ObjReqShouldNe(res.JSON, "dec", 0.0)
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "dec", res.JSON["dec"])

	td.ObjReqMayNe(res.JSON, "dec", 0.0)
	td.ObjReqMayNe(res.JSON, "dec", res.JSON["dec"])

	// testing: LT
	td.ObjReqMustLt(res.JSON, "dec", 100000.0)
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "dec", 0.0)

	td.ObjReqShouldLt(res.JSON, "dec", 100000.0)
	td.Expect(WARN)
	td.ObjReqShouldLt(res.JSON, "dec", 0.0)

	td.ObjReqMayLt(res.JSON, "dec", 100000.0)
	td.ObjReqMayLt(res.JSON, "dec", 0.0)

	// testing: LE
	td.ObjReqMustLe(res.JSON, "dec", 100000.0)
	td.ObjReqMustLe(res.JSON, "dec", res.JSON["dec"])
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "dec", 0.0)

	td.ObjReqShouldLe(res.JSON, "dec", 100000.0)
	td.ObjReqShouldLe(res.JSON, "dec", res.JSON["dec"])
	td.Expect(WARN)
	td.ObjReqShouldLe(res.JSON, "dec", 0.0)

	td.ObjReqMayLe(res.JSON, "dec", 100000.0)
	td.ObjReqMayLe(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMayLe(res.JSON, "dec", 0.0)

	// testing: GT
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "dec", 10000.0)
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMustGt(res.JSON, "dec", 0.0)

	td.Expect(WARN)
	td.ObjReqShouldGt(res.JSON, "dec", 100000.0)
	td.Expect(WARN)
	td.ObjReqShouldGt(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqShouldGt(res.JSON, "dec", 0.0)

	td.ObjReqMayGt(res.JSON, "dec", 100000.0)
	td.ObjReqMayGt(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMayGt(res.JSON, "dec", 0.0)

	// testing: GE
	td.Expect(FAIL)
	td.ObjReqMustGe(res.JSON, "dec", 10000.0)
	td.ObjReqMustGe(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMustGe(res.JSON, "dec", 0.0)

	td.Expect(WARN)
	td.ObjReqShouldGe(res.JSON, "dec", 10000.0)
	td.ObjReqShouldGe(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqShouldGe(res.JSON, "dec", 0.0)

	td.ObjReqMayGe(res.JSON, "dec", 10000.0)
	td.ObjReqMayGe(res.JSON, "dec", res.JSON["dec"])
	td.ObjReqMayGe(res.JSON, "dec", 0.0)

	// testing: OPTIONAL
	td.Expect(FAIL)
	td.ObjOptMustEq(res.JSON, "dec", 10000.0)
	td.ObjOptMustEq(res.JSON, "dec", res.JSON["dec"])
	td.Expect(FAIL)
	td.ObjOptMustNe(res.JSON, "dec", res.JSON["dec"])
	td.ObjOptMustNe(res.JSON, "foo", 0.0)

	// Bool checks
	// //////////////////////////////////////////////////
	td.Log("==== Bool tests ====")
	// testing: EQ
	td.ObjReqMustEq(res.JSON, "bool", res.JSON["bool"])
	td.ObjReqShouldEq(res.JSON, "bool", res.JSON["bool"])
	td.ObjReqMayEq(res.JSON, "bool", res.JSON["bool"])

	td.Expect(FAIL)
	td.ObjReqMustEq(res.JSON, "bool", false)
	td.Expect(WARN)
	td.ObjReqShouldEq(res.JSON, "bool", false)
	td.ObjReqMayEq(res.JSON, "bool", false)

	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "bool", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "bool", "hi")
	td.Expect(FAIL) // bad type
	td.ObjReqMustEq(res.JSON, "xid", true)

	// testing: NE
	td.ObjReqMustNe(res.JSON, "bool", false)
	td.Expect(FAIL)
	td.ObjReqMustNe(res.JSON, "bool", res.JSON["bool"])

	td.ObjReqShouldNe(res.JSON, "bool", false)
	td.Expect(WARN)
	td.ObjReqShouldNe(res.JSON, "bool", res.JSON["bool"])

	td.ObjReqMayNe(res.JSON, "bool", false)
	td.ObjReqMayNe(res.JSON, "bool", res.JSON["bool"])

	// testing: LT
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "bool", false)
	td.Expect(FAIL)
	td.ObjReqMustLt(res.JSON, "bool", true)

	td.Expect(WARN)
	td.ObjReqShouldLt(res.JSON, "bool", false)
	td.Expect(WARN)
	td.ObjReqShouldLt(res.JSON, "bool", true)

	td.ObjReqMayLt(res.JSON, "bool", false)
	td.ObjReqMayLt(res.JSON, "bool", true)

	// testing: LE
	td.Expect(FAIL)
	td.ObjReqMustLe(res.JSON, "bool", false)
	td.ObjReqMustLe(res.JSON, "bool", true)

	td.Expect(WARN)
	td.ObjReqShouldLe(res.JSON, "bool", false)
	td.ObjReqShouldLe(res.JSON, "bool", true)

	td.ObjReqMayLe(res.JSON, "bool", false)
	td.ObjReqMayLe(res.JSON, "bool", true)

	// testing: GT
	td.ObjReqMustGt(res.JSON, "bool", false)
	td.Expect(FAIL)
	td.ObjReqMustGt(res.JSON, "bool", true)

	td.ObjReqShouldGt(res.JSON, "bool", false)
	td.Expect(WARN)
	td.ObjReqShouldGt(res.JSON, "bool", true)

	td.ObjReqMayGt(res.JSON, "bool", false)
	td.ObjReqMayGt(res.JSON, "bool", true)

	// testing: GE
	td.ObjReqMustGe(res.JSON, "bool", false)
	td.ObjReqMustGe(res.JSON, "bool", true)

	td.ObjReqShouldGe(res.JSON, "bool", false)
	td.ObjReqShouldGe(res.JSON, "bool", true)

	td.ObjReqMayGe(res.JSON, "bool", false)
	td.ObjReqMayGe(res.JSON, "bool", true)

	// testing: OPTIONAL
	td.Expect(FAIL)
	td.ObjOptMustEq(res.JSON, "bool", false)
	td.ObjOptMustEq(res.JSON, "bool", true)
	td.Expect(FAIL)
	td.ObjOptMustNe(res.JSON, "bool", true)
	td.ObjOptMustNe(res.JSON, "foo", true)

	// Exist checks
	// //////////////////////////////////////////////////
	td.Log("==== Exist checks ====")

	td.ObjMustExist(res.JSON, "xid")             // any type
	td.ObjMustExist(res.JSON, "registryid", "")  // string
	td.ObjMustExist(res.JSON, "epoch", 5)        // int
	td.ObjMustExist(res.JSON, "createdat", "ts") // ts
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "xid", "ts") // ts
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "foo", "ts") // ts
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "name", 5.5) // float
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "name", true) // bool
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "foo")
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "foo", "foo")
	td.Expect(FAIL)
	td.ObjMustExist(res.JSON, "epoch", "foo") // bad type

	td.ObjShouldExist(res.JSON, "xid")             // any type
	td.ObjShouldExist(res.JSON, "registryid", "")  // string
	td.ObjShouldExist(res.JSON, "epoch", 5)        // int
	td.ObjShouldExist(res.JSON, "createdat", "ts") // ts
	td.Expect(FAIL)
	td.ObjShouldExist(res.JSON, "xid", "ts") // ts
	td.Expect(WARN)
	td.ObjShouldExist(res.JSON, "foo", "ts") // ts
	td.Expect(WARN)
	td.ObjShouldExist(res.JSON, "name", 5.5) // float
	td.Expect(WARN)
	td.ObjShouldExist(res.JSON, "name", true) // bool
	td.Expect(WARN)
	td.ObjShouldExist(res.JSON, "foo")
	td.Expect(WARN)
	td.ObjShouldExist(res.JSON, "foo", "foo")
	td.Expect(FAIL)
	td.ObjShouldExist(res.JSON, "epoch", "foo") // bad type

	td.ObjMayExist(res.JSON, "xid")             // any type
	td.ObjMayExist(res.JSON, "registryid", "")  // string
	td.ObjMayExist(res.JSON, "epoch", 5)        // int
	td.ObjMayExist(res.JSON, "createdat", "ts") // ts
	td.ObjMayExist(res.JSON, "foo", "ts")       // ts
	td.Expect(FAIL)
	td.ObjMayExist(res.JSON, "xid", "ts")  // ts
	td.ObjMayExist(res.JSON, "name", 5.5)  // float
	td.ObjMayExist(res.JSON, "name", true) // bool
	td.ObjMayExist(res.JSON, "foo")
	td.ObjMayExist(res.JSON, "foo", "foo")
	td.Expect(FAIL)
	td.ObjMayExist(res.JSON, "epoch", "foo") // bad type

	// ----

	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "xid") // any type
	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "registryid", "") // string
	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "epoch", 5) // int
	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "createdat", "ts") // ts
	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "xid", "ts")  // ts
	td.ObjMustNotExist(res.JSON, "foo", "ts")  // ts
	td.ObjMustNotExist(res.JSON, "name", 5.5)  // float
	td.ObjMustNotExist(res.JSON, "name", true) // bool
	td.Expect(FAIL)
	td.ObjMustNotExist(res.JSON, "xid", 5) // bad type

	td.Expect(WARN)
	td.ObjShouldNotExist(res.JSON, "xid") // any type
	td.Expect(WARN)
	td.ObjShouldNotExist(res.JSON, "registryid", "") // string
	td.Expect(WARN)
	td.ObjShouldNotExist(res.JSON, "epoch", 5) // int
	td.Expect(WARN)
	td.ObjShouldNotExist(res.JSON, "createdat", "ts") // ts
	td.Expect(FAIL)
	td.ObjShouldNotExist(res.JSON, "xid", "ts")  // ts
	td.ObjShouldNotExist(res.JSON, "foo", "ts")  // ts
	td.ObjShouldNotExist(res.JSON, "name", 5.5)  // float
	td.ObjShouldNotExist(res.JSON, "name", true) // bool
	td.Expect(WARN)
	td.ObjShouldNotExist(res.JSON, "xid")
	td.Expect(FAIL)
	td.ObjShouldNotExist(res.JSON, "xid", 5) // bad type

	td.ObjMayNotExist(res.JSON, "xid")             // any type
	td.ObjMayNotExist(res.JSON, "registryid", "")  // string
	td.ObjMayNotExist(res.JSON, "epoch", 5)        // int
	td.ObjMayNotExist(res.JSON, "createdat", "ts") // ts
	td.Expect(FAIL)
	td.ObjMayNotExist(res.JSON, "xid", "ts") // ts
	td.ObjMayNotExist(res.JSON, "foo", "ts") // ts
	td.Expect(FAIL)
	td.ObjMayNotExist(res.JSON, "epoch", "foo") // int
	td.ObjMayNotExist(res.JSON, "name", 5.5)    // float
	td.ObjMayNotExist(res.JSON, "name", true)   // bool
}
