package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/voxelbrain/goptions"
	"github.com/starkandwayne/shield/agent"
	"golang.org/x/crypto/ssh"
)

type ShieldAgentOpts struct {
	AuthorizedKeysFile string `goptions:"-A, --authorized-keys, obligatory, description='Path to the authorized (public) keys file, for authenticating clients'"`
	HostKeyFile        string `goptions:"-k, --key, obligatory, description='Path to the server host key file'"`
	ListenAddress      string `goptions:"-l, --listen, obligatory, description='Network address and port to listen on'"`
}

func main() {
	fmt.Printf("starting up...\n")

	var opts ShieldAgentOpts
	if err := goptions.Parse(&opts); err != nil {
		fmt.Printf("%s\n", err)
		goptions.PrintHelp()
		return
	}

	authorizedKeys, err := loadAuthorizedKeys(opts.AuthorizedKeysFile)
	if err != nil {
		fmt.Printf("failed to load authorized keys: %s\n", err)
		return
	}

	config, err := configureSSHServer(opts.HostKeyFile, authorizedKeys)
	if err != nil {
		fmt.Printf("failed to configure SSH server: %s\n", err)
		return
	}

	listener, err := net.Listen("tcp", opts.ListenAddress)
	if err != nil {
		fmt.Printf("failed to bind %s: %s", opts.ListenAddress, err)
		return
	}

	fmt.Printf("listening on %s\n", opts.ListenAddress)

	agent := agent.NewAgent(config)
	agent.Serve(listener)
}

func loadAuthorizedKeys(path string) ([]ssh.PublicKey, error) {
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

func configureSSHServer(hostKeyPath string, authorizedKeys []ssh.PublicKey) (*ssh.ServerConfig, error) {
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
