package supervisor

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
)

type Config struct {
	DatabaseType string `yaml:"database_type"`
	DatabaseDSN  string `yaml:"database_dsn"`

	Port string `yaml:"port"`

	PrivateKeyFile string `yaml:"private_key"`
	WebRoot        string `yaml:"web_root"`

	Workers uint `yaml:"workers"`

	PurgeAgent string `yaml:"purge_agent"`

	MaxTimeout uint `yaml:"max_timeout"`

	Auth AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Tokens map[string]string `yaml:"api_tokens"`
	Basic  BasicAuthConfig   `yaml:"basic"`
	OAuth  OAuthConfig       `yaml:"oauth"`
}

type BasicAuthConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type OAuthConfig struct {
	Provider      string         `yaml:"provider"`
	Key           string         `yaml:"key"`
	Secret        string         `yaml:"secret"`
	BaseURL       string         `yaml:"base_url"`
	Sessions      SessionsConfig `yaml:"sessions"`
	SigningKey    string         `yaml:"signing_key"`
	JWTPrivateKey *rsa.PrivateKey
	JWTPublicKey  *rsa.PublicKey
}
type SessionsConfig struct {
	Type   string `yaml:"type"`
	DSN    string `yaml:"dsn"`
	MaxAge int    `yaml:"max_age"`
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
	if config.WebRoot == "" {
		config.WebRoot = "/usr/share/shield/webui"
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

	if config.Auth.Basic.User == "" {
		config.Auth.Basic.User = "admin"
	}

	if config.Auth.Basic.Password == "" {
		config.Auth.Basic.Password = "admin"
	}

	if config.Auth.OAuth.Sessions.MaxAge == 0 {
		config.Auth.OAuth.Sessions.MaxAge = 86400 * 30
	}

	if config.Auth.OAuth.Provider != "" {
		if config.Auth.OAuth.BaseURL == "" {
			return fmt.Errorf("OAuth requested, but no external URL provided. Cannot proceed.")
		}
		if config.Auth.OAuth.SigningKey == "" {
			log.Debugf("No signing key specified, generating a random one")
			privKey, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				return err
			}
			config.Auth.OAuth.JWTPrivateKey = privKey
		} else {
			bytes, err := ioutil.ReadFile(config.Auth.OAuth.SigningKey)
			if err != nil {
				return err
			}
			privKey, err := jwt.ParseRSAPrivateKeyFromPEM(bytes)
			if err != nil {
				return err
			}
			config.Auth.OAuth.JWTPrivateKey = privKey

		}
		config.Auth.OAuth.JWTPublicKey = &config.Auth.OAuth.JWTPrivateKey.PublicKey
	}

	if s.Database == nil {
		s.Database = &db.DB{}
	}

	s.Database.Driver = config.DatabaseType
	s.Database.DSN = config.DatabaseDSN
	s.PrivateKeyFile = config.PrivateKeyFile
	s.Workers = config.Workers
	s.PurgeAgent = config.PurgeAgent
	s.Timeout = time.Duration(config.MaxTimeout) * time.Hour

	ws := WebServer{
		Database:   s.Database.Copy(),
		Port:       config.Port,
		WebRoot:    config.WebRoot,
		Auth:       config.Auth,
		Supervisor: s,
	}
	s.Web = &ws
	return nil
}
