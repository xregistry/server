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

	rc := 0
	for i, server := range servers {
		// Create new config for each server tested
		nextConfig := *config
		nextConfig.Server = server
		nextConfig.TestRuns = map[string]*TD{}

		if i != 0 {
			fmt.Printf("\n")
		}

		rc = rc + testServer(&nextConfig)

		if rc != 0 && nextConfig.FailFast {
			break
		}
	}

	if rc != 0 {
		os.Exit(rc)
	}
}

func testServer(config *TDConfig) int {
	td := NewTD(nil, config.Server)
	td.Config = config

	defer func() {
		// Print the results
		// td.Dump("")
		if config.ConsoleDepth <= 0 {
			// Can't actually do zero, so zero = -1 (all)
			config.ConsoleDepth = 9999999
		}
		td.Print(os.Stdout, "", config.ConsoleDepth-1)
	}()

	reg, xErr := xrlib.GetRegistry(config.Server)
	if xErr != nil {
		td.Fail(xErr.GetTitle())
		return td.ExitCode()
	}

	td.SetRegistry(reg)

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
