package main

import (
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

	restoreArchiveCmd.Flags().StringVarP(&archiveRestoreTo, "to", "", "", "Filter by plugin name")

	// Hookup functions to the subcommands
	deleteArchiveCmd.Run = processDeleteArchiveRequest
	restoreArchiveCmd.Run = processRestoreArchiveRequest

	// Add the subcommands to the base actions
	deleteCmd.AddCommand(deleteArchiveCmd)
	restoreCmd.AddCommand(restoreArchiveCmd)
}

type ListArchiveOptions struct {
	Target string
	Store  string
	Before string
	After  string
	UUID   string
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
		if len(opts.UUID) > 0 && opts.UUID == archive.UUID {
			t.Row(archive.UUID,
				archive.TargetPlugin, target[archive.TargetUUID].Name,
				archive.StorePlugin, store[archive.StoreUUID].Name,
				archive.TakenAt.Format(time.RFC1123Z),
				archive.ExpiresAt.Format(time.RFC1123Z),
				archive.Notes)
			break
		} else if len(opts.UUID) > 0 && opts.UUID != archive.UUID {
			continue
		}

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
