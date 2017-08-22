package stores

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing archive store
var Edit = &commands.Command{
	Summary: "Modify an existing archive store",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.StoreNameFlag,
		},
		JSONInput: `{
			"endpoint":"{\"endpoint\":\"schmendpoint\"}",
			"name":"AnotherStore",
			"plugin":"s3",
			"summary":"A Test Store"
		}`,
		JSONOutput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"AnotherStore",
			"summary":"A Test Store",
			"plugin":"s3",
			"endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}`,
	},
	RunFn: cliEditStore,
	Group: commands.StoresGroup,
}

func cliEditStore(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit store' command")

	s, id, err := internal.FindStore(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Store Name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", s.Plugin, "", internal.FieldIsPluginName)
		in.NewField("Configuration (JSON)", "endpoint", s.Endpoint, "", tui.FieldIsRequired)

		err = in.Show()
		if err != nil {
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
	s, err = api.UpdateStore(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated store")
	return cliGetStore(opts, s.UUID)
}
