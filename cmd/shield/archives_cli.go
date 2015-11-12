package main

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
)

var (

	//== Applicable actions for Archives

	listArchiveCmd = &cobra.Command{
		Use:   "archives",
		Short: "Lists all the archives",
		Long:  "This is a Long Jaun (TBD)",
	}

	showArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Shows information about the specified archive",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Deletes the specified archive",
		Long:  "This is a Long Jaun (TBD)",
	}

	editArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Edit the specified archive",
		Long:  "This is a Long Jaun (TBD)",
	}

	restoreArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Restorss the specified archive",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listArchiveCmd.Run = debug
	showArchiveCmd.Run = debug
	deleteArchiveCmd.Run = debug
	restoreArchiveCmd.Run = debug
	editArchiveCmd.Run = debug

	listCmd.AddCommand(listArchiveCmd)
	showCmd.AddCommand(showArchiveCmd)
	deleteCmd.AddCommand(deleteArchiveCmd)
	restoreCmd.AddCommand(restoreArchiveCmd)
	editCmd.AddCommand(editArchiveCmd)
}
