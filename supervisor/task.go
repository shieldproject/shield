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

type PluginConfig struct {
	Plugin   string
	Endpoint string
}

type Task struct {
	UUID uuid.UUID

	Store  *PluginConfig
	Target *PluginConfig

	Op     Operation
	Status Status

	StartedAt time.Time
	StoppedAt time.Time

	Output []string
}
