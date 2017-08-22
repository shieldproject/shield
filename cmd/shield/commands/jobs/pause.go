package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Pause - Pause a backup job
var Pause = &commands.Command{
	Summary: "Pause a backup job",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
	},
	RunFn: cliPauseJob,
	Group: commands.JobsGroup,
}

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
