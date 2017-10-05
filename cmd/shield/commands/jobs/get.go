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
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.JobNameFlag},
		JSONOutput: `{
			"uuid":"f6623a6f-8dce-46b2-a293-5525bc3a3588",
			"name":"TestJob",
			"summary":"A Test Job",
			"retention_name":"AnotherPolicy",
			"retention_uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"expiry":31536000,
			"schedule":"daily 4am",
			"paused":true,
			"store_uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"store_name":"AnotherStore",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"target_uuid":"84751f04-2be2-428d-b6a3-2022c63bf6ee",
			"target_name":"TestTarget",
			"target_plugin":"postgres",
			"target_endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"agent":"127.0.0.1:1234"
		}`,
	},
	RunFn: cliGetJob,
	Group: commands.JobsGroup,
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
