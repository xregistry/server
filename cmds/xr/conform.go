package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xregistry/server/cmds/xr/xrlib"
)

func conformFunc(cmd *cobra.Command, args []string) {
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
	}

	config.FailFast, _ = cmd.Flags().GetBool("failfast")
	config.ShowWarns, _ = cmd.Flags().GetBool("warns")
	config.ShowSkips, _ = cmd.Flags().GetBool("skips")
	config.ShowLogs, _ = cmd.Flags().GetBool("logs")
	config.ConsoleDepth, _ = cmd.Flags().GetInt("depth")
	config.RunFunc, _ = cmd.Flags().GetString("run")

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

		rc = rc + testServer(&nextConfig)

		if rc != 0 && nextConfig.FailFast {
			break
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

	if config.RunFunc == "" {
		td.Include(TestRegistry)
	} else {
		// Just for testing
		funcs := map[string]TestFn{
			"TestTDAllPass": TestTDAllPass,
			"TestTDDepFail": TestTDDepFail,
			"TestTDMixture": TestTDMixture,
			"TestTDUtils":   TestTDUtils,
		}
		fn := funcs[config.RunFunc]
		if fn == nil {
			panic(fmt.Sprintf("No function by name: %s", config.RunFunc))
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
	conformCmd.Flags().BoolP("logs", "l", false, "Show logs even on success")
	conformCmd.Flags().IntP("depth", "d", 2, "Console depth")
	conformCmd.Flags().BoolVarP(&tdDebug, "tdDebug", "t", tdDebug, "td debug")
	conformCmd.Flags().Bool("warns", false, "Show WARNs in console")
	conformCmd.Flags().Bool("skips", false, "Show SKIPs in console")
	conformCmd.Flags().Bool("failfast", false, "Stop on first failure")
	conformCmd.Flags().StringP("run", "r", "", "Run function")
	conformCmd.Flags().BoolP("nowrap", "", false, "Don't wrap output")

	conformCmd.Flags().MarkHidden("run")
	conformCmd.Flags().MarkHidden("tdDebug")

	parent.AddCommand(conformCmd)
}
