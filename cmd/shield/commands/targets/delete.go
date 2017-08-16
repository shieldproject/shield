package targets

import (
	"strings"

	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/tui"
)

func init() {
	dtarget := commands.Register("delete-target", cliDeleteTarget)
	dtarget.Summarize("Delete a backup target")
	dtarget.Aliases("delete target", "remove target", "rm target")
	dtarget.Help(commands.HelpInfo{
		Flags:      []commands.FlagInfo{commands.TargetNameFlag},
		JSONOutput: `{"ok":"Deleted target"}`,
	})
	dtarget.HelpGroup(commands.TargetsGroup)

}

//Delete a backup target
func cliDeleteTarget(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'delete target' command")

	target, id, err := internal.FindTarget(strings.Join(args, " "), *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		internal.ShowTarget(target)
		if !tui.Confirm("Really delete this target?") {
			return internal.ErrCanceled
		}
	}

	if err := api.DeleteTarget(id); err != nil {
		return err
	}

	commands.OK("Deleted target")
	return nil
}
