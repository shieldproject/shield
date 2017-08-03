package main

import (
	"os"
	"strings"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available backup targets
func cliListTargets(args ...string) error {
	DEBUG("running 'list targets' command")
	DEBUG("  for plugin: '%s'", *opts.Plugin)
	DEBUG("  show unused? %v", *opts.Unused)
	DEBUG("  show in-use? %v", *opts.Used)
	if *opts.Raw {
		DEBUG(" fuzzy search? %v", api.MaybeBools(*opts.Fuzzy, *opts.Raw).Yes)
	}

	targets, err := api.GetTargets(api.TargetFilter{
		Name:       strings.Join(args, " "),
		Plugin:     *opts.Plugin,
		Unused:     api.MaybeBools(*opts.Unused, *opts.Used),
		ExactMatch: api.Opposite(api.MaybeBools(*opts.Fuzzy, *opts.Raw)),
	})

	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(targets)
	}

	t := tui.NewTable("Name", "Summary", "Plugin", "Remote IP", "Configuration")
	for _, target := range targets {
		t.Row(target, target.Name, target.Summary, target.Plugin, target.Agent, PrettyJSON(target.Endpoint))
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a specific backup target
func cliGetTarget(args ...string) error {
	DEBUG("running 'show target' command")

	target, _, err := FindTarget(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(target)
	}

	if *opts.ShowUUID {
		return RawUUID(target.UUID)
	}
	ShowTarget(target)
	return nil
}

//Create a new backup target
func cliCreateTarget(args ...string) error {
	DEBUG("running 'create target' command")

	var err error
	var content string
	if *opts.Raw {
		content, err = readall(os.Stdin)
		if err != nil {
			return err
		}

	} else {
		in := tui.NewForm()
		in.NewField("Target Name", "name", "", "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", "", "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", "", "", FieldIsPluginName)
		in.NewField("Configuration", "endpoint", "", "", tui.FieldIsRequired)
		in.NewField("Remote IP:port", "agent", "", "", tui.FieldIsRequired)
		err := in.Show()
		if err != nil {
			return err
		}

		if !in.Confirm("Really create this target?") {
			return errCanceled
		}

		content, err = in.BuildContent()
		if err != nil {
			return err
		}
	}

	DEBUG("JSON:\n  %s\n", content)

	if *opts.UpdateIfExists {
		t, id, err := FindTarget(content, true)
		if err != nil {
			return err
		}
		if id != nil {
			t, err = api.UpdateTarget(id, content)
			if err != nil {
				return err
			}
			MSG("Updated existing target")
			return cliGetTarget(t.UUID)
		}
	}
	t, err := api.CreateTarget(content)
	if err != nil {
		return err
	}
	MSG("Created new target")
	return cliGetTarget(t.UUID)
}

//Modify an existing backup target
func cliEditTarget(args ...string) error {
	DEBUG("running 'edit target' command")

	t, id, err := FindTarget(strings.Join(args, " "), *opts.Raw)
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
		in.NewField("Target Name", "name", t.Name, "", tui.FieldIsRequired)
		in.NewField("Summary", "summary", t.Summary, "", tui.FieldIsOptional)
		in.NewField("Plugin Name", "plugin", t.Plugin, "", FieldIsPluginName)
		in.NewField("Configuration", "endpoint", t.Endpoint, "", tui.FieldIsRequired)
		in.NewField("Remote IP:port", "agent", t.Agent, "", tui.FieldIsRequired)

		if err := in.Show(); err != nil {
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
	t, err = api.UpdateTarget(id, content)
	if err != nil {
		return err
	}

	MSG("Updated target")
	return cliGetTarget(t.UUID)
}

//Delete a backup target
func cliDeleteTarget(args ...string) error {
	DEBUG("running 'delete target' command")

	target, id, err := FindTarget(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowTarget(target)
		if !tui.Confirm("Really delete this target?") {
			return errCanceled
		}
	}

	if err := api.DeleteTarget(id); err != nil {
		return err
	}

	OK("Deleted target")
	return nil
}
