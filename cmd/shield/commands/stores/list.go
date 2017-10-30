package stores

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available archive stores
var List = &commands.Command{
	Summary: "List available archive stores",
	Flags: commands.FlagList{
		commands.UsedFlag,
		commands.UnusedFlag,
		commands.FuzzyFlag,
	},
	RunFn: cliListStores,
}

func cliListStores(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list stores' command")
	log.DEBUG("  for plugin: '%s'", *opts.Plugin)
	log.DEBUG("  show unused? %v", *opts.Unused)
	log.DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		log.DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	stores, err := api.GetStores(api.StoreFilter{
		Name:       strings.Join(args, " "),
		Plugin:     *opts.Plugin,
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(stores)
		return nil
	}

	t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
	for _, store := range stores {
		t.Row(store, store.Name, store.Summary, store.Plugin, internal.PrettyJSON(store.Endpoint))
	}
	t.Output(os.Stdout)
	return nil
}
