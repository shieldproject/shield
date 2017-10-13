package tenants

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
	Summary: "List shield tenants",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "name", Short: 'n', Valued: true,
				Desc: "Show only tenants with the specified name",
			},
		},
		JSONOutput: `[{ 
		"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf", 
		"name":"Example Tenant", 
	  }]`,
	},
	RunFn: cliListTenants,
	Group: commands.TenantsGroup,
}

func cliListTenants(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'tenants' command")

	if *opts.Limit == "" {
		*opts.Limit = "20"
	}
	log.DEBUG("  for limit: '%s'", *opts.Limit)

	tenants, err := api.GetTenants(api.TenantFilter{
		Limit: *opts.Limit,
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(tenants)
		return nil
	}

	t := tui.NewTable("UUID", "Name")
	for _, tenant := range tenants {
		t.Row(tenant, tenant.UUID, tenant.Name)
	}
	t.Output(os.Stdout)

	return nil
}
