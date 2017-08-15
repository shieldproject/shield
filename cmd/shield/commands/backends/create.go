package backends

import (
	"fmt"
	"os"

	"github.com/geofffranks/spruce/log"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
)

func init() {
	cbackend := commands.Register("create-backend", cliCreateBackend)
	cbackend.Aliases("create backend", "c be", "update backend", "update-backend", "edit-backend", "edit backend")
	cbackend.Summarize("Create or modify a SHIELD backend")
	cbackend.Help(commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "name", Mandatory: true, Positional: true,
				Desc: `The name of the new backend`,
			},
			commands.FlagInfo{
				Name: "uri", Mandatory: true, Positional: true,
				Desc: `The address at which the new backend can be found`,
			},
		},
	})
	cbackend.HelpGroup(commands.BackendsGroup)
}

//Create or modify a SHIELD backend
func cliCreateBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create backend' command")

	if len(args) != 2 {
		return fmt.Errorf("Invalid 'create backend' syntax: `shield backend <name> <uri>")
	}
	err := api.Cfg.AddBackend(args[1], args[0])
	if err != nil {
		return err
	}

	err = api.Cfg.UseBackend(args[0])
	if err != nil {
		return err
	}

	err = api.Cfg.Save()
	if err != nil {
		return err
	}

	ansi.Fprintf(os.Stdout, "Successfully created backend '@G{%s}', pointing to '@G{%s}'\n\n", args[0], args[1])
	Display(api.Cfg)

	return nil
}
