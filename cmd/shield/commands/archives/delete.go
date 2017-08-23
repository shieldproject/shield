package archives

import (
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete a backup archive
var Delete = &commands.Command{
	Summary: "Delete a backup archive",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "uuid", Positional: true, Mandatory: true,
				Desc: "A UUID assigned to a single archive instance",
			},
		},
		JSONOutput: `{"ok":"Deleted archive"}`,
	},
	RunFn: cliDeleteArchive,
	Group: commands.ArchivesGroup,
}

func cliDeleteArchive(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete archive' command")

	internal.Require(len(args) == 1, "USAGE: shield delete archive <UUID>")
	id := uuid.Parse(args[0])
	log.DEBUG("  archive UUID = '%s'", id)

	archive, err := api.GetArchive(id)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowArchive(archive)
		if !tui.Confirm("Really delete this archive?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteArchive(id); err != nil {
		return err
	}

	commands.OK("Deleted archive")
	return nil
}
