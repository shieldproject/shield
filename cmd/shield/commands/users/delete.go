package users

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a local user
var Delete = &commands.Command{
	Summary: "Delete a local user",
	Flags: commands.FlagList{
		commands.AccountFlag,
	},
	RunFn: cliDeleteUser,
}

func cliDeleteUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete-user' command")

	user, id, err := internal.FindUser(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(user, *opts.ShowUUID)
		if !tui.Confirm("Really delete this user?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteUser(id); err != nil {
		return err
	}

	commands.OK("Deleted user")
	return nil
}
