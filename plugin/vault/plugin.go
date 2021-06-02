package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/cloudfoundry-community/vaultkv"
	fmt "github.com/jhunt/go-ansi"

	"github.com/shieldproject/shield/plugin"
)

var ()

func main() {
	p := VaultPlugin{
		Name:    "Vault Backup Plugin",
		Author:  "SHIELD Core Team",
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
	"subtree"             : "secret/some/sub/tree"                  # OPTIONAL
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
			plugin.Field{
				Mode:    "target",
				Name:    "namespace",
				Type:    "string",
				Title:   "Namespace",
				Help:    "If you are using a Vault Enterprise namespace, set this as the namespace to target.",
				Default: "",
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

	output, err := Export(v, subtree)
	if err != nil {
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
			if !vaultkv.IsNotFound(err) {
				return err
			}

			fmt.Fprintf(os.Stderr, "Skipping saving seal keys because they do not exist in the Vault")
			prev = ""
		}
	}

	plugin.DEBUG("Deleting pre-existing contents of %s/* from Vault...", subtree)
	listChan := make(chan string, 500)
	errChan := make(chan error)
	go func() {
		err := recursivelyList(subtree, v, listChan)
		if err != nil {
			errChan <- err
			return
		}

		close(listChan)
	}()

	err = destroyGivenSecrets(v, listChan)
	if err != nil {
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

func connect(endpoint plugin.ShieldEndpoint) (*vaultkv.KV, string, error) {
	vaultURL, err := endpoint.StringValue("url")
	if err != nil {
		return nil, "", err
	}
	plugin.DEBUG("VAULT_URL: '%s'", vaultURL)

	token, err := endpoint.StringValue("token")
	if err != nil {
		return nil, "", err
	}
	if token == "" {
		return nil, "", fmt.Errorf("connect failed: vault token was left empty")
	}
	plugin.DEBUG("AUTH_TOKEN: '%s'", token)

	skipSSLValidation, err := endpoint.BooleanValueDefault("skip_ssl_validation", false)
	if err != nil {
		return nil, "", err
	}
	if skipSSLValidation {
		plugin.DEBUG("Skipping SSL validation")
	}

	namespace, err := endpoint.StringValueDefault("namespace", "")
	if err != nil {
		return nil, "", err
	}
	if namespace != "" {
		plugin.DEBUG("Setting enterprise namespace to '%s'", namespace)
	}

	subtree, err := endpoint.StringValueDefault("subtree", "")
	if err != nil {
		return nil, "", err
	}

	vaultURLParsed, err := url.Parse(strings.TrimSuffix(vaultURL, "/"))
	if err != nil {
		return nil, "", fmt.Errorf("Could not parse Vault URL: %s", err)
	}

	//The default port for Vault is typically 8200 (which is the VaultKV default),
	// but safe has historically ignored that and used the default http or https
	// port, depending on which was specified as the scheme
	if vaultURLParsed.Port() == "" {
		port := ":80"
		if strings.ToLower(vaultURLParsed.Scheme) == "https" {
			port = ":443"
		}
		vaultURLParsed.Host = vaultURLParsed.Host + port
	}

	v := vaultkv.Client{
		VaultURL:  vaultURLParsed,
		AuthToken: token,
		Namespace: namespace,
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipSSLValidation,
				},
				MaxIdleConnsPerHost: 100,
			},
		},
	}

	return v.NewKV(), subtree, err
}

type v1ExportFormat map[string]map[string]interface{}

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
	Deleted   bool                   `json:"deleted,omitempty"`
	Destroyed bool                   `json:"destroyed,omitempty"`
	Value     map[string]interface{} `json:"value,omitempty"`
}

func Export(v *vaultkv.KV, path string) (string, error) {
	vaultData, err := scrape(v, path)
	if err != nil {
		return "", err
	}

	output, err := formatExport(v, vaultData)
	if err != nil {
		return "", err
	}

	return output, nil
}

func scrape(v *vaultkv.KV, path string) (map[string]exportSecret, error) {
	//Standardize and validate path
	path = canonizePath(path)

	listedPaths := make(chan string, 500)
	errChan := make(chan error, 1)

	go func() {
		if err := recursivelyList(path, v, listedPaths); err != nil {
			errChan <- err
			return
		}
		close(listedPaths)
	}()

	numWorkers := getNumWorkers()

	type pathSecretPair struct {
		path   string
		secret *exportSecret
	}

	retrievedSecretsChan := make(chan pathSecretPair, 500)
	getWait := sync.WaitGroup{}
	getWait.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for path := range listedPaths {
				secret, err := getAllVersionsOfSecret(path, v)
				if err != nil {
					errChan <- err
					return
				}

				if secret != nil {
					retrievedSecretsChan <- pathSecretPair{path, secret}
				}
			}

			getWait.Done()
		}()
	}

	go func() {
		getWait.Wait()
		close(retrievedSecretsChan)
	}()

	doneChan := make(chan bool)

	secretsToExport := map[string]exportSecret{}
	go func() {
		for secret := range retrievedSecretsChan {
			secretsToExport[secret.path] = *secret.secret
		}

		doneChan <- true
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-doneChan:
		return secretsToExport, nil
	}
}

