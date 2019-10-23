package agent

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

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
		Interval     int
		ShieldCACert string
		SkipVerify   bool
	}
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

			if err = agent.ResolvePathsIn(command); err != nil {
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
				for {
					s, ok := <-in
					if !ok {
						break
					}
					fmt.Fprintf(out, "%s", s)
					log.Debugf("%s", strings.Trim(s, "\n"))
				}
				close(done)
			}(channel, output, done)

			err = agent.Execute(command, output)
			<-done
			var rc int
			if exitErr, ok := err.(*exec.ExitError); ok {
				sys := exitErr.ProcessState.Sys()
				// os.ProcessState.Sys() may not return syscall.WaitStatus on non-UNIX machines,
				// so currently this feature only works on UNIX, but shouldn't crash on other OSes
				if ws, ok := sys.(syscall.WaitStatus); ok {
					if ws.Exited() {
						rc = ws.ExitStatus()
					} else {
						var signal syscall.Signal
						if ws.Signaled() {
							signal = ws.Signal()
						}
						if ws.Stopped() {
							signal = ws.StopSignal()
						}
						sigStr, ok := SIGSTRING[signal]
						if !ok {
							sigStr = "ABRT" // use ABRT as catch-all signal for any that don't translate
							log.Infof("Task execution terminted due to %s, translating as ABRT for ssh transport", signal)
						} else {
							log.Infof("Task execution terminated due to SIG%s", sigStr)
						}
						sigMsg := struct {
							Signal     string
							CoreDumped bool
							Error      string
							Lang       string
						}{
							Signal:     sigStr,
							CoreDumped: false,
							Error:      fmt.Sprintf("shield-pipe terminated due to SIG%s", sigStr),
							Lang:       "en-US",
						}
						channel.SendRequest("exit-signal", false, ssh.Marshal(&sigMsg))
						channel.Close()
						continue
					}
				}
			} else if err != nil {
				// we got some kind of error that isn't a command execution error,
				// from a UNIX system, use an magical error code to signal this to
				// the shield daemon - 16777216
				log.Infof("Task could not execute: %s", err)
				rc = 16777216
			}

			log.Infof("Task completed with rc=%d", rc)
			channel.SendRequest("exit-status", false, encodeExitCode(rc))
			channel.Close()
		}
	}
}

// Based on what's handled in https://github.com/golang/crypto/blob/master/ssh/session.go#L21
var SIGSTRING = map[syscall.Signal]string{
	syscall.SIGABRT: "ABRT",
	syscall.SIGALRM: "ALRM",
	syscall.SIGFPE:  "FPE",
	syscall.SIGHUP:  "HUP",
	syscall.SIGILL:  "ILL",
	syscall.SIGINT:  "INT",
	syscall.SIGKILL: "KILL",
	syscall.SIGPIPE: "PIPE",
	syscall.SIGQUIT: "QUIT",
	syscall.SIGSEGV: "SEGV",
	syscall.SIGTERM: "TERM",
	syscall.SIGUSR1: "USR1",
	syscall.SIGUSR2: "USR2",
}

func encodeExitCode(rc int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(rc))
	return b
}
