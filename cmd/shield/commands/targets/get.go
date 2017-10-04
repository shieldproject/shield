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

//Get - Print detailed information about a specific backup target
var Get = &commands.Command{
	Summary: "Print detailed information about a specific backup target",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.TargetNameFlag,
			commands.FlagInfo{
				Name: "uuid", Desc: "Return UUID of target",
			},
		},
		JSONOutput: `{
			"uuid":"8add3e57-95cd-4ec0-9144-4cd5c50cd392",
			"name":"SampleTarget",
			"summary":"A Sample Target",
			"plugin":"postgres",
			"endpoint":"{\"endpoint\":\"127.0.0.1:5432\"}",
			"agent":"127.0.0.1:1234"
		}`,
	},
	RunFn: cliGetTarget,
	Group: commands.TargetsGroup,
}

func cliGetTarget(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show target' command")

	target, _, err := internal.FindTarget(strings.Join(args, " "), *commands.Opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(target)
		return nil
	}

	if *opts.ShowUUID {
		internal.RawUUID(target.UUID)
		return nil
	}
	Show(target)
	return nil
}

//Show prints information about the given Target to stdout
func Show(target api.Target) {
	t := tui.NewReport()
	t.Add("Name", target.Name)
	t.Add("Summary", target.Summary)
	t.Break()

	t.Add("Plugin", target.Plugin)
	t.Add("Configuration", target.Endpoint)
	t.Add("Remote IP", target.Agent)
	t.Output(os.Stdout)
}
