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
	OUTPUT
	RESTORE_KEY
)

type WorkerUpdate struct {
	Task      uuid.UUID
	Op        UpdateOp
	StoppedAt time.Time
	Output    string
}

func loadUserKey(path string) ssh.AuthMethod {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		panic("loadUserKey: " + err.Error())
	}

	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		panic("loadUserKey: " + err.Error())
	}

	return ssh.PublicKeys(signer)
}

func worker(id uint, privateKeyFile string, work chan Task, updates chan WorkerUpdate) {
	for t := range work {
		fmt.Printf("worker %d received task %v\n", id, t.UUID.String())

		config := &ssh.ClientConfig{
			Auth: []ssh.AuthMethod{
				loadUserKey(privateKeyFile),
			},
		}

		client, err := ssh.Dial("tcp", "127.0.0.1:2022", config) // FIXME
		if err != nil {
			panic("connection failed: " + err.Error()) // FIXME
		}

		session, err := client.NewSession()
		if err != nil {
			panic("Failed to create session: " + err.Error()) // FIXME
		}
		defer session.Close()

		// start a command and stream output
		rd, wr, err := os.Pipe()
		if err != nil {
			fmt.Printf("worker %d: error: %s\n", id, err)
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
			panic("Failed to run: " + err.Error()) // FIXME
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
				fmt.Printf("uh-oh: %s\n", err)

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
