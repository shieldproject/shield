package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

func init() {
	pause := commands.Register("pause", cliPauseJob).Aliases("pause job")
	pause.Summarize("Pause a backup job")
	pause.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
	})
	pause.HelpGroup(commands.JobsGroup)
}

//Pause a backup job
func cliPauseJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'pause job' command")

	_, id, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}
	if err := api.PauseJob(id); err != nil {
		return err
	}

	return nil
}
