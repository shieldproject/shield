package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/shield/api_agent"
	"os"
)

var (

	//== Applicable actions for Tasks

	listTaskCmd = &cobra.Command{
		Use:   "tasks",
		Short: "Lists all the tasks",
	}

	showTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Shows information about the specified task",
	}

	cancelTaskCmd = &cobra.Command{
		Use:   "task",
		Short: "Cancels the specified task",
	}
)

func init() {
	// Set options for the subcommands
	listTaskCmd.Flags().StringP("status", "s", "", "Filter by status")
	listTaskCmd.Flags().Bool("debug", false, "Turn on debug mode")

	// Hookup functions to the subcommands
	listTaskCmd.Run = processListTasksRequest
	showTaskCmd.Run = processShowTaskRequest
	cancelTaskCmd.Run = processCancelTaskRequest

	// Add the subcommands to the base actions
	listCmd.AddCommand(listTaskCmd)
	showCmd.AddCommand(showTaskCmd)
	cancelCmd.AddCommand(cancelTaskCmd)
}

func processListTasksRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	debug, _ := cmd.Flags().GetBool("debug")
	status, _ := cmd.Flags().GetString("status")

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := api_agent.FetchListTasks(status, debug)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of tasks:\n", err)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of tasks:\n", err)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowTaskRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := api_agent.GetTask(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show task:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render task:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processCancelTaskRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := api_agent.CancelTask(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not cancel task:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Canceled")

	return
}
