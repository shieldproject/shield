package supervisor

import (
	"db"
	"fmt"
	"github.com/pborman/uuid"
	"strings"
	"time"
	"timespec"
)


type JobRepresentation struct {
	UUID uuid.UUID
	Tspec string
	Error error
}
type JobFailedError struct {
	FailedJobs []JobRepresentation
}

func (e JobFailedError) Error() string {
	var jobList []string
	for _, j := range e.FailedJobs {
		jobList = append(jobList, j.UUID.String())
	}
	return fmt.Sprintf("the following job(s) failed: %s", strings.Join(jobList, ", "))
}

func (s *Supervisor) GetAllJobs() ([]*Job, error) {
	l := []*Job{}
	result, err := s.Database.Query(`
		SELECT j.uuid, j.paused,
		       t.plugin, t.endpoint,
		       s.plugin, s.endpoint,
		       sc.timespec, r.expiry
		FROM jobs j
			INNER JOIN targets   t    ON  t.uuid = j.target_uuid
			INNER JOIN stores    s    ON  s.uuid = j.store_uuid
			INNER JOIN schedules sc   ON sc.uuid = j.schedule_uuid
			INNER JOIN retention r    ON  r.uuid = j.retention_uuid
	`)
	if err != nil {
		return l, err
	}
	e := JobFailedError{}
	for result.Next() {
		j := &Job{Target: &PluginConfig{}, Store: &PluginConfig{}}
		var id, tspec string
		var expiry int
		//var paused bool
		err = result.Scan(&id, &j.Paused,
			&j.Target.Plugin, &j.Target.Endpoint,
			&j.Store.Plugin, &j.Store.Endpoint,
			&tspec, &expiry)
		j.UUID = uuid.Parse(id)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, tspec, err})
		}
		j.Spec, err = timespec.Parse(tspec)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, tspec, err})
		}
		l = append(l, j)
	}
	if len(e.FailedJobs) == 0 {
		return l, nil
	}
	return l, e
}


type Supervisor struct {
	tick    chan int
	workers chan Task
	updates chan WorkerUpdate
	jobs chan Job

	Database *db.DB

	tasks map[*uuid.UUID]*Task
	runq  []*Task

	//schedule map[uuid.UUID]*Job
	jobq	[]*Job

	nextWorker uint
}

func NewSupervisor() *Supervisor {
	s := &Supervisor{
		tick:    make(chan int),
		workers: make(chan Task),
		updates: make(chan WorkerUpdate),
		jobs:		 make(chan Job),
		tasks:   make(map[*uuid.UUID]*Task),
		runq:    make([]*Task, 0),
		jobq:		 make([]*Job, 0),
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

			alljobs, err := s.GetAllJobs()
			if err != nil {
				fmt.Printf("%v", err)
			}
			for _, j := range alljobs {
				fmt.Printf(string(j.UUID))
				//fmt.Printf("The spec is: %v\n",j.Spec)
				//j.Spec.Next(time.Now())
				next_run := time.Now()
			  if ( next_run.Equal(time.Now()) ) {
					fmt.Printf("I am in the if loop!\n")
  				/*s.runq = append(s.runq, &Task{
  					uuid:   j.UUID,
  					Op:     BACKUP,
  					status: PENDING,
  					output: make([]string, 0),
  					Store: &PluginConfig{
  						Plugin:   j.Store.Plugin,
  						Endpoint: j.Store.Endpoint,
  					},
  					Target: &PluginConfig{
  						Plugin:   j.Target.Plugin,
  						Endpoint: j.Target.Endpoint,
  					},
  				})*/
			  }
		  }

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
