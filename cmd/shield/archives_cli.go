package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

var (

	//== Applicable actions for Archives

	listArchiveCmd = &cobra.Command{
		Use:   "archives",
		Short: "Lists all the archives",
	}

	showArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Shows information about the specified archive",
	}

	deleteArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Deletes the specified archive",
	}

	editArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Edit the specified archive",
	}

	restoreArchiveCmd = &cobra.Command{
		Use:   "archive",
		Short: "Restores the specified archive",
	}

	// Options
	archiveTargetFilter string
	archiveStoreFilter  string
	archiveAfterFilter  string
	archiveBeforeFilter string
	archiveRestoreTo    string
)

func init() {
	// Set options for the subcommands
	listArchiveCmd.Flags().StringVarP(&archiveTargetFilter, "target", "", "", "Filter by target")
	listArchiveCmd.Flags().StringVarP(&archiveStoreFilter, "store", "", "", "Filter by store")
	listArchiveCmd.Flags().StringVarP(&archiveAfterFilter, "after", "", "", "Filter by after date")
	listArchiveCmd.Flags().StringVarP(&archiveBeforeFilter, "before", "", "", "Filter by before date")

	restoreArchiveCmd.Flags().StringVarP(&archiveRestoreTo, "to", "", "", "Filter by plugin name")

	// Hookup functions to the subcommands
	//createArchiveCmd.Run = processCreateArchiveRequest
	listArchiveCmd.Run = processListArchivesRequest
	showArchiveCmd.Run = processShowArchiveRequest
	editArchiveCmd.Run = processEditArchiveRequest
	deleteArchiveCmd.Run = processDeleteArchiveRequest
	restoreArchiveCmd.Run = processRestoreArchiveRequest

	// Add the subcommands to the base actions
	//createCmd.AddCommand(createArchiveCmd)
	listCmd.AddCommand(listArchiveCmd)
	showCmd.AddCommand(showArchiveCmd)
	editCmd.AddCommand(editArchiveCmd)
	deleteCmd.AddCommand(deleteArchiveCmd)
	restoreCmd.AddCommand(restoreArchiveCmd)
}

func processListArchivesRequest(cmd *cobra.Command, args []string) {

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

	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: Unexpected arguments following command: %v\n", args)
		//FIXME  show help
		os.Exit(1)
	}

	// Fetch
	data, err := FetchListArchives(pluginFilter, unused)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of archives:\n", err)
	}

	t := tui.NewTable("UUID", "Target Type", "Target Name", "Store Type", "Taken at", "Expires at", "Notes")
	target := map[string]Target{}
	targets, _ := GetTargets(TargetFilter{})
	for _, t := range targets {
		target[t.UUID] = t
	}
	for _, archive := range data {
		t.Row(archive.UUID, archive.TargetPlugin, target[archive.TargetUUID].Name, archive.StorePlugin,
			archive.TakenAt.Format(time.RFC1123Z),
			archive.ExpiresAt.Format(time.RFC1123Z),
			archive.Notes)
	}
	t.Output(os.Stdout)
}

func processShowArchiveRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	data, err := GetArchive(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show archive:\n", err)
		os.Exit(1)
	}

	// Print
	output, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render archive:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processEditArchiveRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	// Fetch
	original_data, err := GetArchive(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show archive:\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(original_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render archive:\n", err)
	}

	fmt.Println("Got the following original archive:\n\n", string(data))

	// Invoke editor
	content := invokeEditor(string(data))

	fmt.Println("Got the following edited archive:\n\n", content)

	// Fetch
	update_data, err := UpdateArchive(requested_UUID, content)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not update archives:\n", err)
		os.Exit(1)
	}
	// Print
	output, err := json.MarshalIndent(update_data, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not render archive:\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output[:]))

	return
}

func processDeleteArchiveRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	// Fetch
	err := DeleteArchive(requested_UUID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not delete archive:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Deleted")

	return
}

func processRestoreArchiveRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	if archiveRestoreTo == "" {
		fmt.Fprint(os.Stderr, "\nERROR: Requires a target\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

	// Fetch
	err := RestoreArchive(requested_UUID, archiveRestoreTo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not restore archive:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Restore requested")

	return
}
