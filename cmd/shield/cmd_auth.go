package main

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-table"
	"github.com/spf13/cobra"

	"github.com/shieldproject/shield/client/v2/shield"
	"github.com/shieldproject/shield/tui"
)

var (
	apiSkipSSL bool
	apiCACert  string

	loginProviders bool
	loginUsername  string
	loginPassword  string
	loginToken     string
	loginVia       string
)

var coresCmd = &cobra.Command{
	Use:   "cores",
	Short: "Print list of targeted SHIELD Cores",
	RunE: func(cmd *cobra.Command, args []string) error {
		tbl := table.NewTable("Name", "URL", "Verify TLS?")
		for alias, core := range cliConfig.SHIELDs {
			vfy := fmt.Sprintf("@G{yes}")
			if core.InsecureSkipVerify {
				vfy = fmt.Sprintf("@R{NO}")
			}
			tbl.Row(core, alias, core.URL, vfy)
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var apiCmd = &cobra.Command{
	Use:   "api URL ALIAS",
	Short: "Target a new SHIELD Core",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		alias := args[1]

		if ok, _ := regexp.MatchString("^http", alias); ok {
			url, alias = alias, url
		}

		cacert := ""
		if apiCACert != "" {
			if strings.Contains(apiCACert, "\n") {
				cacert = apiCACert
			} else {
				b, err := ioutil.ReadFile(apiCACert)
				bail(err)
				cacert = string(b)
			}
		}

		c := &shield.Client{
			URL:                url,
			Debug:              optDebug,
			Trace:              optTrace,
			Session:            "",
			InsecureSkipVerify: apiSkipSSL,
			CACertificate:      cacert,
			TrustSystemCAs:     true,
		}
		nfo, err := c.Info()
		bail(err)

		fmt.Printf("@C{%s}  (@B{%s})  @G{OK}\n@W{SHIELD} @Y{%s}\n", alias, url, nfo.Env)
		cliConfig.Add(alias, SHIELD{
			URL:                url,
			InsecureSkipVerify: apiSkipSSL,
			CACertificate:      cacert,
		})
		bail(cliConfig.Write())
		return nil
	},
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to the designated SHIELD Core",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()

		if loginToken == "" {
			loginToken = os.Getenv("SHIELD_CORE_TOKEN")
		}
		if loginUsername == "" {
			loginUsername = os.Getenv("SHIELD_CORE_USERNAME")
		}
		if loginPassword == "" {
			loginPassword = os.Getenv("SHIELD_CORE_PASSWORD")
		}

		if loginProviders {
			providers, err := c.AuthProviders()
			bail(err)

			if optJSON {
				fmt.Printf("%s\n", asJSON(providers))
				return nil
			}

			tbl := table.NewTable("Name", "Description", "Type")
			for _, provider := range providers {
				tbl.Row(provider, provider.Identifier, provider.Name, provider.Type)
			}
			tbl.Output(os.Stdout)
			return nil
		}

		if loginToken != "" {
			err := c.Authenticate(&shield.TokenAuth{Token: loginToken})
			bail(err)

		} else if loginUsername != "" {
			if loginPassword == "" {
				loginPassword = secureprompt("@Y{SHIELD Password:} ")
			}
			err := c.Authenticate(&shield.LocalAuth{
				Username: loginUsername,
				Password: loginPassword,
			})
			bail(err)

		} else if loginVia != "" {
			provider, err := c.AuthProviderAnonymous(loginVia)
			bail(err)

			fmt.Printf("Visit the following URL in your favorite web browser:\n\n")
			fmt.Printf("  @B{%s%s}\n\n", c.URL, provider.CLIEntry)
			fmt.Printf("Then, enter the token you get, below.\n\n")

			err = c.Authenticate(&shield.TokenAuth{
				Token: prompt("@Y{Token:} "),
			})
			bail(err)

		} else if optBatch {
			bail(fmt.Errorf("Unable to login interactively under `--batch` mode"))

		} else {
			loginUsername = prompt("@C{SHIELD Username:} ")
			loginPassword = secureprompt("@Y{SHIELD Password:} ")
			err := c.Authenticate(&shield.LocalAuth{
				Username: loginUsername,
				Password: loginPassword,
			})
			bail(err)
		}

		cliConfig.Current.Session = c.Session
		bail(cliConfig.Write())
		fmt.Printf("logged in successfully\n")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of the current authenticated session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		bail(c.Logout())
		cliConfig.Current.Session = ""
		bail(cliConfig.Write())
		fmt.Printf("logged out successfully\n")
		return nil
	},
}

var idCmd = &cobra.Command{
	Use:   "id",
	Short: "Display information about the current session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		id, err := c.AuthID()
		bail(err)

		if id.Unauthenticated {
			fmt.Printf("@Y{not authenticated}\n")
			return nil
		}

		r := tui.NewReport()
		r.Add("Display Name", id.User.Name)
		r.Add("Username", id.User.Account)
		r.Add("Designation", id.User.Account+"@"+id.User.Backend)
		if id.User.SysRole != "" {
			r.Add("System Role", id.User.SysRole)
		} else {
			r.Add("System Role", fmt.Sprintf("@Y{none}"))
		}
		fmt.Printf("@G{Account Details}\n")
		r.Output(os.Stdout)
		fmt.Printf("\n")

		if len(id.Tenants) == 0 {
			fmt.Printf("@Y{you are not assigned to any tenants}\n")
		} else {
			tbl := table.NewTable("UUID", "Name", "Role")
			for _, tenant := range id.Tenants {
				tbl.Row(tenant, tenant.UUID, tenant.Name, tenant.Role)
			}
			fmt.Printf("@G{Tenants}\n")
			tbl.Output(os.Stdout)
		}
		return nil
	},
}

var passwdCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change your password",
	RunE: func(cmd *cobra.Command, args []string) error {
		if optBatch {
			bail(fmt.Errorf("Password changes cannot be done in batch mode."))
		}
		c := clientFromConfig()

		old := secureprompt("@Y{Current pasword}: ")
		a := secureprompt("@C{Pick a new password}: ")
		b := secureprompt("@C{Confirm new password}: ")
		fmt.Printf("old=%s; a=%s; b=%s;\n", old, a, b)
		if a == "" {
			bail(fmt.Errorf("passwords cannot be blank"))
		}
		if a != b {
			bail(fmt.Errorf("passwords do not match"))
		}

		r, err := c.ChangePassword(old, a)
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(r))
			return nil
		}
		fmt.Printf("%s\n", r.OK)
		return nil
	},
}

var authTokensCmd = &cobra.Command{
	Use:   "auth-tokens",
	Short: "List your personal authentication tokens",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		tokens, err := c.ListAuthTokens()
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(tokens))
			return nil
		}

		tbl := table.NewTable("Name", "Created at", "Last seen")
		for _, token := range tokens {
			tbl.Row(token, token.Name, strftime(token.CreatedAt), strftimenil(token.LastSeen, "(never)"))
		}
		tbl.Output(os.Stdout)
		return nil
	},
}

var createAuthTokenCmd = &cobra.Command{
	Use:   "create-auth-token TOKEN-NAME",
	Short: "Issue a new personal authentication token",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		t, err := c.CreateAuthToken(&shield.AuthToken{Name: args[0]})
		bail(err)

		if optJSON {
			fmt.Printf("%s\n", asJSON(t))
			return nil
		}
		fmt.Printf("@C{%s}\n", t.Session)
		return nil
	},
}

var revokeAuthTokenCmd = &cobra.Command{
	Use:   "revoke-auth-token TOKEN-NAME [OTHER-TOKEN ...]",
	Short: "Revoke an issued authentication token",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := clientFromConfig()
		tokens, err := c.ListAuthTokens()
		bail(err)

		rc := 0
		for _, revoke := range args {
			found := false
			for _, token := range tokens {
				if token.Name == revoke {
					found = true
					if err := c.RevokeAuthToken(token); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %s\n", token.Name, err)
						rc = 3
					} else {
						fmt.Fprintf(os.Stderr, "%s: revoked\n", token.Name)
					}
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "%s: token not found\n", revoke)
				rc = 3
			}
		}
		if rc != 0 {
			os.Exit(rc)
		}
		return nil
	},
}

func init() {
	apiCmd.Flags().BoolVarP(&apiSkipSSL, "skip-ssl-validate", "k", false, "Skip TLS certificate validation")
	apiCmd.Flags().StringVar(&apiCACert, "ca-certificate", "", "CA certificate (path or PEM blob)")
	apiCmd.Flags().StringVar(&apiCACert, "ca-cert", "", "CA certificate (path or PEM blob)")

	loginCmd.Flags().BoolVar(&loginProviders, "providers", false, "List available auth providers")
	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Username")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password")
	loginCmd.Flags().StringVarP(&loginToken, "token", "a", "", "Auth token")
	loginCmd.Flags().StringVar(&loginVia, "via", "", "Auth provider identifier")

	rootCmd.AddCommand(coresCmd)
	rootCmd.AddCommand(apiCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(idCmd)
	rootCmd.AddCommand(passwdCmd)
	rootCmd.AddCommand(authTokensCmd)
	rootCmd.AddCommand(createAuthTokenCmd)
	rootCmd.AddCommand(revokeAuthTokenCmd)
}
