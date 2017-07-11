package main

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available archive stores
func cliListStores(opts Options, args []string, help bool) error {
	if help {
		HelpListMacro("store", "stores")
		FlagHelp("Only show stores using the named store plugin", true, "-P", "--policy=value")
		JSONHelp(`[{"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20","name":"test store","summary":"a test store named \"test store\"","plugin":"s3","endpoint":"{ \"endpoint\": \"doesntmatter\" }"}]`)
		return nil
	}

	DEBUG("running 'list stores' command")
	DEBUG("  for plugin: '%s'", *opts.Plugin)
	DEBUG("  show unused? %v", *opts.Unused)
	DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	stores, err := api.GetStores(api.StoreFilter{
		Name:       strings.Join(args, " "),
		Plugin:     *opts.Plugin,
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(stores)
	}

	t := tui.NewTable("Name", "Summary", "Plugin", "Configuration")
	for _, store := range stores {
		t.Row(store, store.Name, store.Summary, store.Plugin, PrettyJSON(store.Endpoint))
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a specific archive store
func cliGetStore(opts Options, args []string, help bool) error {
	if help {
		JSONHelp(`{"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20","name":"test store","summary":"a test store named \"test store\"","plugin":"s3","endpoint":"{ \"endpoint\": \"doesntmatter\" }"}`)
		HelpShowMacro("store", "stores")
		return nil
	}

	DEBUG("running 'show store' command")

	store, _, err := FindStore(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(store)
	}
	if *opts.ShowUUID {
		return RawUUID(store.UUID)
	}

	ShowStore(store)
	return nil
}

//Create a new archive store
func cliCreateStore(opts Options, args []string, help bool) error {
	if help {
		HelpCreateMacro("store", "stores")
		InputHelp(`{"endpoint":"{\"endpoint\":\"schmendpoint\"}","name":"TestStore","plugin":"s3","summary":"A Test Store"}`)
		JSONHelp(`{"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","name":"TestStore","summary":"A Test Store","plugin":"s3","endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
		return nil
	}

	DEBUG("running 'create store' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Store Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", "", "", FieldIsPluginName)
		in.NewField("Configuration (JSON)", "endpoint", "", "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
			return err
		}

		if !in.Confirm("Really create this archive store?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := FindStore(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateStore(id, content)
			if err != nil {
				return err
			}
			MSG("Updated existing store")
			return cliGetStore(opts, []string{t.UUID}, false)
		}
	}

	s, err := api.CreateStore(content)

	if err != nil {
		return err
	}

	MSG("Created new store")
	return cliGetStore(opts, []string{s.UUID}, false)
}

//Modify an existing archive store
func cliEditStore(opts Options, args []string, help bool) error {
	if help {
		HelpEditMacro("store", "stores")
		InputHelp(`{"endpoint":"{\"endpoint\":\"schmendpoint\"}","name":"AnotherStore","plugin":"s3","summary":"A Test Store"}`)
		JSONHelp(`{"uuid":"355ccd3f-1d2f-49d5-937b-f4a12033a0cf","name":"AnotherStore","summary":"A Test Store","plugin":"s3","endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
		return nil
	}

	DEBUG("running 'edit store' command")

	s, id, err := FindStore(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Store Name", "name", s.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", s.Summary, "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", s.Plugin, "", FieldIsPluginName)
		in.NewField("Configuration (JSON)", "endpoint", s.Endpoint, "", tui.FieldIsRequired)

		err = in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Save these changes?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)
	s, err = api.UpdateStore(id, content)
	if err != nil {
		return err
	}

	MSG("Updated store")
	return cliGetStore(opts, []string{s.UUID}, false)
}

//Delete an archive store
func cliDeleteStore(opts Options, args []string, help bool) error {
	if help {
		HelpDeleteMacro("store", "stores")
		return nil
	}

	DEBUG("running 'delete store' command")

	store, id, err := FindStore(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowStore(store)
		if !tui.Confirm("Really delete this store?") {
			return errCanceled
		}
	}

	if err := api.DeleteStore(id); err != nil {
		return err
	}

	OK("Deleted store")
	return nil
}
