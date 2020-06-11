package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/jhunt/go-log"
	ssg "github.com/jhunt/ssg/pkg/client"
	"golang.org/x/crypto/ssh"

	"github.com/shieldproject/shield/plugin"
	"github.com/shieldproject/shield/plugin/cassandra"
	"github.com/shieldproject/shield/plugin/consul"
	consulsnapshot "github.com/shieldproject/shield/plugin/consul-snapshot"
	"github.com/shieldproject/shield/plugin/etcd"
	"github.com/shieldproject/shield/plugin/fs"
	"github.com/shieldproject/shield/plugin/mongo"
	"github.com/shieldproject/shield/plugin/mysql"
	"github.com/shieldproject/shield/plugin/postgres"
	"github.com/shieldproject/shield/plugin/vault"
	"github.com/shieldproject/shield/plugin/xtrabackup"
)

type Command struct {
	Op             string `json:"operation"`
	TargetPlugin   string `json:"target_plugin,omitempty"`
	TargetEndpoint string `json:"target_endpoint,omitempty"`
	Compression    string `json:"compression,omitempty"` // FIXME
	TaskUUID       string `json:"task_uuid,omitempty"`
	Stream         struct {
		URL   string `json:"url"`
		ID    string `json:"id"`
		Token string `json:"token"`
		Path  string `json:"path,omitempty"`
	} `json:"stream"`
}

