package agent

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/jhunt/go-log"
	"golang.org/x/crypto/ssh"
)

type Agent struct {
	PluginPaths []string

	config *ssh.ServerConfig

	Listen net.Listener

	Name    string
	Version string
	Port    int

	Registration struct {
		URL          string
		Token        string
		Interval     int
		ShieldCACert string
		SkipVerify   bool
	}
}

func NewAgent() *Agent {
	return &Agent{}
}

func (agent *Agent) Run() {
	go agent.Ping()

	for {
		agent.ServeOne(agent.Listen, true)
	}
}

func (agent *Agent) ServeOne(l net.Listener, async bool) {
	c, err := l.Accept()
	if err != nil {
		log.Errorf("failed to accept: %s\n", err)
		return
	}

	conn, chans, reqs, err := ssh.NewServerConn(c, agent.config)
	if err != nil {
		log.Errorf("handshake failed: %s\n", err)
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
			log.Errorf("rejecting unknown channel type: %s\n", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Errorf("failed to accept channel: %s\n", err)
			return
		}

		for req := range requests {
			if req.Type != "exec" {
				log.Errorf("rejecting non-exec channel request (type=%s)\n", req.Type)
				req.Reply(false, nil)
				continue
			}

			req.Reply(true, nil)

			command, err := ParseCommandFromSSHRequest(req)
			if err != nil {
				log.Errorf("%s\n", err)
				fmt.Fprintf(channel, "E:failed: %s\n", err)
				channel.SendRequest("exit-status", false, encodeExitCode(1))
				channel.Close()
				continue
			}

			// drain output to the SSH channel stream
			output := make(chan string)
			done := make(chan int)
			go func(out io.Writer, in chan string, done chan int) {
				for s := range in {
					fmt.Fprintf(out, "%s", s)
					log.Debugf("%s", strings.Trim(s, "\n"))
				}
				close(done)
			}(channel, output, done)

			err = agent.Execute(command, output)
			rc := 0
			if err != nil {
				log.Debugf("task failed: %s", err)
				fmt.Fprintf(channel, "E: task failed: %s\n", err)
				rc = 1
			}
			log.Infof("Task completed with rc=%d", rc)
			<-done
			log.Debugf("sending exit-status(%d) to upstream SSH peer", rc)
			channel.SendRequest("exit-status", false, encodeExitCode(rc))
			channel.Close()
		}
	}
}

func encodeExitCode(rc int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(rc))
	return b
}
