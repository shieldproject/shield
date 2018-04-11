package core

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type AuthConfig struct {
	Name       string `yaml:"name"`
	Identifier string `yaml:"identifier"`
	Backend    string `yaml:"backend"`

	Properties map[interface{}]interface{} `yaml:"properties"`
}

type FailsafeConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
type Config struct {
	SlowLoop int `yaml:"slow_loop"`
	FastLoop int `yaml:"fast_loop"`

	Debug bool `yaml:"debug"`

	DataDir string `yaml:"data_directory"`

	Addr          string `yaml:"listen_addr"`
	KeyFile       string `yaml:"private_key"`
	Workers       int    `yaml:"workers"`
	Purge         string `yaml:"purge_agent"`
	Timeout       int    `yaml:"max_timeout"`
	SkipSSLVerify bool   `yaml:"skip_ssl_verify"`
	WebRoot       string `yaml:"web_root"`

	Environment string `yaml:"env"`
	Color       string `yaml:"color"`
	MOTD        string `yaml:"motd"`

	EncryptionType string `yaml:"encryption_type"`

	VaultAddress string `yaml:"vault_address"`
	VaultCACert  string `yaml:"vault_ca_cert"`

	SessionTimeout int `yaml:"session_timeout"`

	Failsafe FailsafeConfig `yaml:"failsafe"`

	Auth []AuthConfig `yaml:"auth"`
}

func ReadConfig(file string) (Config, error) {
	config := Config{
		FastLoop: 1,
		SlowLoop: 60 * 5,

		DataDir:        "/var/vcap/store/shield",
		Addr:           "*:8888",
		KeyFile:        "worker.key",
		Workers:        2,
		Purge:          "localhost:5444",
		Timeout:        12,
		WebRoot:        "web",
		EncryptionType: "aes256-ctr",
		VaultAddress:   "http://127.0.0.1:8200",
		SessionTimeout: 720,
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
		return config, fmt.Errorf("fast_loop value '%d' is invalid (must be greater than zero)", config.FastLoop)
	}
	if config.SlowLoop <= 0 {
		return config, fmt.Errorf("slow_loop value '%d' is invalid (must be greater than zero)", config.SlowLoop)
	}
	if config.Timeout <= 0 {
		return config, fmt.Errorf("timeout value '%d' is invalid (must be greater than zero)", config.Timeout)
	}
	if config.Workers <= 0 {
		return config, fmt.Errorf("number of workers '%d' is invalid (must be greater than zero)", config.Workers)
	}
	if config.SessionTimeout <= 0 {
		return config, fmt.Errorf("session timeout of '%d' hours is invalid (must be greater than zero)", config.SessionTimeout)
	}
	if config.EncryptionType == "" {
		return config, fmt.Errorf("encryption type '%s' is invalid (see documentation for supported ciphers and modes)", config.EncryptionType)
	}
	if config.DataDir == "" {
		return config, fmt.Errorf("SHIELD data directory '%s' is invalid (must be a valid path)", config.DataDir)
	}
	// FIXME: check existence of WebRoot
	for i, auth := range config.Auth {
		if auth.Name == "local" {
			return config, fmt.Errorf("auth backend configuration #%d is named 'local', which is reserved for internal use by SHIELD itself;please rename this auth backend", i+1)
		}
	}

	return config, nil
}
