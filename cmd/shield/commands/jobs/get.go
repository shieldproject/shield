package jobs

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Get - Print detailed information about a specific backup job
var Get = &commands.Command{
	Summary: "Print detailed information about a specific backup job",
	Flags:   commands.FlagList{commands.JobNameFlag},
	RunFn:   cliGetJob,
}

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

	Show(job, opts.APIVersion == 1)
	return nil
}

//Show displays information about the given job to stdout
func Show(job api.Job, v1 bool) {
	t := tui.NewReport()
	t.Add("Name", job.Name)
	t.Add("Paused", boolString(job.Paused))
	t.Break()

	t.Add("Retention Policy", job.RetentionName)
	t.Add("Expires in", fmt.Sprintf("%d days", job.Expiry/86400))
	t.Break()

	timespec := job.Schedule
	if v1 {
		timespec = job.ScheduleWhen
	}
	t.Add("Schedule", timespec)
	t.Break()

	t.Add("Target", job.TargetPlugin)
	t.Add("Target Endpoint", job.TargetEndpoint)
	t.Add("Remote IP", job.Agent)
	t.Break()

	t.Add("Store", job.StorePlugin)
	t.Add("Store Endpoint", job.StoreEndpoint)
	t.Break()

	t.Add("Notes", job.Summary)

	t.Output(os.Stdout)
}
