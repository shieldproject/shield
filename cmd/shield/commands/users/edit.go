package users

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Edit - Modify an existing user
var Edit = &commands.Command{
	Summary: "Modify an existing user",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{commands.AccountFlag},
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
	RunFn: cliEditUser,
	Group: commands.UsersGroup,
}

func cliEditUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit-user' command")

	u, id, err := internal.FindUser(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Display Name", "name", u.Name, "", tui.FieldIsOptional)
		in.NewField("System Role", "sysrole", u.SysRole, "", tui.FieldIsOptional)

		if err = in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)
	u, err = api.UpdateUser(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated user")
	return cliGetUser(opts, u.UUID)
}
