package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"time"
)

type UpdateOp int

const (
	STOPPED UpdateOp = iota
	OUTPUT
)

type WorkerUpdate struct {
	task      uuid.UUID
	op        UpdateOp
	stoppedAt time.Time
	output    string
}

func worker(id uint, work chan Task, updates chan WorkerUpdate) {
	for t := range work {
		fmt.Printf("worker %d received task %v\n", id, t.uuid.String())

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
					task:   t.uuid,
					op:     OUTPUT,
					output: s,
				}
			}
		}()

		// run the task...
		err := t.Run(stdout, stderr)
		if err != nil {
			fmt.Printf("oops: %s\n", err)
		}

		// signal to the supervisor that we finished
		updates <- WorkerUpdate{
			task:      t.uuid,
			op:        STOPPED,
			stoppedAt: time.Now(),
		}
	}
}

func (s *Supervisor) SpawnWorker() {
	s.nextWorker += 1
	go worker(s.nextWorker, s.workers, s.updates)
}
