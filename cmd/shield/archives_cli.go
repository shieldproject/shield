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

type ListArchiveOptions struct {
	Target string
	Store  string
	Before string
	After  string
}

func ListArchives(opts ListArchiveOptions) error {
	archives, err := GetArchives(ArchiveFilter{})
	if err != nil {
		return fmt.Errorf("\nERROR: Could not fetch list of archives: %s\n", err)
	}

	//Getting the target and store names into the output
	target := map[string]Target{}
	targets, _ := GetTargets(TargetFilter{})
	for _, t := range targets {
		target[t.UUID] = t
	}
	store := map[string]Store{}
	stores, _ := GetStores(StoreFilter{})
	for _, s := range stores {
		store[s.UUID] = s
	}

	//FIXME:
	// Set beforeTime default to 0 and afterTime to Now()
	// Then update if a value is actually passed by the usedFilter
	// Since date is YYYYMMDD make sure the HHMMSS is 23:59:59
	t := tui.NewTable("UUID", "Target Type", "Target Name", "Store Type", "Store Name", "Taken at", "Expires at", "Notes")
	for _, archive := range archives {
		targetSpecified := (len(opts.Target) > 0 && opts.Target == archive.TargetUUID)
		storeSpecified := (len(opts.Store) > 0 && opts.Store == archive.StoreUUID)
		if (targetSpecified && storeSpecified) || targetSpecified || storeSpecified {
			t.Row(archive.UUID,
				archive.TargetPlugin, target[archive.TargetUUID].Name,
				archive.StorePlugin, store[archive.StoreUUID].Name,
				archive.TakenAt.Format(time.RFC1123Z),
				archive.ExpiresAt.Format(time.RFC1123Z),
				archive.Notes)
		} else if len(opts.Target) == 0 && len(opts.Store) == 0 {
			t.Row(archive.UUID,
				archive.TargetPlugin, target[archive.TargetUUID].Name,
				archive.StorePlugin, store[archive.StoreUUID].Name,
				archive.TakenAt.Format(time.RFC1123Z),
				archive.ExpiresAt.Format(time.RFC1123Z),
				archive.Notes)
		}

	}
	t.Output(os.Stdout)
	return nil
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

	archives, err := GetArchives(ArchiveFilter{
		Plugin: pluginFilter,
		Unused: MaybeString(unused),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not fetch list of archives:\n", err)
	}

	t := tui.NewTable("UUID", "Target Type", "Target Name", "Store Type", "Taken at", "Expires at", "Notes")
	target := map[string]Target{}
	targets, _ := GetTargets(TargetFilter{})
	for _, t := range targets {
		target[t.UUID] = t
	}
	for _, archive := range archives {
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

	archive, err := GetArchive(uuid.Parse(args[0]))
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not show archive:\n", err)
		os.Exit(1)
	}

	t := tui.NewReport()
	t.Add("UUID", archive.UUID)
	t.Add("Backup Key", archive.StoreKey)
	t.Break()

	t.Add("Target", archive.TargetPlugin)
	t.Add("Target UUID", archive.TargetUUID)
	t.Add("Target Endpoint", archive.TargetEndpoint)
	t.Break()

	t.Add("Store", archive.StorePlugin)
	t.Add("Store UUID", archive.StoreUUID)
	t.Add("Store Endpoint", archive.StoreEndpoint)
	t.Break()

	t.Add("Taken at", archive.TakenAt.Format(time.RFC1123Z))
	t.Add("Expires at", archive.ExpiresAt.Format(time.RFC1123Z))
	t.Add("Notes", archive.Notes)

	t.Output(os.Stdout)
}

func processEditArchiveRequest(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "\nERROR: Requires a single UUID\n")
		//FIXME  show help
		os.Exit(1)
	}

	requested_UUID := uuid.Parse(args[0])

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

	err := RestoreArchive(requested_UUID, archiveRestoreTo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR: Could not restore archive:\n", err)
		os.Exit(1)
	}

	// Print
	fmt.Println(requested_UUID, " Restore requested")

	return
}
