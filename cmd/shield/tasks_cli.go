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

	cancelTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Cancels the specified task",
	}
)

func init() {
	// Hookup functions to the subcommands
	cancelTaskCmd.Run = processCancelTaskRequest

	// Add the subcommands to the base actions
	cancelCmd.AddCommand(cancelTaskCmd)
}

type ListTaskOptions struct {
	All   bool
	Debug bool
	UUID  string
}

func ListTasks(opts ListTaskOptions) error {
	//FIXME double check debug is/is not working...?
	tasks, err := GetTasks(TaskFilter{
		Debug: Maybe(opts.Debug),
	})
	if err != nil {
		return fmt.Errorf("\nERROR: Could not fetch list of tasks: %s\n", err)
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

		if len(opts.UUID) > 0 && opts.UUID == task.UUID {
			t.Row(task.UUID, task.Owner, task.Op, task.Status, started, stopped)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != task.UUID {
			continue
		}

		if task.Status != "done" {
			t.Row(task.UUID, task.Owner, task.Op, task.Status, started, stopped)
		} else if Maybe(opts.All).Yes {
			t.Row(task.UUID, task.Owner, task.Op, task.Status, started, stopped)
		}

	}
	t.Output(os.Stdout)
	return nil
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
