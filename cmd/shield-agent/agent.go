package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"io"

	"golang.org/x/crypto/ssh"
)

type Agent struct {
	config *ssh.ServerConfig
}

func (agent *Agent) Serve(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Printf("failed to accept: %s\n", err)
			return
		}

		conn, chans, reqs, err := ssh.NewServerConn(c, agent.config)
		if err != nil {
			fmt.Printf("handshake failed: %s\n", err)
			continue
		}

		go agent.handleConn(conn, chans, reqs)
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
			fmt.Printf("channel request: %s\n", req.Type)

			if req.Type != "exec" {
				fmt.Printf("rejecting\n")
				req.Reply(false, nil)
				continue
			}

			request, err := ParseAgentRequest(req)
			if err != nil {
				fmt.Printf("%s\n", err)
				req.Reply(false, nil)
				continue
			}

			fmt.Printf("got an agent-request [%s]\n", request.JSON)
			req.Reply(true, nil)

			// drain output to the SSH channel stream
			output := make(chan string)
			done   := make(chan int)
			go func(out io.Writer, in chan string, done chan int) {
				for {
					s, ok := <-in
					if !ok {
						break
					}
					fmt.Printf("sent: %s", s)
					fmt.Fprintf(out, "%s", s)
				}
				close(done)
			}(channel, output, done)

			// run the agent request
			err = request.Run(output)
			<-done
			if err != nil {
				fmt.Printf("failed: %s\n", err)
				fmt.Fprintf(channel, "%s\n", err)
			}
			channel.SendRequest("exit-status", false, []byte{ 0, 0, 0, 0 })
			channel.Close()
			fmt.Printf("closed channel\n")
		}
		fmt.Printf("out of requests\n")
	}
}

type AgentRequest struct {
	JSON           string
	Operation      string `json:"operation"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	RestoreKey     string `json:"restore_key"`
}

func ParseAgentRequest(req *ssh.Request) (*AgentRequest, error) {
	var raw struct {
		Value []byte
	}
	err := ssh.Unmarshal(req.Payload, &raw)
	if err != nil {
		return nil, err
	}

	request := &AgentRequest{JSON: string(raw.Value)}
	err = json.Unmarshal(raw.Value, &request)
	if err != nil {
		return nil, fmt.Errorf("malformed agent-request %v: %s\n", req.Payload, err)
	}

	if request.Operation == "" {
		return nil, fmt.Errorf("missing required 'operation' value in payload")
	}
	if request.Operation != "backup" && request.Operation != "restore" {
		return nil, fmt.Errorf("unsupported operation: '%s'", request.Operation)
	}
	if request.TargetPlugin == "" {
		return nil, fmt.Errorf("missing required 'target_plugin' value in payload")
	}
	if request.TargetEndpoint == "" {
		return nil, fmt.Errorf("missing required 'target_endpoint' value in payload")
	}
	if request.StorePlugin == "" {
		return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
	}
	if request.StoreEndpoint == "" {
		return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
	}
	if request.Operation == "restore" && request.RestoreKey == "" {
		return nil, fmt.Errorf("missing required 'restore_key' value in payload (for restore operation)")
	}
	return request, nil
}

func (req *AgentRequest) Run(output chan string) error {
	cmd := exec.Command("shield-pipe")
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("USER=%s", os.Getenv("USER")),
		fmt.Sprintf("LANG=%s", os.Getenv("LANG")),

		fmt.Sprintf("SHIELD_OP=%s", req.Operation),
		fmt.Sprintf("SHIELD_STORE_PLUGIN=%s", req.StorePlugin),
		fmt.Sprintf("SHIELD_STORE_ENDPOINT=%s", req.StoreEndpoint),
		fmt.Sprintf("SHIELD_TARGET_PLUGIN=%s", req.TargetPlugin),
		fmt.Sprintf("SHIELD_TARGET_ENDPOINT=%s", req.TargetEndpoint),
		fmt.Sprintf("SHIELD_RESTORE_KEY=%s", req.RestoreKey),
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	drain := func(prefix string, out chan string, in io.Reader) {
		defer wg.Done()
		s := bufio.NewScanner(in)
		for s.Scan() {
			out <- fmt.Sprintf("%s:%s\n", prefix, s.Text())
		}
	}

	wg.Add(2)
	go drain("E", output, stderr)
	go drain("O", output, stdout)

	err = cmd.Start()
	if err != nil {
		return err
	}

	wg.Wait()
	close(output)

	err = cmd.Wait()
	if err != nil {
		return err
	}

	// FIXME: need to pass the command exit status back to the SSH client
	//        via `exit-status` SendRequest() [i think]

	return nil
}
