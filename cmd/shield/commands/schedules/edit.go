package schedules

import (
	"os"
	"strings"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

func init() {
	eSchedule := commands.Register("edit-schedule", cliEditSchedule).Aliases("edit schedule", "update schedule")
	eSchedule.Summarize("Modify an existing backup schedule")
	eSchedule.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.ScheduleNameFlag,
		},
		JSONInput: `{
			"name":"AnotherSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"AnotherSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	})
	eSchedule.HelpGroup(commands.SchedulesGroup)
}

//Modify an existing backup schedule
func cliEditSchedule(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit schedule' command")

	s, id, err := internal.FindSchedule(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Schedule Name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("Time Spec (i.e. 'daily 4am')", "when", s.When, "", tui.FieldIsRequired)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)
	s, err = api.UpdateSchedule(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated schedule")
	return cliGetSchedule(opts, s.UUID)
}
