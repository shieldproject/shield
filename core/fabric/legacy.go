package fabric

import (
	"bufio"
	"encoding/json"

	"github.com/jhunt/go-log"
	"golang.org/x/crypto/ssh"

	"github.com/shieldproject/shield/core/scheduler"
	"github.com/shieldproject/shield/db"
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

	TaskUUID string `json:"task_uuid,omitempty"`

	RestoreKey string `json:"restore_key,omitempty"`

	Compression string `json:"compression,omitempty"` // FIXME
}

func (f LegacyFabric) Backup(task *db.Task) scheduler.Chore {
	op := "backup"

	return f.Execute(op, task.UUID, Command{
		Op: op,

		TargetPlugin:   task.TargetPlugin,
		TargetEndpoint: task.TargetEndpoint,

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,

		TaskUUID: task.UUID,

		Compression: task.Compression, // FIXME
	})
}

func (f LegacyFabric) Restore(task *db.Task) scheduler.Chore {
	op := "restore"

	return f.Execute(op, task.UUID, Command{
		Op: op,

		RestoreKey:     task.RestoreKey,
		TargetPlugin:   task.TargetPlugin,
		TargetEndpoint: task.TargetEndpoint,

		StorePlugin:   task.StorePlugin,
		StoreEndpoint: task.StoreEndpoint,

		TaskUUID: task.UUID,

		Compression: task.Compression, // FIXME
	})
}

func (f LegacyFabric) Status(task *db.Task) scheduler.Chore {
	return f.Execute("agent status", task.UUID, Command{
		Op: "status",
	})
}

func (f LegacyFabric) Execute(op, id string, command Command) scheduler.Chore {
	return scheduler.NewChore(
		id,
		func(chore scheduler.Chore) {
			log.Debugf("starting up legacy agent execution...")
			log.Debugf("checking that we have a SHIELD agent...")
			if f.ip == "" {
				chore.Errorf("ERR> unable to determine SHIELD agent to connect to")
				chore.UnixExit(2)
				return
			}

			log.Debugf("marshaling command into JSON for transport across SSH (legacy) fabric...")
			b, err := json.Marshal(command)
			if err != nil {
				chore.Errorf("ERR> unable to marshal %s task payload: %s", op, err)
				chore.UnixExit(2)
				return
			}
			payload := string(b)

			chore.Errorf("connecting to %s (tcp/ipv4)", f.ip)
			conn, err := ssh.Dial("tcp4", f.ip, f.ssh)
			if err != nil {
				chore.Errorf("ERR> unable to connect to %s: %s", f.ip, err)
				chore.UnixExit(2)
				return
			}
			defer conn.Close()

			chore.Errorf("connected to %s...", f.ip)
			sess, err := conn.NewSession()
			if err != nil {
				chore.Errorf("ERR> unable to create a new execution session against %s: %s", f.ip, err)
				return
			}
			defer sess.Close()

			/* set up an output sink on ssh output pipe */
			pipe, err := sess.StdoutPipe()
			if err != nil {
				chore.Errorf("ERR> unable to redirect standard output from remote execution session: %s", err)
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
					s := b.Text()
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
			chore.Errorf("executing %s task on remote agent.", op)
			err = sess.Run(payload)
			<-wait
			if err != nil {
				chore.Errorf("ERR> remote execution failed: %s", err)
				chore.UnixExit(1)
				return
			}

			chore.UnixExit(0)
		})
}
