package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"

	"github.com/starkandwayne/shield/api"
	"github.com/starkandwayne/shield/cmd/shield/log"
	"gopkg.in/yaml.v2"
)

//cfg is the singleton Config object that the CLI will use. The functions in
//this package manipulate and read from this
var cfg *config
var current *api.Backend

//Config has information about the backends the SHIELD CLI knows about
type config struct {
	Backend    string                 `yaml:"backend"`
	Backends   map[string]string      `yaml:"backends"`
	Aliases    map[string]string      `yaml:"aliases"`
	Properties map[string]backendInfo `yaml:"properties"`

	dirty bool   //True if the config has been changed since saving
	path  string //The filepath that this config was read from
}

//BackendInfo contains all info about a backend that isn't the endpoint or the
// auth token, which are already stored in different tables
type backendInfo struct {
	SkipSSLValidation bool `yaml:"skip_ssl_validation"`
}

//Initialize makes a fresh new blank config object
func Initialize() {
	cfg = &config{
		Backends:   map[string]string{},
		Aliases:    map[string]string{},
		Properties: map[string]backendInfo{},
	}
}

//Load reads in the config file at path p and sets this package's config
// to the unmarshalled contents of the file
func Load(p string) error {
	Initialize()
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

	return nil
}

//Save writes back the config to the file it was loaded in from
func Save() error {
	if !cfg.dirty {
		return nil
	}

	log.DEBUG("Saving config to %s", cfg.path)
	collectGarbage()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		panic("Could not marshal config struct in order to save")
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

	cfg.dirty = false

	return nil
}

//Current returns a Backend struct containing all the information
// about the currently targeted backend. Nil if no backend is targeted.
func Current() *api.Backend {
	if cfg.Backend == "" {
		current = nil
	} else {
		current = Get(cfg.Backend)
		current.Canonize()
	}
	return current
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

	return ret
}

//List returns a slice of backend objects representing each of the backends known
// to the config
func List() (backends []*api.Backend) {
	for name := range cfg.Aliases {
		thisBackend := Get(name)
		if thisBackend == nil {
			panic("Searched for a non-existent backend when listing. Did you mess with the config?")
		}
		backends = append(backends, thisBackend)
	}
	sort.Slice(backends, func(i, j int) bool {
		return backends[i].Name < backends[j].Name
	})
	return backends
}

//Use selects the backend with the given name as the current backend
func Use(be string) error {
	if resolveAlias(be) == "" {
		return fmt.Errorf("Undefined Backend: %s", be)
	}

	if be == cfg.Backend {
		return nil
	}

	cfg.Backend = be
	api.SetBackend(Current())

	cfg.dirty = true
	return nil
}

//Commit puts this backend into the config, overwriting the old backend info if
//present. Does not write to disk. Call Save() for that.
//Attempting to commit an empty config is a nop.
func Commit(b *api.Backend) error {
	if reflect.DeepEqual(*b, api.Backend{}) {
		return nil
	}

	urlRE := regexp.MustCompile(`^https?://[^:]+(:\d+)?/?$`)
	if !urlRE.MatchString(b.Address) {
		return fmt.Errorf("Invalid backend format. Expecting `protocol://hostname:port/`. Got `%s`", b.Address)
	}

	if _, found := cfg.Aliases[b.Name]; found && reflect.DeepEqual(b, Get(b.Name)) {
		return nil
	}

	cfg.Aliases[b.Name] = b.Address
	cfg.Backends[b.Address] = b.Token
	cfg.Properties[b.Address] = backendInfo{
		SkipSSLValidation: b.SkipSSLValidation,
	}

	if cfg.Backend == b.Name {
		current.Address = b.Address
		current.Token = b.Token
		current.SkipSSLValidation = b.SkipSSLValidation
	}

	cfg.dirty = true
	return nil
}

//Delete removes the alias with the given name from the config
func Delete(name string) error {
	if _, found := cfg.Aliases[name]; !found {
		return fmt.Errorf("No backend with alias `%s' was found", name)
	}
	delete(cfg.Aliases, name)

	if cfg.Backend == name {
		cfg.Backend = ""
		//Make current into an empty config
		current.Name = ""
		current.Address = ""
		current.Token = ""
		current.SkipSSLValidation = false
	}

	cfg.dirty = true

	return nil
}

//Path returns the path from which the config was loaded
func Path() string {
	return cfg.path
}

func collectGarbage() {
	referencedAddrs := map[string]bool{}
	for _, addr := range cfg.Aliases {
		referencedAddrs[api.CanonizeURI(addr)] = true
	}

	for addr, token := range cfg.Backends {
		//Keep lines with tokens to preserve old behavior
		if _, found := referencedAddrs[api.CanonizeURI(addr)]; !found && token == "" {
			delete(cfg.Backends, addr)
		}
	}

	for addr := range cfg.Properties {
		if _, found := referencedAddrs[api.CanonizeURI(addr)]; !found {
			delete(cfg.Properties, addr)
		}
	}
}
