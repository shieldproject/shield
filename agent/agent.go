package agent

import (
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
)

type Agent struct {
	PluginPaths []string

	config *ssh.ServerConfig

	Listen net.Listener
}

func NewAgent() *Agent {
	return &Agent{}
}

func (agent *Agent) ResolveBinary(name string) (string, error) {
	for _, path := range agent.PluginPaths {
		candidate := fmt.Sprintf("%s/%s", path, name)
		if stat, err := os.Stat(candidate); err == nil {
			// skip if not executable by someone
			if stat.Mode()&0111 == 0 {
				continue
			}

			// skip if not a regular file
			if stat.Mode()&os.ModeType != 0 {
				continue
			}

			return candidate, nil
		}
	}

	return "", fmt.Errorf("plugin %s not found in path", name)
}

func (agent *Agent) Run() {
	for {
		agent.ServeOne(agent.Listen, true)
	}
}

func (agent *Agent) ServeOne(l net.Listener, async bool) {
	c, err := l.Accept()
	if err != nil {
		fmt.Printf("failed to accept: %s\n", err)
		return
	}

	conn, chans, reqs, err := ssh.NewServerConn(c, agent.config)
	if err != nil {
		fmt.Printf("handshake failed: %s\n", err)
		return
	}

	if async {
		go agent.handleConn(conn, chans, reqs)
	} else {
		agent.handleConn(conn, chans, reqs)
	}
}

func (agent *Agent) handleConn(conn *ssh.ServerConn, chans <-chan ssh.NewChannel, reqs <-chan *ssh.Request) {
	defer conn.Close()

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			fmt.Printf("rejecting unknown channel type: %s\n", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			fmt.Printf("failed to accept channel: %s\n", err)
			return
		}

		defer channel.Close()

		for req := range requests {
			if req.Type != "exec" {
				fmt.Printf("rejecting non-exec channel request (type=%s)\n", req.Type)
				req.Reply(false, nil)
				continue
			}

			request, err := ParseRequest(req)
			if err != nil {
				fmt.Printf("%s\n", err)
				req.Reply(false, nil)
				continue
			}

			if err = request.ResolvePaths(agent); err != nil {
				fmt.Printf("%s\n", err)
				req.Reply(false, nil)
				continue
			}

			//fmt.Printf("got an agent-request [%s]\n", request.JSON)
			req.Reply(true, nil)

			// drain output to the SSH channel stream
			output := make(chan string)
			done := make(chan int)
			go func(out io.Writer, in chan string, done chan int) {
				for {
					s, ok := <-in
					if !ok {
						break
					}
					fmt.Fprintf(out, "%s", s)
				}
				close(done)
			}(channel, output, done)

			// run the agent request
			err = request.Run(output)
			<-done
			rc := []byte{0, 0, 0, 0}
			if err != nil {
				rc[0] = 1
			}
			channel.SendRequest("exit-status", false, rc)
			channel.Close()
		}
	}
}
