package supervisor

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
)

type Supervisor struct {
	tick    chan int          /* scheduler will send a message at regular intervals */
	purge   chan int          /* scheduler for purging archive jobs sends a message at regular intervals */
	resync  chan int          /* api goroutine will send here when the db changes significantly (i.e. new job, updated target, etc.) */
	workers chan Task         /* workers read from this channel to get tasks */
	updates chan WorkerUpdate /* workers write updates to this channel */
	adhoc   chan AdhocTask    /* for submission of new adhoc tasks */

	Database *db.DB

	Port           string /* addr/interface(s) and port to bind */
	PrivateKeyFile string /* path to the SSH private key for talking to remote agents */
	Workers        uint   /* how many workers to spin up */
	PurgeAgent     string /* What agent to use for purge jobs */

	schedq []*Task
	runq   []Task
	jobq   []*Job

	nextWorker uint
	Timeout    time.Duration
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		tick:     make(chan int),
		resync:   make(chan int),
		purge:    make(chan int),
		workers:  make(chan Task),
		adhoc:    make(chan AdhocTask),
		updates:  make(chan WorkerUpdate),
		schedq:   make([]*Task, 0),
		runq:     make([]Task, 0),
		jobq:     make([]*Job, 0),
		Timeout:  12 * time.Hour,
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
			log.Errorf("error encountered while determining next run of %s [%s] which runs %s: %s",
				job.Name, job.UUID.String(), job.Spec.String(), err)
		} else {
			log.Infof("initial run of %s [%s] which runs %s is at %s",
				job.Name, job.UUID.String(), job.Spec.String(), job.NextRun)
		}
	}

	s.jobq = jobq
	return nil
}

func (s *Supervisor) ScheduleTask(t *Task) {
	t.TimeoutAt = time.Now().Add(s.Timeout)
	log.Infof("schedule task %s with deadline %v", t.UUID, t.TimeoutAt)
	s.schedq = append(s.schedq, t)
}

func (s *Supervisor) CheckSchedule() {
	for _, job := range s.jobq {
		if !job.Runnable() {
			continue
		}

		log.Infof("scheduling execution of job %s [%s]", job.Name, job.UUID.String())
		task := job.Task()
		id, err := s.Database.CreateBackupTask("system", job.UUID)
		if err != nil {
			log.Errorf("job -> task conversion / database update failed: %s", err)
			continue
		}

		task.UUID = id
		s.ScheduleTask(task)

		err = job.Reschedule()
		if err != nil {
			log.Errorf("error encountered while determining next run of %s (%s): %s",
				job.UUID.String(), job.Spec.String(), err)
		} else {
			log.Infof("next run of %s [%s] which runs %s is at %s",
				job.Name, job.UUID.String(), job.Spec.String(), job.NextRun)
		}
	}
}

