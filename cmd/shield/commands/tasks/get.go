package tasks

import (
	"github.com/geofffranks/spruce/log"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

func init() {
	task := commands.Register("task", cliGetTask)
	task.Summarize("Print detailed information about a specific task")
	task.Aliases("show task", "view task", "display task", "list task", "ls task")
	task.Help(commands.HelpInfo{
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
	})
	task.HelpGroup(commands.TasksGroup)
}

//Print detailed information about a specific task
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

	internal.ShowTask(task)
	return nil
}
