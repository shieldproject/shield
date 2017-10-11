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
	Help:    &commands.HelpInfo{},
	RunFn:   cliLogin,
	Group:   commands.AccessGroup,
}

func cliLogin(opts *commands.Options, args ...string) error {
	log.DEBUG("running 'login' command")

	internal.Require(len(args) == 0, "USAGE: shield login")

	wipeCurrentToken()
	curBackendCopy := *config.Current()
	authType, err := api.FetchAuthType("")

	var token string

	switch authType {
	case api.AuthV1Basic:
		log.DEBUG("V1 Basic auth detected")
		token, err = v1BasicAuthToken()

	case api.AuthV1OAuth:
		log.DEBUG("V1 OAuth detected")
		return fmt.Errorf("V1 OAuth SHIELD is not supported by this version of the CLI")

	case api.AuthV2Local:
		log.DEBUG("V2 Local User Auth detected")
		token, err = v2LocalAuthSession()
	}

	if err != nil {
		if _, unauthorized := err.(api.ErrUnauthorized); unauthorized {
			return fmt.Errorf("The provided credentials were incorrect")
		}
		return err
	}

	log.DEBUG("Token: %s", token)
	config.Current().Token = token

	if curBackendCopy.APIVersion == 2 {
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

	curBackendCopy.Token = token
	return config.Commit(&curBackendCopy)
}

func promptCreds() (username, password string, err error) {
	fmt.Fprintf(os.Stdout, "Username: ")
	_, err = fmt.Scanln(&username)
	if err != nil {
		return "", "", err
	}
	fmt.Fprintf(os.Stdout, "Password: ")
	tmpPass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", "", err
	}

	password = string(tmpPass)

	fmt.Fprintln(os.Stdout, "") // newline to line-break after the password prompt
	return
}

func v1BasicAuthToken() (token string, err error) {
	username, password, err := promptCreds()
	if err != nil {
		return "", err
	}

	b64enc := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))
	return fmt.Sprintf("Basic %s", b64enc), nil
}

func v2LocalAuthSession() (token string, err error) {
	var username, password string
	username, password, err = promptCreds()
	if err != nil {
		return
	}

	token, _, err = api.Login(username, password)
	return
}
