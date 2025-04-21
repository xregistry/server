package main

import (
	"fmt"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/registry"
)

func addDBCmd(parent *cobra.Command) *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Manage mysql databases",
	}

	parent.AddCommand(dbCmd)

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
