package main

import (
	"github.com/spf13/cobra"
)

var (

	//== Applicable actions for Stores

	listStoreCmd = &cobra.Command{
		Use:   "stores",
		Short: "List all the Stores",
		Long:  "This is a Long Jaun (TBD)",
	}

	showStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Show all the Stores",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Delete all the Stores",
		Long:  "This is a Long Jaun (TBD)",
	}

	editStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Edit all the Stores",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listStoreCmd.Run = debug
	showStoreCmd.Run = debug
	editStoreCmd.Run = debug
	deleteStoreCmd.Run = debug

	listCmd.AddCommand(listStoreCmd)
	showCmd.AddCommand(showStoreCmd)
	editCmd.AddCommand(editStoreCmd)
	deleteCmd.AddCommand(deleteStoreCmd)
}
