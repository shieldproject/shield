package archives

import (
	"fmt"
	"os"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Get - Print detailed information about a backup archive
var Get = &commands.Command{
	Summary: "Print detailed information about a backup archive",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "uuid", Positional: true, Mandatory: true,
				Desc: "A UUID assigned to a single archive instance",
			},
		},
		JSONOutput: `{
			"uuid":"b4a842c5-cb61-4fa1-b0c7-08260fdc3533",
			"key":"thisisastorekey",
			"taken_at":"2016-05-18 11:02:43",
			"expires_at":"2017-05-18 11:02:43",
			"status":"valid",
			"notes":"",
			"target_uuid":"b7aa8269-008d-486a-ba1b-610ee191e4c1",
			"target_plugin":"redis-broker",
			"target_endpoint":"{\"redis_type\":\"broker\"}",
			"store_uuid":"6d52c95f-8d7f-4697-ae32-b9ce51fb4808",
			"store_plugin":"s3",
			"store_endpoint":"{\"endpoint\":\"schmendpoint\"}"
		}`,
	},
	RunFn: cliGetArchive,
	Group: commands.ArchivesGroup,
}

func cliGetArchive(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'show archive' command")

	internal.Require(len(args) == 1, "shield show archive <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  archive UUID = '%s'", id)

	archive, err := api.GetArchive(id)
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(archive)
		return nil
	}
	if *opts.ShowUUID {
		internal.RawUUID(archive.UUID)
		return nil
	}

	Show(archive)
	return nil
}

//Show displays information about the given archive to stdout
func Show(archive api.Archive) {
	t := tui.NewReport()
	t.Add("UUID", archive.UUID)
	t.Add("Backup Key", archive.StoreKey)
	t.Add("Target", fmt.Sprintf("%s %s", archive.TargetPlugin, archive.TargetEndpoint))
	t.Add("Store", fmt.Sprintf("%s %s", archive.StorePlugin, archive.StoreEndpoint))
	if archive.EncryptionType == "" {
		archive.EncryptionType = "(unencrypted)"
	}
	t.Add("Encryption Type", archive.EncryptionType)
	t.Break()

	t.Add("Taken at", archive.TakenAt.Format(time.RFC1123Z))
	t.Add("Expires at", archive.ExpiresAt.Format(time.RFC1123Z))
	t.Add("Notes", archive.Notes)

	t.Output(os.Stdout)
}
