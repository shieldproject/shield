package supervisor

import (
	"timespec"

	"time"

	"github.com/pborman/uuid"
)

type Job struct {
	UUID   uuid.UUID
	Store  *PluginConfig
	Target *PluginConfig
	Spec   *timespec.Spec
	// FIXME retention policy
	Paused bool

	NextRun time.Time
}

// job -> task -> run queue

func (j *Job) Task() *Task {
	return &Task{
		uuid: uuid.NewRandom(),
		Store: &PluginConfig{
			Plugin:   j.Store.Plugin,
			Endpoint: j.Store.Endpoint,
		},
		Target: &PluginConfig{
			Plugin:   j.Target.Plugin,
			Endpoint: j.Target.Endpoint,
		},
		Op:     BACKUP,
		status: PENDING,
		output: make([]string, 0),
	}
}

func (j *Job) Reschedule() error {
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