func (s *Supervisor) ScheduleAdhoc(a AdhocTask) {
	log.Infof("schedule adhoc %s job", a.Op)

	switch a.Op {
	case BACKUP:
		// expect a JobUUID to move to the schedq Immediately
		for _, job := range s.jobq {
			if !uuid.Equal(job.UUID, a.JobUUID) {
				continue
			}

			log.Infof("scheduling immediate (ad hoc) execution of job %s [%s]", job.Name, job.UUID.String())
			task := job.Task()
			id, err := s.Database.CreateBackupTask(a.Owner, job.UUID)
			if err != nil {
				log.Errorf("job -> task conversion / database update failed: %s", err)
				continue
			}

			task.UUID = id
			s.ScheduleTask(task)
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
		s.ScheduleTask(task)
	}
}

func (s *Supervisor) Sweep() error {
	tasks, err := s.Database.GetAllAnnotatedTasks(
		&db.TaskFilter{
			ForStatus: "running",
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to sweep database of running tasks: %s", err)
	}

	now := time.Now()
	for _, task := range tasks {
		log.Warnf("Found task %s in 'running' state at startup; setting to 'failed'", task.UUID)
		if err := s.Database.FailTask(uuid.Parse(task.UUID), now); err != nil {
			return fmt.Errorf("Failed to sweep database of running tasks [%s]: %s", task.UUID, err)
		}
		if task.Op == "backup" && task.ArchiveUUID != "" {
			archive, err := s.Database.GetAnnotatedArchive(uuid.Parse(task.ArchiveUUID))
			if err != nil {
				log.Warnf("Unable to retrieve archive %s (for task %s) from the database: %s",
					task.UUID, task.ArchiveUUID)
				continue
			}
			log.Warnf("Found archive %s for task %s, purging", task.ArchiveUUID, task.UUID)
			if _, err := s.Database.CreatePurgeTask("", archive); err != nil {
				log.Errorf("Failed to purge archive %s (for task %s, which was running at boot): %s",
					archive.UUID, task.UUID, err)
			}
		}
	}

	return nil
}

func (s *Supervisor) Run() error {
	if err := s.Database.Connect(); err != nil {
		return fmt.Errorf("failed to connect to %s database at %s: %s\n",
			s.Database.Driver, s.Database.DSN, err)
	}

	if err := s.Database.CheckCurrentSchema(); err != nil {
		return fmt.Errorf("database failed schema version check: %s\n", err)
	}

	if err := s.Sweep(); err != nil {
		return err
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

		case <-s.purge:
			s.PurgeArchives()

		case <-s.tick:
			s.CheckSchedule()

			// see if any tasks have been running past the timeout period
			if len(s.runq) > 0 {
				ok := true
				lst := make([]Task, 0)
				now := time.Now()

				for _, runtask := range s.runq {
					if now.After(runtask.TimeoutAt) {
						s.Database.CancelTask(runtask.UUID, now)
						log.Errorf("shield timed out task '%s' after running for %v", runtask.UUID, s.Timeout)
						ok = false

					} else {
						lst = append(lst, runtask)
					}
				}

				if !ok {
					s.runq = lst
				}
			}

			// see if we have anything in the schedule queue
		SchedQueue:
			for len(s.schedq) > 0 {
				select {
				case s.workers <- *s.schedq[0]:
					s.Database.StartTask(s.schedq[0].UUID, time.Now())
					s.schedq[0].Attempts++
					log.Infof("sent a task to a worker")
					s.runq = append(s.runq, *s.schedq[0])
					log.Debugf("added task to the runq")
					s.schedq = s.schedq[1:]
				default:
					break SchedQueue
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
				if err := s.Database.FailTask(u.Task, u.StoppedAt); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case OUTPUT:
				log.Errorf("  %s> %s", u.Task, u.Output) // There is only OUTPUT in this case if there is an error
				if err := s.Database.UpdateTaskLog(u.Task, u.Output); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case RESTORE_KEY:
				log.Infof("  %s: restore key is %s", u.Task, u.Output)
				if id, err := s.Database.CreateTaskArchive(u.Task, u.Output, time.Now()); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				} else {
					if !u.TaskSuccess {
						s.Database.InvalidateArchive(id)
					}
				}

			case PURGE_ARCHIVE:
				log.Infof("  %s: archive %s purged from storage", u.Task, u.Archive)
				if err := s.Database.PurgeArchive(u.Archive); err != nil {
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

		status := &StatusAPI{}
		http.Handle("/v1/status", status)

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

func scheduler(c chan int, interval time.Duration) {
	for {
		time.Sleep(time.Second * interval)
		c <- 1
	}
}

func (s *Supervisor) SpawnScheduler() {
	go scheduler(s.tick, 1)
	go scheduler(s.purge, 1800)
}

func (s *Supervisor) SpawnWorkers() {
	var i uint
	for i = 0; i < s.Workers; i++ {
		log.Debugf("spawning worker %d", i)
		s.SpawnWorker()
	}
}

func (s *Supervisor) PurgeArchives() {
	log.Debugf("scanning for archives needing to be expired")

	// mark archives past their retention policy as expired
	toExpire, err := s.Database.GetExpiredArchives()
	if err != nil {
		log.Errorf("error retrieving archives needing to be expired: %s", err.Error())
	}
	for _, archive := range toExpire {
		log.Infof("marking archive %s has expiration date %s, marking as expired", archive.UUID, archive.ExpiresAt)
		err := s.Database.ExpireArchive(uuid.Parse(archive.UUID))
		if err != nil {
			log.Errorf("error marking archive %s as expired: %s", archive.UUID, err)
			continue
		}
	}

	// get archives that are not valid or purged
	toPurge, err := s.Database.GetArchivesNeedingPurge()
	if err != nil {
		log.Errorf("error retrieving archives to purge: %s", err.Error())
	}

	for _, archive := range toPurge {
		log.Infof("requesting purge of archive %s due to status '%s'", archive.UUID, archive.Status)
		err := s.SchedulePurgeTask(archive)
		if err != nil {
			log.Errorf("error scheduling purge of archive %s: %s", archive.UUID, err)
			continue
		}
	}
}

func (s *Supervisor) SchedulePurgeTask(archive *db.AnnotatedArchive) error {
	task := NewPendingTask(PURGE)
	id, err := s.Database.CreatePurgeTask("system", archive)
	if err != nil {
		return err
	}

	task.UUID = id
	task.StorePlugin = archive.StorePlugin
	task.StoreEndpoint = archive.StoreEndpoint
	task.Agent = s.PurgeAgent
	task.RestoreKey = archive.StoreKey
	task.ArchiveUUID = uuid.Parse(archive.UUID)
	s.ScheduleTask(task)
	return nil
}
