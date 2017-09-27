package backends

import (
	"fmt"
	"os"
	"sort"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List configured SHIELD backends
var List = &commands.Command{
	Summary: "List configured SHIELD backend aliases",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "full", Desc: "Display verbose information about all backends",
			},
		},
		JSONOutput: `[{
			"name":"mybackend",
			"uri":"https://10.244.2.2:443",
			"skip_ssl_validation":false
		}]`,
	},
	RunFn: cliListBackends,
	Group: commands.BackendsGroup,
}

func cliListBackends(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backends' command")

	backends := config.List()
	sort.Slice(backends, func(i, j int) bool { return backends[i].Name < backends[j].Name })

	if *opts.Raw {
		internal.RawJSON(backends)
		return nil
	}

	var t *tui.Table
	if *opts.Full {
		t = verboseList(backends)
	} else {
		t = conciseList(backends)
	}

	t.Output(os.Stdout)

	return nil
}

func conciseList(backends []*api.Backend) *tui.Table {
	t := tui.NewTable("Name", "Backend URI")

	for _, backend := range backends {
		isCurrent := config.Current() != nil && backend.Name == config.Current().Name

		if backend.SkipSSLValidation {
			backend.Name = ansi.Sprintf("%s @R{(insecure)}", backend.Name)
		}

		if isCurrent {
			backend.Name = greenify(backend.Name)
			backend.Address = greenify(backend.Address)
		}

		t.Row(backend, backend.Name, backend.Address)
	}

	return &t
}

func verboseList(backends []*api.Backend) *tui.Table {
	t := tui.NewTable("Name", "Backend URI", "Insecure", "Token", "CA Cert")

	for _, backend := range backends {
		isCurrent := config.Current() != nil && backend.Name == config.Current().Name

		isInsecure := fmt.Sprintf("%t", backend.SkipSSLValidation)

		if isCurrent {
			backend.Name = greenify(backend.Name)
			backend.Address = greenify(backend.Address)
			backend.Token = greenify(backend.Token)
			backend.CACert = greenify(backend.CACert)
			isInsecure = greenify(isInsecure)
		}

		if backend.SkipSSLValidation {
			isInsecure = redify(isInsecure)
		}

		t.Row(backend, backend.Name, backend.Address, isInsecure, backend.Token, backend.CACert)
	}

	return &t
}

func greenify(s string) string {
	return ansi.Sprintf("@G{%s}", s)
}

func redify(s string) string {
	return ansi.Sprintf("@R{%s}", s)
}
