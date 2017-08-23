package archives

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

//Restore - Restore a backup archive
var Restore = &commands.Command{
	Summary: "Restore a backup archive",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "target or uuid", Positional: true, Mandatory: true,
				Desc: `The name or UUID of a single target to restore. In raw mode, it
				  must be a UUID assigned to a single archive instance`,
			},
		},
	},
	RunFn: cliRestoreArchive,
	Group: commands.ArchivesGroup,
}

func cliRestoreArchive(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'restore archive' command")

	var id uuid.UUID

	if *opts.Raw {
		internal.Require(len(args) == 1, "USAGE: shield restore archive <UUID>")
		id = uuid.Parse(args[0])
		log.DEBUG("  trying archive UUID '%s'", args[0])

	} else {
		target, _, err := internal.FindTarget(strings.Join(args, " "), false)
		if err != nil {
			return err
		}

		_, id, err = internal.FindArchivesFor(target, 10)
		if err != nil {
			return err
		}
	}
	log.DEBUG("  archive UUID = '%s'", id)

	var params = struct {
		Owner  string `json:"owner,omitempty"`
		Target string `json:"target,omitempty"`
	}{
		Owner: commands.CurrentUser(),
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
		internal.RawJSON(map[string]interface{}{
			"ok":        fmt.Sprintf("Scheduled immediate restore of archive '%s' %s", id, targetMsg),
			"task_uuid": taskUUID,
		})
	} else {
		//`OK` handles raw checking
		commands.OK("Scheduled immediate restore of archive '%s' %s", id, targetMsg)
		if taskUUID != "" {
			ansi.Printf("To view task, type @B{shield task %s}\n", taskUUID)
		}
	}

	return nil
}
