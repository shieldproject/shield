package fakes

import boshtask "github.com/cloudfoundry/bosh-agent/agent/task"

type FakeManager struct {
	taskIDToTaskInfo map[string]boshtask.Info

	AddInfoErr error
}

func NewFakeManager() *FakeManager {
	return &FakeManager{taskIDToTaskInfo: make(map[string]boshtask.Info)}
}

func (m *FakeManager) GetInfos() ([]boshtask.Info, error) {
	var taskInfos []boshtask.Info
	for _, taskInfo := range m.taskIDToTaskInfo {
		taskInfos = append(taskInfos, taskInfo)
	}
	return taskInfos, nil
}

func (m *FakeManager) AddInfo(taskInfo boshtask.Info) error {
	m.taskIDToTaskInfo[taskInfo.TaskID] = taskInfo
	return m.AddInfoErr
}

func (m *FakeManager) RemoveInfo(taskID string) error {
	delete(m.taskIDToTaskInfo, taskID)
	return nil
}
