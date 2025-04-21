package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	// log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
	"github.com/xregistry/server/registry"
)

func addRegistryCmd(parent *cobra.Command) *cobra.Command {
	registryCmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage xRegistries",
	}
	parent.AddCommand(registryCmd)

	createCmd := &cobra.Command{
		Use:   "create ID...",
		Short: "Create one or more xRegistry",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing registry ID arguments")
			}

			force, _ := cmd.Flags().GetBool("force")
			tx, err := registry.NewTx()
			ErrStop(err, "Error talking to the DB: %s", err)

			for _, id := range args {
				reg, err := registry.FindRegistry(tx, id)
				ErrStopTx(err, tx, "Error looking for %q: %s", id, err)

				if reg != nil {
					if force {
						continue
					}

					StopTx(tx, "Registry %q already exists", id)
				}

				Verbose("Creating: %s", id)
				reg, err = registry.NewRegistry(tx, id)
				ErrStopTx(err, tx, "Error creating %q: %s", id, err)
			}
			err = tx.Commit()
			ErrStopTx(err, tx, "Error saving: %s", err)
		},
	}
	createCmd.Flags().BoolP("force", "f", false, "Ignore existing registry")
	registryCmd.AddCommand(createCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete ID...",
		Short: "Delete one or more registry",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing registry ID arguments")
			}

			force, _ := cmd.Flags().GetBool("force")
			tx, err := registry.NewTx()
			ErrStop(err, "Error talking to the DB: %s", err)

			for _, id := range args {
				reg, err := registry.FindRegistry(tx, id)
				ErrStopTx(err, tx, "Error looking for %q: %s", id, err)

				if reg == nil {
					if force {
						continue
					}
					StopTx(tx, "Registry %q doesn't exists", id)
				}

				Verbose("Deleting: %s", id)
				err = reg.Delete()
				ErrStopTx(err, tx, "Error deleting %q: %s", id, err)
			}
			err = tx.Commit()
			ErrStopTx(err, tx, "Error saving: %s", err)
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Ignore missing registry")
	registryCmd.AddCommand(deleteCmd)

	getCmd := &cobra.Command{
		Use:   "get ID",
		Short: "Get details about a registry",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				Stop("Missing registry ID argument")
			}
			if len(args) > 1 {
				Stop("Too many argument on the command line")
			}

			tx, err := registry.NewTx()
			ErrStop(err, "Error talking to the DB: %s", err)

			reg, err := registry.FindRegistry(tx, args[0])
			ErrStop(err, "Error retrieving the registry: %s", err)

			tx.Rollback()

			if reg == nil {
				Stop("Registry %q does not exist", args[0])
			}

			P := func(prefix string, val any) {
				if !registry.IsNil(val) {
					str := fmt.Sprintf("%v", val)
					if str != "" {
						fmt.Printf("%s %s\n", prefix, str)
					}
				}
			}

			P("ID         :", reg.UID)
			P("Name       :", reg.Get("name"))
			P("Description:", reg.Get("descriptio"))
			P("Docs       :", reg.Get("descriptio"))
			P("Created    :", reg.Get("createdat"))
			P("Modified   :", reg.Get("modifiedat"))
		},
	}
	registryCmd.AddCommand(getCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List the registries",
		Run: func(cmd *cobra.Command, args []string) {
			ids := registry.GetRegistryNames()

			tx, err := registry.NewTx()
			ErrStop(err, "Error talking to the DB: %s", err)

			tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
			fmt.Fprintf(tw, "ID\tNAME\tCREATED\tMODIFIED\n")

			for _, id := range ids {
				reg, err := registry.FindRegistry(tx, id)
				ErrStop(err, "Error retrieving registry %q: %s", id, err)

				t, _ := time.Parse(time.RFC3339, reg.GetAsString("createdat"))
				cts := t.Format(time.DateTime)

				t, _ = time.Parse(time.RFC3339, reg.GetAsString("modifiedat"))
				mts := t.Format(time.DateTime)

				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					id, reg.GetAsString("name"), cts, mts)
			}
			tw.Flush()

			tx.Rollback()
		},
	}
	registryCmd.AddCommand(listCmd)

	return registryCmd
}
