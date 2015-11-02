package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"math/rand"
	"time"
)

type UpdateOp int

const (
	STOPPED UpdateOp = iota
	OUTPUT
)

// A structure passed back to the supervisor, by the workers
// to indicate a change in task state
type WorkerUpdate struct {
	task      uuid.UUID
	op        UpdateOp
	stoppedAt time.Time
	output    string
}

type Supervisor struct {
	tick    chan int
	workers chan Task
	updates chan WorkerUpdate

	tasks map[*uuid.UUID]*Task
	runq  []*Task
	// schedule map[uuid.UUID]*Job

	nextWorker uint
}

func NewSupervisor() *Supervisor {
	s := &Supervisor{
		tick:    make(chan int),
		workers: make(chan Task),
		updates: make(chan WorkerUpdate),
		tasks:   make(map[*uuid.UUID]*Task),
		runq:    make([]*Task, 0),
	}

	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
	})

	return s
}

func (s *Supervisor) Run() {
	// multiplex between workers and supervisor
	for {
		select {
		case <-s.tick:
			fmt.Printf("recieved a TICK from the scheduler\n")

		case u := <-s.updates:
			fmt.Printf("received an update for %s from a worker\n", u.task.String())
			if u.op == STOPPED {
				fmt.Printf("  job stopped at %s\n", u.stoppedAt.String())
			} else if u.op == OUTPUT {
				fmt.Printf("  OUTPUT: `%s`\n", u.output)
			} else {
				fmt.Printf("  unrecognized op type\n")
			}

		default:
			if len(s.runq) > 0 {
				select {
				case s.workers <- *s.runq[0]:
					fmt.Printf("sent a task to a worker\n")
					s.runq = s.runq[1:]
				default:
				}
			}
		}
	}
}

func randomTime(min float32, spread float32) time.Duration {
	return time.Duration(1000*1000*1000*spread*rand.Float32() + min)
}

func worker(id uint, work chan Task, updates chan WorkerUpdate) {
	for t := range work {
		fmt.Printf("worker %d received task %v\n", id, t.uuid.String())

		time.Sleep(randomTime(0, 1))
		updates <- WorkerUpdate{
			task:   t.uuid,
			op:     OUTPUT,
			output: "some output, line 1\n",
		}

		time.Sleep(randomTime(1, 2))
		updates <- WorkerUpdate{
			task:   t.uuid,
			op:     OUTPUT,
			output: "some more output (another line)\n",
		}

		time.Sleep(randomTime(0, 1))
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

func scheduler(c chan int) {
	for {
		time.Sleep(time.Millisecond * 200)
		c <- 1
	}
}

func (s *Supervisor) SpawnScheduler() {
	go scheduler(s.tick)
}
