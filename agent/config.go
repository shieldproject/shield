package agent

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	env "github.com/jhunt/go-envirotron"
	"github.com/jhunt/go-log"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Name               string   `yaml:"name"                 env:"SHIELD_AGENT_NAME"`
	AuthorizedKeysFile string   `yaml:"authorized_keys_file" env:"SHIELD_AGENT_AUTHORIZED_KEYS_FILE"`
	AuthorizedKey      string   `yaml:"authorized_key"       env:"SHIELD_AGENT_AUTHORIZED_KEY"`
	HostKeyFile        string   `yaml:"host_key_file"        env:"SHIELD_AGENT_HOST_KEY_FILE"`
	HostKey            string   `yaml:"host_key"             env:"SHIELD_AGENT_HOST_KEY"`
	MACs               []string `yaml:"macs"`
	ListenAddress      string   `yaml:"listen_address"       env:"SHIELD_AGENT_LISTEN_ADDRESS"`
	PluginPaths        []string `yaml:"plugin_paths"`
	PluginPathsEnv     string   `yaml:"-"                    env:"SHIELD_AGENT_PLUGIN_PATHS"`
	Registration       struct {
		Token        string `yaml:"token"           env:"SHIELD_AGENT_REGISTRATION_TOKEN"`
		URL          string `yaml:"url"             env:"SHIELD_AGENT_REGISTRATION_URL"`
		Interval     int    `yaml:"interval"        env:"SHIELD_AGENT_REGISTRATION_INTERVAL"`
		ShieldCACert string `yaml:"shield_ca_cert"  env:"SHIELD_AGENT_REGISTRATION_SHIELD_CA_CERT"`
		SkipVerify   bool   `yaml:"skip_verify"     env:"SHIELD_AGENT_REGISTRATION_SKIP_VERIFY"`
	} `yaml:"registration"`
}

func (agent *Agent) ReadConfig(path string) error {
	var err error
	var config Config

	if path != "" {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if err := yaml.Unmarshal(b, &config); err != nil {
			return err
		}
	}

	env.Override(&config)

	if config.Name == "" {
		return fmt.Errorf("No agent name specified.")
	}
	if config.ListenAddress == "" {
		return fmt.Errorf("No listen address and/or port supplied.")
	}

	if config.AuthorizedKeysFile == "" && config.AuthorizedKey == "" {
		return fmt.Errorf("No authorized keys supplied.")
	}

	if config.PluginPathsEnv != "" {
		p := strings.Split(config.PluginPathsEnv, ":")
		for _, path := range config.PluginPaths {
			p = append(p, path)
		}
		config.PluginPaths = p
	}
	if len(config.PluginPaths) == 0 {
		return fmt.Errorf("No plugin path supplied.")
	}

	var authorizedKeys []ssh.PublicKey
	if config.AuthorizedKey != "" {
		authorizedKeys, err = LoadAuthorizedKeysFromBytes([]byte(config.AuthorizedKey))
	} else {
		authorizedKeys, err = LoadAuthorizedKeysFromFile(config.AuthorizedKeysFile)
	}
	if err != nil {
		log.Errorf("failed to load authorized keys: %s\n", err)
		return err
	}

	var hostKey ssh.Signer
	if config.HostKey != "" {
		hostKey, err = LoadPrivateKeyFromBytes([]byte(config.HostKey))
	} else if config.HostKeyFile != "" {
		hostKey, err = LoadPrivateKeyFromFile(config.HostKeyFile)
	} else {
		hostKey, err = GeneratePrivateKey()
	}
	if err != nil {
		log.Errorf("failed to load host key: %s\n", err)
		return err
	}

	agent.config, err = ConfigureSSHServer(hostKey, authorizedKeys, config.MACs)
	if err != nil {
		log.Errorf("failed to configure SSH server: %s", err)
		return err
	}

	agent.Name = config.Name
	l := strings.Split(config.ListenAddress, ":")
	if len(l) == 1 {
		config.ListenAddress = config.ListenAddress + ":5444"
		agent.Port = 5444

	} else if len(l) != 2 {
		log.Errorf("failed to configure shield-agent: '%s' does not look like a valid address to bind", config.ListenAddress)
		return fmt.Errorf("invalid bind address '%s'", config.ListenAddress)

	} else {
		n, err := strconv.ParseInt(l[1], 10, 0)
		if err != nil {
			log.Errorf("failed to configure shield-agent: '%s' does not look like a valid address to bind: %s", config.ListenAddress, err)
			return err
		}
		agent.Port = int(n)
	}

	agent.Listen, err = net.Listen("tcp4", config.ListenAddress)
	if err != nil {
		log.Errorf("failed to bind %s: %s", config.ListenAddress, err)
		return err
	}

	agent.PluginPaths = config.PluginPaths

	agent.Registration.URL = config.Registration.URL
	agent.Registration.Token = config.Registration.Token
	agent.Registration.Interval = config.Registration.Interval
	agent.Registration.SkipVerify = config.Registration.SkipVerify

	if config.Registration.ShieldCACert != "" {
		if !strings.HasPrefix(config.Registration.ShieldCACert, "---") {
			b, err := ioutil.ReadFile(config.Registration.ShieldCACert)
			if err != nil {
				log.Errorf("failed to configure shield-agent: failed to read CA-Cert with err '%s' ", err)
				return err
			}
			config.Registration.ShieldCACert = string(b)
		}
		agent.Registration.ShieldCACert = config.Registration.ShieldCACert
	}

	return nil
}

func LoadAuthorizedKeysFromFile(path string) ([]ssh.PublicKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadAuthorizedKeysFromBytes(b)
}

func LoadAuthorizedKeysFromBytes(b []byte) ([]ssh.PublicKey, error) {
	var keys []ssh.PublicKey

	for {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(b)
		if err != nil {
			break
		}

		keys = append(keys, key)
		b = rest
	}

	return keys, nil
}

func GeneratePrivateKey() (ssh.Signer, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(k)
}

func LoadPrivateKeyFromFile(path string) (ssh.Signer, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return LoadPrivateKeyFromBytes(b)
}

func LoadPrivateKeyFromBytes(b []byte) (ssh.Signer, error) {
	return ssh.ParsePrivateKey(b)
}

func ConfigureSSHServer(key ssh.Signer, authorizedKeys []ssh.PublicKey, macs []string) (*ssh.ServerConfig, error) {
	certChecker := &ssh.CertChecker{
		IsUserAuthority: func(key ssh.PublicKey) bool {
			return false
		},

		UserKeyFallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			for _, k := range authorizedKeys {
				if bytes.Equal(k.Marshal(), key.Marshal()) {
					return nil, nil
				}
			}

			return nil, fmt.Errorf("unknown public key")
		},
	}

	if len(macs) == 0 {
		macs = []string{"hmac-sha2-256-etm@openssh.com", "hmac-sha2-256", "hmac-sha1"}
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return certChecker.Authenticate(conn, key)
		},
		Config: ssh.Config{MACs: macs},
	}

	config.AddHostKey(key)

	return config, nil
}
