package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Schedules

	createScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Creates a new schedule",
		Long:  "Create a new schedule with ...",
	} // FIXME

	listScheduleCmd = &cobra.Command{
		Use:   "schedules",
		Short: "List all the Schedules",
	}

	showScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Show all the Schedules",
	}

	deleteScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Delete all the Schedules",
	}

	updateScheduleCmd = &cobra.Command{
		Use:   "schedule",
		Short: "Update the Schedules",
	}
)

func init() {
	// Set options for the subcommands
	listScheduleCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused schedules")
	listScheduleCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used schedules")

	// Hookup functions to the subcommands
	createScheduleCmd.Run = processCreateScheduleRequest
	listScheduleCmd.Run = processListSchedulesRequest
	showScheduleCmd.Run = processShowScheduleRequest
	updateScheduleCmd.Run = processUpdateScheduleRequest
	deleteScheduleCmd.Run = processDeleteScheduleRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createScheduleCmd)
	listCmd.AddCommand(listScheduleCmd)
	showCmd.AddCommand(showScheduleCmd)
	updateCmd.AddCommand(updateScheduleCmd)
	deleteCmd.AddCommand(deleteScheduleCmd)
}

func processListSchedulesRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	unused := parseTristateOptions(cmd, "unused", "used")

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := FetchListSchedules(unused)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of schedules:\n", err)
	}

	t := tui.NewTable(4)
	t.Header("UUID", "Name", "Description", "Frequency / Interval")
	for _, schedule := range data {
		t.Row(schedule.UUID, schedule.Name, schedule.Summary, schedule.When)
	}
	t.Output(os.Stdout)
}

func processCreateScheduleRequest(cmd *cobra.Command, args []string) {

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
	"when":    ""
}`)

	fmt.Println("Got the following content:\n\n", content)

	// Fetch
	data, err := CreateSchedule(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of schedules:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of schedules:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowScheduleRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := GetSchedule(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show schedule:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render schedule:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processUpdateScheduleRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	original_data, err := GetSchedule(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show schedule:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render schedule:\n", err)
	}

	fmt.Println("Got the following original schedule:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited schedule:\n\n", content)

	// Fetch
	update_data, err := UpdateSchedule(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update schedules:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render schedule:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteScheduleRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := DeleteSchedule(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete schedule:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
