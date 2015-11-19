package supervisor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/starkandwayne/shield/agent"

	"github.com/pborman/uuid"
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

func worker(id uint, privateKeyFile string, work chan Task, updates chan WorkerUpdate) {
	config, err := agent.ConfigureSSHClient(privateKeyFile)
	if err != nil {
		fmt.Printf("worker %d unable to read user key %s: %s; bailing out.\n",
			id, privateKeyFile, err)
		return
	}

	for t := range work {
		client := agent.NewClient(config)

		remote := t.Agent
		if remote == "" {
			updates <- WorkerUpdate{Task: t.UUID, Op: OUTPUT,
				Output: fmt.Sprintf("TASK FAILED!!  no remote agent specified for task %s\n", t.UUID)}
			updates <- WorkerUpdate{Task: t.UUID, Op: FAILED}
			continue
		}

		err = client.Dial(remote)
		if err != nil {
			updates <- WorkerUpdate{Task: t.UUID, Op: OUTPUT,
				Output: fmt.Sprintf("TASK FAILED!!  shield worker %d unable to connect to %s (%s)\n", id, remote, err)}
			updates <- WorkerUpdate{Task: t.UUID, Op: FAILED}
			continue
		}
		defer client.Close()

		// start a command and stream output
		final := make(chan string)
		partial := make(chan string)

		go func(out chan string, up chan WorkerUpdate, t Task, in chan string) {
			var buffer []string
			for {
				s, ok := <-in
				if !ok {
					break
				}

				switch s[0:2] {
				case "O:":
					buffer = append(buffer, s[2:])
				case "E:":
					up <- WorkerUpdate{
						Task:   t.UUID,
						Op:     OUTPUT,
						Output: s[2:],
					}
				}
			}
			out <- strings.Join(buffer, "")
			close(out)
		}(final, updates, t, partial)

		// exec the command
		err = client.Run(partial, fmt.Sprintf(`
{"operation":"%s",
 "target_plugin":"%s", "target_endpoint":"%s",
 "store_plugin":"%s", "store_endpoint":"%s"}`,
			t.Op,
			t.TargetPlugin, t.TargetEndpoint,
			t.StorePlugin, t.StoreEndpoint))
		if err != nil {
			updates <- WorkerUpdate{Task: t.UUID, Op: OUTPUT,
				Output: fmt.Sprintf("TASK FAILED!!  shield worker %d failed to execute the command against the remote agent %s (%s)\n", id, remote, err)}
			updates <- WorkerUpdate{Task: t.UUID, Op: FAILED}
		}

		out := <-final
		if t.Op == BACKUP {
			// parse JSON from standard output and get the restore key
			// (this might fail, we might not get a key, etc.)
			v := struct {
				Key string
			}{}

			buf := bytes.NewBufferString(out)
			dec := json.NewDecoder(buf)
			err := dec.Decode(&v)

			if err != nil {
				updates <- WorkerUpdate{Task: t.UUID, Op: OUTPUT,
					Output: fmt.Sprintf("WORKER FAILED!!  shield worker %d failed to parse JSON response from remote agent %s (%s)\n", id, remote, err)}

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
	go worker(s.nextWorker, s.PrivateKeyFile, s.workers, s.updates)
}
