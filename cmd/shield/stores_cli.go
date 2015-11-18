package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/shield/api_agent"
	"os"
)

var (

	//== Applicable actions for Stores

	listStoreCmd = &cobra.Command{
		Use:   "stores",
		Short: "List all the Stores",
	}

	showStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Show all the Stores",
	}

	deleteStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Delete all the Stores",
	}

	editStoreCmd = &cobra.Command{
		Use:   "store",
		Short: "Edit all the Stores",
	}
)

func init() {
	// Set options for the subcommands
	listStoreCmd.Flags().StringVarP(&pluginFilter, "plugin", "p", "", "Filter by plugin name")
	listStoreCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused stores")
	listStoreCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used stores")

	// Hookup functions to the subcommands
	listStoreCmd.Run = processListStoresRequest
	showStoreCmd.Run = processShowStoreRequest
	editStoreCmd.Run = processEditStoreRequest
	deleteStoreCmd.Run = processDeleteStoreRequest

	// Add the subcommands to the base actions
	listCmd.AddCommand(listStoreCmd)
	showCmd.AddCommand(showStoreCmd)
	editCmd.AddCommand(editStoreCmd)
	deleteCmd.AddCommand(deleteStoreCmd)
}

func processListStoresRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	unused := parseTristateOptions(cmd, "unused", "used")

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := api_agent.FetchStoresList(pluginFilter, unused)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of stores:\n", err)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of stores:\n", err)
	}

	fmt.Println(string(output[:]))

	return
}

func processCreateStoreRequest(cmd *cobra.Command, args []string) {

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
}`)

	fmt.Println("Got the following content:\n\n", content)

	// Fetch
	data, err := api_agent.CreateStore(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of stores:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of stores:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := api_agent.GetStore(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show store:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render store:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processEditStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	original_data, err := api_agent.GetStore(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show store:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render store:\n", err)
	}

	fmt.Println("Got the following original store:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited store:\n\n", content)

	// Fetch
	update_data, err := api_agent.UpdateStore(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update stores:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render store:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteStoreRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := api_agent.DeleteStore(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete store:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
