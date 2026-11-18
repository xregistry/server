package main

import (
	"fmt"

	"github.com/spf13/cobra"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func addBackendCmd(parent *cobra.Command) *cobra.Command {
	beCmd := &cobra.Command{
		Use:   "backend",
		Short: "Manage backends",
	}

	parent.AddCommand(beCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List the available backends and their options",
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			if !ArrayContains([]string{"table", "json"}, output) {
				Stop("--output must be one of: json, table")
			}
			fmt.Print(registry.ListBackends(XRSConfig, output))
		},
	}
	listCmd.Flags().StringP("output", "o", "table",
		"Output format: json, table*")
	listCmd.Flag("output").DefValue = ""
	beCmd.AddCommand(listCmd)

	return beCmd
}
