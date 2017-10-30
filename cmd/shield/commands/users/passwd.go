package users

import (
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/access"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Passwd - Change the password of a given user
var Passwd = &commands.Command{
	Summary: "Modify an existing user",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "account", Positional: true, Mandatory: false,
			Desc: `A string partially matching the name of a single account
						or a UUID exactly matching the UUID of an account.`,
		},
	},
	RunFn: cliPasswd,
}

//TODO: Update with unique passwd endpoint after cli local auth is a thing
func cliPasswd(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'passwd' command")

	_, id, err := internal.FindUser(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {

		in := tui.NewForm()
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

		if !in.Confirm("Save these changes?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)
	_, err = api.UpdateUser(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated user password")
	return nil
}
