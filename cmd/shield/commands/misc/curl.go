package misc

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/log"
)

var Curl = &commands.Command{
	Summary: "Issue a REST API call and display the output as formatted JSON",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name:   "method",
			Short:  'm',
			Valued: true,
			Desc:   `HTTP request method to use, one of GET, PUT, POST, PATCH, DELETE`,
		},
		commands.FlagInfo{
			Name:       "url",
			Positional: true,
			Mandatory:  true,
			Desc:       `The path component of the URL to curl`,
		},
	},
	RunFn: func(opts *commands.Options, args ...string) error {
		log.DEBUG("running 'curl' command")

		var (
			method string
			url    string
			body   string
			err    error
		)

		if len(args) == 1 {
			method = "GET"
			url = args[0]

		} else if len(args) == 2 {
			method = args[0]
			url = args[1]

		} else {
			return fmt.Errorf("USAGE: shield curl [METHOD] URL\n")
		}

		if method == "PUT" || method == "PATCH" || method == "POST" {
			if isatty.IsTerminal(os.Stdin.Fd()) {
				return fmt.Errorf("%s methods require a request body (usually JSON).\nTry `shield curl %s %s <input.json`",
					method, method, url)
			}
			body, err = internal.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		data, err := api.Curl(method, url, body)
		if err != nil {
			return err
		}

		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", string(out))
		return nil
	},
}
