package info

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

//Status - Query the SHIELD backup server for its status and version info
var Status = &commands.Command{
	Summary: "Query the SHIELD backup server for its status and version info",
	RunFn:   cliStatus,
}

func cliStatus(opts *commands.Options, args ...string) error {
	status, err := api.GetStatus()
	if err != nil {
		return err
	}

	if *commands.Opts.Raw {
		internal.RawJSON(map[string]string{
			"name":    status.Name,
			"version": status.Version,
		})
	} else {
		t := tui.NewReport()
		t.Add("Name", status.Name)
		t.Add("API Version", status.Version)
		t.Output(os.Stdout)
	}

	return nil
}
