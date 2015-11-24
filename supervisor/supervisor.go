package supervisor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"

	"github.com/pborman/uuid"
)

type Supervisor struct {
	tick    chan int          /* scheduler will send a message at regular intervals */
	resync  chan int          /* api goroutine will send here when the db changes significantly (i.e. new job, updated target, etc.) */
	workers chan Task         /* workers read from this channel to get tasks */
	updates chan WorkerUpdate /* workers write updates to this channel */
	adhoc   chan AdhocTask    /* for submission of new adhoc tasks */

	Database *db.DB

	Port           string /* addr/interface(s) and port to bind */
	PrivateKeyFile string /* path to the SSH private key for talking to remote agents */
	Workers        uint   /* how many workers to spin up */

	runq []*Task
	jobq []*Job

	nextWorker uint
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		tick:    make(chan int),
		resync:  make(chan int),
		workers: make(chan Task),
		adhoc:   make(chan AdhocTask),
		updates: make(chan WorkerUpdate),
		runq:    make([]*Task, 0),
		jobq:    make([]*Job, 0),

		Database: &db.DB{},
	}
}

func (s *Supervisor) Resync() error {
	jobq, err := s.GetAllJobs()
	if err != nil {
		return err
	}

	// calculate the initial run of each job
	for _, job := range jobq {
		err := job.Reschedule()
		if err != nil {
			log.Errorf("error encountered while determining next run of %s (%s): %s",
				job.UUID.String(), job.Spec.String(), err)
		} else {
			log.Infof("initial run of %s (%s) is at %s",
				job.UUID.String(), job.Spec.String(), job.NextRun)
		}
	}

	s.jobq = jobq
	return nil
}

func (s *Supervisor) CheckSchedule() {
	for _, job := range s.jobq {
		if !job.Runnable() {
			continue
		}

		log.Infof("scheduling execution of job %s", job.UUID.String())
		task := job.Task()
		id, err := s.Database.CreateBackupTask("system", job.UUID)
		if err != nil {
			log.Errorf("job -> task conversion / database update failed: %s", err)
			continue
		}

		task.UUID = id
		s.runq = append(s.runq, task)

		err = job.Reschedule()
		if err != nil {
			log.Errorf("error encountered while determining next run of %s (%s): %s",
				job.UUID.String(), job.Spec.String(), err)
		} else {
			log.Infof("next run of %s (%s) is at %s",
				job.UUID.String(), job.Spec.String(), job.NextRun)
		}
	}
}

func (s *Supervisor) ScheduleAdhoc(a AdhocTask) {
	log.Infof("schedule adhoc %s job", a.Op)

	switch a.Op {
	case BACKUP:
		// expect a JobUUID to move to the runq Immediately
		for _, job := range s.jobq {
			if !uuid.Equal(job.UUID, a.JobUUID) {
				continue
			}

			log.Infof("scheduling immediate (ad hoc) execution of job %s", job.UUID.String())
			task := job.Task()
			id, err := s.Database.CreateBackupTask(a.Owner, job.UUID)
			if err != nil {
				log.Errorf("job -> task conversion / database update failed: %s", err)
				continue
			}

			task.UUID = id
			s.runq = append(s.runq, task)
		}

	case RESTORE:
		task := NewPendingTask(RESTORE)
		err := s.Database.GetRestoreTaskDetails(
			a.ArchiveUUID, a.TargetUUID,
			&task.StorePlugin, &task.StoreEndpoint, &task.RestoreKey,
			&task.TargetPlugin, &task.TargetEndpoint, &task.Agent)

		id, err := s.Database.CreateRestoreTask(a.Owner, a.ArchiveUUID, a.TargetUUID)
		if err != nil {
			log.Errorf("restore task database creation failed: %s", err)
			return
		}

		task.UUID = id
		s.runq = append(s.runq, task)
	}
}

