package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

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

func CancelTaskByUUID(u string) error {
	err := CancelTask(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: could not cancel task '%s'", u)
	}
	fmt.Fprintf(os.Stdout, "Successfully cancelled task '%s'\n", u)
	return nil
}
