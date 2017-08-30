package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"github.com/starkandwayne/shield/api"
	"gopkg.in/yaml.v2"
)

var cfg *Config

//Config has information about the backends the SHIELD CLI knows about
type Config struct {
	Backend    string                 `yaml:"backend"`
	Backends   map[string]string      `yaml:"backends"`
	Aliases    map[string]string      `yaml:"aliases"`
	Properties map[string]backendInfo `yaml:"properties"`

	path string //The filepath that this config was read from
}

//BackendInfo contains all info about a backend that isn't the endpoint or the
// auth token, which are already stored in different tables
type backendInfo struct {
	SkipSSLValidation bool `yaml:"skip_ssl_validation"`
}

//NewConfig makes a fresh new blank config object
func NewConfig() *Config {
	return &Config{
		Backends:   map[string]string{},
		Aliases:    map[string]string{},
		Properties: map[string]backendInfo{},
	}
}

//Load reads in the config file at path p and sets this package's config
// to the unmarshalled contents of the file
func Load(p string) error {
	cfg = NewConfig()
	cfg.path = p
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return err
	}

	api.SetBackend(Current())

	return nil
}

//Save writes back the config to the file it was loaded in from
func Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(cfg.path), 0755)
	if err != nil {
		return err
	}

	tempFile, err := ioutil.TempFile(path.Dir(cfg.path), "shield_config")
	if err != nil {
		return err
	}

	err = os.Chmod(tempFile.Name(), 0600)
	if err != nil {
		return err
	}

	_, err = tempFile.Write(data)
	if err != nil {
		return err
	}

	err = tempFile.Close()
	if err != nil {
		return err
	}

	err = os.Rename(tempFile.Name(), cfg.path)
	if err != nil {
		return err
	}

	return nil
}

//Current returns a Backend struct containing all the information
// about the currently targeted backend. Nil if no backend is targeted.
func Current() *api.Backend {
	if cfg.Backend == "" {
		return nil
	}

	return Get(cfg.Backend)
}

//resolveAlias retrieves the address associated with the given backend alias
func resolveAlias(be string) string {
	if _, ok := cfg.Backends[be]; ok {
		return be
	} else if aliasedBE, ok := cfg.Aliases[be]; ok {
		if _, ok := cfg.Backends[aliasedBE]; ok {
			return aliasedBE
		}
	}

	return ""
}

//Get creates a backend object from the backend info
// in the config associated with that backend name. Nil if no backend with that
// name exists
func Get(name string) *api.Backend {
	uri := resolveAlias(name)
	if uri == "" {
		return nil
	}

	ret := &api.Backend{
		Name:              name,
		Address:           uri,
		Token:             cfg.Backends[uri],
		SkipSSLValidation: cfg.Properties[uri].SkipSSLValidation,
	}

	//Canonize in case there was an incorrectly formatted address originating from
	//a CLI version before one that regulated this
	ret.Canonize()
	Commit(ret)
	return ret
}

//List returns a slice of backend objects representing each of the backends known
// to the config
func List() []api.Backend {
	backends := []api.Backend{}
	for name := range cfg.Aliases {
		thisBackend := Get(name)
		if thisBackend == nil {
			panic("Searched for a non-existent backend when listing. Did you mess with the config?")
		}
		backends = append(backends, *thisBackend)
	}
	return backends
}

//Use selects the backend with the given name as the current backend
func Use(be string) error {
	if resolveAlias(be) == "" {
		return fmt.Errorf("Undefined Backend: %s", be)
	}

	cfg.Backend = be
	api.SetBackend(Current())
	return nil
}

//Commit puts this backend into the config, overwriting the old backend info if
//present. Does not write to disk. Call Save() for that.
func Commit(b *api.Backend) error {
	//Commit puts this backend into the config, overwriting the old backend info if
	//present. Does not write to disk. Call Save() for that.
	urlRE := regexp.MustCompile(`^https?://[^:]+(:\d+)?/?$`)
	if !urlRE.MatchString(b.Address) {
		return fmt.Errorf("Invalid backend format. Expecting `protocol://hostname:port/`. Got `%s`", b.Address)
	}

	b.Canonize()

	cfg.Aliases[b.Name] = b.Address
	cfg.Backends[b.Address] = b.Token
	cfg.Properties[b.Address] = backendInfo{
		SkipSSLValidation: b.SkipSSLValidation,
	}
	return nil
}
