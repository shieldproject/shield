package backends

import (
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
	Summary: "List configured SHIELD backends",
	Help: &commands.HelpInfo{
		JSONOutput: `[{
			"name":"mybackend",
			"uri":"https://10.244.2.2:443",
			"skip_ssl_validation":false
		}]`,
	},
	RunFn: cliListBackends,
	Group: commands.BackendsGroup,
}

type byAlias []api.Backend

func (a byAlias) Len() int           { return len(a) }
func (a byAlias) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAlias) Less(i, j int) bool { return a[i].Name < a[j].Name }

func cliListBackends(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'backends' command")

	backends := config.List()
	sort.Sort(byAlias(backends))

	if *opts.Raw {
		internal.RawJSON(backends)
		return nil
	}

	t := tui.NewTable("Name", "Backend URI")
	for _, backend := range backends {
		isCurrent := backend.Name == config.Current().Name

		if backend.SkipSSLValidation {
			backend.Name = ansi.Sprintf("%s @R{(insecure)}", backend.Name)
		}

		if isCurrent {
			backend.Name = ansi.Sprintf("@G{%s}", backend.Name)
			backend.Address = ansi.Sprintf("@G{%s}", backend.Address)
		}

		t.Row(backend, backend.Name, backend.Address)
	}
	t.Output(os.Stdout)

	return nil
}
