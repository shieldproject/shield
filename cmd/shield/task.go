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
func cliListTasks(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`Only show tasks with the specified status
									Valid values are one of ['all', 'running', 'pending', 'cancelled']
									If not explicitly set, it defaults to 'running'`,
			true, "-S", "--status=value")
		FlagHelp(`Show all tasks, regardless of state`, true, "-a", "--all")
		FlagHelp("Returns information as a JSON object", true, "--raw")
		FlagHelp("Show only the <value> most recent tasks", true, "--limit=value")
		HelpKMacro()
		JSONHelp(`[{"uuid":"0e3736f3-6905-40ba-9adc-06641a282ff4","owner":"system","type":"backup","job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8","archive_uuid":"2a4147ea-84a6-40fc-8028-143efabcc49d","status":"done","started_at":"2016-05-17 11:00:01","stopped_at":"2016-05-17 11:00:02","timeout_at":"","log":"This is where I would put my plugin output if I had one"}]`)
		return nil
	}

	DEBUG("running 'list tasks' command")

	if *options.Status == "" {
		*options.Status = "running"
	}
	if *options.Status == "all" || *options.All {
		*options.Status = ""
	}
	DEBUG("  for status: '%s'", *opts.Status)

	tasks, err := api.GetTasks(api.TaskFilter{
		Status: *options.Status,
		Limit:  *options.Limit,
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
func cliGetTask(opts Options, args []string, help bool) error {
	if help {
		FlagHelp("The ID number of the task to show info about", false, "<id>")
		HelpKMacro()
		FlagHelp("Returns information as a JSON object", true, "--raw")
		JSONHelp(`{"uuid":"b40ae708-6215-4932-90fb-fe580fac7196","owner":"system","type":"backup","job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8","archive_uuid":"62792b22-c89e-4d69-b874-69a5f056a9ef","status":"done","started_at":"2016-05-18 11:00:01","stopped_at":"2016-05-18 11:00:02","timeout_at":"","log":"This is where I would put my plugin output if I had one"}`)
		return nil
	}

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

func cliCancelTask(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`Outputs the result as a JSON object.
				The cli will not prompt for confirmation in raw mode.`, true, "--raw")
		HelpKMacro()
		JSONHelp(`{"ok":"Cancelled task '81746508-bd18-46a8-842e-97911d4b23a3'\n"}`)
		return nil
	}

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
