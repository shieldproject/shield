package task

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type Info struct {
	TaskID  string
	Method  string
	Payload []byte
}

type ManagerProvider interface {
	NewManager(boshlog.Logger, boshsys.FileSystem, string) Manager
}

type Manager interface {
	GetInfos() ([]Info, error)
	AddInfo(taskInfo Info) error
	RemoveInfo(taskID string) error
}
