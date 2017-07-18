package core

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type AuthConfig struct {
	Name       string                 `yaml:"name"`
	Identifier string                 `yaml:"identifier"`
	Backend    string                 `yaml:"backend"`
	Properties map[string]interface{} `yaml:"properties"`
}

type Config struct {
	SlowLoop int `yaml:"slow_loop"`
	FastLoop int `yaml:"fast_loop"`

	DBType string `yaml:"database_type"`
	DBPath string `yaml:"database_dsn"`

	Addr          string `yaml:"listen_addr"`
	KeyFile       string `yaml:"private_key"`
	Workers       int    `yaml:"workers"`
	Purge         string `yaml:"purge_agent"`
	Timeout       int    `yaml:"max_timeout"`
	SkipSSLVerify bool   `yaml:"skip_ssl_verify"`
	WebRoot       string `yaml:"web_root"`
	MOTD          string `yaml:"motd"`

	Auth []AuthConfig `yaml:"auth"`
}

func ReadConfig(file string) (Config, error) {
	config := Config{
		FastLoop: 1,
		SlowLoop: 60 * 5,

		DBPath:  "shield.db",
		Addr:    "*:8888",
		KeyFile: "worker.key",
		Workers: 2,
		Purge:   "localhost:5444",
		Timeout: 12,
		WebRoot: "web",
	}

	/* optionally read configuration from a file */
	if file != "" {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return config, err
		}

		if err = yaml.Unmarshal(b, &config); err != nil {
			return config, err
		}
	}

	/* validate configuration */
	if config.FastLoop <= 0 {
		return config, fmt.Errorf("fast_loop value '%d' is invalid (must be greater than zero)")
	}
	if config.SlowLoop <= 0 {
		return config, fmt.Errorf("slow_loop value '%d' is invalid (must be greater than zero)")
	}
	if config.Timeout <= 0 {
		return config, fmt.Errorf("timeout value '%d' is invalid (must be greater than zero)")
	}
	if config.Workers <= 0 {
		return config, fmt.Errorf("number of workers '%d' is invalid (must be greater than zero)")
	}
	// FIXME: check existence of WebRoot

	for i, auth := range config.Auth {
		if auth.Name == "local" {
			return config, fmt.Errorf("auth backend configuration #%d is named 'local', which is reserved for internal use by SHIELD itself;please rename this auth backend", i+1)
		}
	}

	return config, nil
}
