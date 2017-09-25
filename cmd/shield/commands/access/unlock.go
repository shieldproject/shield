package access

import (
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Unlock - Unlock the encryption key database
var Unlock = &commands.Command{
	Summary: "Unlock the encryption key database",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "master_password", Positional: true, Mandatory: true,
				Desc: "The master password for unlocking the key database",
			},
		},
	},
	RunFn: cliUnlock,
	Group: commands.AccessGroup,
}

func cliUnlock(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'unseal' command")

	internal.Require(len(args) == 1, "USAGE: shield unseal <master_password>")
	master := args[0]

	if err := api.Unlock(master); err != nil {
		return err
	}

	commands.OK("Unlocked encryption key database")
	return nil
}
