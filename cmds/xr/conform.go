package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
	. "github.com/xregistry/server/common"
)

var depth = 2
var ConfigFile = EnvString("XR_CONFORM_CONFIG", "")
var ShowLogs = EnvBool("XR_SHOWLOGS", false)

func conformFunc(cmd *cobra.Command, args []string) {
	servers := []string{}

	if len(args) > 0 {
		servers = args
	} else {
		servers = []string{GetServer()}
	}

	FailFast, _ = cmd.Flags().GetBool("failfast")
	NoWrap, _ := cmd.Flags().GetBool("nowrap")
	if NoWrap {
		WrapAt = 0
	}

	rc := 0
	for i, server := range servers {
		TDClear()
		if i != 0 {
			fmt.Printf("\n")
		}
		rc = rc + conformServer(cmd, server)
	}
	if rc != 0 {
		os.Exit(rc)
	}
}

func conformServer(cmd *cobra.Command, server string) int {
	td := NewTD(nil, server)

	defer func() {
		// Print the results
		// td.Dump("")
		if depth <= 0 {
			// Can't actually do zero, so zero = -1 (all)
			depth = 9999999
		}
		td.Print(os.Stdout, "", ShowLogs, depth-1)
	}()

	reg, xErr := xrlib.GetRegistry(server)
	if xErr != nil {
		td.Fail(xErr.GetTitle())
		return td.ExitCode()
	}

	td.SetRegistry(reg)

	if ConfigFile != "" {
		Error(reg.LoadConfigFromFile(ConfigFile))
	}

	runFunc, _ := cmd.Flags().GetString("run")
	if runFunc == "" {
		td.Include(TestRegistry)
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

	// Print results via defer
	return td.ExitCode()
}

func addConformCmd(parent *cobra.Command) {
	conformCmd := &cobra.Command{
		Use:     "conform [URL...]",
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

	_, xErr := reg.GetCapabilities()
	Error(xErr)

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
	nTD := NewTD(td, "Bad type tests")

	nTD.Expect(FAIL)
	nTD.ObjReqMustEq(res.JSON, "registryid", 5) // Fail: bad type
	nTD.Expect(FAIL)
	nTD.ObjReqShouldEq(res.JSON, "registryid", 5) // Fail: bad type
	nTD.Expect(FAIL)
	nTD.ObjReqMayEq(res.JSON, "registryid", 5) // Fail: bad type

	// String checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "String tests")

	nnTD := NewTD(nTD, "testing RE")
	// testing: RE
	nnTD.ObjReqMustRe(res.JSON, "registryid", ".*")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustRe(res.JSON, "registryid", ".*zBOGUSz*")
	nnTD.ObjReqShouldRe(res.JSON, "registryid", ".*")
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldRe(res.JSON, "registryid", ".*zBOGUSz*")
	nnTD.ObjReqMayRe(res.JSON, "registryid", ".*")
	nnTD.ObjReqMayRe(res.JSON, "registryid", ".*zBOGUSz*")

	// testing: EQ & must/should/may
	nnTD = NewTD(nTD, "testing EQ & must/should/may")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "registryid", "hi") // Fail
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "registryid", "hi") // Warn
	nnTD.ObjReqMayEq(res.JSON, "registryid", "hi")
	nnTD.Expect(FAIL)
	nnTD.ObjCheck(res.JSON, "registryid", TD_REQUIRED, TD_EQ, "hi") // def=MUST

	// testing: NE
	nnTD = NewTD(nTD, "testing NE")
	nnTD.ObjReqMustNe(res.JSON, "registryid", "")
	nnTD.ObjReqMustNe(res.JSON, "registryid", "hi")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: LT
	nnTD = NewTD(nTD, "testing LT")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "registryid", "") // Fail
	nnTD.ObjReqMustLt(res.JSON, "registryid", "ZZ")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: LE
	nnTD = NewTD(nTD, "testing LE")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "registryid", "") // Fail
	nnTD.ObjReqMustLe(res.JSON, "registryid", "ZZ")
	nnTD.ObjReqMustLe(res.JSON, "registryid", "TestXRConformBasic")

	// testing: GT
	nnTD = NewTD(nTD, "testing GT")
	nnTD.ObjReqMustGt(res.JSON, "registryid", "")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "registryid", "ZZ") // Fail
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "registryid", "TestXRConformBasic") // Fail

	// testing: GE
	nnTD = NewTD(nTD, "testing GE")
	nnTD.ObjReqMustGe(res.JSON, "registryid", "")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "registryid", "ZZ") // Fail
	nnTD.ObjReqMustGe(res.JSON, "registryid", "TestXRConformBasic")

	// testing: EXIST
	nnTD = NewTD(nTD, "testing EXIST")
	nnTD.ObjCheck(res.JSON, "registryid", TD_MUST, TD_EXIST)
	nnTD.Expect(FAIL)
	nnTD.ObjCheck(res.JSON, "name", TD_MUST, TD_EXIST) // Fail: must exist
	nnTD.Expect(FAIL)
	nnTD.ObjCheck(res.JSON, "registryid", TD_EXIST, 5) // Fail: bad type

	// testing: OPTIONAL
	nnTD = NewTD(nTD, "testing OPTIONAL")
	nnTD.ObjOptMustEq(res.JSON, "registryid", "TestXRConformBasic")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustEq(res.JSON, "registryid", "")
	nnTD.ObjOptMustNe(res.JSON, "registryid", "")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustNe(res.JSON, "registryid", "TestXRConformBasic")
	nnTD.ObjOptMustNe(res.JSON, "name", "")

	// testing: REQUIRED
	nnTD = NewTD(nTD, "testing REQUIRED")
	nnTD.ObjReqMustNe(res.JSON, "registryid", "")
	nnTD.ObjReqMustEq(res.JSON, "registryid", "TestXRConformBasic")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "name", "") // Fail: missing

	// Timestamp checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "Timestamp testsQ")
	// testing: EQ
	nnTD = NewTD(nTD, "EQ")
	nnTD.ObjReqMustEq(res.JSON, "createdat", "ts")
	nnTD.ObjReqShouldEq(res.JSON, "createdat", "ts")
	nnTD.ObjReqMayEq(res.JSON, "createdat", "ts")

	nnTD.ObjReqMustEq(res.JSON, "createdat", "YYYY-MM-DD")
	nnTD.ObjReqShouldEq(res.JSON, "createdat", "YYYY-MM-DD")
	nnTD.ObjReqMayEq(res.JSON, "createdat", "YYYY-MM-DD")

	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "createdat", "")
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "createdat", "")
	nnTD.ObjReqMayEq(res.JSON, "createdat", "")

	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.ObjReqMayEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")

	nnTD.Expect(FAIL) // exp is a string, not a ts
	nnTD.ObjReqMustEq(res.JSON, "createdat", "2026/01/01T12:00:00")
	nnTD.Expect(FAIL) // registryid isn't a TS
	nnTD.ObjReqMustEq(res.JSON, "registryid", "ts")
	nnTD.Expect(FAIL) // optional description isn't a TS
	nnTD.ObjReqMustEq(res.JSON, "description", "ts")

	nnTD = NewTD(nTD, "NE")
	// testing: NE
	nnTD.ObjReqMustNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "createdat", "ts")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "createdat", "YYYY-MM-DD")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "foo", "YYYY-MM-DD")

	nnTD.ObjReqShouldNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "createdat", "ts")
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "createdat", "YYYY-MM-DD")
	nnTD.Expect(FAIL)
	nnTD.ObjReqShouldNe(res.JSON, "foo", "YYYY-MM-DD")

	nnTD.ObjReqMayNe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.ObjReqMayNe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.ObjReqMayNe(res.JSON, "createdat", "ts")
	nnTD.ObjReqMayNe(res.JSON, "createdat", "YYYY-MM-DD")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMayNe(res.JSON, "foo", "YYYY-MM-DD")

	// testing: LT
	nnTD = NewTD(nTD, "LT")
	nnTD.ObjReqMustLt(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "createdat", "ts")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: LE
	nnTD = NewTD(nTD, "LE")
	nnTD.ObjReqMustLe(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.ObjReqMustLe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "createdat", "ts")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: GT
	nnTD = NewTD(nTD, "GT")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	nnTD.ObjReqMustGt(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "createdat", "ts")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: GE
	nnTD = NewTD(nTD, "GE")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "createdat", "3026-01-01T12:00:00Z")
	nnTD.ObjReqMustGe(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.ObjReqMustGe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "createdat", "ts")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "createdat", "YYYY-MM-DD")

	// testing: OPTIONAL
	nnTD = NewTD(nTD, "OPTIONAL")
	nnTD.ObjOptMustEq(res.JSON, "createdat", "ts")
	nnTD.ObjOptMustEq(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustEq(res.JSON, "createdat", "2026-01-01T12:00:00Z")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustNe(res.JSON, "createdat", res.JSON["createdat"])
	nnTD.ObjOptMustNe(res.JSON, "foo", "ts")

	// Int checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "Int checks")

	// testing: EQ
	nnTD = NewTD(nTD, "EQ")
	nnTD.ObjReqMustEq(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqShouldEq(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMayEq(res.JSON, "epoch", res.JSON["epoch"])

	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "epoch", 0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "epoch", 0)
	nnTD.ObjReqMayEq(res.JSON, "epoch", 0)

	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "epoch", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "epoch", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "xid", 1)

	// testing: NE
	nnTD = NewTD(nTD, "NE")
	nnTD.ObjReqMustNe(res.JSON, "epoch", 0)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "epoch", res.JSON["epoch"])

	nnTD.ObjReqShouldNe(res.JSON, "epoch", 0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "epoch", res.JSON["epoch"])

	nnTD.ObjReqMayNe(res.JSON, "epoch", 0)
	nnTD.ObjReqMayNe(res.JSON, "epoch", res.JSON["epoch"])

	// testing: LT
	nnTD = NewTD(nTD, "LT")
	nnTD.ObjReqMustLt(res.JSON, "epoch", 100000)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "epoch", 0)

	nnTD.ObjReqShouldLt(res.JSON, "epoch", 100000)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLt(res.JSON, "epoch", 0)

	nnTD.ObjReqMayLt(res.JSON, "epoch", 100000)
	nnTD.ObjReqMayLt(res.JSON, "epoch", 0)

	// testing: LE
	nnTD = NewTD(nTD, "LE")
	nnTD.ObjReqMustLe(res.JSON, "epoch", 100000)
	nnTD.ObjReqMustLe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "epoch", 0)

	nnTD.ObjReqShouldLe(res.JSON, "epoch", 100000)
	nnTD.ObjReqShouldLe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLe(res.JSON, "epoch", 0)

	nnTD.ObjReqMayLe(res.JSON, "epoch", 100000)
	nnTD.ObjReqMayLe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMayLe(res.JSON, "epoch", 0)

	// testing: GT
	nnTD = NewTD(nTD, "GT")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "epoch", 10000)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMustGt(res.JSON, "epoch", 0)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGt(res.JSON, "epoch", 100000)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGt(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqShouldGt(res.JSON, "epoch", 0)

	nnTD.ObjReqMayGt(res.JSON, "epoch", 100000)
	nnTD.ObjReqMayGt(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMayGt(res.JSON, "epoch", 0)

	// testing: GE
	nnTD = NewTD(nTD, "GE")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "epoch", 10000)
	nnTD.ObjReqMustGe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMustGe(res.JSON, "epoch", 0)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGe(res.JSON, "epoch", 10000)
	nnTD.ObjReqShouldGe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqShouldGe(res.JSON, "epoch", 0)

	nnTD.ObjReqMayGe(res.JSON, "epoch", 10000)
	nnTD.ObjReqMayGe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjReqMayGe(res.JSON, "epoch", 0)

	// testing: OPTIONAL
	nnTD = NewTD(nTD, "OPTIONAL")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustEq(res.JSON, "epoch", 10000)
	nnTD.ObjOptMustEq(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustNe(res.JSON, "epoch", res.JSON["epoch"])
	nnTD.ObjOptMustNe(res.JSON, "foo", 1)

	// Float checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "Float/decimal tests")

	// testing: EQ
	nnTD = NewTD(nTD, "EQ")
	nnTD.ObjReqMustEq(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqShouldEq(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMayEq(res.JSON, "dec", res.JSON["dec"])

	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "dec", 0.0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "dec", 0.0)
	nnTD.ObjReqMayEq(res.JSON, "dec", 0.0)

	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "dec", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "dec", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "xid", 0.0)

	// testing: NE
	nnTD = NewTD(nTD, "NE")
	nnTD.ObjReqMustNe(res.JSON, "dec", 0.0)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "dec", res.JSON["dec"])

	nnTD.ObjReqShouldNe(res.JSON, "dec", 0.0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "dec", res.JSON["dec"])

	nnTD.ObjReqMayNe(res.JSON, "dec", 0.0)
	nnTD.ObjReqMayNe(res.JSON, "dec", res.JSON["dec"])

	// testing: LT
	nnTD = NewTD(nTD, "LT")
	nnTD.ObjReqMustLt(res.JSON, "dec", 100000.0)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "dec", 0.0)

	nnTD.ObjReqShouldLt(res.JSON, "dec", 100000.0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLt(res.JSON, "dec", 0.0)

	nnTD.ObjReqMayLt(res.JSON, "dec", 100000.0)
	nnTD.ObjReqMayLt(res.JSON, "dec", 0.0)

	// testing: LE
	nnTD = NewTD(nTD, "LE")
	nnTD.ObjReqMustLe(res.JSON, "dec", 100000.0)
	nnTD.ObjReqMustLe(res.JSON, "dec", res.JSON["dec"])
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "dec", 0.0)

	nnTD.ObjReqShouldLe(res.JSON, "dec", 100000.0)
	nnTD.ObjReqShouldLe(res.JSON, "dec", res.JSON["dec"])
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLe(res.JSON, "dec", 0.0)

	nnTD.ObjReqMayLe(res.JSON, "dec", 100000.0)
	nnTD.ObjReqMayLe(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMayLe(res.JSON, "dec", 0.0)

	// testing: GT
	nnTD = NewTD(nTD, "GT")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "dec", 10000.0)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMustGt(res.JSON, "dec", 0.0)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGt(res.JSON, "dec", 100000.0)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGt(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqShouldGt(res.JSON, "dec", 0.0)

	nnTD.ObjReqMayGt(res.JSON, "dec", 100000.0)
	nnTD.ObjReqMayGt(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMayGt(res.JSON, "dec", 0.0)

	// testing: GE
	nnTD = NewTD(nTD, "GE")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGe(res.JSON, "dec", 10000.0)
	nnTD.ObjReqMustGe(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMustGe(res.JSON, "dec", 0.0)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGe(res.JSON, "dec", 10000.0)
	nnTD.ObjReqShouldGe(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqShouldGe(res.JSON, "dec", 0.0)

	nnTD.ObjReqMayGe(res.JSON, "dec", 10000.0)
	nnTD.ObjReqMayGe(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjReqMayGe(res.JSON, "dec", 0.0)

	// testing: OPTIONAL
	nnTD = NewTD(nTD, "OPTIONAL")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustEq(res.JSON, "dec", 10000.0)
	nnTD.ObjOptMustEq(res.JSON, "dec", res.JSON["dec"])
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustNe(res.JSON, "dec", res.JSON["dec"])
	nnTD.ObjOptMustNe(res.JSON, "foo", 0.0)

	// Bool checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "Bool checks")

	// testing: EQ
	nnTD = NewTD(nTD, "EQ")
	nnTD.ObjReqMustEq(res.JSON, "bool", res.JSON["bool"])
	nnTD.ObjReqShouldEq(res.JSON, "bool", res.JSON["bool"])
	nnTD.ObjReqMayEq(res.JSON, "bool", res.JSON["bool"])

	nnTD.Expect(FAIL)
	nnTD.ObjReqMustEq(res.JSON, "bool", false)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldEq(res.JSON, "bool", false)
	nnTD.ObjReqMayEq(res.JSON, "bool", false)

	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "bool", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "bool", "hi")
	nnTD.Expect(FAIL) // bad type
	nnTD.ObjReqMustEq(res.JSON, "xid", true)

	// testing: NE
	nnTD = NewTD(nTD, "NE")
	nnTD.ObjReqMustNe(res.JSON, "bool", false)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustNe(res.JSON, "bool", res.JSON["bool"])

	nnTD.ObjReqShouldNe(res.JSON, "bool", false)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldNe(res.JSON, "bool", res.JSON["bool"])

	nnTD.ObjReqMayNe(res.JSON, "bool", false)
	nnTD.ObjReqMayNe(res.JSON, "bool", res.JSON["bool"])

	// testing: LT
	nnTD = NewTD(nTD, "LT")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "bool", false)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLt(res.JSON, "bool", true)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLt(res.JSON, "bool", false)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLt(res.JSON, "bool", true)

	nnTD.ObjReqMayLt(res.JSON, "bool", false)
	nnTD.ObjReqMayLt(res.JSON, "bool", true)

	// testing: LE
	nnTD = NewTD(nTD, "LE")
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustLe(res.JSON, "bool", false)
	nnTD.ObjReqMustLe(res.JSON, "bool", true)

	nnTD.Expect(WARN)
	nnTD.ObjReqShouldLe(res.JSON, "bool", false)
	nnTD.ObjReqShouldLe(res.JSON, "bool", true)

	nnTD.ObjReqMayLe(res.JSON, "bool", false)
	nnTD.ObjReqMayLe(res.JSON, "bool", true)

	// testing: GT
	nnTD = NewTD(nTD, "GT")
	nnTD.ObjReqMustGt(res.JSON, "bool", false)
	nnTD.Expect(FAIL)
	nnTD.ObjReqMustGt(res.JSON, "bool", true)

	nnTD.ObjReqShouldGt(res.JSON, "bool", false)
	nnTD.Expect(WARN)
	nnTD.ObjReqShouldGt(res.JSON, "bool", true)

	nnTD.ObjReqMayGt(res.JSON, "bool", false)
	nnTD.ObjReqMayGt(res.JSON, "bool", true)

	// testing: GE
	nnTD = NewTD(nTD, "GE")
	nnTD.ObjReqMustGe(res.JSON, "bool", false)
	nnTD.ObjReqMustGe(res.JSON, "bool", true)

	nnTD.ObjReqShouldGe(res.JSON, "bool", false)
	nnTD.ObjReqShouldGe(res.JSON, "bool", true)

	nnTD.ObjReqMayGe(res.JSON, "bool", false)
	nnTD.ObjReqMayGe(res.JSON, "bool", true)

	// testing: OPTIONAL
	nnTD = NewTD(nTD, "OPTIONAL")
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustEq(res.JSON, "bool", false)
	nnTD.ObjOptMustEq(res.JSON, "bool", true)
	nnTD.Expect(FAIL)
	nnTD.ObjOptMustNe(res.JSON, "bool", true)
	nnTD.ObjOptMustNe(res.JSON, "foo", true)

	// Exist checks
	// //////////////////////////////////////////////////
	nTD = NewTD(td, "Exists checks")

	nnTD = NewTD(nTD, "MUST")
	nnTD.ObjMustExist(res.JSON, "xid")             // any type
	nnTD.ObjMustExist(res.JSON, "registryid", "")  // string
	nnTD.ObjMustExist(res.JSON, "epoch", 5)        // int
	nnTD.ObjMustExist(res.JSON, "createdat", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "xid", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "foo", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "name", 5.5) // float
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "name", true) // bool
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "foo")
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "foo", "foo")
	nnTD.Expect(FAIL)
	nnTD.ObjMustExist(res.JSON, "epoch", "foo") // bad type

	nnTD = NewTD(nTD, "SHOULD")
	nnTD.ObjShouldExist(res.JSON, "xid")             // any type
	nnTD.ObjShouldExist(res.JSON, "registryid", "")  // string
	nnTD.ObjShouldExist(res.JSON, "epoch", 5)        // int
	nnTD.ObjShouldExist(res.JSON, "createdat", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjShouldExist(res.JSON, "xid", "ts") // ts
	nnTD.Expect(WARN)
	nnTD.ObjShouldExist(res.JSON, "foo", "ts") // ts
	nnTD.Expect(WARN)
	nnTD.ObjShouldExist(res.JSON, "name", 5.5) // float
	nnTD.Expect(WARN)
	nnTD.ObjShouldExist(res.JSON, "name", true) // bool
	nnTD.Expect(WARN)
	nnTD.ObjShouldExist(res.JSON, "foo")
	nnTD.Expect(WARN)
	nnTD.ObjShouldExist(res.JSON, "foo", "foo")
	nnTD.Expect(FAIL)
	nnTD.ObjShouldExist(res.JSON, "epoch", "foo") // bad type

	nnTD = NewTD(nTD, "MAY")
	nnTD.ObjMayExist(res.JSON, "xid")             // any type
	nnTD.ObjMayExist(res.JSON, "registryid", "")  // string
	nnTD.ObjMayExist(res.JSON, "epoch", 5)        // int
	nnTD.ObjMayExist(res.JSON, "createdat", "ts") // ts
	nnTD.ObjMayExist(res.JSON, "foo", "ts")       // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMayExist(res.JSON, "xid", "ts")  // ts
	nnTD.ObjMayExist(res.JSON, "name", 5.5)  // float
	nnTD.ObjMayExist(res.JSON, "name", true) // bool
	nnTD.ObjMayExist(res.JSON, "foo")
	nnTD.ObjMayExist(res.JSON, "foo", "foo")
	nnTD.Expect(FAIL)
	nnTD.ObjMayExist(res.JSON, "epoch", "foo") // bad type

	// ----

	nnTD = NewTD(nTD, "MUST NOT")
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "xid") // any type
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "registryid", "") // string
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "epoch", 5) // int
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "createdat", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "xid", "ts")  // ts
	nnTD.ObjMustNotExist(res.JSON, "foo", "ts")  // ts
	nnTD.ObjMustNotExist(res.JSON, "name", 5.5)  // float
	nnTD.ObjMustNotExist(res.JSON, "name", true) // bool
	nnTD.Expect(FAIL)
	nnTD.ObjMustNotExist(res.JSON, "xid", 5) // bad type

	nnTD = NewTD(nTD, "SHOULD NOT")
	nnTD.Expect(WARN)
	nnTD.ObjShouldNotExist(res.JSON, "xid") // any type
	nnTD.Expect(WARN)
	nnTD.ObjShouldNotExist(res.JSON, "registryid", "") // string
	nnTD.Expect(WARN)
	nnTD.ObjShouldNotExist(res.JSON, "epoch", 5) // int
	nnTD.Expect(WARN)
	nnTD.ObjShouldNotExist(res.JSON, "createdat", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjShouldNotExist(res.JSON, "xid", "ts")  // ts
	nnTD.ObjShouldNotExist(res.JSON, "foo", "ts")  // ts
	nnTD.ObjShouldNotExist(res.JSON, "name", 5.5)  // float
	nnTD.ObjShouldNotExist(res.JSON, "name", true) // bool
	nnTD.Expect(WARN)
	nnTD.ObjShouldNotExist(res.JSON, "xid")
	nnTD.Expect(FAIL)
	nnTD.ObjShouldNotExist(res.JSON, "xid", 5) // bad type

	nnTD = NewTD(nTD, "MAY NOT")
	nnTD.ObjMayNotExist(res.JSON, "xid")             // any type
	nnTD.ObjMayNotExist(res.JSON, "registryid", "")  // string
	nnTD.ObjMayNotExist(res.JSON, "epoch", 5)        // int
	nnTD.ObjMayNotExist(res.JSON, "createdat", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMayNotExist(res.JSON, "xid", "ts") // ts
	nnTD.ObjMayNotExist(res.JSON, "foo", "ts") // ts
	nnTD.Expect(FAIL)
	nnTD.ObjMayNotExist(res.JSON, "epoch", "foo") // int
	nnTD.ObjMayNotExist(res.JSON, "name", 5.5)    // float
	nnTD.ObjMayNotExist(res.JSON, "name", true)   // bool
}
