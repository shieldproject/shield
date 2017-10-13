package tenants

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
	Summary: "Print detailed information about a tenant",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "uuid", Positional: true, Mandatory: true,
				Desc: "A UUID assigned to a tenant",
			},
		},
		JSONOutput: `{ 
		"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf", 
		"name":"Example Tenant", 
	  }`,
	},
	RunFn: cliGetTenant,
	Group: commands.TenantsGroup,
}

func cliGetTenant(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'tenant' command")

	internal.Require(len(args) == 1, "shield tenant <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  tenant UUID = '%s'", id)

	user, err := api.GetTenant(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(user)
		return nil
	}

	ShowTenant(user, *opts.ShowUUID)
	return nil
}

func ShowTenant(tenant api.Tenant, showTennantUUID bool) {
	t := tui.NewReport()
	t.Add("UUID", tenant.UUID)
	t.Add("Name", tenant.Name)

	t.Output(os.Stdout)
}
