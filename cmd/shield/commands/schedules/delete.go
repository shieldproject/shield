package schedules

import (
	"strings"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

func init() {
	dSchedule := commands.Register("delete-schedule", cliDeleteSchedule)
	dSchedule.Summarize("Delete a backup schedule")
	dSchedule.Aliases("delete schedule", "remove schedule", "rm schedule")
	dSchedule.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.ScheduleNameFlag,
		},
		JSONOutput: `{"ok":"Deleted schedule"}`,
	})
	dSchedule.HelpGroup(commands.SchedulesGroup)
}

//Delete a backup schedule
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
