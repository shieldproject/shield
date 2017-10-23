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
	Flags: commands.FlagList{
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
        when using this backend. Not compatible with --ca-cert`,
		},
		commands.FlagInfo{
			Name: "ca-cert",
			Desc: `If this flag is given, this backend will always trust the root CA
        cert found in the given file. Not compatible with --skip-ssl-validation`,
		},
	},
	RunFn: cliCreateBackend,
}

func cliCreateBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create backend' command")

	if len(args) != 2 {
		return fmt.Errorf("Invalid 'create-backend' syntax: `shield create-backend <name> <uri>")
	}

	name := args[0]
	uri := args[1]

	toCommit := &api.Backend{
		Name:              name,
		Address:           uri,
		Token:             config.TokenForURI(uri),
		SkipSSLValidation: *opts.SkipSSLValidation,
	}

	if *opts.CACert != "" {
		var err error
		toCommit.CACert, err = ParseCACertFlag(*opts.CACert)
		if err != nil {
			return fmt.Errorf("could not use CA Cert: %s", err.Error())
		}

		if *opts.SkipSSLValidation {
			return fmt.Errorf("cannot use CA Cert flag and Skip SSL Validation flag")
		}
	}

	err := config.Commit(toCommit)
	if err != nil {
		return err
	}

	err = config.Use(name)
	if err != nil {
		panic("We just created this backend. Why can't we use it?")
	}

	ansi.Fprintf(os.Stdout, "Successfully created backend '@G{%s}', pointing to '@G{%s}'\n\n", args[0], args[1])
	DisplayCurrent()

	return nil
}
