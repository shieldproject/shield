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

//Get - Print detailed information about a specific retention policy
var Get = &commands.Command{
	Summary: "Print detailed information about a specific retention policy",
	Flags: commands.FlagList{
		commands.PolicyNameFlag,
	},
	RunFn: cliGetPolicy,
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

	Show(policy)
	return nil
}

//Show displays information about the given retention policy to stdout
func Show(policy api.RetentionPolicy) {
	t := tui.NewReport()
	t.Add("Name", policy.Name)
	t.Add("Summary", policy.Summary)
	t.Add("Expiration", fmt.Sprintf("%d days", policy.Expires/86400))
	t.Output(os.Stdout)
}
