package users

import (
	"os"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Get - Print detailed information about a local user
var Get = &commands.Command{
	Summary: "Print detailed information about a local user",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "uuid", Positional: true, Mandatory: true,
				Desc: "A UUID assigned to a single local user",
			},
		},
		JSONOutput: `{
			"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf",
			"name":"Example User",
			"account":"exampleuser",
			"sysrole":"admin/manager/technician"
		}`,
	},
	RunFn: cliGetUser,
	Group: commands.UsersGroup,
}

func cliGetUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'user' command")

	internal.Require(len(args) == 1, "shield user <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  user UUID = '%s'", id)

	user, err := api.GetUser(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(user)
		return nil
	}

	ShowUser(user, *opts.ShowUUID)
	return nil
}

func ShowUser(user api.User, showTennantUUID bool) {
	t := tui.NewReport()
	t.Add("UUID", user.UUID)
	t.Add("Name", user.Name)
	t.Add("Account", user.Account)
	t.Add("Backend", user.Backend)
	t.Add("System Role", user.SysRole)
	t.Add("Tennants", api.LocalTenantsToString(user.Tenants, showTennantUUID))

	t.Output(os.Stdout)
}
