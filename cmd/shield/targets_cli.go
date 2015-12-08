package main

import (
	//"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

type ListTargetOptions struct {
	Unused bool
	Used   bool
	Plugin string
	UUID   string
}

func ListTargets(opts ListTargetOptions) error {
	//FIXME (un?)used flags do not work; --plugin name works fine.
	targets, err := GetTargets(TargetFilter{
		Plugin: opts.Plugin,
		Unused: MaybeBools(opts.Unused, opts.Used),
	})

	if err != nil {
		return fmt.Errorf("failed to retrieve targets from SHIELD: %s", err)
	}

	t := tui.NewTable("UUID", "Target Name", "Description", "Plugin", "Endpoint", "SHIELD Agent")
	for _, target := range targets {
		//FIXME implement with GetTarget(UUID)
		if len(opts.UUID) > 0 && opts.UUID == target.UUID {
			t.Row(target.UUID, target.Name, target.Summary, target.Plugin, target.Endpoint, target.Agent)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != target.UUID {
			continue
		}
		t.Row(target.UUID, target.Name, target.Summary, target.Plugin, target.Endpoint, target.Agent)
	}
	t.Output(os.Stdout)
	return nil
}

func CreateNewTarget() error {
	content := invokeEditor(`{
	"name":     "Empty Target",
	"summary":  "How many licks does it take to reach the center",
	"plugin":   "of a tootsie pop",
	"endpoint": "{\"the world\":\"may never know\"}",
	"agent":    "blackwidow"
  }`)

	newTarget, err := CreateTarget(content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not create new target: %s", err)
	}

	fmt.Fprintf(os.Stdout, "Created new target.\n")
	t := tui.NewTable("UUID", "Target Name", "Description", "Plugin", "Endpoint", "SHIELD Agent")
	t.Row(newTarget.UUID, newTarget.Name, newTarget.Summary, newTarget.Plugin, newTarget.Endpoint, newTarget.Agent)
	t.Output(os.Stdout)
	return nil
}

func EditExstingTarget(u string) error {
	t, err := GetTarget(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Could not retrieve target '%s': %s", u, err)
	}
	content := invokeEditor(`{
	"name":     "` + t.Name + `",
	"summary":  "` + t.Summary + `",
	"plugin":   "` + t.Plugin + `",
	"endpoint": "` + t.Endpoint + `",
	"agent":    "` + t.Agent + `"
  }`)

	t, err = UpdateTarget(uuid.Parse(u), content)
	if err != nil {
		return fmt.Errorf("ERROR: Could not update target '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Updated target.\n")
	table := tui.NewTable("UUID", "Target Name", "Description", "Plugin", "Endpoint", "SHIELD Agent")
	table.Row(t.UUID, t.Name, t.Summary, t.Plugin, t.Endpoint, t.Agent)
	table.Output(os.Stdout)
	return nil
}

func DeleteTargetByUUID(u string) error {
	err := DeleteTarget(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Could not delete target '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted target '%s'\n", u)
	return nil
}
