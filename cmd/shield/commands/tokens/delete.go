package tokens

import (
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Delete - Delete token with the given name
var Delete = &commands.Command{
	Summary: "Delete token with the given name",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "tokenname", Desc: "The name of the token to delete",
				Positional: true, Mandatory: true,
			},
		},
		//TODO: Update this
		JSONOutput: `[{
			"uuid":"6e83bfb7-7ae1-4f0f-88a8-84f0fe4bae20",
			"name":"test store",
			"summary":"a test store named \"test store\"",
			"plugin":"s3",
			"endpoint":"{ \"endpoint\": \"doesntmatter\" }"
		}]`,
	},
	RunFn: cliDeleteToken,
	Group: commands.TokensGroup,
}

func cliDeleteToken(opts *commands.Options, args ...string) error {
	log.DEBUG("running `delete-token' command")
	internal.Require(len(args) == 1, "shield delete-token <tokenname>")

	token, uuid, err := internal.FindToken(args[0], *opts.Raw)
	if err != nil {
		return err
	}

	if !*opts.Raw {
		Show(&token)
		if !tui.Confirm("Really delete this token?") {
			return internal.ErrCanceled
		}
	}

	err = api.DeleteToken(uuid.String())
	if err != nil {
		return err
	}

	commands.OK("Deleted token")
	return nil
}