func ParseCommand(b []byte) (*Command, error) {
	cmd := &Command{}
	if err := json.Unmarshal(b, &cmd); err != nil {
		return nil, fmt.Errorf("malformed agent command: %s", err)
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

		if cmd.Stream.URL == "" {
			return nil, fmt.Errorf("missing required 'stream.url' value in payload")
		}
		if cmd.Stream.ID == "" {
			return nil, fmt.Errorf("missing required 'stream.id' value in payload")
		}
		if cmd.Stream.Token == "" {
			return nil, fmt.Errorf("missing required 'stream.token' value in payload")
		}
		if cmd.Stream.Path == "" {
			return nil, fmt.Errorf("missing required 'stream.path' value in payload")
		}

	case "restore", "shield-restore":
		if cmd.TargetPlugin == "" {
			return nil, fmt.Errorf("missing required 'target_plugin' value in payload")
		}
		if cmd.TargetEndpoint == "" {
			return nil, fmt.Errorf("missing required 'target_endpoint' value in payload")
		}

		if cmd.Stream.URL == "" {
			return nil, fmt.Errorf("missing required 'stream.url' value in payload")
		}
		if cmd.Stream.ID == "" {
			return nil, fmt.Errorf("missing required 'stream.id' value in payload")
		}
		if cmd.Stream.Token == "" {
			return nil, fmt.Errorf("missing required 'stream.token' value in payload")
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
		return fmt.Sprintf("backup of target '%s' with task_uuid '%s'",
			c.TargetPlugin, c.TaskUUID)

	case "restore":
		return fmt.Sprintf("restore target '%s'",
			c.TargetPlugin)

	default:
		return fmt.Sprintf("%s op", c.Op)
	}
}

func (agent *Agent) Execute(c *Command, out chan string) error {
	defer close(out)

	// Select the target plugin.
	var pT plugin.Plugin
	switch c.TargetPlugin {
	case "cassandra":
		log.Debugf("cassandra selected")
		pT = cassandra.CassandraPlugin{}
	case "consul":
		log.Debugf("consul selected")
		pT = consul.ConsulPlugin{}
	case "consul-snapshot":
		log.Debugf("consul-snapshot selected")
		pT = consulsnapshot.ConsulPlugin{}
	case "etcd":
		log.Debugf("etcd selected")
		pT = etcd.EtcdPlugin{}
	case "fs":
		log.Debugf("fs selected")
		pT = fs.FSPlugin{}
	case "mongo":
		log.Debugf("mongo selected")
		pT = mongo.MongoPlugin{}
	case "mysql":
		log.Debugf("mysql selected")
		pT = mysql.MySQLPlugin{}
	case "postgres":
		log.Debugf("postgres selected")
		pT = postgres.PostgresPlugin{}
	case "vault":
		pT = vault.VaultPlugin{}
	case "xtrabackup":
		pT = xtrabackup.XtraBackupPlugin{}
	case "":
		break
	default:
		return fmt.Errorf("unrecognized target plugin %s", c.TargetPlugin)
	}

	// If the target plugin exists parse the endpoint json data.
	var targetEndpoint plugin.ShieldEndpoint
	if c.TargetPlugin != "" {
		err := json.Unmarshal([]byte(c.TargetEndpoint), &targetEndpoint)
		if err != nil {
			return fmt.Errorf("error parsing target endpoint json data: %s", err)
		}
	}

	var wg sync.WaitGroup

	// The errors channel collects the errors from the operations performs and
	// passes it to the core if any errors exist.
	errors := make(chan error, 2)

	// inputStream and outputStream are used by while doing backup, store and
	// restore, retrieve operations for passing the data between the
	// the target and cloud storage.
	inputStream, outputStream := io.Pipe()

	// logReaderStream and logWriterStream are used by all the plugin functions to
	// collect the task logs on the status of the plugin operations.
	logReaderStream, logWriterStream := io.Pipe()

	// This is a goroutine for sending task logs to the core.
	done := make(chan int)
	go func() {
		s := bufio.NewScanner(logReaderStream)
		for s.Scan() {
			out <- fmt.Sprintf("E:%s\n", s.Text())
		}
		close(done)
	}()

	if agent.Version == "" {
		agent.Version = "dev"
	}

	switch c.Op {
	case "status":
		fmt.Fprintf(logWriterStream, "Running SHIELD Agent "+agent.Version+" Health Checks\n")
		v := struct {
			Name    string                       `json:"name"`
			Version string                       `json:"version"`
			Health  string                       `json:"health"`
			Plugins map[string]plugin.PluginInfo `json:"plugins"`
		}{
			Name:    agent.Name,
			Version: agent.Version,
			Health:  "ok",
			Plugins: map[string]plugin.PluginInfo{
				"cassandra":       cassandra.New().Meta(),
				"consul":          consul.New().Meta(),
				"consul-snapshot": consulsnapshot.New().Meta(),
				"etcd":            etcd.New().Meta(),
				"fs":              fs.New().Meta(),
				"mongo":           mongo.New().Meta(),
				"mysql":           mysql.New().Meta(),
				"postgres":        postgres.New().Meta(),
				"vault":           vault.New().Meta(),
				"xtrabackup":      xtrabackup.New().Meta(),
			},
		}
		b, err := json.Marshal(&v)
		if err != nil {
			errors <- fmt.Errorf("failed to marshall agent data: %s", err)
			break
		}
		out <- fmt.Sprintf("O:%s\n", b)

	case "backup":
		fmt.Fprintf(logWriterStream, "Validating "+c.TargetPlugin+" plugin\n")
		err := pT.Validate(logWriterStream, targetEndpoint)
		if err != nil {
			errors <- fmt.Errorf("target plugin validation failed: %s", err)
			break
		}

		switch c.Compression {
		case "compression":
			fmt.Fprintf(logWriterStream, "Running backup task using compression\n")
		default:
			fmt.Fprintf(logWriterStream, "Running backup task without compression\n")
		}

		// This goroutine performs the backup for the target plugin. The data
		// that is being backed-up is written to the outputStream.
		wg.Add(1)
		go func() {
			if err = pT.Backup(outputStream, logWriterStream, targetEndpoint); err != nil {
				errors <- fmt.Errorf("backup operation failed: %s", err)
			}
			outputStream.Close()
			wg.Done()
		}()

		// This goroutine uploads the archive to the storage gateway.
		wg.Add(1)
		go func() {
			defer func() {
				inputStream.Close()
				wg.Done()
			}()

			gw := ssg.Client{URL: c.Stream.URL}
			size, err := gw.Put(c.Stream.ID, c.Stream.Token, inputStream, true)
			if err != nil {
				errors <- fmt.Errorf("store failed: %s", err)
				return
			}
			v := struct {
				Key         string `json:"key"`
				Size        int64  `json:"archive_size"`
				Compression string `json:"compression"`
			}{
				Key:         c.Stream.Path,
				Size:        size,
				Compression: c.Compression,
			}
			s, err := json.Marshal(&v)
			if err != nil {
				errors <- fmt.Errorf("could not make json encoding of key, size and compression: %s", err)
				return
			}
			out <- fmt.Sprintf("O:%s\n", string(s))
		}()

	case "restore":
		fmt.Fprintf(logWriterStream, "Validating "+c.TargetPlugin+" plugin\n")
		err := pT.Validate(logWriterStream, targetEndpoint)
		if err != nil {
			errors <- fmt.Errorf("target plugin validation failed: %s", err)
			break
		}

		switch c.Compression {
		case "compression":
			fmt.Fprintf(logWriterStream, "Running restore task using compression\n")
		default:
			fmt.Fprintf(logWriterStream, "Running restore task without compression\n")
		}

		// This goroutine retrieves the archive data from the storage gateway,
		// and writes it to outputStream for the target plugin to read.
		wg.Add(1)
		errors := make(chan error, 2)
		go func() {
			gw := ssg.Client{URL: c.Stream.URL}
			dl, err := gw.Get(c.Stream.ID, c.Stream.Token)
			if err != nil {
				errors <- fmt.Errorf("could not retrieve archive: %s", err)
			}

			io.Copy(outputStream, dl)
			dl.Close()
			outputStream.Close()
			wg.Done()
		}()

		// This goroutine is reading data from the inputStream which tracks the
		// data written to outputStream. The data read is then put in the respective
		// target system.
		wg.Add(1)
		go func() {
			err = pT.Restore(inputStream, logWriterStream, targetEndpoint)
			if err != nil {
				errors <- fmt.Errorf("restore operation failed: %s", err)
			}
			inputStream.Close()
			wg.Done()
		}()

	default:
		errors <- fmt.Errorf("unrecognized operation %s", c.Op)
	}

	wg.Wait()
	logWriterStream.Close()
	<-done
	select {
	case err := <-errors:
		return err
	default:
		return nil
	}
}
