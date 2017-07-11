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
func cliListArchives(opts Options, args []string, help bool) error {
	if help {
		HelpListMacro("archive", "archives")
		FlagHelp(`Only show archives with the specified state of validity.
									Accepted values are one of ['all', 'valid']
									If not explicitly set, it defaults to 'valid'`,
			true, "-S", "--status=value")
		FlagHelp("Show only archives created from the specified target", true, "-t", "--target=value")
		FlagHelp("Show only archives sent to the specified store", true, "-s", "--store=value")
		FlagHelp("Show only the <value> most recent archives", true, "--limit=value")
		FlagHelp(`Show only the archives taken before this point in time
				Specify in the format YYYYMMDD`, true, "-B", "--before=value")
		FlagHelp(`Show only the archives taken after this point in time
				Specify in the format YYYYMMDD`, true, "-A", "--after=value")
		FlagHelp(`Show all archives, regardless of validity.
									Equivalent to '--status=all'`, true, "-a", "--all")
		JSONHelp(`[{"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533","key":"thisisastorekey","taken_at":"2016-05-18 11:02:43","expires_at":"2017-05-18 11:02:43","status":"valid","notes":"","target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1","target_plugin":"redis-broker","target_endpoint":"{\"redis_type\":\"broker\"}","store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}"}]`)
		return nil
	}

	DEBUG("running 'list archives' command")

	if *options.Status == "" {
		*options.Status = "valid"
	}
	if *options.Status == "all" || *options.All {
		*options.Status = ""
	}
	DEBUG("  for status: '%s'", *opts.Status)

	if *options.Limit == "" {
		*options.Limit = "20"
	}
	DEBUG("  for limit: '%s'", *opts.Limit)

	archives, err := api.GetArchives(api.ArchiveFilter{
		Target: *options.Target,
		Store:  *options.Store,
		Before: *options.Before,
		After:  *options.After,
		Status: *options.Status,
		Limit:  *options.Limit,
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
func cliGetArchive(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`A UUID assigned to a single archive instance`, false, "<uuid>")
		FlagHelp("Returns information as a JSON object", true, "--raw")
		HelpKMacro()
		JSONHelp(`{"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533","key":"thisisastorekey","taken_at":"2016-05-18 11:02:43","expires_at":"2017-05-18 11:02:43","status":"valid","notes":"","target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1","target_plugin":"redis-broker","target_endpoint":"{\"redis_type\":\"broker\"}","store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808","store_plugin":"s3","store_endpoint":"{\"endpoint\":\"schmendpoint\"}"}`)
		return nil
	}

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
func cliRestoreArchive(opts Options, args []string, help bool) error {
	if help {
		MessageHelp("Note: If raw mode is specified and the targeted SHIELD backend does not support handing back the task uuid, the task_uuid in the JSON will be the empty string")
		FlagHelp(`Outputs the result as a JSON object.`, true, "--raw")
		FlagHelp(`The name or UUID of a single target to restore. In raw mode, it must be a UUID assigned to a single archive instance`, false, "<target or uuid>")
		HelpKMacro()
		return nil
	}

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

func cliDeleteArchive(opts Options, args []string, help bool) error {
	if help {
		FlagHelp(`A UUID assigned to a single archive instance`, false, "<uuid>")
		FlagHelp(`Outputs the result as a JSON object.
				The cli will not prompt for confirmation in raw mode.`, true, "--raw")
		HelpKMacro()
		JSONHelp(`{"ok":"Deleted archive"}`)
		return nil
	}

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
