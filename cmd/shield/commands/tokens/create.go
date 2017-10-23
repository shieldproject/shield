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
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "tokenname", Desc: "The name of the token to create",
			Positional: true, Mandatory: true,
		},
	},
	RunFn: cliCreateToken,
}

func cliCreateToken(opts *commands.Options, args ...string) error {
	log.DEBUG("running `create-auth-token' command")

	internal.Require(len(args) == 1, "shield create-auth-token NAME")
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

//Show displays a token
func Show(token *api.Token) {
	t := tui.NewReport()
	if token.Session != "" {
		t.Add("Token", token.Session)
	}
	t.Add("Name", token.Name)
	t.Add("Created At", token.CreatedAt)
	if token.Session == "" {
		if token.LastSeen == "" {
			token.LastSeen = "never"
		}
		t.Add("Last Seen", token.LastSeen)
	}
	t.Output(os.Stdout)
}
