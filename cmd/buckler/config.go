package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type LegacyConfig struct {
	Backend  string            `yaml:"backend"`
	Backends map[string]string `yaml:"backends"`
	Aliases  map[string]string `yaml:"aliases"`

	Properties map[string]struct {
		InsecureSkipVerify bool   `yaml:"skip_ssl_validation,omitempty"`
		CACert             string `yaml:"ca_cert,omitempty"`
	} `yaml:"properties"`
}

type SHIELD struct {
	URL                string `yaml:"url"`
	Session            string `yaml:"session"`
	InsecureSkipVerify bool   `yaml:"skip_verify"`
	CACertificate      string `yaml:"cacert"`
}
type Config struct {
	Path    string
	Current *SHIELD
	SHIELDs map[string]*SHIELD
}

func newConfig(path string) *Config {
	return &Config{
		Path:    path,
		SHIELDs: map[string]*SHIELD{},
	}
}

func convertLegacyConfig(path, oldpath string) (*Config, error) {
	cfg := newConfig(path)

	b, err := ioutil.ReadFile(oldpath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	legacy := &LegacyConfig{}
	if err := yaml.Unmarshal(b, legacy); err != nil {
		return nil, err
	}

	for alias, url := range legacy.Aliases {
		shield := &SHIELD{URL: url}
		if session, ok := legacy.Backends[url]; ok {
			shield.Session = session
		}
		if props, ok := legacy.Properties[alias]; ok {
			shield.CACertificate = props.CACert
			shield.InsecureSkipVerify = props.InsecureSkipVerify
		}
		cfg.SHIELDs[alias] = shield
	}

	return cfg, cfg.Write()
}

func ReadConfig(path, legacy string) (*Config, error) {
	cfg := newConfig(path)

	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return convertLegacyConfig(path, legacy)
		}
		return nil, err
	}

	if err := yaml.Unmarshal(b, &cfg.SHIELDs); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Write() error {
	b, err := yaml.Marshal(c.SHIELDs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(c.Path, b, 0666)
}

func (c *Config) Select(alias string) error {
	if core, ok := c.SHIELDs[alias]; ok {
		c.Current = core
		return nil
	}
	return fmt.Errorf("Unknown SHIELD Core '%s'", alias)
}

func (c *Config) Add(alias string, core SHIELD) {
	c.SHIELDs[alias] = &core
}
