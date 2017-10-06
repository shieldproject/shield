package users

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/access"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
	"golang.org/x/crypto/ssh/terminal"
)

//Create - Create a new local user
var Create = &commands.Command{
	Summary: "Create a new local user",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.UpdateIfExistsFlag,
		},
		JSONInput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"Example User",
			"account":"exampleuser"
			"password":"foobar"
			"sysrole":"admin/manager/technician"
		}`,
		JSONOutput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"Example User",
			"account":"exampleuser"
			"sysrole":"admin/manager/technician"
		}`,
	},
	RunFn: cliCreateUser,
	Group: commands.UsersGroup,
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
				ansi.Fprintf(os.Stderr, "\n")
				userpass = a
				break
			}
			ansi.Fprintf(os.Stderr, "\n@Y{oops, passwords do not match: try again }(Ctrl-C to cancel)\n\n")
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
