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
	v, subtree, err := connect(endpoint)
	if err != nil {
		return err
	}

	if subtree == "" {
		subtree = "secret"
	}

	plugin.DEBUG("Reading %s/* from the Vault...", subtree)
	var output string
	if output, err = Export(v, subtree); err != nil {
		return err
	}

	return Write(output)
}

func (p VaultPlugin) Restore(endpoint plugin.ShieldEndpoint) error {
	var (
		preserve   bool
		data, prev string
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
	var input []byte
	if input, err = Read(subtree); err != nil {
		return err
	}

	if preserve {
		plugin.DEBUG("Saving seal keys for current Vault...")
		prev, err = Export(v, "secret/vault/seal/keys")
		if err != nil {
			if !vault.IsNotFound(err) {
				return err
			}
			prev = ""
		}
	}

	plugin.DEBUG("Deleting pre-existing contents of %s/* from Vault...", subtree)
	if err = v.DeleteTree(subtree, vault.DeleteOpts{Destroy: true, All: true}); err != nil && !vault.IsNotFound(err) {
		return err
	}

	plugin.DEBUG("Restoring %d paths to the Vault...", len(data))
	if err = Import(v, input); err != nil {
		return err
	}

	if prev != "" {
		plugin.DEBUG("Replacing seal keys for current Vault (overwriting those from the backup archive)...")
		return Import(v, []byte(prev))
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

type v1ExportFormat map[string]*vault.Secret

type v2ExportFormat struct {
	ExportVersion uint `json:"export_version"`
	//map from path string to map from version number to version info
	Data               map[string]exportSecret `json:"data"`
	RequiresVersioning map[string]bool         `json:"requires_versioning"`
}

type exportSecret struct {
	FirstVersion uint            `json:"first,omitempty"`
	Versions     []exportVersion `json:"versions"`
}

type exportVersion struct {
	Deleted   bool              `json:"deleted,omitempty"`
	Destroyed bool              `json:"destroyed,omitempty"`
	Value     map[string]string `json:"value,omitempty"`
}

func Export(v *vault.Vault, path string) (string, error) {
	var toExport interface{}

	//Standardize and validate path
	path = vault.Canonicalize(path)
	_, key, version := vault.ParsePath(path)
	if key != "" {
		return "", fmt.Errorf("Cannot export path with key (%s)", path)
	}

	if version > 0 {
		return "", fmt.Errorf("Cannot export path with version (%s)", path)
	}

	secrets, err := v.ConstructSecrets(path, vault.TreeOpts{
		FetchKeys:           true,
		FetchAllVersions:    true,
		GetDeletedVersions:  true,
		AllowDeletedSecrets: true,
	})
	if err != nil {
		return "", err
	}

	var mustV2Export bool
	//Determine if we can get away with a v1 export
	for _, s := range secrets {
		if len(s.Versions) > 1 {
			mustV2Export = true
			break
		}
	}

	v1Export := func() error {
		export := v1ExportFormat{}
		for _, s := range secrets {
			export[s.Path] = s.Versions[0].Data
		}

		toExport = export
		return nil
	}

	v2Export := func() error {
		export := v2ExportFormat{ExportVersion: 2, Data: map[string]exportSecret{}, RequiresVersioning: map[string]bool{}}

		for _, secret := range secrets {
			if len(secret.Versions) > 1 {
				mount, _ := v.Client().MountPath(secret.Path)
				export.RequiresVersioning[mount] = true
			}

			thisSecret := exportSecret{FirstVersion: secret.Versions[0].Number}
			//We want to omit the `first` key in the json if it's 1
			if thisSecret.FirstVersion == 1 {
				thisSecret.FirstVersion = 0
			}

			for _, version := range secret.Versions {
				thisVersion := exportVersion{
					Deleted:   version.State == vault.SecretStateDeleted,
					Destroyed: version.State == vault.SecretStateDestroyed,
					Value:     map[string]string{},
				}

				for _, key := range version.Data.Keys() {
					thisVersion.Value[key] = version.Data.Get(key)
				}

				thisSecret.Versions = append(thisSecret.Versions, thisVersion)
			}

			export.Data[secret.Path] = thisSecret

			//Wrap export in array so that older versions of safe don't try to import this improperly.
			toExport = []v2ExportFormat{export}
		}

		return nil
	}

	if mustV2Export {
		err = v2Export()
	} else {
		err = v1Export()
	}

	if err != nil {
		return "", err
	}
	b, err := json.Marshal(&toExport)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Import(v *vault.Vault, backup []byte) error {
	type importFunc func([]byte) error

	v1Import := func(input []byte) error {
		data := v1ExportFormat{}
		err := json.Unmarshal(input, &data)
		if err != nil {
			return err
		}
		for path, s := range data {
			err = v.Write(path, s)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "wrote %s\n", path)
		}
		return nil
	}

	v2Import := func(input []byte) error {
		var unmarshalTarget []v2ExportFormat
		err := json.Unmarshal(input, &unmarshalTarget)
		if err != nil {
			return fmt.Errorf("Could not interpret export file: %s", err)
		}

		if len(unmarshalTarget) != 1 {
			return fmt.Errorf("Improperly formatted export file")
		}

		data := unmarshalTarget[0]

		//Verify that the mounts that require versioning actually support it. We
		//can't really detect if v1 mounts exist at this stage unless we assume
		//the token given has mount listing privileges. Not a big deal, because
		//it will become very apparent once we start trying to put secrets in it
		for mount, needsVersioning := range data.RequiresVersioning {
			if needsVersioning {
				mountVersion, err := v.MountVersion(mount)
				if err != nil {
					return fmt.Errorf("Could not determine existing mount version: %s", err)
				}

				if mountVersion != 2 {
					return fmt.Errorf("Export for mount `%s' has secrets with multiple versions, but the mount either\n"+
						"does not exist or does not support versioning", mount)
				}
			}
		}

		//Put the secrets in the places, writing the versions in the correct order and deleting/destroying secrets that
		// need to be deleted/destroyed.
		for path, secret := range data.Data {
			s := vault.SecretEntry{
				Path: path,
			}

			firstVersion := secret.FirstVersion
			if firstVersion == 0 {
				firstVersion = 1
			}

			for i := range secret.Versions {
				state := vault.SecretStateAlive
				if secret.Versions[i].Destroyed {
					state = vault.SecretStateDestroyed
				} else if secret.Versions[i].Deleted {
					state = vault.SecretStateDeleted
				}
				data := vault.NewSecret()
				for k, v := range secret.Versions[i].Value {
					data.Set(k, v, false)
				}
				s.Versions = append(s.Versions, vault.SecretVersion{
					Number: firstVersion + uint(i),
					State:  state,
					Data:   data,
				})
			}

			err := s.Copy(v, s.Path, vault.TreeCopyOpts{
				Clear: true,
				Pad:   true,
			})
			if err != nil {
				return err
			}
		}

		return nil
	}

	var fn importFunc
	//determine which version of the export format this is
	var typeTest interface{}
	json.Unmarshal(backup, &typeTest)
	switch v := typeTest.(type) {
	case map[string]interface{}:
		fn = v1Import
	case []interface{}:
		if len(v) == 1 {
			if meta, isMap := (v[0]).(map[string]interface{}); isMap {
				version, isFloat64 := meta["export_version"].(float64)
				if isFloat64 && version == 2 {
					fn = v2Import
				}
			}
		}
	}

	if fn == nil {
		return fmt.Errorf("Unknown export file format - aborting")
	}

	return fn(backup)
}

func Read(subtree string) ([]byte, error) {
	return ioutil.ReadAll(os.Stdin)
}

func Write(output string) error {
	fmt.Printf("%s\n", output)
	return nil
}
