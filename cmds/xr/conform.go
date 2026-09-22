package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
)

var conformRunAliases = []string{"all", "smoke", "entities"}

var conformRunFunctions = map[string]TestFn{
	"all":      TestTDAll,
	"smoke":    TestTDSmoke,
	"entities": TestTDEntities,

	// Retain the existing internal TD test entry points.
	"TestTDAllPass": TestTDAllPass,
	"TestTDDepFail": TestTDDepFail,
	"TestTDMixture": TestTDMixture,
	"TestTDUtils":   TestTDUtils,
}

func conformFunc(cmd *cobra.Command, args []string) {
	runNames, _ := cmd.Flags().GetStringArray("run")

	servers := []string{}

	if len(args) > 0 {
		servers = args
	} else {
		servers = []string{GetServer()}
	}

	if NoWrap, _ := cmd.Flags().GetBool("nowrap"); NoWrap {
		terminalWidth = 0
	}

	config := &TDConfig{
		// FailFast:   false,
		IgnoreWarn: true,
		NextStatus: 0,
		Out:        os.Stdout,
		// ShowSkips:  false,
		// ShowWarns:  false,
		// ShowLogs:   false, // EnvBool("XR_SHOWLOGS", false)
		// ShowStats:  false,
	}

	config.FailFast, _ = cmd.Flags().GetBool("failfast")
	config.ShowWarns, _ = cmd.Flags().GetBool("warns")
	config.ShowSkips, _ = cmd.Flags().GetBool("skips")
	config.ShowLogs, _ = cmd.Flags().GetBool("logs")
	config.ShowStats, _ = cmd.Flags().GetBool("stats")
	config.ConsoleDepth, _ = cmd.Flags().GetInt("depth")
	config.RunFuncs = resolveConformRunFuncs(runNames)

	rc := runConform(servers, config)
	if rc != 0 {
		os.Exit(rc)
	}
}

func runConform(servers []string, config *TDConfig) int {
	out := config.Out
	if out == nil {
		out = os.Stdout
	}

	rc := 0
	for i, server := range servers {
		// Create new config for each server tested
		nextConfig := *config
		nextConfig.Server = server
		nextConfig.Registry = nil
		nextConfig.Model = nil
		nextConfig.Capabilities = nil
		nextConfig.NextStatus = 0
		nextConfig.Out = out
		nextConfig.TestRuns = map[string]*TD{}

		if i != 0 {
			fmt.Fprintln(out)
		}

		if testServer(&nextConfig) != 0 {
			rc = 1
			if nextConfig.FailFast {
				break
			}
		}
	}

	return rc
}

func testServer(config *TDConfig) int {
	td := NewTD(nil, config.Server)
	td.Config = config

	defer func() {
		// Print the results
		// td.Dump("")
		printDepth := config.ConsoleDepth
		if printDepth <= 0 {
			// Can't actually do zero, so zero = -1 (all)
			printDepth = 9999999
		}
		td.Print(config.Out, "", printDepth-1)
	}()

	td.SetRegistry(xrlib.DefineRegistry(config.Server))

	runFuncs := config.RunFuncs
	if len(runFuncs) == 0 {
		runFuncs = []TestFn{TestRegistry}
	}

	for _, fn := range runFuncs {
		result := td.Run(fn)
		if config.FailFast && result.Status == FAIL {
			break
		}
	}

	// Print results via defer
	return td.ExitCode()
}

func resolveConformRunFuncs(names []string) []TestFn {
	functions := make([]TestFn, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		fn := conformRunFunctions[name]
		if fn == nil {
			Error("Unknown --run value: %q. Valid values: %s",
				name, strings.Join(conformRunAliases, ", "))
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		functions = append(functions, fn)
	}
	return functions
}

func TestTDAll(td *TD) {
	td.DependsOn(TestSniff)
	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)
	td.DependsOn(TestRegistryRoot)
	td.DependsOn(TestGroups)
	td.DependsOn(TestResources)
}

func TestTDSmoke(td *TD) {
	td.DependsOn(TestSniff)
	td.DependsOn(TestModel)
	td.DependsOn(TestCapabilities)
	td.DependsOn(TestRegistryRoot)
}

func TestTDEntities(td *TD) {
	td.DependsOn(TestRegistryRoot)
	td.DependsOn(TestGroups)
	td.DependsOn(TestResources)
}

func addConformCmd(parent *cobra.Command) {
	conformCmd := &cobra.Command{
		Use:     "conform [URL...]",
		Short:   "xRegistry Conformance Tester",
		Run:     conformFunc,
		GroupID: "Admin",
	}
	conformCmd.Flags().BoolP("logs", "l", false, "Show logs even on success")
	conformCmd.Flags().IntP("depth", "d", 2, "Console depth (0=all)")
	conformCmd.Flags().BoolVarP(&tdDebug, "tdDebug", "t", tdDebug, "td debug")
	conformCmd.Flags().Bool("warns", false, "Show WARNs in console")
	conformCmd.Flags().Bool("skips", false, "Show SKIPs in console")
	conformCmd.Flags().Bool("failfast", false, "Stop on first failure")
	conformCmd.Flags().Bool("stats", false, "Show full stats on all groups")
	conformCmd.Flags().StringArrayP("run", "r", nil,
		"Run test (all, smoke, entities)")
	conformCmd.Flags().BoolP("nowrap", "", false, "Don't wrap output")

	conformCmd.Flags().MarkHidden("tdDebug")

	parent.AddCommand(conformCmd)
}
