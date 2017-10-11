package access

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/ansi"
	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/commands"
	"github.com/starkandwayne/shield/cmd/shield/config"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"github.com/starkandwayne/shield/tui"
)

//Whoami - Get information about the currently authenticated user
var Whoami = &commands.Command{
	Summary: "Get information about the currently authenticated user",
	Help:    &commands.HelpInfo{},
	RunFn:   cliWhoami,
	Group:   commands.AccessGroup,
}

func cliWhoami(opts *commands.Options, args ...string) error {
	log.DEBUG("running command `whoami'")

	var err error
	if config.Current().APIVersion == 1 {
		err = v1WhoAmI()
	} else {
		err = v2WhoAmI()
	}

	return err
}

func v1WhoAmI() error {
	token := config.Current().Token
	if token == "" {
		return api.NewErrUnauthorized("")
	}

	b64Token := strings.TrimPrefix(token, "Basic ")
	asciiToken, err := base64.StdEncoding.DecodeString(b64Token)
	if err != nil {
		return err
	}

	user := strings.Split(string(asciiToken), ":")[0]
	fmt.Println("")
	ansi.Printf("@G{USER:}\n")
	userReport := tui.NewReport()
	userReport.Add("Name", user)
	userReport.Output(os.Stdout)
	fmt.Println("")
	return nil
}

func v2WhoAmI() error {
	userInfo, err := api.AuthID()
	if err != nil {
		return err
	}

	sysrole := userInfo.User.Sysrole
	if sysrole == "" {
		sysrole = "none"
	}

	fmt.Println("")
	ansi.Printf("@G{USER:}\n")
	userReport := tui.NewReport()
	userReport.Add("Name", userInfo.User.Name)
	userReport.Add("Account", userInfo.User.Account)
	userReport.Add("Backend", userInfo.User.Backend)
	userReport.Add("Sysrole", sysrole)

	userReport.Output(os.Stdout)

	fmt.Println("")
	ansi.Printf("@G{TENANTS:}\n")
	tenantsTable := tui.NewTable("Name", "Role", "UUID")
	for _, tenant := range userInfo.Tenants {
		tenantsTable.Row(tenant, tenant.Name, tenant.Role, tenant.UUID)
	}

	tenantsTable.Output(os.Stdout)
	fmt.Println("")

	return nil
}
