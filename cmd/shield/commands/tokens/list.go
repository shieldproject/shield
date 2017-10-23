package tokens

import (
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//List - List tokens created for the currently authenticated user
var List = &commands.Command{
	Summary: "List tokens created for the currently authenticated user",
	RunFn:   cliListTokens,
}

func cliListTokens(opts *commands.Options, args ...string) error {
	log.DEBUG("running `auth-tokens' command")
	tokens, err := api.ListTokens()
	if err != nil {
		return err
	}

	if *opts.Raw {
		internal.RawJSON(tokens)
		return nil
	}

	t := tui.NewTable("Name", "Created At", "Last Seen")
	for _, token := range tokens {
		if token.LastSeen == "" {
			token.LastSeen = "never"
		}
		t.Row(token, token.Name, token.CreatedAt, token.LastSeen)
	}
	t.Output(os.Stdout)

	return nil
}
