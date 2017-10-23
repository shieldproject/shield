package backends

import (
	"fmt"

	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Use - Select a particular backend for use
var Use = &commands.Command{
	Summary: "Select a particular backend for use",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "name", Mandatory: true, Positional: true,
			Desc: "The name of the backend to target",
		},
	},
	RunFn: cliUseBackend,
}

func cliUseBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backend' command")

	if len(args) == 0 {
		DisplayCurrent()
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("Invalid 'backend' syntax: `shield backend <name>`")
	}
	err := config.Use(args[0])
	if err != nil {
		return err
	}

	DisplayCurrent()
	return nil
}
