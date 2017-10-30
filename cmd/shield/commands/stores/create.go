package stores

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new archive store
var Create = &commands.Command{
	Summary: "Create a new archive store",
	Flags: commands.FlagList{
		commands.UpdateIfExistsFlag,
	},
	RunFn: cliCreateStore,
}

func cliCreateStore(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create store' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Store Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", "", "", internal.FieldIsPluginName)
		in.NewField("Configuration (JSON)", "endpoint", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this archive store?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := internal.FindStore(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateStore(id, content)
			if err != nil {
				return err
			}
			commands.MSG("Updated existing store")
			return cliGetStore(opts, t.UUID)
		}
	}

	s, err := api.CreateStore(content)

	if err != nil {
		return err
	}

	commands.MSG("Created new store")
	return cliGetStore(opts, s.UUID)
}
