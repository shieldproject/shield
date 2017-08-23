package policies

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new retention policy
var Create = &commands.Command{
	Summary: "Create a new retention policy",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.UpdateIfExistsFlag,
		},
		JSONInput: `{
			"expires":31536000,
			"name":"TestPolicy",
			"summary":"A Test Policy"
		}`,
		JSONOutput: `{
			"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"name":"TestPolicy",
			"summary":"A Test Policy",
			"expires":31536000
		}`,
	},
	RunFn: cliCreatePolicy,
	Group: commands.PoliciesGroup,
}

func cliCreatePolicy(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create retention policy' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Policy Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe, in days", "expires", "", "", internal.FieldIsRetentionTimeframe)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this retention policy?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := internal.FindRetentionPolicy(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateRetentionPolicy(id, content)
			if err != nil {
				return err
			}
			commands.MSG("Updated existing retention policy")
			return cliGetPolicy(opts, t.UUID)
		}
	}

	p, err := api.CreateRetentionPolicy(content)

	if err != nil {
		return err
	}

	commands.MSG("Created new retention policy")
	return cliGetPolicy(opts, p.UUID)
}
