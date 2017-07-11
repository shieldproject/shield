package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List configured SHIELD backends
func cliListBackends(opts Options, args []string, help bool) error {
	if help {
		JSONHelp(`[{"name":"mybackend","uri":"https://10.244.2.2:443"}]`)
		FlagHelp("Outputs information as JSON object", true, "--raw")
		return nil
	}

	DEBUG("running 'backends' command")

	var indices []string
	for key := range api.Cfg.Aliases {
		indices = append(indices, key)
	}
	sort.Strings(indices)

	if *opts.Raw {
		arr := []map[string]string{}
		for _, alias := range indices {
			arr = append(arr, map[string]string{"name": alias, "uri": api.Cfg.Aliases[alias]})
		}
		return RawJSON(arr)
	}

	t := tui.NewTable("Name", "Backend URI")
	for _, alias := range indices {
		be := map[string]string{"name": alias, "uri": api.Cfg.Aliases[alias]}
		t.Row(be, be["name"], be["uri"])
	}
	t.Output(os.Stdout)

	return nil
}

//Create or modify a SHIELD backend
func cliCreateBackend(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`The name of the new backend`, false, "<name>")
		FlagHelp(`The address at which the new backend can be found`, false, "<uri>")

		return nil
	}

	DEBUG("running 'create backend' command")

	if len(args) != 2 {
		return fmt.Errorf("Invalid 'create backend' syntax: `shield backend <name> <uri>")
	}
	err := api.Cfg.AddBackend(args[1], args[0])
	if err != nil {
		return err
	}

	err = api.Cfg.UseBackend(args[0])
	if err != nil {
		return err
	}

	err = api.Cfg.Save()
	if err != nil {
		return err
	}

	ansi.Fprintf(os.Stdout, "Successfully created backend '@G{%s}', pointing to '@G{%s}'\n\n", args[0], args[1])
	DisplayBackend(api.Cfg)

	return nil
}

//Select a particular backend for use
func cliUseBackend(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`The name of the backend to target`, false, "<name>")
		return nil
	}

	DEBUG("running 'backend' command")

	if len(args) == 0 {
		DisplayBackend(api.Cfg)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("Invalid 'backend' syntax: `shield backend <name>`")
	}
	err := api.Cfg.UseBackend(args[0])
	if err != nil {
		return err
	}
	api.Cfg.Save()

	DisplayBackend(api.Cfg)
	return nil
}

func loadBackend() {
	if len(api.Cfg.Backends) == 0 {
		backend := os.Getenv("SHIELD_API")
		if *options.Shield != "" {
			backend = *options.Shield
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
		DEBUG("Unable to save shield config: %s", err)
	}
}
