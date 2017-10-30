package tasks

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Get - Print detailed information about a specific task
var Get = &commands.Command{
	Summary: "Print detailed information about a specific task",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "taskuuid", Desc: "The UUID of the task to get information for",
			Mandatory: true, Positional: true,
		},
	},
	RunFn: cliGetTask,
}

func cliGetTask(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show task' command")

	internal.Require(len(args) == 1, "shield show task <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  task UUID = '%s'", id)

	task, err := api.GetTask(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(task)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(task.UUID)
		return nil
	}

	Show(task)
	return nil
}

//Show displays information about the given task to stdout
func Show(task api.Task) {
	t := tui.NewReport()
	t.Add("UUID", task.UUID)
	t.Add("Owner", task.Owner)
	t.Add("Type", task.Op)
	t.Add("Status", task.Status)
	t.Break()

	started := "(pending)"
	stopped := "(not yet started)"
	if !task.StartedAt.IsZero() {
		stopped = "(running)"
		started = task.StartedAt.Format(time.RFC1123Z)
	}
	if !task.StoppedAt.IsZero() {
		stopped = task.StoppedAt.Format(time.RFC1123Z)
	}
	t.Add("Started at", started)
	t.Add("Stopped at", stopped)
	t.Break()

	if job, err := api.GetJob(uuid.Parse(task.JobUUID)); err == nil {
		t.Add("Job", fmt.Sprintf("%s (%s)", job.Name, task.JobUUID))
	}
	if task.ArchiveUUID != "" {
		t.Add("Archive UUID", task.ArchiveUUID)
	}
	t.Break()

	t.Add("Log", task.Log)
	t.Output(os.Stdout)
}
