package main

import (
	"github.com/spf13/cobra"
	//"github.com/spf13/viper"
)

var (

	//== Applicable actions for Retention Policies

	listRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
		Long:  "This is a Long Jaun (TBD)",
	}

	listRetentionPoliciesCmd = &cobra.Command{
		Use:   "policies",
		Short: "List all the Retention Policies",
		Long:  "This is a Long Jaun (TBD)",
	}

	showRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
		Long:  "This is a Long Jaun (TBD)",
	}
	showRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Show details for the given retention policy",
		Long:  "This is a Long Jaun (TBD)",
	}

	deleteRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
		Long:  "This is a Long Jaun (TBD)",
	}
	deleteRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Delete details for the given retention policy",
		Long:  "This is a Long Jaun (TBD)",
	}

	updateRetentionCmd = &cobra.Command{
		Use:   "retention",
		Short: "List all the Retentions",
		Long:  "This is a Long Jaun (TBD)",
	}
	updateRetentionPolicyCmd = &cobra.Command{
		Use:   "policy",
		Short: "Update details for the given retention policy",
		Long:  "This is a Long Jaun (TBD)",
	}
)

func init() {
	listRetentionPoliciesCmd.Run = debug
	showRetentionPolicyCmd.Run = debug
	updateRetentionPolicyCmd.Run = debug
	deleteRetentionPolicyCmd.Run = debug

	listCmd.AddCommand(listRetentionCmd)
	showCmd.AddCommand(showRetentionCmd)
	updateCmd.AddCommand(updateRetentionCmd)
	deleteCmd.AddCommand(deleteRetentionCmd)
	listRetentionCmd.AddCommand(listRetentionPoliciesCmd)
	showRetentionCmd.AddCommand(showRetentionPolicyCmd)
	updateRetentionCmd.AddCommand(updateRetentionPolicyCmd)
	deleteRetentionCmd.AddCommand(deleteRetentionPolicyCmd)
}
