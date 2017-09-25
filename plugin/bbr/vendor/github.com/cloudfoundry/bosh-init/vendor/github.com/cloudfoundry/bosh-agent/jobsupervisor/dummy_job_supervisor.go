package jobsupervisor

type dummyJobSupervisor struct {
	status    string
	processes []Process
}

func NewDummyJobSupervisor() JobSupervisor {
	return &dummyJobSupervisor{status: "unknown"}
}

func (s *dummyJobSupervisor) Reload() error {
	return nil
}

func (s *dummyJobSupervisor) Start() error {
	s.status = "running"
	s.processes = []Process{}
	return nil
}

func (s *dummyJobSupervisor) Stop() error {
	s.status = "stopped"
	return nil
}

func (s *dummyJobSupervisor) Unmonitor() error {
	return nil
}

func (s *dummyJobSupervisor) Status() (status string) {
	return s.status
}

func (s *dummyJobSupervisor) Processes() ([]Process, error) {
	return s.processes, nil
}

func (s *dummyJobSupervisor) AddJob(jobName string, jobIndex int, configPath string) error {
	return nil
}

func (s *dummyJobSupervisor) RemoveAllJobs() error {
	return nil
}

func (s *dummyJobSupervisor) MonitorJobFailures(handler JobFailureHandler) error {
	return nil
}
