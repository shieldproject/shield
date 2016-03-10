package agent

import (
	"bufio"
	"io"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	config  *ssh.ClientConfig
	session *ssh.Session
	conn    *ssh.Client
}

func NewClient(config *ssh.ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

func (c *Client) Dial(endpoint string) error {
	conn, err := ssh.Dial("tcp4", endpoint, c.config)
	if err != nil {
		return err
	}

	session, err := conn.NewSession()
	if err != nil {
		conn.Close()
		return err
	}

	c.conn = conn
	c.session = session
	return nil
}

func (c *Client) Close() error {
	var sessErr, connErr error
	if c.conn != nil {
		if c.session != nil {
			sessErr = c.session.Close()
		}
		connErr = c.conn.Conn.Close()
	}
	if connErr != nil {
		return connErr
	}
	if sessErr != nil {
		return sessErr
	}
	return nil
}

func (c *Client) Run(out chan string, command string) error {
	rd, err := c.session.StdoutPipe()
	if err != nil {
		return err
	}

	go func(out chan string, in io.Reader) {
		b := bufio.NewScanner(in)
		for b.Scan() {
			out <- b.Text()
		}
		close(out)
	}(out, rd)

	err = c.session.Start(command)
	if err != nil {
		return err
	}

	err = c.session.Wait()
	if err != nil {
		return err
	}

	c.Close()
	return nil
}
