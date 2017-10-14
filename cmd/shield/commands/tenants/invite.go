package tenants

import (
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

var Invite = &commands.Command{
	Summary: "Invite a user to a tenant",
	Help: &commands.HelpInfo{
		JSONInput: `{
			users: [{
				"account":"userAccountName",
				"role":"admin",
				"uuid":"84751f04-2be2-428d-b6a3-2022c63ffaa3",
				}],
		}`,
		JSONOutput: `{"ok":"Invitations sent"}`,
	},
	RunFn: cliInviteUser,
	Group: commands.TenantsGroup,
}

func cliInviteUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'invite' command")

	var err error
	var content string

	_, id, err := internal.FindTenant(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("User UUID", "uuid", "", "", tui.FieldIsRequired)
		in.NewField("Role", "role", "", "", tui.FieldIsRequired)
		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really invite this user?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
		content = fmt.Sprintf(`{"users":[%s]}`, content)
	}

	log.DEBUG("JSON:\n  %s\n", content)

	err = api.Invite(id, content)
	if err != nil {
		return err
	}

	commands.MSG("invited user")
	return nil
}
