package jobs

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available backup jobs
var List = &commands.Command{
	Summary: "List available backup jobs",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			{
				Name: "target", Short: 't', Valued: true,
				Desc: "Show only jobs using the specified target",
			},
			{
				Name: "store", Short: 's', Valued: true,
				Desc: "Show only jobs using the specified store",
			},
			{
				Name: "policy", Short: 'p', Valued: true,
				Desc: "Show only jobs using the specified retention policy",
			},
			{Name: "paused", Desc: "Show only jobs which are paused"},
			{Name: "unpaused", Desc: "Show only jobs which are unpaused"},
			commands.FuzzyFlag,
		},
		JSONOutput: `[{
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
		}]`,
	},
	RunFn: cliListJobs,
	Group: commands.JobsGroup,
}

func cliListJobs(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list jobs' command")
	log.DEBUG("  for target:      '%s'", *opts.Target)
	log.DEBUG("  for store:       '%s'", *opts.Store)
	log.DEBUG("  for ret. policy: '%s'", *opts.Retention)
	log.DEBUG("  show paused?      %v", *opts.Paused)
	log.DEBUG("  show unpaused?    %v", *opts.Unpaused)
	if *opts.Raw {
		log.DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	jobs, err := api.GetJobs(api.JobFilter{
		Name:       strings.Join(args, " "),
		Paused:     api.MaybeBools(*opts.Paused, *opts.Unpaused),
		Target:     *opts.Target,
		Store:      *opts.Store,
		Retention:  *opts.Retention,
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(jobs)
		return nil
	}

	t := tui.NewTable("Name", "P?", "Summary", "Retention Policy", "Schedule", "Remote IP", "Target")
	for _, job := range jobs {
		t.Row(job, job.Name, boolString(job.Paused), job.Summary,
			job.RetentionName, job.ScheduleName, job.Agent, internal.PrettyJSON(job.TargetEndpoint))
	}
	t.Output(os.Stdout)
	return nil
}

func boolString(tf bool) string {
	if tf {
		return "Y"
	}
	return "N"
}
