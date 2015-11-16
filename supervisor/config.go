package supervisor

import (
	"github.com/starkandwayne/shield/db"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	DatabaseType string `yaml:"database_type"`
	DatabaseDSN  string `yaml:"database_dsn"`

	Listen string `yaml:"listen"`

	PrivateKeyFile string `yaml:"private_key"`
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

	if config.Listen == "" {
		config.Listen = ":8888"
	}
	if config.PrivateKeyFile == "" {
		config.PrivateKeyFile = "/etc/shield/ssh/server.key"
	}

	if s.Database == nil {
		s.Database = &db.DB{}
	}

	s.Database.Driver = config.DatabaseType
	s.Database.DSN = config.DatabaseDSN
	s.Listen = config.Listen
	s.PrivateKeyFile = config.PrivateKeyFile
	return nil
}
