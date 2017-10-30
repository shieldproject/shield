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

//Edit - Modify an existing tenant
var Edit = &commands.Command{
	Summary: "Modify an existing tenant",
	RunFn:   cliEditTenant,
}

func cliEditTenant(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'edit-tenant' command")

	t, id, err := internal.FindTenant(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Display Name", "name", t.Name, "", tui.FieldIsOptional)

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
	t, err = api.UpdateTenant(id, content)
	if err != nil {
		return err
	}

	commands.MSG("Updated tenant")
	return cliGetTenant(opts, t.UUID)
}
