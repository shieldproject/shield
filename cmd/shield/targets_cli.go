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

	listTargetCmd = &cobra.Command{
		Use:   "targets",
		Short: "Lists the available targets",
		//		Long:  "This is a Long Jaun (TBD)",
	}

	showTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Show all the Targets",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Delete all the Targets",
		Long:  "This is a Long Jaun (TBD)",
	}

	editTargetCmd = &cobra.Command{
		Use:   "target",
		Short: "Edit all the Targets",
		Long:  "This is a Long Jaun (TBD)",
	}

	// Options
	pluginFilter string
	unusedFilter bool
	usedFilter   bool
)

func init() {
	// Set options for the subcommands
	listTargetCmd.Flags().StringVarP(&pluginFilter, "plugin", "p", "", "Filter by plugin name")
	listTargetCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused targets")
	listTargetCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used targets")

	// Hookup functions to the subcommands
	listTargetCmd.Run = processListTargetsRequest
	showTargetCmd.Run = debug
	editTargetCmd.Run = debug
	deleteTargetCmd.Run = debug

	// Add the subcommands to the base actions
	listCmd.AddCommand(listTargetCmd)
	showCmd.AddCommand(showTargetCmd)
	editCmd.AddCommand(editTargetCmd)
	deleteCmd.AddCommand(deleteTargetCmd)
}

func processListTargetsRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	unused := ""
	if unusedFilter {
		unused = "t"
	}
	if usedFilter {
		if unused == "" {
			unused = "f"
		} else {
			fmt.Fprintf(os.Stderr, "\nERROR: Cannot specify --used and --unused at the same time\n\n")
			os.Exit(1)
		}
	}

	// Fetch
	data, err := api_agent.FetchListTargets(pluginFilter, unused)
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
