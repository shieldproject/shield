package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/starkandwayne/goutils/log"
	"io"
	"log/syslog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

type Request struct {
	JSON           string
	Operation      string `json:"operation"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	RestoreKey     string `json:"restore_key"`
}

func ParseRequestValue(value []byte) (*Request, error) {
	request := &Request{JSON: string(value)}
	err := json.Unmarshal(value, &request)
	if err != nil {
		return nil, fmt.Errorf("malformed agent-request %s: %s\n", value, err)
	}

	if request.Operation == "" {
		return nil, fmt.Errorf("missing required 'operation' value in payload")
	}
	if request.Operation != "backup" && request.Operation != "restore" && request.Operation != "purge" {
		return nil, fmt.Errorf("unsupported operation: '%s'", request.Operation)
	}
	if request.Operation != "purge" {
		if request.TargetPlugin == "" {
			return nil, fmt.Errorf("missing required 'target_plugin' value in payload")
		}
		if request.TargetEndpoint == "" {
			return nil, fmt.Errorf("missing required 'target_endpoint' value in payload")
		}
	}
	if request.StorePlugin == "" {
		return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
	}
	if request.StoreEndpoint == "" {
		return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
	}
	if (request.Operation == "restore" || request.Operation == "purge") && request.RestoreKey == "" {
		return nil, fmt.Errorf("missing required 'restore_key' value in payload (for restore operation)")
	}
	return request, nil
}

func ParseRequest(req *ssh.Request) (*Request, error) {
	var raw struct {
		Value []byte
	}
	err := ssh.Unmarshal(req.Payload, &raw)
	if err != nil {
		return nil, err
	}

	return ParseRequestValue(raw.Value)
}

func (req *Request) Run(output chan string) error {
	cmd := exec.Command("shield-pipe")

	log.Infof("Executing %s request using target %s and store %s via shield-pipe", req.Operation, req.TargetPlugin, req.StorePlugin)
	log.Debugf("Target Endpoint config: %s", req.TargetEndpoint)
	log.Debugf("Store Endpoint config: %s", req.StoreEndpoint)

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

	if log.LogLevel() == syslog.LOG_DEBUG {
		cmd.Env = append(cmd.Env, "DEBUG=true")
	}

	log.Debugf("ENV: %s", strings.Join(cmd.Env, ","))

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
		close(output)
		return err
	}

	wg.Wait()
	close(output)

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (r *Request) ResolvePaths(agent *Agent) error {
	if r.Operation != "purge" {
		bin, err := agent.ResolveBinary(r.TargetPlugin)
		if err != nil {
			return err
		}
		r.TargetPlugin = bin
	}

	bin, err := agent.ResolveBinary(r.StorePlugin)
	if err != nil {
		return err
	}
	r.StorePlugin = bin

	return nil
}
