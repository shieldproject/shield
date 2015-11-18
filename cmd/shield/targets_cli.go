package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/shield/api_agent"
	"os"
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

	// Validate Request
	unused := parseTristateOptions(cmd, "unused", "used")

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := api_agent.FetchTargetsList(pluginFilter, unused)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of targets:\n", err)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of targets:\n", err)
	}

	fmt.Println(string(output[:]))

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

	// Fetch
	data, err := api_agent.CreateTarget(content)
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

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := api_agent.GetTarget(requested_UUID)
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

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	original_data, err := api_agent.GetTarget(requested_UUID)
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

	// Fetch
	update_data, err := api_agent.UpdateTarget(requested_UUID, content)
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

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := api_agent.DeleteTarget(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete target:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
