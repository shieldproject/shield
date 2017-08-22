package schedules

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Get - Print detailed information about a specific backup schedule
var Get = &commands.Command{
	Summary: "Print detailed information about a specific backup schedule",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.ScheduleNameFlag,
		},
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	},
	RunFn: cliGetSchedule,
	Group: commands.SchedulesGroup,
}

func cliGetSchedule(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show schedule' command")

	schedule, _, err := internal.FindSchedule(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(schedule)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(schedule.UUID)
		return nil
	}

	internal.ShowSchedule(schedule)
	return nil
}
