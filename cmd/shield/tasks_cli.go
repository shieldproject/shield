package main

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
)

var (

	//== Applicable actions for Tasks

	listTaskCmd = &cobra.Command{
		Use:   "tasks",
		Short: "Lists all the tasks",
		Long:  "This is a Long Jaun (TBD)",
	}

	showTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Shows information about the specified task",
		Long:  "This is a Long Jaun (TBD)",
	}

	cancelTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Cancels the specified task",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listTaskCmd.Run = debug
	showTaskCmd.Run = debug
	cancelTaskCmd.Run = debug

	listCmd.AddCommand(listTaskCmd)
	showCmd.AddCommand(showTaskCmd)
	cancelCmd.AddCommand(cancelTaskCmd)
}
