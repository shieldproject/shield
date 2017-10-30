package targets

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing backup target
var Edit = &commands.Command{
	Summary: "Modify an existing backup target",
	Flags:   commands.FlagList{commands.TargetNameFlag},
	RunFn:   cliEditTarget,
}

func cliEditTarget(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit target' command")

	t, id, err := internal.FindTarget(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Target Name", "name", t.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", t.Summary, "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", t.Plugin, "", internal.FieldIsPluginName)
		in.NewField("Configuration", "endpoint", t.Endpoint, "", tui.FieldIsRequired)
		in.NewField("Remote IP:port", "agent", t.Agent, "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
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
	t, err = api.UpdateTarget(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated target")
	return cliGetTarget(opts, t.UUID)
}
