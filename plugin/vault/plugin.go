package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"

	"github.com/starkandwayne/safe/vault"
	"github.com/starkandwayne/shield/plugin"
)

var ()

func main() {
	p := VaultPlugin{
		Name:    "Vault Backup Plugin",
		Author:  "Stark & Wayne",
		Version: "0.0.1",
		Features: plugin.PluginFeatures{
			Target: "yes",
			Store:  "no",
		},
		Example: `
{
	"url"                 : "https://vault.myorg.mycompany.com",    # REQUIRED
	"token"               : "b8714fec-0df9-3f66-d262-35a57e414120", # REQUIRED
	"skip_ssl_validation" : true,                                   # REQUIRED

	"subtree"             : "secret/some/sub/tree",                 # OPTIONAL
}
`,
		Defaults: `
{
	"subtree"             : "secret",
	"skip_ssl_validation" : false
}
`,
		Fields: []plugin.Field{
			plugin.Field{
				Mode:     "target",
				Name:     "url",
				Type:     "string",
				Title:    "Vault URL",
				Help:     "The url address of your Vault server, including the protocol.",
				Example:  "https://vault.myorg.mycompany.com",
				Required: true,
			},
			plugin.Field{
				Mode:     "target",
				Name:     "token",
				Type:     "password",
				Title:    "Auth Token",
				Help:     "The auth token for a user with privileges to read and write the entire secret/ tree.",
				Required: true,
			},
			plugin.Field{
				Mode:    "target",
				Name:    "subtree",
				Type:    "string",
				Title:   "Vault Path Subtree",
				Help:    "A subtree to limit the backup operation to.",
				Default: "",
			},
			plugin.Field{
				Mode:    "target",
				Name:    "skip_ssl_validation",
				Type:    "bool",
				Title:   "Skip SSL Validation",
				Help:    "If your Vault certificate is invalid, expired, or signed by an unknown Certificate Authority, you can disable SSL validation.  This is not recommended from a security standpoint, however.",
				Default: "false",
			},
		},
	}

	plugin.Run(p)
}

type VaultPlugin plugin.PluginInfo

func (p VaultPlugin) Meta() plugin.PluginInfo {
	return plugin.PluginInfo(p)
}

func (p VaultPlugin) Validate(endpoint plugin.ShieldEndpoint) error {
	var (
		s    string
		err  error
		fail bool
	)

	s, err = endpoint.StringValue("url")
	if err != nil {
		fmt.Printf("@R{\u2717 url                  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 url}                  @C{%s}\n", s)
	}

	s, err = endpoint.StringValue("token")
	if err != nil {
		fmt.Printf("@R{\u2717 token                %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 token}                @C{%s}\n", plugin.Redact(s))
	}

	s, err = endpoint.StringValueDefault("subtree", "")
	if err != nil {
		fmt.Printf("@R{\u2717 subtree              %s}\n", err)
		fail = true
	} else if s == "" {
		fmt.Printf("@G{\u2713 subtree}              @C{secret}/* (everything)\n")
	} else {
		fmt.Printf("@G{\u2713 subtree}              @C{%s}/*\n", s)
	}

	yes, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		fmt.Printf("@R{\u2717 skip_ssl_validation  %s}\n", err)
		fail = true
	} else {
		fmt.Printf("@G{\u2713 skip_ssl_validation}  @C{%t}\n", yes)
	}

	if fail {
		return fmt.Errorf("vault: invalid configuration")
	}
	return nil
}

func (p VaultPlugin) Backup(endpoint plugin.ShieldEndpoint) error {
	var data Vault

	v, subtree, err := connect(endpoint)
	if err != nil {
		return err
	}

	if subtree == "" {
		subtree = "secret"
	}

	plugin.DEBUG("Reading %s/* from the Vault...", subtree)
	data = make(Vault)
	if err = data.Export(v, subtree); err != nil {
		return err
	}

	plugin.DEBUG("Exported %d paths from the Vault successfully.", len(data))
	return data.Write()
}

func (p VaultPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	var (
		preserve   bool
		data, prev Vault
	)

	v, subtree, err := connect(endpoint)
	if err != nil {
		return err
	}

	if subtree == "" {
		subtree = "secret"
		preserve = true
	}

	plugin.DEBUG("Reading contents of backup archive...")
	data = make(Vault)
	if err = data.Read(subtree); err != nil {
		return err
	}

	if preserve {
		prev = make(Vault)
		plugin.DEBUG("Saving seal keys for current Vault...")
		err = prev.Export(v, "secret/vault/seal/keys")
		if err != nil {
			if !vault.IsNotFound(err) {
				return err
			}
			prev = nil
		}
	}

	plugin.DEBUG("Deleting pre-existing contents of %s/* from Vault...", subtree)
	if err = v.DeleteTree(subtree); err != nil && !vault.IsNotFound(err) {
		return err
	}

	plugin.DEBUG("Restoring %d paths to the Vault...", len(data))
	if err = data.Import(v); err != nil {
		return err
	}

	if prev != nil {
		plugin.DEBUG("Replacing seal keys for current Vault (overwriting those from the backup archive)...")
		return prev.Import(v)
	}
	return nil
}

func (p VaultPlugin) Store(endpoint plugin.ShieldEndpoint) (string, int64, error) {
	return "", 0, plugin.UNIMPLEMENTED
}

func (p VaultPlugin) Retrieve(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func (p VaultPlugin) Purge(endpoint plugin.ShieldEndpoint, file string) error {
	return plugin.UNIMPLEMENTED
}

func connect(endpoint plugin.ShieldEndpoint) (*vault.Vault, string, error) {
	url, err := endpoint.StringValue("url")
	if err != nil {
		return nil, "", err
	}
	plugin.DEBUG("VAULT_URL: '%s'", url)

	token, err := endpoint.StringValue("token")
	if err != nil {
		return nil, "", err
	}
	plugin.DEBUG("AUTH_TOKEN: '%s'", token)

	skipSslValidation, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, "", err
	}
	if skipSslValidation {
		plugin.DEBUG("Skipping SSL validation")
		os.Setenv("VAULT_SKIP_VERIFY", "1")
	}

	subtree, err := endpoint.StringValueDefault("subtree", "")
	if err != nil {
		return nil, "", err
	}

	v, err := vault.NewVault(url, token, true)
	return v, subtree, err
}

type Vault map[string]*vault.Secret

func (v Vault) Export(from *vault.Vault, path string) error {
	tree, err := from.Tree(path, vault.TreeOptions{
		StripSlashes: true,
	})
	if err != nil {
		return err
	}

	for _, path := range tree.Paths("/") {
		s, err := from.Read(path)
		if err != nil {
			return err
		}
		plugin.DEBUG(" -- read %s", path)
		v[path] = s
	}
	return nil
}

func (v Vault) Import(to *vault.Vault) error {
	for path, s := range v {
		err := to.Write(path, s)
		if err != nil {
			return err
		}
		plugin.DEBUG(" -- wrote contents to %s", path)
	}
	return nil
}

func (v Vault) Read(subtree string) error {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	if !strings.HasSuffix(subtree, "/") {
		subtree += "/"
	}

	for key := range v {
		if !strings.HasPrefix(key, subtree) {
			plugin.DEBUG(" -- IGNORING %s (not under %s*)", key, subtree)
			delete(v, key)
		}
	}

	return nil
}

func (v Vault) Write() error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", string(b))
	return nil
}
