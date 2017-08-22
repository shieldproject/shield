package schedules

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a backup schedule
var Delete = &commands.Command{
	Summary: "Delete a backup schedule",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.ScheduleNameFlag,
		},
		JSONOutput: `{"ok":"Deleted schedule"}`,
	},
	RunFn: cliDeleteSchedule,
	Group: commands.SchedulesGroup,
}

func cliDeleteSchedule(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete schedule' command")

	schedule, id, err := internal.FindSchedule(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowSchedule(schedule)
		if !tui.Confirm("Really delete this schedule?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteSchedule(id); err != nil {
		return err
	}

	commands.OK("Deleted schedule")
	return nil
}
