package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

func init() {
	unpause := commands.Register("unpause", cliUnpauseJob).Aliases("unpause job")
	unpause.Summarize("Unpause a backup job")
	unpause.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
	})
	unpause.HelpGroup(commands.JobsGroup)
}

//Unpause a backup job
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
