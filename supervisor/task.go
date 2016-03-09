package supervisor

import (
	"github.com/pborman/uuid"
	"time"

	"github.com/starkandwayne/shield/db"
)

type AdhocTask struct {
	Op    string
	Owner string

	TargetUUID  uuid.UUID
	ArchiveUUID uuid.UUID
	RestoreKey  string

	JobUUID uuid.UUID
}

type Task struct {
	UUID uuid.UUID

	StorePlugin    string
	StoreEndpoint  string
	ArchiveUUID    uuid.UUID
	TargetPlugin   string
	TargetEndpoint string
	RestoreKey     string
	Agent          string

	Op       string
	Status   string
	Attempts int

	StartedAt time.Time
	StoppedAt time.Time
	TimeoutAt time.Time

	Output []string
}

func NewPendingTask(Op string) *Task {
	return &Task{
		Op:     Op,
		Status: db.PendingStatus,
		Output: make([]string, 0),
	}
}
