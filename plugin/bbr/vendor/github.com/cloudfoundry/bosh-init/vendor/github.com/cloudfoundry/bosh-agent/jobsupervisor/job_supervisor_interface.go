package jobsupervisor

import (
	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
)

type Process struct {
	Name   string       `json:"name"`
	State  string       `json:"state"`
	Uptime UptimeVitals `json:"uptime,omitempty"`
	Memory MemoryVitals `json:"mem,omitempty"`
	CPU    CPUVitals    `json:"cpu,omitempty"`
}

type UptimeVitals struct {
	Secs int `json:"secs,omitempty"`
}

type MemoryVitals struct {
	Kb      int     `json:"kb,omitempty"`
	Percent float64 `json:"percent"`
}

type CPUVitals struct {
	Total float64 `json:"total"`
}

type JobFailureHandler func(boshalert.MonitAlert) error

type JobSupervisor interface {
	Reload() error

	// Actions taken on all services
	Start() error
	Stop() error

	// Start and Stop should still function after Unmonitor.
	// Calling Start after Unmonitor should re-monitor all jobs.
	// Calling Stop after Unmonitor should not re-monitor all jobs.
	// (Monit complies to above requirements.)
	Unmonitor() error

	Status() string
	Processes() ([]Process, error)
	// Job management
	AddJob(jobName string, jobIndex int, configPath string) error
	RemoveAllJobs() error

	MonitorJobFailures(handler JobFailureHandler) error
}
