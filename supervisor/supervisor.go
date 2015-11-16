package supervisor

import (
	"fmt"
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
	"github.com/starkandwayne/shield/timespec"
	"net/http"
	"os"
	"strings"
	"time"
)

type JobRepresentation struct {
	UUID  uuid.UUID
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
		j := &Job{}
		var id, tspec string
		var expiry int
		err = result.Scan(&id, &j.Paused,
			&j.TargetPlugin, &j.TargetEndpoint,
			&j.StorePlugin, &j.StoreEndpoint,
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

type AdhocTask struct {
	Op          Operation

	TargetUUID  uuid.UUID
	ArchiveUUID uuid.UUID
	RestoreKey  string

	JobUUID     uuid.UUID
}

type Supervisor struct {
	tick    chan int          /* scheduler will send a message at regular intervals */
	resync  chan int          /* api goroutine will send here when the db changes significantly (i.e. new job, updated target, etc.) */
	workers chan Task         /* workers read from this channel to get tasks */
	updates chan WorkerUpdate /* workers write updates to this channel */
	adhoc   chan AdhocTask    /* for submission of new adhoc tasks */

	Database *db.DB

	Listen         string /* addr/interface(s) and port to bind */
	PrivateKeyFile string /* path to the SSH private key for talking to remote agents */

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
			fmt.Printf("error encountered while determining next run of %s (%s): %s\n",
				job.UUID.String(), job.Spec.String(), err)
		} else {
			fmt.Printf("initial run of %s (%s) is at %s\n",
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

		fmt.Printf("scheduling execution of job %s\n", job.UUID.String())
		task := job.Task()
		id, err := s.Database.CreateTask(
			"system", // owner
			"backup",
			"ARGS", // FIXME: need real args
			job.UUID,
		)
		if err != nil {
			fmt.Printf("job -> task conversion / database update failed: %s\n", err)
			continue
		}

		task.UUID = id
		s.runq = append(s.runq, task)

		err = job.Reschedule()
		if err != nil {
			fmt.Printf("error encountered while determining next run of %s (%s): %s\n",
				job.UUID.String(), job.Spec.String(), err)
		} else {
			fmt.Printf("next run of %s (%s) is at %s\n",
				job.UUID.String(), job.Spec.String(), job.NextRun)
		}
	}
}

func (s *Supervisor) ScheduleAdhoc(a AdhocTask) {
	fmt.Printf("schedule adhoc %s job\n", a.Op)

	switch a.Op {
	case BACKUP:
		// expect a JobUUID to move to the runq Immediately
		for _, job := range s.jobq {
			if !uuid.Equal(job.UUID, a.JobUUID) {
				continue
			}

			fmt.Printf("scheduling immediate (ad hoc) execution of job %s\n", job.UUID.String())
			task := job.Task()
			id, err := s.Database.CreateTask(
				"adhoc", // FIXME: need a better owner
				"backup",
				"ARGS", // FIXME: need real args
				job.UUID,
			)
			if err != nil {
				fmt.Printf("job -> task conversion / database update failed: %s\n", err)
				continue
			}

			task.UUID = id
			s.runq = append(s.runq, task)
		}

	case RESTORE:
		// FIXME: support for RESTORE tasks
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
				fmt.Printf("resync error: %s\n", err)
			}

		case <-s.tick:
			s.CheckSchedule()

		case adhoc := <-s.adhoc:
			s.ScheduleAdhoc(adhoc)

		case u := <-s.updates:
			switch u.Op {
			case STOPPED:
				fmt.Printf("  %s: job stopped at %s\n", u.Task, u.StoppedAt.String())
				if err := s.Database.CompleteTask(u.Task, u.StoppedAt); err != nil {
					fmt.Printf("  %s: !! failed to update database - %s\n", u.Task, err)
				}

			case FAILED:
				fmt.Printf("  %s: task failed!\n", u.Task)
				if err := s.Database.FailTask(u.Task, time.Now()); err != nil {
					fmt.Printf("  %s: !! failed to update database - %s\n", u.Task, err)
				}

			case OUTPUT:
				fmt.Printf("  %s> %s\n", u.Task, u.Output)
				if err := s.Database.UpdateTaskLog(u.Task, u.Output); err != nil {
					fmt.Printf("  %s: !! failed to update database - %s\n", u.Task, err)
				}

			case RESTORE_KEY:
				fmt.Printf("  %s: restore key is %s\n", u.Task, u.Output)
				if err := s.Database.CreateTaskArchive(u.Task, u.Output, time.Now()); err != nil {
					fmt.Printf("  %s: !! failed to update database - %s\n", u.Task, err)
				}

			default:
				fmt.Printf("  %s: !! unrecognized op type\n", u.Task)
			}

		default:
			if len(s.runq) > 0 {
				select {
				case s.workers <- *s.runq[0]:
					s.Database.StartTask(s.runq[0].UUID, time.Now())
					fmt.Printf("sent a task to a worker\n")
					s.runq = s.runq[1:]
				default:
				}
			}
		}
	}
}

func (s *Supervisor) SpawnAPI() {
	go func(s *Supervisor) {
		db := s.Database.Copy()
		if err := db.Connect(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect to %s database at %s: %s\n",
				db.Driver, db.DSN, err)
			return
		}

		ping := &PingAPI{}
		http.Handle("/v1/ping", ping)

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

		http.ListenAndServe(s.Listen, nil)
	}(s)
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
