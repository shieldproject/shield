package users

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List shield users
var List = &commands.Command{
	Summary: "List shield users",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "sysrole", Short: 'r', Valued: true,
				Desc: "Show only users with the specified system role.",
			},
			commands.FlagInfo{
				Name: "account", Valued: true,
				Desc: "Show only users with the specified account.",
			},
		},
		JSONOutput: `[{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"Example User",
			"account":"exampleuser"
			"sysrole":"admin/manager/technician"
		}]`,
	},
	RunFn: cliListUsers,
	Group: commands.UsersGroup,
}

func cliListUsers(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'users' command")

	if *opts.Limit == "" {
		*opts.Limit = "20"
	}
	log.DEBUG("  for limit: '%s'", *opts.Limit)

	users, err := api.GetUsers(api.UserFilter{
		SysRole: *opts.SysRole,
		Account: *opts.Account,
		Limit:   *opts.Limit,
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(users)
		return nil
	}

	if *opts.ShowUUID {
		t := tui.NewTable("UUID", "Name", "Account", "System Role", "Tennants")
		for _, user := range users {
			t.Row(user, user.UUID, user.Name, user.Account, user.SysRole, api.LocalTenantsToString(user.Tenants, *opts.ShowUUID))
		}
		t.Output(os.Stdout)
	} else {
		t := tui.NewTable("Name", "Account", "System Role", "Tennants")
		for _, user := range users {
			t.Row(user, user.Name, user.Account, user.SysRole, api.LocalTenantsToString(user.Tenants, *opts.ShowUUID))
		}
		t.Output(os.Stdout)
	}

	return nil
}
