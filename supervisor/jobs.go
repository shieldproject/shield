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
	UUID   uuid.UUID

	StorePlugin string
	StoreEndpoint string
	TargetPlugin string
	TargetEndpoint string

	Spec   *timespec.Spec
	Paused bool

	NextRun time.Time
}

// job -> task -> run queue

func (j *Job) Task() *Task {
	return &Task{
		Op:     BACKUP,
		StorePlugin:   j.StorePlugin,
		StoreEndpoint: j.StoreEndpoint,
		TargetPlugin:   j.TargetPlugin,
		TargetEndpoint: j.TargetEndpoint,
		Status: PENDING,
		Output: make([]string, 0),
	}
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
