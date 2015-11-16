package supervisor

import (
	"github.com/pborman/uuid"
	"time"
)

type Operation int

const (
	BACKUP Operation = iota
	RESTORE
)

func (o Operation) String() string {
	switch o {
	case BACKUP:
		return "backup"
	case RESTORE:
		return "restore"
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

type Task struct {
	UUID uuid.UUID

	StorePlugin    string
	StoreEndpoint  string
	TargetPlugin   string
	TargetEndpoint string
	RestoreKey     string

	Op     Operation
	Status Status

	StartedAt time.Time
	StoppedAt time.Time

	Output []string
}

func NewPendingTask(Op Operation) *Task {
	return &Task{
		Op: Op,
		Status: PENDING,
		Output: make([]string, 0),
	}
}
