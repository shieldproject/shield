package tokens

import (
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete token with the given name
var Delete = &commands.Command{
	Summary: "Revoke an auth token",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "NAME", Desc: "The name of the auth token to revoke",
			Positional: true, Mandatory: true,
		},
	},
	RunFn: cliDeleteToken,
}

func cliDeleteToken(opts *commands.Options, args ...string) error {
	log.DEBUG("running `revoke-token' command")
	internal.Require(len(args) == 1, "shield revoke-token NAME")

	token, uuid, err := internal.FindToken(args[0], *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(&token)
		if !tui.Confirm("Really revoke this token?") {
			return internal.ErrCanceled
		}
	}

	err = api.DeleteToken(uuid.String())
	if err != nil {
		return err
	}

	commands.OK("Revoked token")
	return nil
}
