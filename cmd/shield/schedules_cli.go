package main

import (
	"github.com/spf13/cobra"
)

var (

	//== Applicable actions for Schedules

	listScheduleCmd = &cobra.Command{
		Use:   "schedules",
		Short: "List all the Schedules",
		Long:  "This is a Long Jaun (TBD)",
	}

	showScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Show all the Schedules",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Delete all the Schedules",
		Long:  "This is a Long Jaun (TBD)",
	}

	editScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Edit all the Schedules",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listScheduleCmd.Run = debug
	showScheduleCmd.Run = debug
	editScheduleCmd.Run = debug
	deleteScheduleCmd.Run = debug

	listCmd.AddCommand(listScheduleCmd)
	showCmd.AddCommand(showScheduleCmd)
	editCmd.AddCommand(editScheduleCmd)
	deleteCmd.AddCommand(deleteScheduleCmd)
}
