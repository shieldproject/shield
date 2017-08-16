package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

//Print detailed information about a specific backup job
func cliGetJob(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show job' command")

	job, _, err := internal.FindJob(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(job)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(job.UUID)
		return nil
	}

	internal.ShowJob(job)
	return nil
}
