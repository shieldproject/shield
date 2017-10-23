package access

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"golang.org/x/crypto/ssh/terminal"
)

//Init - Initializes the encryption key database
var Init = &commands.Command{
	Summary: "Initialize the encryption key database",
	RunFn:   cliInit,
}

func cliInit(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'init' command")

	internal.Require(len(args) == 0, "USAGE: shield init")
	master := ""
	for {
		a := SecurePrompt("%s @Y{[hidden]:} ", "New Master Password")
		b := SecurePrompt("%s @C{[confirm]:} ", "New Master Password")

		if a != "" && (a == b || !terminal.IsTerminal(int(os.Stdin.Fd()))) {
			ansi.Fprintf(os.Stderr, "\n")
			master = a
			break
		}
		ansi.Fprintf(os.Stderr, "\n@Y{oops, passwords do not match: try again }(Ctrl-C to cancel)\n\n")
	}

	if err := api.Init(master); err != nil {
		return err
	}

	commands.OK("Initialized encryption key database")
	return nil
}
