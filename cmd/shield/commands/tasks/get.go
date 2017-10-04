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
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "taskuuid", Desc: "The UUID of the task to get information for",
				Mandatory: true, Positional: true,
			},
		},
		JSONOutput: `{
			"uuid":"0e3736f3-6905-40ba-9adc-06641a282ff4",
			"owner":"system",
			"type":"backup",
			"job_uuid":"9b39b2ed-04dc-4de4-9ee8-265a3f9000e8",
			"archive_uuid":"2a4147ea-84a6-40fc-8028-143efabcc49d",
			"status":"done",
			"started_at":"2016-05-17 11:00:01",
			"stopped_at":"2016-05-17 11:00:02",
			"timeout_at":"",
			"log":"This is where I would put my plugin output if I had one"
		}`,
	},
	RunFn: cliGetTask,
	Group: commands.TasksGroup,
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
