package supervisor

import (
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/timespec"
	"os"
	"time"
)

var DEV_MODE_SCHEDULING bool = false

func init() {
	if os.Getenv("SHIELD_MODE") == "DEV" {
		DEV_MODE_SCHEDULING = true
	}
}

type Job struct {
	UUID uuid.UUID

	StorePlugin    string
	StoreEndpoint  string
	TargetPlugin   string
	TargetEndpoint string
	Agent          string

	Spec   *timespec.Spec
	Paused bool

	NextRun time.Time
}

// job -> task -> run queue

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
