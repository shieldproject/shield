package archives

import (
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
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

	internal.ShowArchive(archive)
	return nil
}
