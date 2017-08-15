package stores

import (
	"strings"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

func init() {
	dStore := commands.Register("delete-store", cliDeleteStore)
	dStore.Summarize("Delete an archive store")
	dStore.Aliases("delete store", "remove store", "rm store")
	dStore.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.StoreNameFlag,
		},
		JSONOutput: `{"ok":"Deleted store"}`,
	})
	dStore.HelpGroup(commands.StoresGroup)
}

//Delete an archive store
func cliDeleteStore(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete store' command")

	store, id, err := internal.FindStore(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowStore(store)
		if !tui.Confirm("Really delete this store?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteStore(id); err != nil {
		return err
	}

	commands.OK("Deleted store")
	return nil
}
