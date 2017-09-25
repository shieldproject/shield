package jobsupervisor

import (

	//boshmonit "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit"
	//boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	//boshlog "github.com/cloudfoundry/bosh-utils/logger"
	//boshsys "github.com/cloudfoundry/bosh-utils/system"
	"encoding/json"
	"github.com/cloudfoundry/bosh-agent/settings/directories"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/cloudfoundry/bosh-utils/system"
	"path/filepath"
)

const wrapperJobSupervisorLogTag = "wrapperJobSupervisor"

type wrapperJobSupervisor struct {
	delegate      JobSupervisor
	fs            system.FileSystem
	dirProvider   directories.Provider
	logger        boshlog.Logger
	pollRunning   bool
	pollUnmonitor bool
}

func NewWrapperJobSupervisor(delegate JobSupervisor, fs system.FileSystem, dirProvider directories.Provider, logger boshlog.Logger) JobSupervisor {
	return &wrapperJobSupervisor{
		delegate:    delegate,
		fs:          fs,
		dirProvider: dirProvider,
		logger:      logger,
	}
}

func (w *wrapperJobSupervisor) Reload() error {
	return w.delegate.Reload()
}
func (w *wrapperJobSupervisor) Start() error {

	err := w.delegate.Start()
	w.HealthRecorder(w.delegate.Status())

	return err
}
func (w *wrapperJobSupervisor) Stop() error {
	err := w.delegate.Stop()
	w.HealthRecorder(w.delegate.Status())

	return err
}
func (w *wrapperJobSupervisor) StopAndWait() error {
	return w.delegate.StopAndWait()
}
func (w *wrapperJobSupervisor) Unmonitor() error {
	err := w.delegate.Unmonitor()
	if err != nil {
		return err
	}

	w.HealthRecorder(w.delegate.Status())
	return err
}
func (w *wrapperJobSupervisor) Status() string {
	return w.delegate.Status()
}
func (w *wrapperJobSupervisor) Processes() ([]Process, error) {
	return w.delegate.Processes()
}
func (w *wrapperJobSupervisor) AddJob(jobName string, jobIndex int, configPath string) error {
	return w.delegate.AddJob(jobName, jobIndex, configPath)
}
func (w *wrapperJobSupervisor) RemoveAllJobs() error {
	return w.delegate.RemoveAllJobs()
}
func (w *wrapperJobSupervisor) MonitorJobFailures(handler JobFailureHandler) error {
	return w.delegate.MonitorJobFailures(handler)
}

func (w *wrapperJobSupervisor) HealthRecorder(status string) {

	healthRaw, err := json.Marshal(Health{State: status})
	if err != nil {
		w.logger.Error(wrapperJobSupervisorLogTag, err.Error())
	}
	err = w.fs.WriteFile(filepath.Join(w.dirProvider.InstanceDir(), "health.json"), healthRaw)
	if err != nil {
		w.logger.Error(wrapperJobSupervisorLogTag, err.Error())
	}
}
