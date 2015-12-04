package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Tasks

	listTaskCmd = &cobra.Command{
		Use:   "tasks",
		Short: "Lists all the tasks",
	}

	showTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Shows information about the specified task",
	}

	cancelTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Cancels the specified task",
	}
)

func init() {
	// Set options for the subcommands
	listTaskCmd.Flags().StringP("status", "s", "", "Filter by status")
	listTaskCmd.Flags().Bool("debug", false, "Turn on debug mode")

	// Hookup functions to the subcommands
	listTaskCmd.Run = processListTasksRequest
	showTaskCmd.Run = processShowTaskRequest
	cancelTaskCmd.Run = processCancelTaskRequest

	// Add the subcommands to the base actions
	listCmd.AddCommand(listTaskCmd)
	showCmd.AddCommand(showTaskCmd)
	cancelCmd.AddCommand(cancelTaskCmd)
}

func processListTasksRequest(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	debug, _ := cmd.Flags().GetBool("debug")
	status, _ := cmd.Flags().GetString("status")
	tasks, err := GetTasks(TaskFilter{
		Status: status,
		Debug:  Maybe(debug),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of tasks:\n", err)
	}

	t := tui.NewTable("UUID", "Owner", "Type", "Status", "Started", "Stopped")
	for _, task := range tasks {
		started := "(pending)"
		if !task.StartedAt.IsZero() {
			started = task.StartedAt.Format(time.RFC1123Z)
		}

		stopped := "(running)"
		if !task.StoppedAt.IsZero() {
			stopped = task.StoppedAt.Format(time.RFC1123Z)
		}

		t.Row(task.UUID, task.Owner, task.Op, task.Status, started, stopped)
	}
	t.Output(os.Stdout)
}

func processShowTaskRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	task, err := GetTask(uuid.Parse(args[0]))
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show task:\n", err)
		os.Exit(1)
	}

	t := tui.NewReport()
	t.Add("UUID", task.UUID)
	t.Add("Owner", task.Owner)
	t.Add("Type", task.Op)
	t.Add("Status", task.Status)
	t.Break()

	started := "(pending)"
	if !task.StartedAt.IsZero() {
		started = task.StartedAt.Format(time.RFC1123Z)
	}
	stopped := "(running)"
	if !task.StoppedAt.IsZero() {
		stopped = task.StoppedAt.Format(time.RFC1123Z)
	}
	t.Add("Started at", started)
	t.Add("Stopped at", stopped)
	t.Break()

	t.Add("Job UUID", task.JobUUID)
	t.Add("Archive UUID", task.ArchiveUUID)
	t.Break()

	t.Add("Log", task.Log)
	t.Output(os.Stdout)
}

func processCancelTaskRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	err := CancelTask(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not cancel task:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Canceled")

	return
}
