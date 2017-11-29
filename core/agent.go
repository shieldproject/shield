package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"sync"

	"golang.org/x/crypto/ssh"
)

type AgentCommand struct {
	Op             string `json:"operation"`
	TargetPlugin   string `json:"target_plugin,omitempty"`
	TargetEndpoint string `json:"target_endpoint,omitempty"`
	StorePlugin    string `json:"store_plugin,omitempty"`
	StoreEndpoint  string `json:"store_endpoint,omitempty"`
	RestoreKey     string `json:"restore_key,omitempty"`
	EncryptType    string `json:"encrypt_type,omitempty"`
	EncryptKey     string `json:"encrypt_key,omitempty"`
	EncryptIV      string `json:"encrypt_iv,omitempty"`
}

type AgentClient struct {
	config *ssh.ClientConfig
	key    ssh.Signer
}

func NewAgentClient(keyfile string) (*AgentClient, error) {
	raw, err := ioutil.ReadFile(keyfile)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, err
	}

	return &AgentClient{
		key: signer,
		config: &ssh.ClientConfig{
			Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		},
	}, nil
}

// FIXME: add a stderr here and move O:/E: out of core/core.go
func (c *AgentClient) Run(host string, stdout, stderr chan string, command *AgentCommand) error {
	raw, err := json.Marshal(command)
	if err != nil {
		return err
	}

	conn, err := ssh.Dial("tcp4", host, c.config)
	if err != nil {
		return err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	rd, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func(stdout, stderr chan string, in io.Reader) {
		defer wg.Done()
		var buf bytes.Buffer
		b := bufio.NewScanner(in)
		for b.Scan() {
			s := b.Text() + "\n"
			switch s[:2] {
			case "O:":
				buf.WriteString(s[2:])
			case "E:":
				stderr <- s[2:]
			}
		}
		close(stderr)
		stdout <- buf.String()
		close(stdout)
	}(stdout, stderr, rd)

	err = session.Run(string(raw))
	wg.Wait()
	return err
}
