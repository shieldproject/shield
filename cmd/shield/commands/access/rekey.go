package access

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Rekey - Rekeys the encryption database keys
var Rekey = &commands.Command{
	Summary: "Rekey the encryption database keys",
	RunFn:   cliRekey,
}

func cliRekey(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'rekey' command")

	internal.Require(len(args) == 0, "USAGE: shield rekey")

	curmaster := SecurePrompt("%s @Y{[hidden]:} ", "Current Master Password")

	newmaster := ""
	for {
		a := SecurePrompt("%s @Y{[hidden]:} ", "New Master Password")
		b := SecurePrompt("%s @C{[confirm]:} ", "New Master Password")

		if a != "" && (a == b || !terminal.IsTerminal(int(os.Stdin.Fd()))) {
			fmt.Fprintf(os.Stderr, "\n")
			newmaster = a
			break
		}
		fmt.Fprintf(os.Stderr, "\n@Y{oops, passwords do not match: try again }(Ctrl-C to cancel)\n\n")
	}
	if err := api.Rekey(curmaster, newmaster); err != nil {
		return err
	}

	commands.OK("Successfully rekeyed the encryption database")
	return nil
}