func (s *Supervisor) Run() error {
	if err := s.Database.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s database at %s: %s\n",
			s.Database.Driver, s.Database.DSN, err)
	}

	if err := s.Database.CheckCurrentSchema(); err != nil {
		return fmt.Errorf("database failed schema version check: %s\n", err)
	}

	if err := s.Resync(); err != nil {
		return err
	}
	if DEV_MODE_SCHEDULING {
		for _, job := range s.jobq {
			job.NextRun = time.Now()
		}
	}

	for {
		select {
		case <-s.resync:
			if err := s.Resync(); err != nil {
				log.Errorf("resync error: %s", err)
			}

		case <-s.tick:
			s.CheckSchedule()

			// see if we have anything in the run queue
		RunQueue:
			for len(s.runq) > 0 {
				select {
				case s.workers <- *s.runq[0]:
					s.Database.StartTask(s.runq[0].UUID, time.Now())
					log.Infof("sent a task to a worker")
					s.runq = s.runq[1:]
				default:
					break RunQueue
				}
			}

		case adhoc := <-s.adhoc:
			s.ScheduleAdhoc(adhoc)

		case u := <-s.updates:
			switch u.Op {
			case STOPPED:
				log.Infof("  %s: job stopped at %s", u.Task, u.StoppedAt.String())
				if err := s.Database.CompleteTask(u.Task, u.StoppedAt); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case FAILED:
				log.Warnf("  %s: task failed!", u.Task)
				if err := s.Database.FailTask(u.Task, time.Now()); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case OUTPUT:
				log.Errorf("  %s> %s", u.Task, u.Output) // There is only OUTPUT in this case if there is an error
				if err := s.Database.UpdateTaskLog(u.Task, u.Output); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case RESTORE_KEY:
				log.Infof("  %s: restore key is %s", u.Task, u.Output)
				if err := s.Database.CreateTaskArchive(u.Task, u.Output, time.Now()); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			default:
				log.Errorf("  %s: !! unrecognized op type", u.Task)
			}
		}
	}
}

func (s *Supervisor) SpawnAPI() {
	go func(s *Supervisor) {
		db := s.Database.Copy()
		if err := db.Connect(); err != nil {
			log.Errorf("failed to connect to %s database at %s: %s", db.Driver, db.DSN, err)
			return
		}

		ping := &PingAPI{}
		http.Handle("/v1/ping", ping)

		meta := &MetaAPI{
			PrivateKeyFile: s.PrivateKeyFile,
		}
		http.Handle("/v1/meta/", meta)

		jobs := &JobAPI{
			Data:       db,
			ResyncChan: s.resync,
			AdhocChan:  s.adhoc,
		}
		http.Handle("/v1/jobs", jobs)
		http.Handle("/v1/job/", jobs)

		retention := &RetentionAPI{
			Data:       db,
			ResyncChan: s.resync,
		}
		http.Handle("/v1/retention", retention)
		http.Handle("/v1/retention/", retention)

		archives := &ArchiveAPI{
			Data:       db,
			ResyncChan: s.resync,
			AdhocChan:  s.adhoc,
		}
		http.Handle("/v1/archives", archives)
		http.Handle("/v1/archive/", archives)

		schedules := &ScheduleAPI{
			Data:       db,
			ResyncChan: s.resync,
		}
		http.Handle("/v1/schedules", schedules)
		http.Handle("/v1/schedule/", schedules)

		stores := &StoreAPI{
			Data:       db,
			ResyncChan: s.resync,
		}
		http.Handle("/v1/stores", stores)
		http.Handle("/v1/store/", stores)

		targets := &TargetAPI{
			Data:       db,
			ResyncChan: s.resync,
		}
		http.Handle("/v1/targets", targets)
		http.Handle("/v1/target/", targets)

		tasks := &TaskAPI{
			Data: db,
		}
		http.Handle("/v1/tasks", tasks)
		http.Handle("/v1/task/", tasks)

		err := http.ListenAndServe(":"+s.Port, nil)
		if err != nil {
			log.Critf("HTTP API failed %s", err.Error())
			panic("HTTP API failed: " + err.Error())
		}
	}(s)
}

func scheduler(c chan int) {
	for {
		time.Sleep(time.Second)
		c <- 1
	}
}

func (s *Supervisor) SpawnScheduler() {
	go scheduler(s.tick)
}

func (s *Supervisor) SpawnWorkers() {
	var i uint
	for i = 0; i < s.Workers; i++ {
		log.Debugf("spawning worker %d", i)
		s.SpawnWorker()
	}
}
