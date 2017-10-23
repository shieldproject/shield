package tasks

import (
	"fmt"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Cancel - Cancel a running or pending task
var Cancel = &commands.Command{
	Summary: "Cancel a running or pending task",
	RunFn:   cliCancelTask,
}

func cliCancelTask(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'cancel task' command")

	internal.Require(len(args) == 1, "shield cancel task <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  task UUID = '%s'", id)

	task, err := api.GetTask(id)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(task)
		if !tui.Confirm("Really cancel this task?") {
			return fmt.Errorf("Task '%s' was not canceled", id)
		}
	}

	if err := api.CancelTask(id); err != nil {
		return err
	}

	commands.OK("Cancelled task '%s'\n", id)
	return nil
}
