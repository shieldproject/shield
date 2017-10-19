package tokens

import (
	"os"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Create - Create token for the currently authenticated user
var Create = &commands.Command{
	Summary: "Create token for the currently authenticated user",
	Help: &commands.HelpInfo{
		Flags: []commands.FlagInfo{
			commands.FlagInfo{
				Name: "tokenname", Desc: "The name of the token to create",
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
	RunFn: cliCreateToken,
	Group: commands.TokensGroup,
}

func cliCreateToken(opts *commands.Options, args ...string) error {
	log.DEBUG("running `create-token' command")

	internal.Require(len(args) == 1, "shield create-token <tokenname>")
	token, err := api.CreateToken(args[0])
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(token)
		return nil
	}

	ansi.Fprintf(os.Stderr, "@Y{This is the only time you can see the value of this token, so store it somewhere safe}\n")

	Show(token)

	return nil
}

//Show displays information about the given Store to stdout
func Show(token *api.Token) {
	t := tui.NewReport()
	t.Add("Name", token.Name)
	if token.Token != "" {
		t.Add("Token", token.Token)
	}
	t.Add("Created At", token.CreatedAt)
	if token.Token == "" {
		if token.LastUsedAt == "" {
			token.LastUsedAt = "never"
		}
		t.Add("Last Used At", token.LastUsedAt)
	}
	t.Output(os.Stdout)
}
