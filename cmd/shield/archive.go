package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/tui"
)

//List available backup archives
func cliListArchives(args ...string) error {
	DEBUG("running 'list archives' command")

	if *opts.Status == "" {
		*opts.Status = "valid"
	}
	if *opts.Status == "all" || *opts.All {
		*opts.Status = ""
	}
	DEBUG("  for status: '%s'", *opts.Status)

	if *opts.Limit == "" {
		*opts.Limit = "20"
	}
	DEBUG("  for limit: '%s'", *opts.Limit)

	archives, err := api.GetArchives(api.ArchiveFilter{
		Target: *opts.Target,
		Store:  *opts.Store,
		Before: *opts.Before,
		After:  *opts.After,
		Status: *opts.Status,
		Limit:  *opts.Limit,
	})
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(archives)
	}

	// Map out the target names, for prettier output
	target := map[string]api.Target{}
	targets, _ := api.GetTargets(api.TargetFilter{})
	for _, t := range targets {
		target[t.UUID] = t
	}

	// Map out the store names, for prettier output
	store := map[string]api.Store{}
	stores, _ := api.GetStores(api.StoreFilter{})
	for _, s := range stores {
		store[s.UUID] = s
	}

	t := tui.NewTable("UUID", "Target", "Restore IP", "Store", "Taken at", "Expires at", "Status", "Notes")
	for _, archive := range archives {
		if *opts.Target != "" && archive.TargetUUID != *opts.Target {
			continue
		}
		if *opts.Store != "" && archive.StoreUUID != *opts.Store {
			continue
		}

		t.Row(archive, archive.UUID,
			fmt.Sprintf("%s (%s)", target[archive.TargetUUID].Name, archive.TargetPlugin),
			target[archive.TargetUUID].Agent,
			fmt.Sprintf("%s (%s)", store[archive.StoreUUID].Name, archive.StorePlugin),
			archive.TakenAt.Format(time.RFC1123Z),
			archive.ExpiresAt.Format(time.RFC1123Z),
			archive.Status, archive.Notes)
	}
	t.Output(os.Stdout)
	return nil
}

//Print detailed information about a backup archive
func cliGetArchive(args ...string) error {
	DEBUG("running 'show archive' command")

	require(len(args) == 1, "shield show archive <UUID>")
	id := uuid.Parse(args[0])
	DEBUG("  archive UUID = '%s'", id)

	archive, err := api.GetArchive(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		return RawJSON(archive)
	}
	if *opts.ShowUUID {
		return RawUUID(archive.UUID)
	}

	ShowArchive(archive)
	return nil
}

//Restore a backup archive
func cliRestoreArchive(args ...string) error {
	DEBUG("running 'restore archive' command")

	var id uuid.UUID

	if *opts.Raw {
		require(len(args) == 1, "USAGE: shield restore archive <UUID>")
		id = uuid.Parse(args[0])
		DEBUG("  trying archive UUID '%s'", args[0])

	} else {
		target, _, err := FindTarget(strings.Join(args, " "), false)
		if err != nil {
			return err
		}

		_, id, err = FindArchivesFor(target, 10)
		if err != nil {
			return err
		}
	}
	DEBUG("  archive UUID = '%s'", id)

	var params = struct {
		Owner  string `json:"owner,omitempty"`
		Target string `json:"target,omitempty"`
	}{
		Owner: CurrentUser(),
	}

	if *opts.To != "" {
		params.Target = *opts.To
	}

	b, err := json.Marshal(params)
	if err != nil {
		return err
	}

	taskUUID, err := api.RestoreArchive(id, string(b))
	if err != nil {
		return err
	}

	targetMsg := ""
	if params.Target != "" {
		targetMsg = fmt.Sprintf("to target '%s'", params.Target)
	}
	if *opts.Raw {
		RawJSON(map[string]interface{}{
			"ok":        fmt.Sprintf("Scheduled immediate restore of archive '%s' %s", id, targetMsg),
			"task_uuid": taskUUID,
		})
	} else {
		//`OK` handles raw checking
		OK("Scheduled immediate restore of archive '%s' %s", id, targetMsg)
		if taskUUID != "" {
			ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
		}
	}

	return nil
}

func cliDeleteArchive(args ...string) error {
	DEBUG("running 'delete archive' command")

	require(len(args) == 1, "USAGE: shield delete archive <UUID>")
	id := uuid.Parse(args[0])
	DEBUG("  archive UUID = '%s'", id)

	archive, err := api.GetArchive(id)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		ShowArchive(archive)
		if !tui.Confirm("Really delete this archive?") {
			return errCanceled
		}
	}

	if err := api.DeleteArchive(id); err != nil {
		return err
	}

	OK("Deleted archive")
	return nil
}
