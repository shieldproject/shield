package task

import (
	"encoding/json"
	"path"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type concreteManagerProvider struct{}

func NewManagerProvider() ManagerProvider {
	return concreteManagerProvider{}
}

func (provider concreteManagerProvider) NewManager(
	logger boshlog.Logger,
	fs boshsys.FileSystem,
	dir string,
) Manager {
	return NewManager(logger, fs, path.Join(dir, "tasks.json"))
}

type concreteManager struct {
	logger boshlog.Logger

	fs        boshsys.FileSystem
	fsSem     chan func()
	tasksPath string

	// Access to taskInfos must be synchronized via fsSem
	taskInfos map[string]Info
}

func NewManager(logger boshlog.Logger, fs boshsys.FileSystem, tasksPath string) Manager {
	m := &concreteManager{
		logger:    logger,
		fs:        fs,
		fsSem:     make(chan func()),
		tasksPath: tasksPath,
		taskInfos: make(map[string]Info),
	}

	go m.processFsFuncs()

	return m
}

func (m *concreteManager) GetInfos() ([]Info, error) {
	taskInfosChan := make(chan map[string]Info)
	errCh := make(chan error)

	m.fsSem <- func() {
		taskInfos, err := m.readInfos()
		m.taskInfos = taskInfos
		taskInfosChan <- taskInfos
		errCh <- err
	}

	taskInfos := <-taskInfosChan
	err := <-errCh

	if err != nil {
		return nil, err
	}

	var r []Info
	for _, taskInfo := range taskInfos {
		r = append(r, taskInfo)
	}

	return r, nil
}

func (m *concreteManager) AddInfo(taskInfo Info) error {
	errCh := make(chan error)

	m.fsSem <- func() {
		m.taskInfos[taskInfo.TaskID] = taskInfo
		err := m.writeInfos(m.taskInfos)
		errCh <- err
	}
	return <-errCh
}

func (m *concreteManager) RemoveInfo(taskID string) error {
	errCh := make(chan error)

	m.fsSem <- func() {
		delete(m.taskInfos, taskID)
		err := m.writeInfos(m.taskInfos)
		errCh <- err
	}
	return <-errCh
}

func (m *concreteManager) processFsFuncs() {
	defer m.logger.HandlePanic("Task Manager Process Fs Funcs")

	for {
		do := <-m.fsSem
		do()
	}
}

func (m *concreteManager) readInfos() (map[string]Info, error) {
	taskInfos := make(map[string]Info)

	exists := m.fs.FileExists(m.tasksPath)
	if !exists {
		return taskInfos, nil
	}

	tasksJSON, err := m.fs.ReadFile(m.tasksPath)
	if err != nil {
		return nil, bosherr.WrapError(err, "Reading tasks json")
	}

	err = json.Unmarshal(tasksJSON, &taskInfos)
	if err != nil {
		return nil, bosherr.WrapError(err, "Unmarshaling tasks json")
	}

	return taskInfos, nil
}

func (m *concreteManager) writeInfos(taskInfos map[string]Info) error {
	newTasksJSON, err := json.Marshal(taskInfos)
	if err != nil {
		return bosherr.WrapError(err, "Marshalling tasks json")
	}

	err = m.fs.WriteFile(m.tasksPath, newTasksJSON)
	if err != nil {
		return bosherr.WrapError(err, "Writing tasks json")
	}

	return nil
}
