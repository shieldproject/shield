package fakes

import (
	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
)

type FakeJobSupervisor struct {
	Reloaded  bool
	ReloadErr error

	AddJobArgs []AddJobArgs

	RemovedAllJobs    bool
	RemovedAllJobsErr error

	Started  bool
	StartErr error

	Stopped bool
	StopErr error

	Unmonitored  bool
	UnmonitorErr error

	StatusStatus    string
	ProcessesStatus []boshjobsuper.Process
	ProcessesError  error

	JobFailureAlert *boshalert.MonitAlert
}

type AddJobArgs struct {
	Name       string
	Index      int
	ConfigPath string
}

func NewFakeJobSupervisor() *FakeJobSupervisor {
	return &FakeJobSupervisor{}
}

func (m *FakeJobSupervisor) Reload() error {
	m.Reloaded = true
	return m.ReloadErr
}

func (m *FakeJobSupervisor) AddJob(jobName string, jobIndex int, configPath string) error {
	args := AddJobArgs{
		Name:       jobName,
		Index:      jobIndex,
		ConfigPath: configPath,
	}
	m.AddJobArgs = append(m.AddJobArgs, args)
	return nil
}

func (m *FakeJobSupervisor) RemoveAllJobs() error {
	m.RemovedAllJobs = true
	return m.RemovedAllJobsErr
}

func (m *FakeJobSupervisor) Start() error {
	m.Started = true
	return m.StartErr
}

func (m *FakeJobSupervisor) Stop() error {
	m.Stopped = true
	return m.StopErr
}

func (m *FakeJobSupervisor) Unmonitor() error {
	m.Unmonitored = true
	return m.UnmonitorErr
}

func (m *FakeJobSupervisor) Status() string {
	return m.StatusStatus
}

func (m *FakeJobSupervisor) Processes() ([]boshjobsuper.Process, error) {
	return m.ProcessesStatus, m.ProcessesError
}

func (m *FakeJobSupervisor) MonitorJobFailures(handler boshjobsuper.JobFailureHandler) error {
	if m.JobFailureAlert != nil {
		return handler(*m.JobFailureAlert)
	}
	return nil
}
