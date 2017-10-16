package tenants

import (
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a tenant
var Delete = &commands.Command{
	Summary: "Delete a tenant",
	Help: &commands.HelpInfo{
		JSONOutput: `{"ok":"Deleted Tenant"}`,
	},
	RunFn: cliDeleteTenant,
	Group: commands.TenantsGroup,
}

func cliDeleteTenant(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete-tenant' command")

	tenant, id, err := internal.FindTenant(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(tenant, *opts.ShowUUID)
		if !tui.Confirm("Really delete this tenant?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteTenant(id); err != nil {
		return err
	}

	commands.OK("Deleted tenant")
	return nil
}
