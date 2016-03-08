package supervisor

import (
	//"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/db"
	//"github.com/starkandwayne/shield/timespec"
)

type Job struct {
	Job *db.Job
}

func (s *Supervisor) GetAllJobs() ([]*Job, error) {
	jobs, err := s.Database.GetAllJobs(&db.JobFilter{})
	if err != nil {
		return nil, err
	}

	l := make([]*Job, len(jobs))
	for i, j := range jobs {
		l[i] = &Job{Job:j}
	}
	return l, nil
}

func (j *Job) Task() *Task {
	t := NewPendingTask(BACKUP)
	t.StorePlugin = j.Job.StorePlugin
	t.StoreEndpoint = j.Job.StoreEndpoint
	t.TargetPlugin = j.Job.TargetPlugin
	t.TargetEndpoint = j.Job.TargetEndpoint
	t.Agent = j.Job.Agent
	return t
}

func (j *Job) Reschedule() error {
	return j.Job.Reschedule()
}

func (j *Job) Runnable() bool {
	return j.Job.Runnable()
}