func getNumWorkers() int {
	numWorkers := runtime.NumCPU()
	if numWorkers < 1 {
		numWorkers = 1
	}

	return numWorkers
}

func formatExport(v *vaultkv.KV, secrets map[string]exportSecret) (string, error) {
	var toExport []v2ExportFormat

	export := v2ExportFormat{ExportVersion: 2, Data: map[string]exportSecret{}, RequiresVersioning: map[string]bool{}}

	for path, secret := range secrets {
		if len(secret.Versions) > 1 {
			mount, err := v.MountPath(path)
			if err != nil {
				return "", err
			}

			export.RequiresVersioning[mount] = true
		}

		export.Data[path] = secret
	}

	//Wrap export in array so that older versions of safe don't try to import this improperly.
	toExport = []v2ExportFormat{export}

	b, err := json.Marshal(&toExport)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func recursivelyList(path string, v *vaultkv.KV, listedPaths chan string) error {
	l, err := v.List(path)
	if err != nil {
		if vaultkv.IsNotFound(err) {
			mount, mountErr := v.MountPath(path)
			if mountErr != nil {
				fmt.Fprintf(os.Stderr, "Error when determining mount for path `%s': %s\n", path, mountErr)
				return mountErr
			}
			if canonizePath(mount) == canonizePath(path) {
				//then this is just a mount with no secrets in it
				return nil
			}
		}
		fmt.Fprintf(os.Stderr, "Error when listing path `%s': %s\n", path, err)
		return err
	}

	for _, val := range l {
		if !strings.HasSuffix(val, "/") {
			listedPaths <- canonizePath(fmt.Sprintf("%s/%s", path, val))
			continue
		}

		recursivelyList(canonizePath(fmt.Sprintf("%s/%s", path, val)), v, listedPaths)
		//only care about 404s and 403s at the top level. below that, it's not
		// unreasonable that permissions might not allow scraping _everything_ in
		// the tree.
		if err != nil && !(vaultkv.IsNotFound(err) || vaultkv.IsForbidden(err)) {
			return err
		}
	}

	return nil
}

func getAllVersionsOfSecret(path string, v *vaultkv.KV) (*exportSecret, error) {
	versions, err := v.Versions(path)
	if err != nil {
		if vaultkv.IsNotFound(err) || vaultkv.IsForbidden(err) {
			return nil, nil
		}

		return nil, err
	}

	secretToExport := &exportSecret{FirstVersion: versions[0].Version}
	if secretToExport.FirstVersion == 1 {
		secretToExport.FirstVersion = 0
	}

	for _, version := range versions {
		versionToExport := exportVersion{
			Destroyed: version.Destroyed,
			Deleted:   version.Deleted,
			Value:     map[string]interface{}{},
		}

		if !version.Destroyed {
			if version.Deleted {
				err = v.Undelete(path, []uint{version.Version})
				if err != nil {
					return nil, fmt.Errorf("Error unmarking deletion from path `%s', version `%d': %s", path, version.Version, err)
				}
			}

			_, err = v.Get(
				path,
				&versionToExport.Value,
				&vaultkv.KVGetOpts{Version: version.Version},
			)
			if err != nil {
				return nil, fmt.Errorf("Error getting path `%s', version `%d': %s", path, version.Version, err)
			}

			if version.Deleted {
				err = v.Delete(path, &vaultkv.KVDeleteOpts{Versions: []uint{version.Version}})
				if err != nil {
					return nil, fmt.Errorf("Error marking as deleted for path `%s', version `%d': %s", path, version.Version, err)
				}
			}
		}

		secretToExport.Versions = append(secretToExport.Versions, versionToExport)
	}

	return secretToExport, nil
}

func destroyGivenSecrets(v *vaultkv.KV, toDelete <-chan string) error {
	doneChan := make(chan bool)
	errChan := make(chan error)

	numWorkers := getNumWorkers()
	destroyWaitGroup := sync.WaitGroup{}
	destroyWaitGroup.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			for path := range toDelete {
				err := v.DestroyAll(path)
				if err != nil {
					errChan <- fmt.Errorf("Error destroying all versions of secret at path `%s': %s", path, err)
					return
				}
			}

			destroyWaitGroup.Done()
		}()
	}

	go func() {
		destroyWaitGroup.Wait()
		doneChan <- true
	}()

	var err error
	select {
	case err = <-errChan:
	case <-doneChan:
	}

	return err
}

