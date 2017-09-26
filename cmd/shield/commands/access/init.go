package access

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Init - Initializes the encryption key database
var Init = &commands.Command{
	Summary: "Initialize the encryption key database",
	Help:    &commands.HelpInfo{},
	RunFn:   cliInit,
	Group:   commands.AccessGroup,
}

func cliInit(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'init' command")

	internal.Require(len(args) == 0, "USAGE: shield init")
	master := ""
	for {
		a := SecurePrompt("%s @Y{[hidden]:} ", "master_password")
		b := SecurePrompt("%s @C{[confirm]:} ", "master_password")

		if a == b && a != "" {
			ansi.Fprintf(os.Stderr, "\n")
			master = a
			break
		}
		ansi.Fprintf(os.Stderr, "\n@Y{oops, try again }(Ctrl-C to cancel)\n\n")
	}

	if err := api.Init(master); err != nil {
		return err
	}

	commands.OK("Initialized encryption key database")
	return nil
}
