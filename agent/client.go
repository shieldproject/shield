package agent

import (
	"bufio"
	"io"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	config  *ssh.ClientConfig
	session *ssh.Session
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
		return err
	}

	c.session = session
	return nil
}

func (c *Client) Close() error {
	if c.session == nil {
		return nil
	}
	return c.session.Close()
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
