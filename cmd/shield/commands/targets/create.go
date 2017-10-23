package targets

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new backup target
var Create = &commands.Command{
	Summary: "Create a new backup target",
	Flags: commands.FlagList{
		commands.UpdateIfExistsFlag,
	},
	RunFn: cliCreateTarget,
}

func cliCreateTarget(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create target' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Target Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", "", "", internal.FieldIsPluginName)
		in.NewField("Configuration", "endpoint", "", "", tui.FieldIsRequired)
		in.NewField("Remote IP:port", "agent", "", "", tui.FieldIsRequired)
		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really create this target?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := internal.FindTarget(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateTarget(id, content)
			if err != nil {
				return err
			}
			commands.MSG("Updated existing target")
			return cliGetTarget(opts, t.UUID)
		}
	}
	t, err := api.CreateTarget(content)
	if err != nil {
		return err
	}
	commands.MSG("Created new target")
	return cliGetTarget(opts, t.UUID)
}
