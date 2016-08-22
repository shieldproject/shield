package supervisor

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/log"

	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timestamp"
)

type Supervisor struct {
	tick  *time.Ticker
	purge *time.Ticker

	resync  chan int          /* api goroutine will send here when the db changes significantly (i.e. new job, updated target, etc.) */
	workers chan *db.Task     /* workers read from this channel to get tasks */
	updates chan WorkerUpdate /* workers write updates to this channel */
	adhoc   chan *db.Task     /* for submission of new adhoc tasks */

	Database *db.DB

	PrivateKeyFile string /* path to the SSH private key for talking to remote agents */
	Workers        uint   /* how many workers to spin up */
	PurgeAgent     string /* What agent to use for purge jobs */

	Web *WebServer /* Webserver that gets spawned to handle http requests */

	schedq []*db.Task
	runq   []*db.Task
	jobq   []*db.Job

	nextWorker uint
	Timeout    time.Duration
}

func NewSupervisor() *Supervisor {
	return &Supervisor{
		tick:     time.NewTicker(time.Second * 1),
		purge:    time.NewTicker(time.Second * 1800),
		resync:   make(chan int),
		workers:  make(chan *db.Task),
		adhoc:    make(chan *db.Task),
		updates:  make(chan WorkerUpdate),
		schedq:   make([]*db.Task, 0),
		runq:     make([]*db.Task, 0),
		jobq:     make([]*db.Job, 0),
		Timeout:  12 * time.Hour,
		Database: &db.DB{},
	}
}

func (s *Supervisor) Resync() error {
	jobq, err := s.Database.GetAllJobs(nil)
	if err != nil {
		return err
	}

	// calculate the initial run of each job
	for _, job := range jobq {
		err := job.Reschedule()
		if err != nil {
			log.Errorf("error encountered while determining next run of %s [%s] which runs %s: %s",
				job.Name, job.UUID, job.Spec, err)
		} else {
			log.Infof("initial run of %s [%s] which runs %s is at %s",
				job.Name, job.UUID, job.Spec, job.NextRun)
		}
	}

	s.jobq = jobq
	return nil
}

func (s *Supervisor) RemoveTaskFromRunq(id uuid.UUID) {
	l := make([]*db.Task, 0)
	for _, task := range s.runq {
		if uuid.Equal(task.UUID, id) {
			continue
		}
		l = append(l, task)
	}
	s.runq = l
}

func (s *Supervisor) ScheduleTask(t *db.Task) {
	t.TimeoutAt = timestamp.Now().Add(s.Timeout)
	log.Infof("schedule task %s with deadline %v", t.UUID, t.TimeoutAt)
	s.schedq = append(s.schedq, t)
}

func (s *Supervisor) CheckSchedule() {
	for _, job := range s.jobq {
		if !job.Runnable() {
			continue
		}

		log.Infof("scheduling execution of job %s [%s]", job.Name, job.UUID)
		task, err := s.Database.CreateBackupTask("system", job)
		if err != nil {
			log.Errorf("job -> task conversion / database update failed: %s", err)
			continue
		}
		s.ScheduleTask(task)

		err = job.Reschedule()
		if err != nil {
			log.Errorf("error encountered while determining next run of %s (%s): %s",
				job.UUID, job.Spec, err)
		} else {
			log.Infof("next run of %s [%s] which runs %s is at %s",
				job.Name, job.UUID, job.Spec, job.NextRun)
		}
	}
}

func (s *Supervisor) ScheduleAdhoc(a *db.Task) {
	log.Infof("schedule adhoc %s job", a.Op)

	switch a.Op {
	case db.BackupOperation:
		// expect a JobUUID to move to the schedq Immediately
		for _, job := range s.jobq {
			if !uuid.Equal(job.UUID, a.JobUUID) {
				continue
			}

			log.Infof("scheduling immediate (ad hoc) execution of job %s [%s]", job.Name, job.UUID)
			task, err := s.Database.CreateBackupTask(a.Owner, job)
			if err != nil {
				log.Errorf("job -> task conversion / database update failed: %s", err)
				if a.TaskUUIDChan != nil {
					a.TaskUUIDChan <- &db.TaskInfo{
						Err:  true,
						Info: err.Error(),
					}
				}
				continue
			}
			if a.TaskUUIDChan != nil {
				a.TaskUUIDChan <- &db.TaskInfo{
					Err:  false,
					Info: task.UUID.String(),
				}
			}

			s.ScheduleTask(task)
		}

	case db.RestoreOperation:
		archive, err := s.Database.GetArchive(a.ArchiveUUID)
		if err != nil {
			log.Errorf("unable to find archive %s for restore task: %s", a.ArchiveUUID, err)
			return
		}
		target, err := s.Database.GetTarget(a.TargetUUID)
		if err != nil {
			log.Errorf("unable to find target %s for restore task: %s", a.TargetUUID, err)
			return
		}
		task, err := s.Database.CreateRestoreTask(a.Owner, archive, target)
		if err != nil {
			log.Errorf("restore task database creation failed: %s", err)
			if a.TaskUUIDChan != nil {
				a.TaskUUIDChan <- &db.TaskInfo{
					Err:  true,
					Info: err.Error(),
				}
			}
			return
		}
		if a.TaskUUIDChan != nil {
			a.TaskUUIDChan <- &db.TaskInfo{
				Err:  false,
				Info: task.UUID.String(),
			}
		}

		s.ScheduleTask(task)
	}
}

