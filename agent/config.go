package agent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"

	"github.com/jhunt/go-log"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Name               string   `yaml:"name"`
	AuthorizedKeysFile string   `yaml:"authorized_keys_file"`
	HostKeyFile        string   `yaml:"host_key_file"`
	ListenAddress      string   `yaml:"listen_address"`
	PluginPaths        []string `yaml:"plugin_paths"`
	Registration       struct {
		URL          string `yaml:"url"`
		Interval     int    `yaml:"interval"`
		ShieldCACert string `yaml:"shield_ca_cert"`
		SkipVerify   bool   `yaml:"skip_verify"`
	} `yaml:"registration"`
}

func (agent *Agent) ReadConfig(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var config Config
	err = yaml.Unmarshal(b, &config)

	if err != nil {
		return err
	}

	if config.Name == "" {
		return fmt.Errorf("No agent name specified.")
	}
	if config.AuthorizedKeysFile == "" {
		return fmt.Errorf("No authorized keys file supplied.")
	}
	if config.HostKeyFile == "" {
		return fmt.Errorf("No host key file supplied.")
	}
	if config.ListenAddress == "" {
		return fmt.Errorf("No listen address and/or port supplied.")
	}
	if len(config.PluginPaths) == 0 {
		return fmt.Errorf("No plugin path supplied.")
	}

	authorizedKeys, err := LoadAuthorizedKeys(config.AuthorizedKeysFile)
	if err != nil {
		log.Errorf("failed to load authorized keys: %s\n", err)
		return err
	}

	server, err := ConfigureSSHServer(config.HostKeyFile, authorizedKeys)
	if err != nil {
		log.Errorf("failed to configure SSH server: %s", err)
		return err
	}
	agent.config = server

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

	listener, err := net.Listen("tcp4", config.ListenAddress)
	if err != nil {
		log.Errorf("failed to bind %s: %s", config.ListenAddress, err)
		return err
	}
	agent.Listen = listener

	agent.PluginPaths = config.PluginPaths

	agent.Registration.URL = config.Registration.URL
	agent.Registration.Interval = config.Registration.Interval
	agent.Registration.SkipVerify = config.Registration.SkipVerify

	if config.Registration.ShieldCACert != "" {
		b, err := ioutil.ReadFile(config.Registration.ShieldCACert)
		if err != nil {
			log.Errorf("failed to configure shield-agent: failed to read CA-Cert with err '%s' ", err)
			return err
		}
		agent.Registration.ShieldCACert = string(b)
	}

	return nil
}

func LoadAuthorizedKeys(path string) ([]ssh.PublicKey, error) {
	authorizedKeysBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var authorizedKeys []ssh.PublicKey

	for {
		key, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			break
		}

		authorizedKeys = append(authorizedKeys, key)

		authorizedKeysBytes = rest
	}

	return authorizedKeys, nil
}

func ConfigureSSHServer(hostKeyPath string, authorizedKeys []ssh.PublicKey) (*ssh.ServerConfig, error) {
	certChecker := &ssh.CertChecker{
		IsAuthority: func(key ssh.PublicKey) bool {
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

	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return certChecker.Authenticate(conn, key)
		},
	}

	privateBytes, err := ioutil.ReadFile(hostKeyPath)
	if err != nil {
		return nil, err
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, err
	}

	config.AddHostKey(private)

	return config, nil
}

func ConfigureSSHClient(privateKeyPath string) (*ssh.ClientConfig, error) {
	raw, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
	}, nil
}
