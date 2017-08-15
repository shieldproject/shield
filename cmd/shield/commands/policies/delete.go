package policies

import (
	"strings"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

func init() {
	dPolicy := commands.Register("delete-policy", cliDeletePolicy)
	dPolicy.Summarize("Delete a retention policy")
	dPolicy.Aliases("delete retention policy", "remove retention policy", "rm retention policy")
	dPolicy.Aliases("delete policy", "remove policy", "rm policy")
	dPolicy.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.PolicyNameFlag,
		},
		JSONOutput: `{"ok":"Deleted policy"}`,
	})
	dPolicy.HelpGroup(commands.PoliciesGroup)
}

//Delete a retention policy
func cliDeletePolicy(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete retention policy' command")

	policy, id, err := internal.FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowRetentionPolicy(policy)
		if !tui.Confirm("Really delete this retention policy?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteRetentionPolicy(id); err != nil {
		return err
	}

	commands.OK("Deleted policy")
	return nil
}
