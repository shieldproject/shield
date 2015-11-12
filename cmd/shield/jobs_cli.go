package main

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
)

var (

	//== Applicable actions for Jobs

	listJobCmd = &cobra.Command{
		Use:   "jobs",
		Short: "Lists all the jobs",
		Long:  "This is a Long Jaun (TBD)",
	}

	showJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Shows information about the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Deletes the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}

	pauseJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Pauses the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}

	unpauseJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Unpauses the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}

	pausedJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Returns \"true\" with exit code 0 if specified job is paused, \"false\"/1 otherwise",
		Long:  "This is a Long Jaun (TBD)",
	}

	runJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Runs the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}

	editJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Edit the specified job",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listJobCmd.Run = debug
	showJobCmd.Run = debug
	deleteJobCmd.Run = debug
	pauseJobCmd.Run = debug
	unpauseJobCmd.Run = debug
	pausedJobCmd.Run = debug
	runJobCmd.Run = debug
	editJobCmd.Run = debug

	listCmd.AddCommand(listJobCmd)
	showCmd.AddCommand(showJobCmd)
	deleteCmd.AddCommand(deleteJobCmd)
	pauseCmd.AddCommand(pauseJobCmd)
	unpauseCmd.AddCommand(unpauseJobCmd)
	pausedCmd.AddCommand(pausedJobCmd)
	runCmd.AddCommand(runJobCmd)
	editCmd.AddCommand(editJobCmd)
}
