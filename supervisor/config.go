package supervisor

import (
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/starkandwayne/shield/db"
)

type Config struct {
	DatabaseType string `yaml:"database_type"`
	DatabaseDSN  string `yaml:"database_dsn"`

	Port string `yaml:"port"`

	PrivateKeyFile string `yaml:"private_key"`

	Workers uint `yaml:"workers"`

	PurgeAgent string `yaml:"purge_agent"`

	MaxTimeout uint `yaml:"max_timeout"`
}

func (s *Supervisor) ReadConfig(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var config Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return err
	}

	if config.Port == "" {
		config.Port = "8888"
	}
	if config.PrivateKeyFile == "" {
		config.PrivateKeyFile = "/etc/shield/ssh/server.key"
	}
	if config.Workers == 0 {
		config.Workers = 5
	}

	if config.PurgeAgent == "" {
		config.PurgeAgent = "localhost:5444"
	}

	if config.MaxTimeout == 0 {
		config.MaxTimeout = 12
	}

	if s.Database == nil {
		s.Database = &db.DB{}
	}

	s.Database.Driver = config.DatabaseType
	s.Database.DSN = config.DatabaseDSN
	s.Port = config.Port
	s.PrivateKeyFile = config.PrivateKeyFile
	s.Workers = config.Workers
	s.PurgeAgent = config.PurgeAgent
	s.Timeout = time.Duration(config.MaxTimeout) * time.Hour
	return nil
}
