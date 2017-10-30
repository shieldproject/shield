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

//List - List available backup targets
var List = &commands.Command{
	Summary: "List available backup targets",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "plugin", Short: 'P', Valued: true,
			Desc: "Only show targets using the named target plugin",
		},
		commands.UsedFlag,
		commands.UnusedFlag,
		commands.FuzzyFlag,
	},
	RunFn: cliListTargets,
}

func cliListTargets(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list targets' command")
	log.DEBUG("  for plugin: '%s'", *opts.Plugin)
	log.DEBUG("  show unused? %v", *opts.Unused)
	log.DEBUG("  show in-use? %v", *opts.Used)
	if *commands.Opts.Raw {
		log.DEBUG(" fuzzy search? %v", api.MaybeBools(*commands.Opts.Fuzzy, *commands.Opts.Raw).Yes)
	}

	targets, err := api.GetTargets(api.TargetFilter{
		Name:       strings.Join(args, " "),
		Plugin:     *opts.Plugin,
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})

	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(targets)
		return nil
	}

	t := tui.NewTable("Name", "Summary", "Plugin", "Remote IP", "Configuration")
	for _, target := range targets {
		t.Row(target, target.Name, target.Summary, target.Plugin, target.Agent, internal.PrettyJSON(target.Endpoint))
	}
	t.Output(os.Stdout)
	return nil
}