func parseAsV2Structure(b []byte) (*v2ExportFormat, error) {
	var preliminaryDecode interface{}
	err := json.Unmarshal(b, &preliminaryDecode)
	if err != nil {
		return nil, err
	}

	var ret *v2ExportFormat

	switch preliminaryDecode.(type) {
	case map[string]interface{}:
		v1Format := v1ExportFormat{}
		err = json.Unmarshal(b, &v1Format)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshaling into v1 backup format: %s", err)
		}
		ret = convertV1ToV2(v1Format)

	case []interface{}:
		v2FormatWrapped := &[]v2ExportFormat{}
		//unmarshal and verify
		err = json.Unmarshal(b, v2FormatWrapped)
		if err != nil {
			return nil, fmt.Errorf("Error unmarshaling into v2 backup format: %s", err)
		}

		v2Format := &v2ExportFormat{}
		if len(*v2FormatWrapped) > 0 {
			v2Format = &(*v2FormatWrapped)[0]
		}

		if v2Format.ExportVersion != 2 {
			return nil, fmt.Errorf("Unsupported export version `%d', expected `2'", v2Format.ExportVersion)
		}

		ret = v2Format

	default:
		err = fmt.Errorf("Unknown Vault backup format")
	}

	return ret, err
}

func convertV1ToV2(exp v1ExportFormat) *v2ExportFormat {
	ret := &v2ExportFormat{
		ExportVersion:      2,
		RequiresVersioning: map[string]bool{},
		Data:               map[string]exportSecret{},
	}

	//Expose v1 (non-versioned) format as a v2 (versioned) backup where every
	// secret has exactly one version which can not possibly have been
	// deleted/destroyed
	for path, secret := range exp {
		ret.Data[path] = exportSecret{
			FirstVersion: 1,
			Versions: []exportVersion{
				{Value: secret},
			},
		}
	}

	return ret
}

func Import(v *vaultkv.KV, input []byte) error {
	exp, err := parseAsV2Structure(input)
	if err != nil {
		return err
	}

	//Verify that the mounts that require versioning actually support it. We
	//can't really detect if v1 mounts exist at this stage unless we assume
	//the token given has mount listing privileges. Not a big deal, because
	//it will become very apparent once we start trying to put secrets in it
	for mount, needsVersioning := range exp.RequiresVersioning {
		if needsVersioning {
			if err := verifyMountIsVersion2(v, mount); err != nil {
				return err
			}
		}
	}

	//Put the secrets in the places, writing the versions in the correct order and deleting/destroying secrets that
	// need to be deleted/destroyed.
	for path, secret := range exp.Data {
		err := writeSecret(v, path, expandVersionList(secret))
		if err != nil {
			return err
		}
	}

	return nil
}

func verifyMountIsVersion2(v *vaultkv.KV, mount string) error {
	mountVersion, err := v.MountVersion(mount)
	if err != nil {
		return fmt.Errorf("Could not determine existing mount version: %s", err)
	}

	if mountVersion != 2 {
		return fmt.Errorf("Export for mount `%s' has secrets with multiple versions, but the mount either\n"+
			"does not exist or does not support versioning", mount)
	}

	return nil
}

func expandVersionList(secret exportSecret) []exportVersion {
	ret := []exportVersion{}

	for vers := uint(1); vers < secret.FirstVersion; vers++ {
		ret = append(ret, exportVersion{
			Destroyed: true,
		})
	}

	return append(ret, secret.Versions...)
}

//this is the value that gets written for secret versions that need to be
//destroyed
var placeholderDestroyedValue = map[string]interface{}{"placeholder": "garbage"}

func writeSecret(v *vaultkv.KV, path string, versions []exportVersion) error {
	var versionsToDelete, versionsToDestroy []uint

	for i, vers := range versions {
		if vers.Destroyed {
			vers.Value = placeholderDestroyedValue
		}

		meta, err := v.Set(path, &vers.Value, nil)
		if err != nil {
			return fmt.Errorf("Error writing path `%s', version `%d': %s", path, i+1, err)
		}

		if vers.Destroyed {
			versionsToDestroy = append(versionsToDestroy, meta.Version)
		} else if vers.Deleted {
			versionsToDelete = append(versionsToDelete, meta.Version)
		}
	}

	if len(versionsToDelete) > 0 {
		err := v.Delete(path, &vaultkv.KVDeleteOpts{Versions: versionsToDelete})
		if err != nil {
			return fmt.Errorf("Error marking versions as deleted for path `%s': %s", path, err)
		}
	}

	if len(versionsToDestroy) > 0 {
		err := v.Destroy(path, versionsToDestroy)
		if err != nil {
			return fmt.Errorf("Error destroying placeholder versions for path `%s': %s", path, err)
		}
	}

	return nil
}

func Read(subtree string) ([]byte, error) {
	return ioutil.ReadAll(os.Stdin)
}

func Write(output string) error {
	fmt.Printf("%s\n", output)
	return nil
}

func canonizePath(path string) string {
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")

	re := regexp.MustCompile("//+")
	path = re.ReplaceAllString(path, "/")

	return path
}
