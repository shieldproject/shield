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

		output := make(chan string)
		stderr := make(chan string)
		stdout := make(chan string)

		// drain stdout to the output[] array
		go func(out chan string, in chan string) {
			var b []string
			for {
				s, ok := <-in
				if !ok {
					break
				}
				b = append(b, s)
			}

			out <- strings.Join(b, "")
			close(out)
		}(output, stdout)

		// relay messages on stderr to the updates
		// channel, wrapped in a WorkerUpdate struct
		go func(t Task, in chan string) {
			for {
				s, ok := <-in
				if !ok {
					break
				}
				updates <- WorkerUpdate{
					Task:   t.UUID,
					Op:     OUTPUT,
					Output: s,
				}
			}
		}(t, stderr)

		// run the task...
		err := t.Run(stdout, stderr)
		if err != nil {
			fmt.Printf("oops: %s\n", err)
		}

		if t.Op == BACKUP {
			// parse JSON from standard output and get the restore key
			// (this might fail, we might not get a key, etc.)
			v := struct {
				Key string
			}{}

			buf := bytes.NewBufferString(<-output)
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
