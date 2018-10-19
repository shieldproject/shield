package fabric

import (
	"bufio"
	"encoding/json"

	"golang.org/x/crypto/ssh"

	"github.com/starkandwayne/shield/core/scheduler"
	"github.com/starkandwayne/shield/core/vault"
	"github.com/starkandwayne/shield/db"
)

type LegacyFabric struct {
	ip     string
	config *ssh.ClientConfig
}

func Legacy(ip, key string) Fabric {
	signer, err := ssh.ParsePrivateKey([]byte(key))
	if err != nil {
		return ErrorFabric{e: err}
	}

	return LegacyFabric{
		ip: ip,
		config: &ssh.ClientConfig{
			Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		},
	}
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

func (f LegacyFabric) Backup(target *db.Target, store *db.Store, compression string, encryption *vault.Parameters) scheduler.Chore {
	op := "backup"

	t, err := target.ConfigJSON()
	if err != nil {
		return f.Error(op, err)
	}

	s, err := store.ConfigJSON()
	if err != nil {
		return f.Error(op, err)
	}

	return f.Execute(op, Command{
		Op: "backup",

		TargetPlugin:   target.Plugin,
		TargetEndpoint: t,

		StorePlugin:   store.Plugin,
		StoreEndpoint: s,

		Compression: compression,

		EncryptType: encryption.Type,
		EncryptKey:  encryption.Key,
		EncryptIV:   encryption.IV,
	})
}

func (f LegacyFabric) Restore(archive *db.Archive, target *db.Target, encryption *vault.Parameters) scheduler.Chore {
	op := "restore"

	t, err := target.ConfigJSON()
	if err != nil {
		return f.Error(op, err)
	}

	return f.Execute(op, Command{
		Op: "restore",

		RestoreKey:     archive.StoreKey,
		TargetPlugin:   target.Plugin,
		TargetEndpoint: t,

		StorePlugin:   archive.StorePlugin,
		StoreEndpoint: archive.StoreEndpoint,

		Compression: archive.Compression,

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

func (f LegacyFabric) Purge(archive *db.Archive) scheduler.Chore {
	return f.Execute("archive purge", Command{
		Op: "purge",

		StorePlugin:   archive.StorePlugin,
		StoreEndpoint: archive.StoreEndpoint,
	})
}

func (f LegacyFabric) TestStore(store *db.Store) scheduler.Chore {
	op := "storage test"

	s, err := store.ConfigJSON()
	if err != nil {
		return f.Error(op, err)
	}

	return f.Execute(op, Command{
		Op: "test-store",

		StorePlugin:   store.Plugin,
		StoreEndpoint: s,
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
			b, err := json.Marshal(command)
			if err != nil {
				chore.Errorf("ERR> unable to marshal %s task payload: %s\n", op, err)
				chore.UnixExit(2)
				return
			}
			payload := string(b)

			conn, err := ssh.Dial("tcp4", f.ip, f.config)
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
