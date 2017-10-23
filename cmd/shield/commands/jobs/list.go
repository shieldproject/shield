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
	Flags: commands.FlagList{
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
	RunFn: cliListJobs,
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
		timespec := job.Schedule
		if opts.APIVersion == 1 {
			timespec = job.ScheduleWhen
		}
		t.Row(job, job.Name, boolString(job.Paused), job.Summary,
			job.RetentionName, timespec, job.Agent, internal.PrettyJSON(job.TargetEndpoint))
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
