package policies

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a retention policy
var Delete = &commands.Command{
	Summary: "Delete a retention policy",
	Flags: commands.FlagList{
		commands.PolicyNameFlag,
	},
	RunFn: cliDeletePolicy,
}

func cliDeletePolicy(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete retention policy' command")

	policy, id, err := internal.FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(policy)
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
