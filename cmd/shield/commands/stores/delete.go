package stores

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete an archive store
var Delete = &commands.Command{
	Summary: "Delete an archive store",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.StoreNameFlag,
		},
		JSONOutput: `{"ok":"Deleted store"}`,
	},
	RunFn: cliDeleteStore,
	Group: commands.StoresGroup,
}

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
