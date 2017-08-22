package schedules

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available backup schedules
var List = &commands.Command{
	Summary: "List available backup schedules",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.UsedFlag,
			commands.UnusedFlag,
			commands.FuzzyFlag,
		},
		JSONOutput: `[{
			"uuid":"86ff3fec-76c5-48c4-880d-c37563033613",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}]`,
	},
	RunFn: cliListSchedules,
	Group: commands.SchedulesGroup,
}

func cliListSchedules(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list schedules' command")
	log.DEBUG("  show unused? %v", *opts.Unused)
	log.DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		log.DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	schedules, err := api.GetSchedules(api.ScheduleFilter{
		Name:       strings.Join(args, " "),
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(schedules)
		return nil
	}

	t := tui.NewTable("Name", "Summary", "Frequency / Interval (UTC)")
	for _, schedule := range schedules {
		t.Row(schedule, schedule.Name, schedule.Summary, schedule.When)
	}
	t.Output(os.Stdout)
	return nil
}
