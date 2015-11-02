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
	uuid uuid.UUID

	Store  *PluginConfig
	Target *PluginConfig

	Op     Operation
	status Status

	startedAt time.Time
	stoppedAt time.Time

	output []string
}
