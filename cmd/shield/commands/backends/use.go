package backends

import (
	"fmt"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Use - Select a particular backend for use
var Use = &commands.Command{
	Summary: "Select a particular backend for use",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "name", Mandatory: true, Positional: true,
				Desc: "The name of the backend to target",
			},
		},
	},
	RunFn: cliUseBackend,
	Group: commands.BackendsGroup,
}

func cliUseBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backend' command")

	if len(args) == 0 {
		Display(api.Cfg)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("Invalid 'backend' syntax: `shield backend <name>`")
	}
	err := api.Cfg.UseBackend(args[0])
	if err != nil {
		return err
	}
	api.Cfg.Save()

	Display(api.Cfg)
	return nil
}
