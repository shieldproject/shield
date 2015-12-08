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

	//== Applicable actions for Retention Policies

	createRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "Create all the Retentions",
	}
	createRetentionPoliciesCmd = &cobra.Command{
		Use:   "policies",
		Short: "Create all the Retention Policies",
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

	// Hookup functions to the subcommands
	createRetentionPoliciesCmd.Run = processCreateRetentionRequest
	updateRetentionPolicyCmd.Run = processUpdateRetentionRequest

	// Add the subcommands to the base actions
	createCmd.AddCommand(createRetentionCmd)
	updateCmd.AddCommand(updateRetentionCmd)
	createRetentionCmd.AddCommand(createRetentionPoliciesCmd)
	updateRetentionCmd.AddCommand(updateRetentionPolicyCmd)
}

type ListRetentionOptions struct {
	Unused bool
	Used   bool
	UUID   string
}

func ListRetentionPolicies(opts ListRetentionOptions) error {
	policies, err := GetRetentionPolicies(RetentionPoliciesFilter{
		Unused: MaybeBools(opts.Unused, opts.Used),
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve retention policies from SHIELD: %s", err)
	}
	t := tui.NewTable("UUID", "Name", "Description", "Expires in")
	for _, policy := range policies {
		if len(opts.UUID) > 0 && opts.UUID == policy.UUID {
			t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
			break
		} else if len(opts.UUID) > 0 && opts.UUID != policy.UUID {
			continue
		}
		t.Row(policy.UUID, policy.Name, policy.Summary, fmt.Sprintf("%d days", policy.Expires/86400))
	}
	t.Output(os.Stdout)
	return nil
}

func DeleteRetentionPolicyByUUID(u string) error {
	err := DeleteRetentionPolicy(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete retention policy '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted retention policy '%s'\n", u)
	return nil
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

func processUpdateRetentionRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

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
