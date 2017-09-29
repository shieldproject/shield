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

//Rotate - Rotates the encryption database keys
var Rotate = &commands.Command{
	Summary: "Rotate the encryption database keys",
	Help:    &commands.HelpInfo{},
	RunFn:   cliRotate,
	Group:   commands.AccessGroup,
}

func cliRotate(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'rotate' command")

	internal.Require(len(args) == 0, "USAGE: shield rotate")

	curmaster := SecurePrompt("%s @Y{[hidden]:} ", "current_master_password")

	newmaster := ""
	for {
		a := SecurePrompt("%s @Y{[hidden]:} ", "master_password")
		b := SecurePrompt("%s @C{[confirm]:} ", "master_password")

		if a != "" && (a == b || !terminal.IsTerminal(int(os.Stdin.Fd()))) {
			ansi.Fprintf(os.Stderr, "\n")
			newmaster = a
			break
		}
		ansi.Fprintf(os.Stderr, "\n@Y{oops, passwords do not match: try again }(Ctrl-C to cancel)\n\n")
	}
	if err := api.Rotate(curmaster, newmaster); err != nil {
		return err
	}

	commands.OK("Successfully rotated the encryption database keys")
	return nil
}
