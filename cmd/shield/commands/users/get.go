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

//Get - Print detailed information about a local user
var Get = &commands.Command{
	Summary: "Print detailed information about a local user",
	Flags: commands.FlagList{
		commands.UserNameFlag,
	},
	RunFn: cliGetUser,
}

func cliGetUser(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'user' command")

	user, _, err := internal.FindUser(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(user)
		return nil
	}

	Show(user, *opts.ShowUUID)
	return nil
}

func Show(user api.User, showTennantUUID bool) {
	t := tui.NewReport()
	t.Add("UUID", user.UUID)
	t.Add("Name", user.Name)
	t.Add("Account", user.Account)
	t.Add("Backend", user.Backend)
	t.Add("System Role", user.SysRole)
	t.Add("Tenants", api.LocalTenantsToString(user.Tenants, showTennantUUID))

	t.Output(os.Stdout)
}
