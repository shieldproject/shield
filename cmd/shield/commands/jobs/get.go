package jobs

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
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
			"schedule_name":"AnotherSched",
			"schedule_uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"schedule_when":"daily 4am",
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

	internal.ShowJob(job)
	return nil
}
