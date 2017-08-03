package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available tasks
func cliListTasks(args ...string) error {
	DEBUG("running 'list tasks' command")

	if *opts.Status == "" {
		*opts.Status = "running"
	}
	if *opts.Status == "all" || *opts.All {
		*opts.Status = ""
	}
	DEBUG("  for status: '%s'", *opts.Status)

	tasks, err := api.GetTasks(api.TaskFilter{
		Status: *opts.Status,
		Limit:  *opts.Limit,
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(tasks)
	}

	job := map[string]api.Job{}
	jobs, _ := api.GetJobs(api.JobFilter{})
	for _, j := range jobs {
		job[j.UUID] = j
	}

	t := tui.NewTable("UUID", "Owner", "Type", "Remote IP", "Status", "Started", "Stopped")
	for _, task := range tasks {
		started := "(pending)"
		stopped := "(not yet started)"
		if !task.StartedAt.IsZero() {
			stopped = "(running)"
			started = task.StartedAt.Format(time.RFC1123Z)
		}

		if !task.StoppedAt.IsZero() {
			stopped = task.StoppedAt.Format(time.RFC1123Z)
		}

		t.Row(task, task.UUID, task.Owner, task.Op, job[task.JobUUID].Agent, task.Status, started, stopped)
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a specific task
func cliGetTask(args ...string) error {
	DEBUG("running 'show task' command")

	require(len(args) == 1, "shield show task <UUID>")
	id := uuid.Parse(args[0])
	DEBUG("  task UUID = '%s'", id)

	task, err := api.GetTask(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(task)
	}
	if *opts.ShowUUID {
		return RawUUID(task.UUID)
	}

	ShowTask(task)
	return nil
}

func cliCancelTask(args ...string) error {
	DEBUG("running 'cancel task' command")

	require(len(args) == 1, "shield cancel task <UUID>")
	id := uuid.Parse(args[0])
	DEBUG("  task UUID = '%s'", id)

	task, err := api.GetTask(id)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowTask(task)
		if !tui.Confirm("Really cancel this task?") {
			return fmt.Errorf("Task '%s' was not canceled", id)
		}
	}

	if err := api.CancelTask(id); err != nil {
		return err
	}

	OK("Cancelled task '%s'\n", id)
	return nil
}
