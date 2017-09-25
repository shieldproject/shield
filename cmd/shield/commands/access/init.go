package access

import (
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Init - Initializes the encryption key database
var Init = &commands.Command{
	Summary: "Initialize the encryption key database",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "master_password", Positional: true, Mandatory: true,
				Desc: "The master password for initializing the key database",
			},
		},
	},
	RunFn: cliInit,
	Group: commands.AccessGroup,
}

func cliInit(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'init' command")

	internal.Require(len(args) == 1, "USAGE: shield init <master_password>")
	master := args[0]

	if err := api.Init(master); err != nil {
		return err
	}

	commands.OK("Initialized encryption key database")
	return nil
}
