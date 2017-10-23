package tasks

import (
	"os"
	"time"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available tasks
var List = &commands.Command{
	Summary: "List available tasks",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "status", Short: 'S', Valued: true,
			Desc: `Only show tasks with the specified status
							Valid values are one of ['all', 'running', 'pending', 'cancelled']
							If not explicitly set, it defaults to 'running'`,
		},
		commands.FlagInfo{Name: "all", Short: 'a', Desc: "Show all tasks, regardless of state"},
		commands.FlagInfo{Name: "limit", Desc: "Show only the <value> most recent tasks"},
	},
	RunFn: cliListTasks,
}

func cliListTasks(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list tasks' command")

	if *opts.Status == "" {
		*opts.Status = "running"
	}
	if *opts.Status == "all" || *opts.All {
		*opts.Status = ""
	}
	log.DEBUG("  for status: '%s'", *opts.Status)

	tasks, err := api.GetTasks(api.TaskFilter{
		Status: *opts.Status,
		Limit:  *opts.Limit,
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(tasks)
		return nil
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
