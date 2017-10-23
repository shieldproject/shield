package policies

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available retention policies
var List = &commands.Command{
	Summary: "List available retention policies",
	Flags: commands.FlagList{
		commands.UnusedFlag,
		commands.UsedFlag,
		commands.FuzzyFlag,
	},
	RunFn: cliListPolicies,
}

func cliListPolicies(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list retention policies' command")
	log.DEBUG("  show unused? %v", *opts.Unused)
	log.DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		log.DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	policies, err := api.GetRetentionPolicies(api.RetentionPolicyFilter{
		Name:       strings.Join(args, " "),
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(policies)
		return nil
	}

	t := tui.NewTable("Name", "Summary", "Expires in")
	for _, policy := range policies {
		t.Row(policy, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
	}
	t.Output(os.Stdout)
	return nil
}
