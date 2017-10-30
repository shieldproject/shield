package archives

import (
	"fmt"
	"os"
	"time"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List available backup archives
var List = &commands.Command{
	Summary: "List available backup archives",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "status", Short: 'S', Valued: true,
			Desc: `Only show archives with the specified state of validity.
								 Accepted values are one of ['all', 'valid']. If not
								 explicitly set, it defaults to 'valid'`,
		},
		commands.FlagInfo{
			Name: "target", Short: 't', Valued: true,
			Desc: "Show only archives created from the specified target",
		},
		commands.FlagInfo{
			Name: "store", Short: 's', Valued: true,
			Desc: "Show only archives sent to the specified store",
		},
		commands.FlagInfo{
			Name: "limit", Valued: true,
			Desc: "Show only the <value> most recent archives",
		},
		commands.FlagInfo{
			Name: "before", Short: 'B', Valued: true,
			Desc: `Show only the archives taken before this point in time. Specify
				  in the format YYYYMMDD`,
		},
		commands.FlagInfo{
			Name: "after", Short: 'A', Valued: true,
			Desc: `Show only the archives taken after this point in time. Specify
				  in the format YYYYMMDD`,
		},
		commands.FlagInfo{
			Name: "all", Short: 'a',
			Desc: "Show all archives, regardless of validity. Equivalent to '--status=all'",
		},
	},
	RunFn: cliListArchives,
}

func cliListArchives(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'list archives' command")

	if *opts.Status == "" {
		*opts.Status = "valid"
	}
	if *opts.Status == "all" || *opts.All {
		*opts.Status = ""
	}
	log.DEBUG("  for status: '%s'", *opts.Status)

	if *opts.Limit == "" {
		*opts.Limit = "20"
	}
	log.DEBUG("  for limit: '%s'", *opts.Limit)

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
		internal.RawJSON(archives)
		return nil
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

	t := tui.NewTable("UUID", "Target", "Restore IP", "Store", "Size", "Taken at", "Expires at", "Encryption Type", "Status", "Notes")
	for _, archive := range archives {
		if *opts.Target != "" && archive.TargetUUID != *opts.Target {
			continue
		}
		if *opts.Store != "" && archive.StoreUUID != *opts.Store {
			continue
		}
		encType := archive.EncryptionType
		if encType == "" {
			encType = "(unencrypted)"
		}

		t.Row(archive, archive.UUID,
			fmt.Sprintf("%s (%s)", target[archive.TargetUUID].Name, archive.TargetPlugin),
			target[archive.TargetUUID].Agent,
			fmt.Sprintf("%s (%s)", store[archive.StoreUUID].Name, archive.StorePlugin),
			fmt.Sprintf("%d", archive.Size),
			archive.TakenAt.Format(time.RFC1123Z),
			archive.ExpiresAt.Format(time.RFC1123Z),
			encType,
			archive.Status, archive.Notes)
	}
	t.Output(os.Stdout)
	return nil
}
