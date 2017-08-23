package backends

import (
	"os"
	"sort"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List configured SHIELD backends
var List = &commands.Command{
	Summary: "List configured SHIELD backends",
	Help: &commands.HelpInfo{
		JSONOutput: `[{
			"name":"mybackend",
			"uri":"https://10.244.2.2:443"
		}]`,
	},
	RunFn: cliListBackends,
	Group: commands.BackendsGroup,
}

func cliListBackends(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backends' command")

	var indices []string
	for key := range api.Cfg.Aliases {
		indices = append(indices, key)
	}
	sort.Strings(indices)

	if *opts.Raw {
		arr := []map[string]string{}
		for _, alias := range indices {
			arr = append(arr, map[string]string{"name": alias, "uri": api.Cfg.Aliases[alias]})
		}
		internal.RawJSON(arr)
		return nil
	}

	t := tui.NewTable("Name", "Backend URI")
	for _, alias := range indices {
		be := map[string]string{"name": alias, "uri": api.Cfg.Aliases[alias]}
		t.Row(be, be["name"], be["uri"])
	}
	t.Output(os.Stdout)

	return nil
}
