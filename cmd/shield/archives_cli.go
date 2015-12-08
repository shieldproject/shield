package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

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
		return fmt.Errorf("ERROR: Could not fetch list of archives: %s", err)
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

func RestoreArchiveByUUID(opts ListArchiveOptions) error {
	targetJSON := "{}"
	toTargetJSONmsg := ""
	if len(opts.Target) > 0 {
		targetJSON = opts.Target
		toTargetJSONmsg = fmt.Sprintf("to target '%s'", targetJSON)
	}
	err := RestoreArchive(uuid.Parse(opts.UUID), targetJSON)
	if err != nil {
		return fmt.Errorf("ERROR: Cannot restore archive '%s': '%s'", opts.UUID, err)
	}
	fmt.Fprintf(os.Stdout, "Restoring archive '%s' %s\n", opts.UUID, toTargetJSONmsg)
	return nil
}

func DeleteArchiveByUUID(u string) error {
	err := DeleteArchive(uuid.Parse(u))
	if err != nil {
		return fmt.Errorf("ERROR: Cannot delete archive '%s': %s", u, err)
	}
	fmt.Fprintf(os.Stdout, "Deleted archive '%s'\n", u)
	return nil
}
