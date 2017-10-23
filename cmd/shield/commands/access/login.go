package access

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/commands/internal"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"golang.org/x/crypto/ssh/terminal"
)

//Login - Authenticate with the currently targeted SHIELD backend for future commands
var Login = &commands.Command{
	Summary: "Authenticate with the currently targeted SHIELD backend for future commands",
	Flags: commands.FlagList{
		commands.FlagInfo{
			Name: "provider", Desc: "Provider to authenticate against. Uses local user auth if not given",
		},
		commands.FlagInfo{
			Name: "username", Short: 'U', Desc: "Username to use for local user or basic auth login",
		},
		commands.FlagInfo{
			Name: "password", Desc: "Password to use for local user or basic auth login",
		},
	},
	RunFn: cliLogin,
}

func cliLogin(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'login' command")

	internal.Require(len(args) == 0, "USAGE: shield login [--provider=VALUE] [--token=VALUE]")

	err := Logout.RunFn(opts)
	if err != nil {
		return err
	}

	curBackend := config.Current()
	authType, _, err := api.FetchAuthType(*opts.Provider)
	if err != nil {
		return err
	}

	var token string

	switch authType {
	case api.AuthV1Basic:
		log.DEBUG("V1 Basic auth detected")
		token, err = v1BasicAuthToken(*opts.User, *opts.Password)

	case api.AuthV1OAuth:
		log.DEBUG("V1 OAuth detected")
		return fmt.Errorf("V1 OAuth SHIELD is not supported by this version of the CLI")

	case api.AuthV2Local:
		log.DEBUG("V2 Local User Auth detected")
		token, err = v2LocalAuthSession(*opts.User, *opts.Password)

	default:
		log.DEBUG("Unknown auth type")
		return fmt.Errorf("Unable to detect auth type")
	}

	if err != nil {
		if _, unauthorized := err.(api.ErrUnauthorized); unauthorized {
			return fmt.Errorf("The provided credentials were incorrect")
		}
		return err
	}

	log.DEBUG("Token: %s", token)
	curBackend.Token = token
	err = api.SetBackend(curBackend)
	if err != nil {
		return err
	}

	log.DEBUG("curToken: %s", config.Current().Token)

	if curBackend.APIVersion == 2 {
		err = Whoami.RunFn(opts)
	} else {
		_, err = api.GetStatus()
	}

	if err != nil {
		if _, unauthorized := err.(api.ErrUnauthorized); unauthorized {
			return fmt.Errorf("The provided credentials were incorrect")
		}
		return err
	}

	return config.Commit(curBackend)
}

func promptUser() (username string, err error) {
	fmt.Fprintf(os.Stdout, "Username: ")
	_, err = fmt.Scanln(&username)
	return
}

func promptPassword() (password string, err error) {
	fmt.Fprintf(os.Stdout, "Password: ")
	tmpPass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return
	}

	password = string(tmpPass)
	return
}

func v1BasicAuthToken(username, password string) (token string, err error) {
	if username == "" {
		username, err = promptUser()
		if err != nil {
			return
		}
	}

	if password == "" {
		password, err = promptPassword()
		if err != nil {
			return
		}
	}

	fmt.Println("")

	b64enc := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	return fmt.Sprintf("Basic %s", b64enc), nil
}

func v2LocalAuthSession(username, password string) (token string, err error) {
	if username == "" {
		username, err = promptUser()
		if err != nil {
			return
		}
	}

	if password == "" {
		password, err = promptPassword()
		if err != nil {
			return
		}
	}

	fmt.Println("")
	token, _, err = api.Login(username, password)
	return
}