func (s *Supervisor) FailUnfinishedTasks() error {
	tasks, err := s.Database.GetAllTasks(
		&db.TaskFilter{
			ForStatus: db.RunningStatus,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to sweep database of running tasks: %s", err)
	}

	now := time.Now()
	for _, task := range tasks {
		log.Warnf("Found task %s in 'running' state at startup; setting to 'failed'", task.UUID)
		if err := s.Database.FailTask(task.UUID, now); err != nil {
			return fmt.Errorf("Failed to sweep database of running tasks [%s]: %s", task.UUID, err)
		}
		if task.Op == db.BackupOperation && task.ArchiveUUID != nil {
			archive, err := s.Database.GetArchive(task.ArchiveUUID)
			if err != nil {
				log.Warnf("Unable to retrieve archive %s (for task %s) from the database: %s",
					task.ArchiveUUID, task.UUID, err)
				continue
			}
			log.Warnf("Found archive %s for task %s, purging", archive.UUID, task.UUID)
			task, err := s.Database.CreatePurgeTask("", archive, s.PurgeAgent)
			if err != nil {
				log.Errorf("Failed to purge archive %s (for task %s, which was running at boot): %s",
					archive.UUID, task.UUID, err)
			} else {
				s.ScheduleTask(task)
			}
		}
	}

	return nil
}

func (s *Supervisor) ReschedulePendingTasks() error {
	tasks, err := s.Database.GetAllTasks(
		&db.TaskFilter{
			ForStatus: db.PendingStatus,
		},
	)
	if err != nil {
		return fmt.Errorf("Failed to sweep database of pending tasks: %s", err)
	}

	for _, task := range tasks {
		log.Warnf("Found task %s in 'pending' state at startup; rescheduling", task.UUID)
		s.ScheduleTask(task)
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

	if err := s.Resync(); err != nil {
		return err
	}
	if err := s.FailUnfinishedTasks(); err != nil {
		return err
	}
	if err := s.ReschedulePendingTasks(); err != nil {
		return err
	}

	for {
		select {
		case <-s.resync:
			if err := s.Resync(); err != nil {
				log.Errorf("resync error: %s", err)
			}

		case <-s.purge.C:
			s.PurgeArchives()

		case <-s.tick.C:
			s.CheckSchedule()

			// see if any tasks have been running past the timeout period
			if len(s.runq) > 0 {
				ok := true
				lst := make([]*db.Task, 0)
				now := timestamp.Now()

				for _, runtask := range s.runq {
					if now.After(runtask.TimeoutAt) {
						s.Database.CancelTask(runtask.UUID, now.Time())
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
			for i := 0; i < len(s.schedq); i++ {
				t := s.schedq[i]

				runnable, err := s.Database.IsTaskRunnable(t)
				if err != nil {
					continue
				}
				if !runnable {
					continue
				}

				select {
				case s.workers <- t:
					s.Database.StartTask(t.UUID, time.Now())
					s.schedq[i].Attempts++
					log.Infof("sent a task to a worker")
					s.runq = append(s.runq, s.schedq[i])
					log.Debugf("added task to the runq")
					s.schedq = append(s.schedq[:i], s.schedq[i+1:]...)
					i -= 1 // adjust index after changing schedq's size
				default:
					break SchedQueue
				}
			}

		case adhoc := <-s.adhoc:
			s.ScheduleAdhoc(adhoc)

		case u := <-s.updates:
			switch u.Op {
			case STOPPED:
				log.Infof("  %s: job stopped at %s", u.Task, u.StoppedAt)
				s.RemoveTaskFromRunq(u.Task)
				if err := s.Database.CompleteTask(u.Task, u.StoppedAt); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case FAILED:
				log.Warnf("  %s: task failed!", u.Task)
				s.RemoveTaskFromRunq(u.Task)
				if err := s.Database.FailTask(u.Task, u.StoppedAt); err != nil {
					log.Errorf("  %s: !! failed to update database - %s", u.Task, err)
				}

			case OUTPUT:
				log.Infof("  %s> %s", u.Task, strings.Trim(u.Output, "\n"))
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
	go s.Web.Start()
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
		err := s.Database.ExpireArchive(archive.UUID)
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
		task, err := s.Database.CreatePurgeTask("system", archive, s.PurgeAgent)
		if err != nil {
			log.Errorf("error scheduling purge of archive %s: %s", archive.UUID, err)
			continue
		}
		s.ScheduleTask(task)
	}
}
