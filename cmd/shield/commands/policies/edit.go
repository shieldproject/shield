package policies

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing retention policy
var Edit = &commands.Command{
	Summary: "Modify an existing retention policy",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.PolicyNameFlag,
		},
		JSONInput: `{
			"expires":31536000,
			"name":"AnotherPolicy",
			"summary":"A Test Policy"
		}`,
		JSONOutput: `{
			"uuid":"18a446c4-c068-4c09-886c-cb77b6a85274",
			"name":"AnotherPolicy",
			"summary":"A Test Policy",
			"expires":31536000
		}`,
	},
	RunFn: cliEditPolicy,
	Group: commands.PoliciesGroup,
}

func cliEditPolicy(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit retention policy' command")

	p, id, err := internal.FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Policy Name", "name", p.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", p.Summary, "", tui.FieldIsOptional)
		in.NewField("Retention Timeframe, in days", "expires", p.Expires/86400, "", internal.FieldIsRetentionTimeframe)

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
	p, err = api.UpdateRetentionPolicy(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated retention policy")
	return cliGetPolicy(opts, p.UUID)
}
