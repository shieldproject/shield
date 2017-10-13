package tenants

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create a new tenant
var Create = &commands.Command{
	Summary: "Create a new tenant",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.UpdateIfExistsFlag,
		},
		JSONInput: `{ 
		"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf", 
		"name":"Example User", 
	  }`,
		JSONOutput: `{ 
		"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf", 
		"name":"Example User", 
	  }`,
	},
	RunFn: cliCreateTenant,
	Group: commands.TenantsGroup,
}

func cliCreateTenant(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'create-tenant' command")
	var err error
	var content string
	if *opts.Raw {
		content, err = internal.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Tenant Name", "name", "", "", tui.FieldIsRequired)

		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really create this tenant?") {
			return internal.ErrCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	log.DEBUG("JSON:\n  %s\n", content)

	tenant, err := api.CreateTenant(content)
	if err != nil {
		return err
	}

	commands.MSG("Created new tenant")
	return cliGetTenant(opts, tenant.UUID)
}
