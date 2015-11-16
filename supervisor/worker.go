package supervisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"os"
	"io"
	"io/ioutil"
	"bufio"

	"github.com/pborman/uuid"
	"golang.org/x/crypto/ssh"
)

type UpdateOp int

const (
	STOPPED UpdateOp = iota
	FAILED
	OUTPUT
	RESTORE_KEY
)

type WorkerUpdate struct {
	Task      uuid.UUID
	Op        UpdateOp
	StoppedAt time.Time
	Output    string
}

func loadUserKey(path string) (ssh.AuthMethod, error) {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

func worker(id uint, privateKeyFile string, work chan Task, updates chan WorkerUpdate) {
	auth, err := loadUserKey(privateKeyFile)
	if err != nil {
		fmt.Printf("worker %d unable to read user key %s: %s; bailing out.\n",
			id, privateKeyFile, err)
		return
	}

	for t := range work {
		config := &ssh.ClientConfig{
			Auth: []ssh.AuthMethod{ auth },
		}

		remote := "127.0.0.1:2022" // eIXME: hard-coded value
		client, err := ssh.Dial("tcp", remote, config)
		if err != nil {
			fmt.Printf("worker %d unable to connect to %s: %s; ignoring this task.\n", id, remote, err)
			updates <- WorkerUpdate{ Task: t.UUID, Op: FAILED }
			continue
		}

		session, err := client.NewSession()
		if err != nil {
			fmt.Printf("worker %d (on %s): unable to create remote session: %s; ignoring this task.\n", id, remote, err)
			updates <- WorkerUpdate{ Task: t.UUID, Op: FAILED }
			continue
		}
		defer session.Close()

		// start a command and stream output
		rd, wr, err := os.Pipe()
		if err != nil {
			fmt.Printf("worker %d (on %s): general error: %s\n", id, remote, err)
			updates <- WorkerUpdate{ Task: t.UUID, Op: FAILED }
			session.Close()
			continue
		}

		session.Stdout = wr

		output := make(chan string)
		go func(out chan string, up chan WorkerUpdate, t Task, rd io.Reader) {
			var buffer []string
			b := bufio.NewScanner(rd)
			for b.Scan() {
				s := b.Text()
				switch s[0:2] {
				case "O:":
					buffer = append(buffer, s[2:])
				case "E:":
					up <- WorkerUpdate{
						Task:   t.UUID,
						Op:     OUTPUT,
						Output: s[2:],
					}
				default:
					fmt.Printf("Unhandled output `%s`\n", s)
				}
			}
			out <- strings.Join(buffer, "")
			close(out)
		}(output, updates, t, rd)

		// exec the command
		err = session.Start(fmt.Sprintf(`
{"operation":"%s",
 "target_plugin":"%s", "target_endpoint":"%s",
 "store_plugin":"%s", "store_endpoint":"%s"}`,
			t.Op,
			t.Target.Plugin, t.Target.Endpoint,
			t.Store.Plugin, t.Store.Endpoint))

		if err != nil {
			fmt.Printf("worker %d (on %s): run failed: %s\n", id, remote, err)
			updates <- WorkerUpdate{ Task: t.UUID, Op: FAILED }
		}

		session.Wait()
		session.Close()

		final := <-output
		// run the task...
		if t.Op == BACKUP {
			// parse JSON from standard output and get the restore key
			// (this might fail, we might not get a key, etc.)
			v := struct {
				Key string
			}{}

			buf := bytes.NewBufferString(final)
			dec := json.NewDecoder(buf)
			err := dec.Decode(&v)

			if err != nil {
				fmt.Printf("worker %d (on %s): %s\n", id, remote, err)

			} else {
				updates <- WorkerUpdate{
					Task:   t.UUID,
					Op:     RESTORE_KEY,
					Output: v.Key,
				}
			}
		}

		// signal to the supervisor that we finished
		updates <- WorkerUpdate{
			Task:      t.UUID,
			Op:        STOPPED,
			StoppedAt: time.Now(),
		}
	}
}

func (s *Supervisor) SpawnWorker() {
	s.nextWorker += 1
	go worker(s.nextWorker, s.privateKeyFile, s.workers, s.updates)
}
