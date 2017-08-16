package stores

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
)

func init() {
	store := commands.Register("store", cliGetStore)
	store.Summarize("Print detailed information about a specific archive store")
	store.Aliases("show store", "view store", "display store", "list store", "ls store")
	store.Help(commands.HelpInfo{
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
	})
	store.HelpGroup(commands.StoresGroup)
}

//Print detailed information about a specific archive store
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

	internal.ShowStore(store)
	return nil
}
