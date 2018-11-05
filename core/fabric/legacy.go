package fabric

import (
	"bufio"
	"encoding/json"

	"golang.org/x/crypto/ssh"
	"github.com/jhunt/go-log"

	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

func Legacy(ip string, config *ssh.ClientConfig, db *db.DB) LegacyFabric {
	return LegacyFabric{
		ip:  ip,
		ssh: config,
		db:  db,
	}
}

type LegacyFabric struct {
	ip  string
	ssh *ssh.ClientConfig
	db  *db.DB
}

type Command struct {
	Op string `json:"operation"`

	TargetPlugin   string `json:"target_plugin,omitempty"`
	TargetEndpoint string `json:"target_endpoint,omitempty"`

	StorePlugin   string `json:"store_plugin,omitempty"`
	StoreEndpoint string `json:"store_endpoint,omitempty"`

	RestoreKey string `json:"restore_key,omitempty"`

	EncryptType string `json:"encrypt_type,omitempty"`
	EncryptKey  string `json:"encrypt_key,omitempty"`
	EncryptIV   string `json:"encrypt_iv,omitempty"`

	Compression string `json:"compression,omitempty"`
}

func (f LegacyFabric) Backup(task *db.Task, encryption vault.Parameters) scheduler.Chore {
	op := "backup"

	return f.Execute(op, Command{
		Op: op,

		TargetPlugin:   task.TargetPlugin,
		TargetEndpoint: task.TargetEndpoint,

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,

		Compression: task.Compression,

		EncryptType: encryption.Type,
		EncryptKey:  encryption.Key,
		EncryptIV:   encryption.IV,
	})
}

func (f LegacyFabric) Restore(task *db.Task, encryption vault.Parameters) scheduler.Chore {
	op := "restore"

	return f.Execute(op, Command{
		Op: op,

		RestoreKey:     task.RestoreKey,
		TargetPlugin:   task.TargetPlugin,
		TargetEndpoint: task.TargetEndpoint,

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,

		Compression: task.Compression,

		EncryptType: encryption.Type,
		EncryptKey:  encryption.Key,
		EncryptIV:   encryption.IV,
	})
}

func (f LegacyFabric) Status() scheduler.Chore {
	return f.Execute("agent status", Command{
		Op: "status",
	})
}

func (f LegacyFabric) Purge(task *db.Task) scheduler.Chore {
	return f.Execute("archive purge", Command{
		Op: "purge",

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,
	})
}

func (f LegacyFabric) TestStore(task *db.Task) scheduler.Chore {
	op := "storage test"

	return f.Execute(op, Command{
		Op: "test-store",

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,
	})
}

func (f LegacyFabric) Error(op string, err error) scheduler.Chore {
	return scheduler.NewChore(
		func(chore scheduler.Chore) {
			chore.Errorf("ERR> unable to marshal %s task payload: %s\n", op, err)
			chore.UnixExit(2)
			return
		})
}

func (f LegacyFabric) Execute(op string, command Command) scheduler.Chore {
	return scheduler.NewChore(
		func(chore scheduler.Chore) {
			log.Debugf("starting up legacy agent execution...")
			log.Debugf("marshaling command into JSON for transport across SSH (legacy) fabric")
			b, err := json.Marshal(command)
			if err != nil {
				chore.Errorf("ERR> unable to marshal %s task payload: %s\n", op, err)
				chore.UnixExit(2)
				return
			}
			payload := string(b)

			chore.Infof("connecing to %s (tcp/ipv4)\n", f.ip)
			conn, err := ssh.Dial("tcp4", f.ip, f.ssh)
			if err != nil {
				chore.Errorf("ERR> unable to connect to %s: %s\n", err)
				chore.UnixExit(2)
				return
			}
			defer conn.Close()

			chore.Errorf("connected to %s...\n", f.ip)
			sess, err := conn.NewSession()
			if err != nil {
				chore.Errorf("ERR> unable to create a new execution session against %s: %s\n", f.ip, err)
				return
			}
			defer sess.Close()

			/* set up an output sink on ssh output pipe */
			pipe, err := sess.StdoutPipe()
			if err != nil {
				chore.Errorf("ERR> unable to redirect standard output from remote execution session: %s\n", err)
				return
			}

			/* we do this in a goroutine so that we can
			   exec the payload in the main thread. */
			wait := make(chan bool)
			go func() {
				/* on the other side of the ssh session,
				   the shield-agent process combines standard
				   output and standard error into a single
				   stream, prefixing each line with either
				   "O:" (stdout) or "E:" (stderr). */
				b := bufio.NewScanner(pipe)
				for b.Scan() {
					s := b.Text() + "\n"
					switch s[:2] {
					case "O:":
						chore.Infof("%s", s[2:])
					case "E:":
						chore.Errorf("%s", s[2:])
					}
				}

				wait <- true
			}()

			/* execute the payload remotely */
			chore.Errorf("executing %s task on remote agent.\n", op)
			err = sess.Run(payload)
			<-wait
			if err != nil {
				chore.Errorf("ERR> remote execution failed: %s\n", err)
			}
		})
}
