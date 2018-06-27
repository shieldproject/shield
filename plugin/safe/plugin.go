package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/safe/vault"
	"github.com/starkandwayne/shield/plugin"
)

var ()

func main() {
	p := SafePlugin{
		Name:    "Safe Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
	"vault_url"           : "https://safe.myorg.mycompany.com",     # REQUIRED
	"auth_token"          : "b8714fec-0df9-3f66-d262-35a57e414120", # REQUIRED
	"skip_ssl_validation" : true,                                   # REQUIRED
}
`,
		Defaults: `
{
	"skip_ssl_validation" : false,
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "vault_url",
				Type:     "string",
				Title:    "Vault URL",
				Help:     "The url address of your Vault server, including the protocol.",
				Example:  "https://safe.myorg.mycompany.com",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "auth_token",
				Type:     "string",
				Title:    "Auth Token",
				Help:     "The auth token for a user with privileges to read and write the entire secret/ tree",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "skip_ssl_validation",
				Type:    "bool",
				Title:   "Skip SSL Validation",
				Help:    "Set to true if using a self-signed cert",
				Default: "false",
			},
		},
	}

	plugin.Run(p)
}

type SafePlugin plugin.PluginInfo

type SafeConnectionInfo struct {
	URL            string
	Token          string
	SkipValidation bool
}

func (p SafePlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p SafePlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("vault_url")
	if err != nil {
		fmt.Printf("@R{\u2717 vault_url            %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 vault_url}            @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("auth_token")
	if err != nil {
		fmt.Printf("@R{\u2717 auth_token           %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 auth_token}           @C{%s}\n", plugin.Redact(s))
	}

	skipValidation, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{%t}\n", skipValidation)
	}

	if fail {
		return fmt.Errorf("safe: invalid configuration")
	}
	return nil
}

func (p SafePlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	target, err := safeConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	v, err := safeConnect(target)
	if err != nil {
		return err
	}

	plugin.DEBUG("Reading secrets from the Vault...")
	data := make(SafeContents)
	if err = data.ReadFromVault(v, "secret"); err != nil {
		return err
	}

	return data.WriteToStdout()
}

func (p SafePlugin) Restore(endpoint plugin.ShieldEndpoint) error {

	target, err := safeConnectionInfo(endpoint)
	if err != nil {
		return err
	}

	v, err := safeConnect(target)
	if err != nil {
		return err
	}

	plugin.DEBUG("Reading secrets backup from stdin...")
	data := make(SafeContents)
	err = data.ReadFromStdin()
	if err != nil {
		return err
	}

	plugin.DEBUG("Grabbing Safe metadata for current Vault")
	sealKeysData := make(SafeContents)
	sealKeysErr := sealKeysData.ReadFromVault(v, "secret/vault/seal/keys")
	if sealKeysErr != nil && !vault.IsNotFound(sealKeysErr) {
		return sealKeysErr
	}

	plugin.DEBUG("Cleaning out the old secrets from the Vault...")
	if err = v.DeleteTree("secret"); err != nil && !vault.IsNotFound(err) {
		return err
	}

	plugin.DEBUG("Restoring backup contents to the Vault")
	err = data.WriteToVault(v)
	if err != nil {
		return err
	}

	if sealKeysErr == nil {
		plugin.DEBUG("Rewriting Safe metadata to the Vault")
		return sealKeysData.WriteToVault(v)
	}
	return nil
}

func (p SafePlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p SafePlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p SafePlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func safeConnect(target *SafeConnectionInfo) (*vault.Vault, error) {
	if target.SkipValidation {
		os.Setenv("VAULT_SKIP_VERIFY", "1")
	}
	v, err := vault.NewVault(target.URL, target.Token, true)
	return v, err
}

func safeConnectionInfo(endpoint plugin.ShieldEndpoint) (*SafeConnectionInfo, error) {
	url, err := endpoint.StringValue("vault_url")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("VAULT_URL: '%s'", url)

	token, err := endpoint.StringValue("auth_token")
	if err != nil {
		return nil, err
	}
	plugin.DEBUG("AUTH_TOKEN: '%s'", token)

	skipSslValidation, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, err
	}
	if skipSslValidation {
		plugin.DEBUG("Skipping SSL validation")
	}

	return &SafeConnectionInfo{
		URL:            url,
		Token:          token,
		SkipValidation: skipSslValidation,
	}, nil
}

type SafeContents map[string]*vault.Secret

func (sc SafeContents) ReadFromVault(v *vault.Vault, path string) error {

	tree, err := v.Tree(path, vault.TreeOptions{
		StripSlashes: true,
	})
	if err != nil {
		return err
	}
	for _, sub := range tree.Paths("/") {
		s, err := v.Read(sub)
		if err != nil {
			return err
		}
		sc[sub] = s
	}
	return nil
}

func (sc SafeContents) ReadFromStdin() error {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &sc)
}

func (sc SafeContents) WriteToVault(v *vault.Vault) error {
	for path, s := range sc {
		err := v.Write(path, s)
		if err != nil {
			return err
		}
		plugin.DEBUG(" -- wrote contents to %s", path)
	}
	return nil
}

func (sc SafeContents) WriteToStdout() error {
	b, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(b))
	return nil
}
