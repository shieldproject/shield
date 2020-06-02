package agent

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/jhunt/go-log"
	"github.com/shieldproject/shield/plugin"
	"github.com/shieldproject/shield/plugin/cassandra"
	"github.com/shieldproject/shield/plugin/consul"
	consulsnapshot "github.com/shieldproject/shield/plugin/consul-snapshot"
	"github.com/shieldproject/shield/plugin/etcd"
	"github.com/shieldproject/shield/plugin/fs"
	"github.com/shieldproject/shield/plugin/mongo"
	"github.com/shieldproject/shield/plugin/mysql"
	"github.com/shieldproject/shield/plugin/postgres"
	"github.com/shieldproject/shield/plugin/ssg"
	"github.com/shieldproject/shield/plugin/vault"
	"github.com/shieldproject/shield/plugin/webdav"
	"github.com/shieldproject/shield/plugin/xtrabackup"
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
	Compression    string `json:"compression,omitempty"`
	TaskUUID       string `json:"task_uuid,omitempty"`
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

	case "restore", "shield-restore":
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

	case "test-store":
		if cmd.StorePlugin == "" {
			return nil, fmt.Errorf("missing required 'store_plugin' value in payload")
		}
		if cmd.StoreEndpoint == "" {
			return nil, fmt.Errorf("missing required 'store_endpoint' value in payload")
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
		return fmt.Sprintf("backup of target '%s' to store '%s' with task_uuid '%s'",
			c.TargetPlugin, c.StorePlugin, c.TaskUUID)

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

func appendEndpointVariables(env []string, prefix, raw string) []string {
	cooked := make(map[string]interface{})
	err := json.Unmarshal([]byte(raw), &cooked)
	if err != nil {
		return env
	}

	re := regexp.MustCompile(`[^A-Z0-9]+`)
	for k, v := range cooked {
		k = re.ReplaceAllString(strings.ToUpper(k), "_")
		env = append(env, fmt.Sprintf("%s%s=%v", prefix, k, v))
	}
	return env
}

func (agent *Agent) Execute(c *Command, out chan string) error {
	//   c.OP                      Operation: either 'backup' or 'restore'
	//   c.TargetPlugin            Target plugin to use
	//   c.Targetendpoint          The target endpoint config (probably JSON)
	//   c.StorePlugin             Store plugin to use
	//   c.StoreEndpoint           The store endpoint config (probably JSON)
	//   c.RestoreKey              Archive key for 'restore' operations
	//   c.Compression             What type of compression to perform
	//   c.EncryptType             Cipher and mode to be used for archive encryption
	//   c.EncryptKey              Encryption key for archive encryption
	//   c.EncryptIV               Initialization vector for archive encryption

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

	// Select the store plugin.
	var pS plugin.Plugin
	switch c.StorePlugin {
	case "ssg":
		log.Debugf("ssg plugin selected")
		pS = ssg.SsgPlugin{}
	case "webdav":
		log.Debugf("webdav plugin selected")
		pS = webdav.WebDAVPlugin{}
	case "":
		break
	default:
		return fmt.Errorf("unrecognized store plugin %s", c.StorePlugin)
	}

	// If the target plugin exists parse the endpoint json data.
	var targetEndpoint plugin.ShieldEndpoint
	if c.TargetPlugin != "" {
		err := json.Unmarshal([]byte(c.TargetEndpoint), &targetEndpoint)
		if err != nil {
			return fmt.Errorf("error parsing target endpoint json data: %s", err)
		}
	}

	// If the store plugin exists parse the endpoint json data.
	var storeEndpoint plugin.ShieldEndpoint
	if c.StorePlugin != "" {
		err := json.Unmarshal([]byte(c.StoreEndpoint), &storeEndpoint)
		if err != nil {
			return fmt.Errorf("error parsing store endpoint json data :%s", err)
		}
	}

	var wg sync.WaitGroup

	// The errors channel collects the errors from the operations performs and
	// passes it to the core if any errors exist.
	errors := make(chan error, 2)

	// inputStream and outputStream are used by while doing backup, store and
	// restore, retrieve operations for passing the data between the
	// the target and the store plugins.
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

	fmt.Fprintf(logWriterStream, "Validating environment...\n")
	if c.EncryptType == "" {
		fmt.Fprintf(logWriterStream, "SHIELD encryption type not set...\n")
	} else {
		fmt.Fprintf(logWriterStream, "Encryption type ... %s\n", c.EncryptType)
	}

	if c.EncryptKey == "" {
		fmt.Fprintf(logWriterStream, "SHIELD encryption key not set...\n")
	} else {
		fmt.Fprintf(logWriterStream, "Encryption key ... found\n")
	}

	if c.EncryptIV == "" {
		fmt.Fprintf(logWriterStream, "SHIELD encryption iv not set...\n")
	} else {
		fmt.Fprintf(logWriterStream, "Encryption IV ... found\n")
	}

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
				"ssg":             ssg.New().Meta(),
				"vault":           vault.New().Meta(),
				"webdav":          webdav.New().Meta(),
				"xtrabackup":      xtrabackup.New().Meta(),
			},
		}
		b, err := json.Marshal(&v)
		if err != nil {
			errors <- fmt.Errorf("failed to marshall agent data: %s", err)
			break
		}
		out <- fmt.Sprintf("O:%s\n", b)

	case "test-store":
		fmt.Fprintf(logWriterStream, "Validating "+c.StorePlugin+" plugin\n")
		err := pS.Validate(logWriterStream, storeEndpoint)
		if err != nil {
			out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
			errors <- fmt.Errorf("store plugin validation failed: %s", err)
			break
		}

		fmt.Fprintf(logWriterStream, "Performing store / retrieve / purge test\n")
		fmt.Fprintf(logWriterStream, "generating an input bit pattern\n")
		var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		b := make([]byte, 25)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		input := "test::" + base64.URLEncoding.EncodeToString(b)

		wg.Add(1)
		go func() {
			defer wg.Done()
			outputStream.Write([]byte(input))
			outputStream.Close()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			key, _, err := pS.Store(inputStream, logWriterStream, storeEndpoint)
			if err != nil {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("store operation failed: %s", err)
				return
			}

			// os.Pipe() works for now because our test input is always 25 bytes.
			// This is storing less than one pipe's worth of data in the archive.
			// Should the size increase to more than one pipe's worth in the future
			// for testing purposes, it will block and we will deadlock.
			rd, wr, err := os.Pipe()
			if err != nil {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("failed to initialize pipe: %s", err)
				return
			}
			err = pS.Retrieve(wr, logWriterStream, storeEndpoint, key)
			if err != nil {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("restore operation failed: %s", err)
				return
			}
			wr.Close()

			err = pS.Purge(logWriterStream, storeEndpoint, key)
			if err != nil {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("purge operation failed: %s", err)
				return
			}

			s := bufio.NewScanner(rd)
			var output string
			for s.Scan() {
				output = output + s.Text()
			}

			fmt.Fprintf(logWriterStream, "INPUT:  %s\n", input)
			fmt.Fprintf(logWriterStream, "OUTPUT: %s\n", output)
			fmt.Fprintf(logWriterStream, "KEY:    %s\n", key)

			if output == "" {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("unable to read from storage")
				return
			}

			if input != output {
				out <- fmt.Sprintf("O:%s\n", "{\"healthy\":false}")
				errors <- fmt.Errorf("input string does not match output string")
				return
			}
			out <- fmt.Sprintf("O:%s\n", "{\"healthy\":true}")
		}()

	case "backup":
		fmt.Fprintf(logWriterStream, "Validating "+c.TargetPlugin+" plugin\n")
		err := pT.Validate(logWriterStream, targetEndpoint)
		if err != nil {
			errors <- fmt.Errorf("target plugin validation failed: %s", err)
			break
		}

		fmt.Fprintf(logWriterStream, "Validating "+c.StorePlugin+" plugin\n")
		err = pS.Validate(logWriterStream, storeEndpoint)
		if err != nil {
			errors <- fmt.Errorf("store plugin validation failed: %s", err)
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
			defer func() {
				wg.Done()
				outputStream.Close()
			}()
			err = pT.Backup(outputStream, logWriterStream, targetEndpoint)
			if err != nil {
				errors <- fmt.Errorf("backup operation failed: %s", err)
			}
		}()

		// This goroutine performs the store function on the store plugin. The
		// store plugin reads from the inputStream which is continuously getting
		// data from the outputStream. Once all the data is read, it returns the
		// restore key and the size of the archive.
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, size, err := pS.Store(inputStream, logWriterStream, storeEndpoint)
			if err != nil {
				errors <- fmt.Errorf("store operation failed: %s", err)
				return
			}
			v := struct {
				Key         string `json:"key"`
				Size        int64  `json:"archive_size"`
				Compression string `json:"compression"`
			}{
				Key:         key,
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

		fmt.Fprintf(logWriterStream, "Validating "+c.StorePlugin+" plugin\n")
		err = pS.Validate(logWriterStream, storeEndpoint)
		if err != nil {
			errors <- fmt.Errorf("store plugin validation failed: %s", err)
			break
		}

		switch c.Compression {
		case "compression":
			fmt.Fprintf(logWriterStream, "Running restore task using compression\n")
		default:
			fmt.Fprintf(logWriterStream, "Running restore task without compression\n")
		}

		// This goroutine is retrieving the archive data from the respective store
		// plugin. The data is being written to outputStream.
		wg.Add(1)
		errors := make(chan error, 2)
		go func() {
			defer func() {
				wg.Done()
				outputStream.Close()
			}()
			err = pS.Retrieve(outputStream, logWriterStream, storeEndpoint, c.RestoreKey)
			if err != nil {
				errors <- fmt.Errorf("could not retrieve from path %s, error: %s", c.RestoreKey, err)
			}
		}()

		// This goroutine is reading data from the inputStream which tracks the
		// data written to outputStream. The data read is then put in the respective
		// target system.
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = pT.Restore(inputStream, logWriterStream, targetEndpoint)
			if err != nil {
				errors <- fmt.Errorf("restore operation failed: %s", err)
			}
		}()

	case "purge":
		fmt.Fprintf(logWriterStream, "Validating "+c.StorePlugin+" plugin\n")
		err := pS.Validate(logWriterStream, storeEndpoint)
		if err != nil {
			errors <- fmt.Errorf("store plugin validation failed: %s", err)
			break
		}

		fmt.Fprintf(logWriterStream, "Running purge task\n")
		err = pS.Purge(logWriterStream, storeEndpoint, c.RestoreKey)
		if err != nil {
			errors <- fmt.Errorf("purge operatoin failed: %s", err)
			break
		}

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
