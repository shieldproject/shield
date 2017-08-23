package backends

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Load loads the backend information into the api config
func Load() {
	if len(api.Cfg.Backends) == 0 {
		backend := os.Getenv("SHIELD_API")
		if *commands.Opts.Shield != "" {
			backend = *commands.Opts.Shield
		}

		if backend != "" {
			ansi.Fprintf(os.Stderr, "@C{Initializing `default` backend as `%s`}\n", backend)
			err := api.Cfg.AddBackend(backend, "default")
			if err != nil {
				ansi.Fprintf(os.Stderr, "@R{Error creating `default` backend: %s}", err)
			}
			api.Cfg.UseBackend("default")
		}
	}

	if api.Cfg.BackendURI() == "" {
		ansi.Fprintf(os.Stderr, "@R{No backend targeted. Use `shield list backends` and `shield backend` to target one}\n")
		os.Exit(1)
	}

	err := api.Cfg.Save()
	if err != nil {
		log.DEBUG("Unable to save shield config: %s", err)
	}
}

//Display displays information about the currently targeted backend to
//the screen
func Display(cfg *api.Config) {
	if cfg.BackendURI() == "" {
		ansi.Fprintf(os.Stderr, "No current SHIELD backend\n\n")
	} else {
		ansi.Fprintf(os.Stderr, "Using @G{%s} (%s) as SHIELD backend\n\n", cfg.BackendURI(), cfg.Backend)
	}
}
