package api

import (
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

var Cfg *Config

type Config struct {
	Backend  string            `yaml:"backend"`
	Backends map[string]string `yaml:"backends"`
	Aliases  map[string]string `yaml:"aliases"`
	Path     string            `yaml:"-"` // omit this little guy
}

func LoadConfig(p string) error {
	Cfg = &Config{
		Path:     p,
		Backends: map[string]string{},
		Aliases:  map[string]string{},
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, Cfg)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) Save() error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(cfg.Path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (cfg *Config) BackendURI() string {
	if cfg.Backend == "" {
		return ""
	}

	return strings.TrimSuffix(cfg.ResolveAlias(cfg.Backend), "/")
}

func (cfg *Config) BackendToken() string {
	return cfg.Backends[cfg.BackendURI()]
}

func (cfg *Config) ResolveAlias(be string) string {
	if _, ok := cfg.Backends[be]; ok {
		return be
	} else if aliasedBE, ok := cfg.Aliases[be]; ok {
		if _, ok := cfg.Backends[aliasedBE]; ok {
			return aliasedBE
		}
	}

	return ""
}

func (cfg *Config) UpdateBackend(be string, token string) error {
	backend := cfg.ResolveAlias(be)
	if backend == "" {
		return fmt.Errorf("Could not find backend '%s' to update", be)
	}
	cfg.Backends[backend] = token
	return nil
}

func (cfg *Config) UpdateCurrentBackend(token string) error {
	return cfg.UpdateBackend(cfg.BackendURI(), token)
}

func (cfg *Config) AddBackend(be string, alias string) error {
	urlRE := regexp.MustCompile(`^https?://[^:]+(:\d+)?/?$`)
	if !urlRE.MatchString(be) {
		return fmt.Errorf("Invalid backend format. Expecting `protocol://hostname:port/`. Got `%s`", be)
	}

	// only reset to "" if it didn't exist previously
	if _, ok := cfg.Backends[be]; !ok {
		cfg.Backends[be] = ""
	}
	cfg.Aliases[alias] = be
	return nil
}

func (cfg *Config) UseBackend(be string) error {
	if cfg.ResolveAlias(be) == "" {
		return fmt.Errorf("Undefined Backend: %s", be)
	}

	cfg.Backend = be
	return nil
}

func BasicAuthToken(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}
