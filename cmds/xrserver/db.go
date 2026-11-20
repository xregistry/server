package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func addDBCmd(parent *cobra.Command) *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage databases",
		// Cobra only runs the nearest PersistentPreRun so call the root's
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cmd.Root().PersistentPreRun(cmd, args)
			if !registry.BackendSupportsDBs(XRSConfig) {
				Stop("Backend %q doesn't support DBs",
					registry.BackendName(XRSConfig))
			}
		},
	}

	parent.AddCommand(dbCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List the databases",
		Run: func(cmd *cobra.Command, args []string) {
			output, _ := cmd.Flags().GetString("output")
			if !ArrayContains([]string{"table", "json"}, output) {
				Stop("--output must be one of: json, table")
			}

			dbs, err := registry.DBList(XRSConfig)
			ErrStop(err, "Error talking to backend: %s", err)

			sort.Strings(dbs)

			if output == "table" {
				tw := tabwriter.NewWriter(os.Stdout, 0, 1, 3, ' ', 0)
				fmt.Fprintf(tw, "DB NAME\n")

				for _, name := range dbs {
					fmt.Fprintf(tw, "%s\n", name)
				}
				tw.Flush()
			} else {
				res := []string{}
				for _, name := range dbs {
					res = append(res, name)
				}

				fmt.Printf("%s\n", ToJSON(res))
			}
		},
	}
	listCmd.Flags().StringP("output", "o", "table",
		"Output format: json, table*")
	listCmd.Flag("output").DefValue = ""
	dbCmd.AddCommand(listCmd)

	createCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName := args[0]

			exists, xErr := registry.DBExists(XRSConfig, DBName)
			ErrStop(xErr, "Error creating DB %q: %s", DBName, xErr)

			if exists {
				if val, _ := cmd.Flags().GetBool("force"); !val {
					Stop("DB %q already exists", DBName)
				}

				Verbose("Deleting DB: %s", DBName)
				xErr := registry.DBDelete(XRSConfig, DBName)
				ErrStop(xErr, "Error deleting DB %q: %s", DBName, xErr)
			}

			Verbose("Creating DB: %s", DBName)
			xErr = registry.DBCreate(XRSConfig, DBName)
			ErrStop(xErr, "Error creating DB %q: %s", DBName, xErr)
		},
	}
	createCmd.Flags().BoolP("force", "f", false, "Delete existing DB first")
	dbCmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName := args[0]

			exists, xErr := registry.DBExists(XRSConfig, DBName)
			ErrStop(xErr, "Error deleting DB %q: %s", DBName, xErr)

			if !exists {
				if val, _ := cmd.Flags().GetBool("force"); !val {
					Stop("DB %q doesn't exist", DBName)
				}
				return
			}

			Verbose("Deleting DB: %s", DBName)
			xErr = registry.DBDelete(XRSConfig, DBName)
			ErrStop(xErr, "Error deleting DB %q: %s", DBName, xErr)
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Ignore DB missing error")
	dbCmd.AddCommand(deleteCmd)

	getCmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get details about a DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName := args[0]

			exists, xErr := registry.DBExists(XRSConfig, DBName)
			ErrStop(xErr, "Error retrieving DB %q: %s", DBName, xErr)

			if !exists {
				Stop("DB %q doesn't exist", DBName)
			}

			fmt.Printf("DB %q exists\n", DBName)
		},
	}
	dbCmd.AddCommand(getCmd)

	/*
		listCmd := &cobra.Command{
			Use:   "list",
			Short: "List the DBs",
			Run: func(cmd *cobra.Command, args []string) {
				Stop("TBD... list 'em")
			},
		}
		dbCmd.AddCommand(listCmd)
	*/

	return dbCmd
}
