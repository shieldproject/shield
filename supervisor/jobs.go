package supervisor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/timespec"
)

var DEV_MODE_SCHEDULING bool = false

func init() {
	if os.Getenv("SHIELD_MODE") == "DEV" {
		DEV_MODE_SCHEDULING = true
	}
}

type Job struct {
	UUID uuid.UUID
	Name string

	StorePlugin    string
	StoreEndpoint  string
	TargetPlugin   string
	TargetEndpoint string
	Agent          string

	Spec   *timespec.Spec
	Paused bool

	NextRun time.Time
}

type JobRepresentation struct {
	UUID  uuid.UUID
	Name  string
	Tspec string
	Error error
}

type JobFailedError struct {
	FailedJobs []JobRepresentation
}

func (e JobFailedError) Error() string {
	var jobList []string
	for _, j := range e.FailedJobs {
		jobList = append(jobList, fmt.Sprintf("%s (%s)", j.Name, j.UUID))
	}
	return fmt.Sprintf("the following job(s) failed: %s", strings.Join(jobList, ", "))
}

func (s *Supervisor) GetAllJobs() ([]*Job, error) {
	l := []*Job{}
	result, err := s.Database.Query(`
		SELECT j.uuid, j.name, j.paused,
		       t.plugin, t.endpoint,
		       s.plugin, s.endpoint,
		       sc.timespec, r.expiry, t.agent
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
		err = result.Scan(&id, &j.Name, &j.Paused,
			&j.TargetPlugin, &j.TargetEndpoint,
			&j.StorePlugin, &j.StoreEndpoint,
			&tspec, &expiry, &j.Agent)
		j.UUID = uuid.Parse(id)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, j.Name, tspec, err})
		}
		j.Spec, err = timespec.Parse(tspec)
		if err != nil {
			e.FailedJobs = append(e.FailedJobs, JobRepresentation{j.UUID, j.Name, tspec, err})
		}
		l = append(l, j)
	}
	if len(e.FailedJobs) == 0 {
		return l, nil
	}
	return l, e
}

func (j *Job) Task() *Task {
	t := NewPendingTask(BACKUP)
	t.StorePlugin = j.StorePlugin
	t.StoreEndpoint = j.StoreEndpoint
	t.TargetPlugin = j.TargetPlugin
	t.TargetEndpoint = j.TargetEndpoint
	t.Agent = j.Agent
	return t
}

func (j *Job) Reschedule() error {
	if DEV_MODE_SCHEDULING {
		j.NextRun = time.Now().Add(time.Minute)
		return nil
	}

	next, err := j.Spec.Next(time.Now())
	if err != nil {
		return err
	}
	j.NextRun = next
	return nil
}

func (j *Job) Runnable() bool {
	return j.Paused == false && !j.NextRun.After(time.Now())
}
