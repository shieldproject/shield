package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a backup job
var Delete = &commands.Command{
	Summary: "Delete a backup job",
	Help: &commands.HelpInfo{
		Flags:      []commands.FlagInfo{commands.JobNameFlag},
		JSONOutput: `{"ok":"Deleted job"}`,
	},
	RunFn: cliDeleteJob,
	Group: commands.JobsGroup,
}

func cliDeleteJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete job' command")

	job, id, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowJob(job)
		if !tui.Confirm("Really delete this backup job?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteJob(id); err != nil {
		return err
	}

	commands.OK("Deleted job")
	return nil
}
