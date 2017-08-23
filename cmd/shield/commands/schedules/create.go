package schedules

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new backup schedule
var Create = &commands.Command{
	Summary: "Create a new backup schedule",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.UpdateIfExistsFlag,
		},
		JSONInput: `{
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
		JSONOutput: `{
			"uuid":"9a58a3fa-7457-431c-b094-e201b42b5c7b",
			"name":"TestSched",
			"summary":"A Test Schedule",
			"when":"daily 4am"
		}`,
	},
	RunFn: cliCreateSchedule,
	Group: commands.SchedulesGroup,
}

func cliCreateSchedule(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create schedule' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Schedule Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Time Spec (i.e. 'daily 4am')", "when", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this schedule?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := internal.FindSchedule(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateSchedule(id, content)
			if err != nil {
				return err
			}
			commands.MSG("Updated existing schedule")
			return cliGetSchedule(opts, t.UUID)
		}
	}

	s, err := api.CreateSchedule(content)
	if err != nil {
		return err
	}

	commands.MSG("Created new schedule")
	return cliGetSchedule(opts, s.UUID)
}
