package supervisor

import (
	"timespec"

	"github.com/pborman/uuid"
)

type Job struct {
	UUID   uuid.UUID
	Store  *PluginConfig
	Target *PluginConfig
	Spec   timespec.Spec
	// FIXME retention policy
	Paused bool
}

// job -> task -> run queue

func (j *Job) Task() *Task {
	return &Task{uuid: uuid.NewRandom(),
		store:  j.Store,
		target: j.Target,
		Op:     BACKUP,
		status: PENDING}
}
