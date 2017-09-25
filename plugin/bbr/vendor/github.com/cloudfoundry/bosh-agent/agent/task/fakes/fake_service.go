package fakes

import (
	boshtask "github.com/cloudfoundry/bosh-agent/agent/task"
)

type FakeService struct {
	StartedTasks        map[string]boshtask.Task
	CreateTaskErr       error
	CreateTaskWithIDErr error
}

func NewFakeService() *FakeService {
	return &FakeService{
		StartedTasks: make(map[string]boshtask.Task),
	}
}

func (s *FakeService) CreateTask(
	taskFunc boshtask.Func,
	cancelFunc boshtask.CancelFunc,
	endFunc boshtask.EndFunc,
) (boshtask.Task, error) {
	if s.CreateTaskErr != nil {
		return boshtask.Task{}, s.CreateTaskErr
	}
	return s.CreateTaskWithID("fake-generated-task-id", taskFunc, cancelFunc, endFunc), nil
}

func (s *FakeService) CreateTaskWithID(
	id string,
	taskFunc boshtask.Func,
	cancelFunc boshtask.CancelFunc,
	endFunc boshtask.EndFunc,
) boshtask.Task {
	return boshtask.Task{
		ID:         id,
		State:      boshtask.StateRunning,
		Func:       taskFunc,
		CancelFunc: cancelFunc,
		EndFunc:    endFunc,
	}
}

func (s *FakeService) StartTask(task boshtask.Task) {
	s.StartedTasks[task.ID] = task
}

func (s *FakeService) FindTaskWithID(id string) (boshtask.Task, bool) {
	task, found := s.StartedTasks[id]
	return task, found
}
