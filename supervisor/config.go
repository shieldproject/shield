package supervisor

import (
	"github.com/starkandwayne/shield/db"

	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DatabaseType string `yaml:"database_type"`
	DatabaseDSN  string `yaml:"database_dsn"`

	Port string `yaml:"port"`

	PrivateKeyFile string `yaml:"private_key"`

	Workers uint `yaml:"workers"`

	PurgeAgent string `yaml:"purge_agent"`
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

	if s.Database == nil {
		s.Database = &db.DB{}
	}

	s.Database.Driver = config.DatabaseType
	s.Database.DSN = config.DatabaseDSN
	s.Port = config.Port
	s.PrivateKeyFile = config.PrivateKeyFile
	s.Workers = config.Workers
	s.PurgeAgent = config.PurgeAgent
	return nil
}
