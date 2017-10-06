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

//Get - Print detailed information about a specific archive store
var Get = &commands.Command{
	Summary: "Print detailed information about a specific archive store",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.StoreNameFlag,
		},
		JSONOutput: `{
			"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20",
			"name":"test store",
			"summary":"a test store named \"test store\"",
			"plugin":"s3",
			"endpoint":"{ \"endpoint\": \"doesntmatter\" }"
		}`,
	},
	RunFn: cliGetStore,
	Group: commands.StoresGroup,
}

func cliGetStore(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show store' command")

	store, _, err := internal.FindStore(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(store)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(store.UUID)
		return nil
	}

	Show(store)
	return nil
}

//Show displays information about the given Store to stdout
func Show(store api.Store) {
	t := tui.NewReport()
	t.Add("Name", store.Name)
	t.Add("Summary", store.Summary)
	t.Break()

	t.Add("Plugin", store.Plugin)
	t.Add("Configuration", store.Endpoint)
	t.Output(os.Stdout)
}
