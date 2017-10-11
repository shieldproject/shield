package access

import (
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
)

//Logout - End your authentication session with the SHIELD backend manually
var Logout = &commands.Command{
	Summary: "End your authentication session with the SHIELD backend manually",
	Help:    &commands.HelpInfo{},
	RunFn:   cliLogout,
	Group:   commands.AccessGroup,
}

func cliLogout(opts *commands.Options, args ...string) error {
	if config.Current().APIVersion == 2 {
		api.Logout() //Ignore logout response. We're just making an effort to clear the session
	}
	wipeCurrentToken()
	return nil
}

//wipeCurrentToken commits an empty string token to the currently selected
//config.
func wipeCurrentToken() {
	curBackend := config.Current()
	curBackend.Token = ""
	err := config.Commit(curBackend)
	if err != nil {
		panic("Couldn't clear token from backend")
	}
}
