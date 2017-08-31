package backends

import (
	"fmt"
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Create - Create or modify a SHIELD backend
var Create = &commands.Command{
	Summary: "Create or modify a SHIELD backend alias",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "name", Mandatory: true, Positional: true,
				Desc: `The name of the new backend`,
			},
			commands.FlagInfo{
				Name: "uri", Mandatory: true, Positional: true,
				Desc: `The address at which the new backend can be found`,
			},
			commands.FlagInfo{
				Name: "skip-ssl-validation", Short: 'k',
				Desc: `If this flag is specified, SSL validation will always be skipped
				when using this backend`,
			},
		},
	},
	RunFn: cliCreateBackend,
	Group: commands.BackendsGroup,
}

func cliCreateBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create backend' command")

	if len(args) != 2 {
		return fmt.Errorf("Invalid 'create-backend' syntax: `shield create-backend <name> <uri>")
	}

	name := args[0]
	uri := args[1]
	err := config.Commit(&api.Backend{
		Name:              name,
		Address:           uri,
		SkipSSLValidation: *opts.SkipSSLValidation,
	})
	if err != nil {
		return err
	}

	err = config.Use(name)
	if err != nil {
		return err
	}

	ansi.Fprintf(os.Stdout, "Successfully created backend '@G{%s}', pointing to '@G{%s}'\n\n", args[0], args[1])
	DisplayCurrent()

	return nil
}
