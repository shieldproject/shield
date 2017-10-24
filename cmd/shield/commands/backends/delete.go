package backends

import (
	"os"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Delete - Delete a SHIELD backend
var Delete = &commands.Command{
	Summary: "Delete a SHIELD backend alias",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "name", Mandatory: true, Positional: true,
			Desc: `The name of the backend to delete`,
		},
	},
	RunFn: cliDeleteBackend,
}

func cliDeleteBackend(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete-backend' command")

	if len(args) != 1 {
		return fmt.Errorf("Invalid 'delete-backend' syntax: `shield delete-backend <name>`")
	}

	name := args[0]
	err := config.Delete(name)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Successfully deleted backend '@G{%s}'\n", name)
	if config.Current() == nil {
		fmt.Fprintf(os.Stdout, "@Y{You are no longer targeting any backend}\n")
	}

	fmt.Println("")

	return nil
}
