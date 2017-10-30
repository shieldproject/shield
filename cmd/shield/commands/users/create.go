package users

import (
	"os"

	fmt "github.com/jhunt/go-ansi"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/access"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new local user
var Create = &commands.Command{
	Summary: "Create a new local user",
	Flags: []commands.FlagInfo{
		commands.UpdateIfExistsFlag,
	},
	RunFn: cliCreateUser,
}

func cliCreateUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create-user' command")
	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Display Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Username", "account", "", "", tui.FieldIsRequired)
		in.NewField("System Role", "sysrole", "", "", tui.FieldIsRequired)

		err := in.Show()
		if err != nil {
			return err
		}

		userpass := ""
		for {
			a := access.SecurePrompt("%s @Y{[hidden]:} ", "Password:")
			b := access.SecurePrompt("%s @C{[confirm]:} ", "Confirm Password:")

			if a != "" && (a == b || !terminal.IsTerminal(int(os.Stdin.Fd()))) {
				fmt.Fprintf(os.Stderr, "\n")
				userpass = a
				break
			}
			fmt.Fprintf(os.Stderr, "\n@Y{oops, passwords do not match: try again }(Ctrl-C to cancel)\n\n")
		}

		pass, _ := in.NewField("Password", "password", userpass, "", tui.FieldIsRequired)
		pass.Hidden = true

		if !in.Confirm("Really create this user?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	user, err := api.CreateUser(content)
	if err != nil {
		return err
	}

	commands.MSG("Created new local user")
	return cliGetUser(opts, user.UUID)
}
