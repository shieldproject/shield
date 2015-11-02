package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"time"
)

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
		Store: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
		Target: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
		Store: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
		Target: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
		Store: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
		Target: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
	})
	s.runq = append(s.runq, &Task{
		uuid:   uuid.NewRandom(),
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
		Store: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
		Target: &PluginConfig{
			Plugin:   "src/supervisor/test/bin/dummy",
			Endpoint: "(endpoint here)",
		},
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

func scheduler(c chan int) {
	for {
		time.Sleep(time.Millisecond * 200)
		c <- 1
	}
}

func (s *Supervisor) SpawnScheduler() {
	go scheduler(s.tick)
}
