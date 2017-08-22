package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Unpause - Unpause a backup job
var Unpause = &commands.Command{
	Summary: "Unpause a backup job",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
	},
	RunFn: cliUnpauseJob,
	Group: commands.JobsGroup,
}

func cliUnpauseJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'unpause job' command")

	_, id, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}
	if err := api.UnpauseJob(id); err != nil {
		return err
	}

	commands.OK("Unpaused job")
	return nil
}
