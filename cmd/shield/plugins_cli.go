package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/shield/api_agent"
	"os"
)

var (

	//== Applicable actions for Plugins

	listPluginCmd = &cobra.Command{
		Use:   "plugins",
		Short: "Lists all the plugins",
	}

	showPluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Shows information about the specified plugin",
	}
)

func init() {
	listPluginCmd.Run = processListPluginsRequest
	showPluginCmd.Run = processShowPluginRequest

	listCmd.AddCommand(listPluginCmd)
	showCmd.AddCommand(showPluginCmd)
}

func processListPluginsRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := api_agent.FetchListPlugins()
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of plugins:\n", err)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of plugins:\n", err)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowPluginRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single Name\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_name := args[0]

	// Fetch
	data, err := api_agent.GetPlugin(requested_name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show plugin:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render plugin:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}
