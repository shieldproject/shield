package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"time"
	"bytes"
	"strings"
	"encoding/json"
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

func worker(id uint, work chan Task, updates chan WorkerUpdate) {
	for t := range work {
		fmt.Printf("worker %d received task %v\n", id, t.UUID.String())

		var output []string
		stderr := make(chan string)
		stdout := make(chan string)

		// drain stdout to the output[] array
		go func() {
			for {
				s, ok := <-stdout
				if !ok {
					break
				}
				output = append(output, s)
			}
		}()

		// relay messages on stderr to the updates
		// channel, wrapped in a WorkerUpdate struct
		go func() {
			for {
				s, ok := <-stderr
				if !ok {
					break
				}
				updates <- WorkerUpdate{
					Task:   t.UUID,
					Op:     OUTPUT,
					Output: s,
				}
			}
		}()

		// run the task...
		err := t.Run(stdout, stderr)
		if err != nil {
			fmt.Printf("oops: %s\n", err)
		}

		if t.Op == BACKUP {
			// parse JSON from standard output and get the restore key
			// (this might fail, we might not get a key, etc.)

			// FIXME: stop the drain goroutine for stdout.
			// FIXME: (geoff noticed some data races here, so that may just happen
			//         when he fixes those)
			v := struct {
				Key string
			}{}

			buf := bytes.NewBufferString(strings.Join(output, ""))
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
	go worker(s.nextWorker, s.workers, s.updates)
}
