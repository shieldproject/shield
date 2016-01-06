package supervisor

import (
	"github.com/pborman/uuid"
	"time"
)

type Operation int

const (
	BACKUP Operation = iota
	RESTORE
	PURGE
)

func (o Operation) String() string {
	switch o {
	case BACKUP:
		return "backup"
	case RESTORE:
		return "restore"
	case PURGE:
		return "purge"
	default:
		return "UNKNOWN"
	}
}

type Status int

const (
	PENDING Status = iota
	RUNNING
	CANCELED
	DONE
)

type AdhocTask struct {
	Op    Operation
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

	Op       Operation
	Status   Status
	Attempts int

	StartedAt time.Time
	StoppedAt time.Time
	TimeoutAt time.Time

	Output []string
}

func NewPendingTask(Op Operation) *Task {
	return &Task{
		Op:     Op,
		Status: PENDING,
		Output: make([]string, 0),
	}
}
