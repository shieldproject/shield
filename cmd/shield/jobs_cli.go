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

	//== Applicable actions for Jobs
	createJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Creates a new job",
		Long:  "Create a new job with ...",
	} // FIXME

	listJobCmd = &cobra.Command{
		Use:   "jobs",
		Short: "Lists all the jobs",
	}

	showJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Shows information about the specified job",
	}

	deleteJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Deletes the specified job",
	}

	pauseJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Pauses the specified job",
	}

	unpauseJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Unpauses the specified job",
	}

	pausedJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Returns \"true\" with exit code 0 if specified job is paused, \"false\"/1 otherwise",
	}

	runJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Runs the specified job",
	}

	editJobCmd = &cobra.Command{
		Use:   "job",
		Short: "Edit the specified job",
	}
)

func init() {

	// Set options for the subcommands
	listJobCmd.Flags().Bool("paused", false, "Show only paused jobs")
	listJobCmd.Flags().Bool("unpaused", false, "Show only unpaused jobs")
	listJobCmd.Flags().String("store", "", "Filter by store UUID")
	listJobCmd.Flags().String("target", "", "Filter by store UUID")
	listJobCmd.Flags().String("schedule", "", "Filter by schedule UUID")
	listJobCmd.Flags().String("retention", "", "Filter by retention policy UUID")

	// Hookup functions to the subcommands
	createJobCmd.Run = processCreateJobRequest
	listJobCmd.Run = processListJobsRequest
	showJobCmd.Run = processShowJobRequest
	deleteJobCmd.Run = processDeleteJobRequest
	pauseJobCmd.Run = processPauseJobRequest
	unpauseJobCmd.Run = processUnpauseJobRequest
	pausedJobCmd.Run = processPausedJobRequest
	runJobCmd.Run = processRunJobRequest
	editJobCmd.Run = processEditJobRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createJobCmd)
	listCmd.AddCommand(listJobCmd)
	showCmd.AddCommand(showJobCmd)
	deleteCmd.AddCommand(deleteJobCmd)
	pauseCmd.AddCommand(pauseJobCmd)
	unpauseCmd.AddCommand(unpauseJobCmd)
	pausedCmd.AddCommand(pausedJobCmd)
	runCmd.AddCommand(runJobCmd)
	editCmd.AddCommand(editJobCmd)
}

func processCreateJobRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Invoke editor
	content := invokeEditor(`{
  "name"      : "Job Name",
  "summary"   : "a short description",

  "store"     : "uuid_of_store_to_use",
  "target"    : "uuid_of_target_to_use",
  "retention" : "uuid_of_retention_policy_to_use",
  "schedule"  : "uuid_of_schedule_to_use",

  "paused"    : false
}`)

	fmt.Println("Got the following content:\n\n", content)

	// Fetch
	data, err := CreateJob(content)
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

func processListJobsRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	pausedFilter := parseTristateOptions(cmd, "paused", "unpaused")
	storeUUID, _ := cmd.Flags().GetString("store")
	targetUUID, _ := cmd.Flags().GetString("target")
	scheduleUUID, _ := cmd.Flags().GetString("schedule")
	retentionUUID, _ := cmd.Flags().GetString("retention")

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := FetchListJobs(targetUUID, storeUUID, scheduleUUID, retentionUUID, pausedFilter)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of jobs:\n", err)
	}

	t := tui.NewTable(7)
	t.Header("UUID", "P?", "Name", "Description", "Retention Policy", "Schedule", "Target", "Agent")
	for _, job := range data {
		paused := "-"
		if job.Paused {
			paused = "Y"
		}

		t.Row(job.UUID, paused, job.Name, job.Summary,
			job.RetentionName, job.ScheduleName, job.TargetEndpoint, job.Agent)
	}
	t.Output(os.Stdout)
}

func processShowJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := GetJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show job:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render job:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processEditJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	original_data, err := GetJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show job:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render job:\n", err)
	}

	fmt.Println("Got the following original job:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited job:\n\n", content)

	// Fetch
	update_data, err := UpdateJob(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update jobs:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render job:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := DeleteJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete job:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, "deleted")

	return
}

func processRunJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// FIXME when owner can be passed in or otherwise fetched
	content := "{\"owner\":\"anon\"}"

	// Fetch
	err := RunJob(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not run job:\n", err)
		os.Exit(1)
	}

	fmt.Println(requested_UUID, "scheduled")

	return
}

func processPauseJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := PauseJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not pause job:\n", err)
		os.Exit(1)
	}

	fmt.Println(requested_UUID, "pause")

	return
}

func processUnpauseJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := UnpauseJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not run jobs:\n", err)
		os.Exit(1)
	}

	fmt.Println(requested_UUID, "unpaused")

	return
}

func processPausedJobRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	paused, err := IsPausedJob(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not pause job:\n", err)
		os.Exit(1)
	}

	if paused == true {
		fmt.Println("Job", requested_UUID, "is paused")
		os.Exit(0)
	} else {
		fmt.Println("Job", requested_UUID, "is not paused")
		os.Exit(1)
	}
	return
}
