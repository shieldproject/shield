package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pborman/uuid"
	"github.com/spf13/cobra"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Targets

	editTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Edit all the Targets",
	}
)

func init() {

	// Hookup functions to the subcommands
	editTargetCmd.Run = processEditTargetRequest

	// Add the subcommands to the base actions
	editCmd.AddCommand(editTargetCmd)
}

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

func DeleteTargetByUUID(u string) error {
	err := DeleteTarget(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Could not delete target '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted target '%s'\n", u)
	return nil
}

func processEditTargetRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	original_data, err := GetTarget(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show target:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render target:\n", err)
	}

	fmt.Println("Got the following original target:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited target:\n\n", content)

	update_data, err := UpdateTarget(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update targets:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render target:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}
