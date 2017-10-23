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
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "uuid", Positional: true, Mandatory: true,
			Desc: "A UUID assigned to a single archive instance",
		},
	},
	RunFn: cliGetArchive,
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
	t.Add("Size", fmt.Sprintf("%d", archive.Size))
	t.Add("Encryption Type", archive.EncryptionType)
	t.Break()

	t.Add("Taken at", archive.TakenAt.Format(time.RFC1123Z))
	t.Add("Expires at", archive.ExpiresAt.Format(time.RFC1123Z))
	t.Add("Notes", archive.Notes)

	t.Output(os.Stdout)
}
