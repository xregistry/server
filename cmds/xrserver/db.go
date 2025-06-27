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
		Short: "Manage mysql databases",
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

			dbs, err := registry.ListDBs()
			ErrStop(err, "Error talking to mysql: %s", err)

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
		"Output format: json, table")
	dbCmd.AddCommand(listCmd)

	createCmd := &cobra.Command{
		Use:   "create NAME",
		Short: "Create a new mysql DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName = args[0]

			if registry.DBExists(DBName) {
				if val, _ := cmd.Flags().GetBool("force"); !val {
					Stop("DB %q already exists", DBName)
				}

				Verbose("Deleting DB: %s", DBName)
				err := registry.DeleteDB(DBName)
				ErrStop(err, "Error deleting DB %q: %s", DBName, err)
			}

			Verbose("Creating DB: %s", DBName)
			err := registry.CreateDB(DBName)
			ErrStop(err, "Error creating DB %q: %s", DBName, err)
		},
	}
	createCmd.Flags().BoolP("force", "f", false, "Delete existing DB first")
	dbCmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete NAME",
		Short: "Delete a mysql DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName = args[0]

			if !registry.DBExists(DBName) {
				if val, _ := cmd.Flags().GetBool("force"); !val {
					Stop("DB %q doesn't exists", DBName)
				}
				return
			}

			Verbose("Deleting DB: %s", DBName)
			err := registry.DeleteDB(DBName)
			ErrStop(err, "Error deleting DB %q: %s", DBName, err)
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Ignore DB missing error")
	dbCmd.AddCommand(deleteCmd)

	getCmd := &cobra.Command{
		Use:   "get NAME",
		Short: "Get details about a mysql DB",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing DB NAME argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}
			DBName = args[0]

			if !registry.DBExists(DBName) {
				Stop("DB %q doesn't exist", DBName)
			}

			fmt.Printf("DB %q exists\n", DBName)
		},
	}
	dbCmd.AddCommand(getCmd)

	/*
		listCmd := &cobra.Command{
			Use:   "list",
			Short: "List the mysql DBs",
			Run: func(cmd *cobra.Command, args []string) {
				Stop("TBD... list 'em")
			},
		}
		dbCmd.AddCommand(listCmd)
	*/

	return dbCmd
}
