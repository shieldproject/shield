package tenants

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Get - Print detailed information about a local tenant
var Get = &commands.Command{
	Summary: "Print detailed information about a tenant",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.TenantNameFlag,
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

	tenant, _, err := internal.FindTenant(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(tenant)
		return nil
	}

	Show(tenant, *opts.ShowUUID)
	return nil
}

func Show(tenant api.Tenant, showTennantUUID bool) {
	t := tui.NewReport()
	t.Add("UUID", tenant.UUID)
	t.Add("Name", tenant.Name)

	t.Output(os.Stdout)
}
