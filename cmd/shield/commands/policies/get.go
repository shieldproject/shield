package policies

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

var Get = &commands.Command{
	Summary: "Print detailed information about a specific retention policy",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.PolicyNameFlag,
		},
		JSONOutput: `{
			"uuid":"8c6f894f-9c27-475f-ad5a-8c0db37926ec",
			"name":"apolicy",
			"summary":"a policy",
			"expires":5616000
		}`,
	},
	RunFn: cliGetPolicy,
	Group: commands.PoliciesGroup,
}

func cliGetPolicy(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show retention policy' command")

	policy, _, err := internal.FindRetentionPolicy(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(policy)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(policy.UUID)
		return nil
	}

	internal.ShowRetentionPolicy(policy)
	return nil
}
