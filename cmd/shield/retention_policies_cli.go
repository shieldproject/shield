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

	//== Applicable actions for Retention Policies

	createRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "Create all the Retentions",
	}
	createRetentionPoliciesCmd = &cobra.Command{
		Use:   "policies",
		Short: "Create all the Retention Policies",
	}

	listRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
	}
	listRetentionPoliciesCmd = &cobra.Command{
		Use:   "policies",
		Short: "List all the Retention Policies",
	}

	showRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
	}
	showRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Show details for the given retention policy",
	}

	deleteRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
	}
	deleteRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Delete details for the given retention policy",
	}

	updateRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
	}
	updateRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Update details for the given retention policy",
	}
)

func init() {
	// Set options for the subcommands
	listRetentionPoliciesCmd.Flags().BoolVar(&unusedFilter, "unused", false, "Show only unused retentions")
	listRetentionPoliciesCmd.Flags().BoolVar(&usedFilter, "used", false, "Show only used retentions")

	// Hookup functions to the subcommands
	createRetentionPoliciesCmd.Run = processCreateRetentionRequest
	listRetentionPoliciesCmd.Run = processListRetentionsRequest
	showRetentionPolicyCmd.Run = processShowRetentionRequest
	updateRetentionPolicyCmd.Run = processUpdateRetentionRequest
	deleteRetentionPolicyCmd.Run = processDeleteRetentionRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createRetentionCmd)
	listCmd.AddCommand(listRetentionCmd)
	showCmd.AddCommand(showRetentionCmd)
	updateCmd.AddCommand(updateRetentionCmd)
	deleteCmd.AddCommand(deleteRetentionCmd)
	createRetentionCmd.AddCommand(createRetentionPoliciesCmd)
	listRetentionCmd.AddCommand(listRetentionPoliciesCmd)
	showRetentionCmd.AddCommand(showRetentionPolicyCmd)
	updateRetentionCmd.AddCommand(updateRetentionPolicyCmd)
	deleteRetentionCmd.AddCommand(deleteRetentionPolicyCmd)
}

func processListRetentionsRequest(cmd *cobra.Command, args []string) {

	// Validate Request
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := GetRetentionPolicies(RetentionPoliciesFilter{
		Unused: MaybeString(parseTristateOptions(cmd, "unused", "used")),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of retentions:\n", err)
	}

	t := tui.NewTable("UUID", "Name", "Description", "Expires in")
	for _, policy := range data {
		t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
	}
	t.Output(os.Stdout)
}

func processCreateRetentionRequest(cmd *cobra.Command, args []string) {

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
	"expires": 
}`)

	fmt.Println("Got the following content:\n\n", content)

	// Fetch
	data, err := CreateRetentionPolicy(content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of retentions:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render list of retentions:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processShowRetentionRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	data, err := GetRetentionPolicy(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show retention:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render retention:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processUpdateRetentionRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	original_data, err := GetRetentionPolicy(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show retention:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render retention:\n", err)
	}

	fmt.Println("Got the following original retention:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited retention:\n\n", content)

	// Fetch
	update_data, err := UpdateRetentionPolicy(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update retentions:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render retention:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteRetentionRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	//FIXME validate args is a valid UUID
	requested_UUID := args[0]

	// Fetch
	err := DeleteRetentionPolicy(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete retention:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}
