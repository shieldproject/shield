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

	createTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Creates a new target",
		Long:  "Create a new target with ...",
	} // FIXME

	listTargetCmd = &cobra.Command{
		Use:   "targets",
		Short: "Lists the available targets",
	}

	showTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Show all the Targets",
	}

	deleteTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Delete all the Targets",
	}

	editTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Edit all the Targets",
	}
)

func init() {
	// Set options for the subcommands
	listTargetCmd.Flags().StringVarP(&pluginFilter, "plugin", "p", "", "Filter by plugin name")
	listTargetCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused targets")
	listTargetCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used targets")

	// Hookup functions to the subcommands
	createTargetCmd.Run = processCreateTargetRequest
	listTargetCmd.Run = processListTargetsRequest
	showTargetCmd.Run = processShowTargetRequest
	editTargetCmd.Run = processEditTargetRequest
	deleteTargetCmd.Run = processDeleteTargetRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createTargetCmd)
	listCmd.AddCommand(listTargetCmd)
	showCmd.AddCommand(showTargetCmd)
	editCmd.AddCommand(editTargetCmd)
	deleteCmd.AddCommand(deleteTargetCmd)
}

func processListTargetsRequest(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	targets, err := GetTargets(TargetFilter{
		Plugin: pluginFilter,
		Unused: MaybeString(parseTristateOptions(cmd, "unused", "used")),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of targets:\n", err)
	}

	t := tui.NewTable("UUID", "Target Name", "Description", "Plugin", "Endpoint", "SHIELD Agent")
	for _, target := range targets {
		t.Row(target.UUID, target.Name, target.Summary, target.Plugin, target.Endpoint, target.Agent)
	}
	t.Output(os.Stdout)
	return
}

func processCreateTargetRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Invoke editor
	content := invokeEditor(`{
	"name":     "",
	"summary":  "",
	"plugin":   "",
	"endpoint": "{\"\":\"\"}",
	"agent":    ""
}`)

	fmt.Println("Got the following content:\n\n", content)

	data, err := CreateTarget(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of targets:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of targets:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowTargetRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	data, err := GetTarget(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show target:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render target:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
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

func processDeleteTargetRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	err := DeleteTarget(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete target:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
