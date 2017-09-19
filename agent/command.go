package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/syslog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/starkandwayne/goutils/log"

	"golang.org/x/crypto/ssh"
)

type Command struct {
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

func ParseCommand(b []byte) (*Command, error) {
	cmd := &Command{}
	if err := json.Unmarshal(b, &cmd); err != nil {
		return nil, fmt.Errorf("malformed agent command: %s\n", err)
	}

	switch cmd.Op {
	case "":
		return nil, fmt.Errorf("missing required 'operation' value in payload")
	case "backup":
		if cmd.TargetPlugin == "" {
			return nil, fmt.Errorf("missing required 'target_plugin' value in payload")
		}
		if cmd.TargetEndpoint == "" {
			return nil, fmt.Errorf("missing required 'target_endpoint' value in payload")
		}

		if cmd.StorePlugin == "" {
			return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
		}
		if cmd.StoreEndpoint == "" {
			return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
		}
		if cmd.EncryptType == "" {
			return nil, fmt.Errorf("missing required 'encrypt_cipher' value in payload")
		}
		if cmd.EncryptKey == "" {
			return nil, fmt.Errorf("missing required 'encrypt_key' value in payload")
		}
		if cmd.EncryptIV == "" {
			return nil, fmt.Errorf("missing required 'encrypt_iv' value in payload")
		}

	case "restore":
		if cmd.TargetPlugin == "" {
			return nil, fmt.Errorf("missing required 'target_plugin' value in payload")
		}
		if cmd.TargetEndpoint == "" {
			return nil, fmt.Errorf("missing required 'target_endpoint' value in payload")
		}

		if cmd.StorePlugin == "" {
			return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
		}
		if cmd.StoreEndpoint == "" {
			return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
		}

		if cmd.RestoreKey == "" {
			return nil, fmt.Errorf("missing required 'restore_key' value in payload (for restore operation)")
		}
		if cmd.EncryptType == "" {
			return nil, fmt.Errorf("missing required 'encrypt_cipher' value in payload")
		}
		if cmd.EncryptKey == "" {
			return nil, fmt.Errorf("missing required 'encrypt_key' value in payload")
		}
		if cmd.EncryptIV == "" {
			return nil, fmt.Errorf("missing required 'encrypt_iv' value in payload")
		}
	case "purge":
		if cmd.StorePlugin == "" {
			return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
		}
		if cmd.StoreEndpoint == "" {
			return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
		}

		if cmd.RestoreKey == "" {
			return nil, fmt.Errorf("missing required 'restore_key' value in payload (for purge operation)")
		}

	case "status":
		/* nothing to validate */

	default:
		return nil, fmt.Errorf("unsupported operation: '%s'", cmd.Op)
	}

	return cmd, nil
}

func ParseCommandFromSSHRequest(r *ssh.Request) (*Command, error) {
	var raw struct{ Value []byte }

	if err := ssh.Unmarshal(r.Payload, &raw); err != nil {
		return nil, err
	}

	return ParseCommand(raw.Value)
}

func (c *Command) Details() string {
	switch c.Op {
	case "backup":
		return fmt.Sprintf("backup of target '%s' to store '%s'",
			c.TargetPlugin, c.StorePlugin)

	case "restore":
		return fmt.Sprintf("restore of [%s] from store '%s' to target '%s'",
			c.RestoreKey, c.StorePlugin, c.TargetPlugin)

	case "purge":
		return fmt.Sprintf("purge of [%s] from store '%s'",
			c.RestoreKey, c.StorePlugin)

	default:
		return fmt.Sprintf("%s op", c.Op)
	}
}

func (agent *Agent) Execute(c *Command, out chan string) error {
	cmd := exec.Command("shield-pipe")

	log.Infof("Executing %s via shield-pipe", c.Details())
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("USER=%s", os.Getenv("USER")),
		fmt.Sprintf("LANG=%s", os.Getenv("LANG")),

		fmt.Sprintf("SHIELD_OP=%s", c.Op),
		fmt.Sprintf("SHIELD_STORE_PLUGIN=%s", c.StorePlugin),
		fmt.Sprintf("SHIELD_STORE_ENDPOINT=%s", c.StoreEndpoint),
		fmt.Sprintf("SHIELD_TARGET_PLUGIN=%s", c.TargetPlugin),
		fmt.Sprintf("SHIELD_TARGET_ENDPOINT=%s", c.TargetEndpoint),
		fmt.Sprintf("SHIELD_RESTORE_KEY=%s", c.RestoreKey),
		fmt.Sprintf("SHIELD_PLUGINS_PATH=%s", strings.Join(agent.PluginPaths, ":")),
		fmt.Sprintf("SHIELD_AGENT_NAME=%s", agent.Name),
		fmt.Sprintf("SHIELD_AGENT_VERSION=%s", agent.Version),
		fmt.Sprintf("SHIELD_ENCRYPT_TYPE=%s", c.EncryptType),
		fmt.Sprintf("SHIELD_ENCRYPT_KEY=%s", c.EncryptKey),
		fmt.Sprintf("SHIELD_ENCRYPT_IV=%s", c.EncryptIV),
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
	go drain("E", out, stderr)
	go drain("O", out, stdout)

	err = cmd.Start()
	if err != nil {
		close(out)
		return err
	}

	wg.Wait()
	close(out)

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (agent *Agent) ResolvePathsIn(c *Command) error {
	if c.TargetPlugin != "" {
		bin, err := agent.ResolveBinary(c.TargetPlugin)
		if err != nil {
			return err
		}
		c.TargetPlugin = bin
	}

	if c.StorePlugin != "" {
		bin, err := agent.ResolveBinary(c.StorePlugin)
		if err != nil {
			return err
		}
		c.StorePlugin = bin
	}

	return nil
}
