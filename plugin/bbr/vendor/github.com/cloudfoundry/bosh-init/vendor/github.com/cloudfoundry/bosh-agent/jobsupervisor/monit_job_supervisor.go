package jobsupervisor

import (
	"fmt"
	"path"
	"time"

	"github.com/pivotal/go-smtpd/smtpd"

	boshalert "github.com/cloudfoundry/bosh-agent/agent/alert"
	boshmonit "github.com/cloudfoundry/bosh-agent/jobsupervisor/monit"
	boshdir "github.com/cloudfoundry/bosh-agent/settings/directories"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

const monitJobSupervisorLogTag = "monitJobSupervisor"

type monitJobSupervisor struct {
	fs          boshsys.FileSystem
	runner      boshsys.CmdRunner
	client      boshmonit.Client
	logger      boshlog.Logger
	dirProvider boshdir.Provider

	jobFailuresServerPort int

	reloadOptions MonitReloadOptions
}

type MonitReloadOptions struct {
	// Number of times `monit reload` will be executed
	MaxTries int

	// Number of times monit incarnation will be checked
	// for difference after executing `monit reload`
	MaxCheckTries int

	// Length of time between checking for incarnation difference
	DelayBetweenCheckTries time.Duration
}

func NewMonitJobSupervisor(
	fs boshsys.FileSystem,
	runner boshsys.CmdRunner,
	client boshmonit.Client,
	logger boshlog.Logger,
	dirProvider boshdir.Provider,
	jobFailuresServerPort int,
	reloadOptions MonitReloadOptions,
) JobSupervisor {
	return monitJobSupervisor{
		fs:          fs,
		runner:      runner,
		client:      client,
		logger:      logger,
		dirProvider: dirProvider,

		jobFailuresServerPort: jobFailuresServerPort,

		reloadOptions: reloadOptions,
	}
}

func (m monitJobSupervisor) Reload() error {
	var currentIncarnation int

	oldIncarnation, err := m.getIncarnation()
	if err != nil {
		return bosherr.WrapError(err, "Getting monit incarnation")
	}

	// Monit process could be started in the same second as `monit reload` runs
	// so it's ideal for MaxCheckTries * DelayBetweenCheckTries to be greater than 1 sec
	// because monit incarnation id is just a timestamp with 1 sec resolution.
	for reloadI := 0; reloadI < m.reloadOptions.MaxTries; reloadI++ {
		// Exit code or output cannot be trusted
		_, _, _, err := m.runner.RunCommand("monit", "reload")
		if err != nil {
			m.logger.Error(monitJobSupervisorLogTag, "Failed to reload monit %s", err.Error())
		}

		for checkI := 0; checkI < m.reloadOptions.MaxCheckTries; checkI++ {
			currentIncarnation, err = m.getIncarnation()
			if err != nil {
				return bosherr.WrapError(err, "Getting monit incarnation")
			}

			// Incarnation id can decrease or increase because
			// monit uses time(...) and system time can be changed
			if oldIncarnation != currentIncarnation {
				return nil
			}

			m.logger.Debug(
				monitJobSupervisorLogTag,
				"Waiting for monit to reload: before=%d after=%d",
				oldIncarnation, currentIncarnation,
			)

			time.Sleep(m.reloadOptions.DelayBetweenCheckTries)
		}
	}

	return bosherr.Errorf(
		"Failed to reload monit: before=%d after=%d",
		oldIncarnation, currentIncarnation,
	)
}

func (m monitJobSupervisor) Start() error {
	services, err := m.client.ServicesInGroup("vcap")
	if err != nil {
		return bosherr.WrapError(err, "Getting vcap services")
	}

	for _, service := range services {
		m.logger.Debug(monitJobSupervisorLogTag, "Starting service %s", service)
		err = m.client.StartService(service)
		if err != nil {
			return bosherr.WrapErrorf(err, "Starting service %s", service)
		}
	}

	err = m.fs.RemoveAll(m.stoppedFilePath())
	if err != nil {
		return bosherr.WrapError(err, "Removing stopped File")
	}

	return nil
}

func (m monitJobSupervisor) Stop() error {
	services, err := m.client.ServicesInGroup("vcap")
	if err != nil {
		return bosherr.WrapError(err, "Getting vcap services")
	}

	for _, service := range services {
		m.logger.Debug(monitJobSupervisorLogTag, "Stopping service %s", service)
		err = m.client.StopService(service)
		if err != nil {
			return bosherr.WrapErrorf(err, "Stopping service %s", service)
		}
	}

	err = m.fs.WriteFileString(m.stoppedFilePath(), "")
	if err != nil {
		return bosherr.WrapError(err, "Creating stopped File")
	}

	return nil
}

func (m monitJobSupervisor) Unmonitor() error {
	services, err := m.client.ServicesInGroup("vcap")
	if err != nil {
		return bosherr.WrapError(err, "Getting vcap services")
	}

	for _, service := range services {
		m.logger.Debug(monitJobSupervisorLogTag, "Unmonitoring service %s", service)
		err := m.client.UnmonitorService(service)
		if err != nil {
			return bosherr.WrapErrorf(err, "Unmonitoring service %s", service)
		}
	}

	return nil
}

func (m monitJobSupervisor) Status() (status string) {
	status = "running"

	m.logger.Debug(monitJobSupervisorLogTag, "Getting monit status")
	monitStatus, err := m.client.Status()
	if err != nil {
		status = "unknown"
		return
	}

	if m.fs.FileExists(m.stoppedFilePath()) {
		status = "stopped"

	} else {
		services := monitStatus.ServicesInGroup("vcap")
		for _, service := range services {
			if service.Status == "starting" {
				return "starting"
			}
			if !service.Monitored || service.Status != "running" {
				status = "failing"
			}
		}
	}

	return
}

func (m monitJobSupervisor) Processes() (processes []Process, err error) {
	processes = []Process{}

	monitStatus, err := m.client.Status()
	if err != nil {
		return processes, bosherr.WrapError(err, "Getting service status")
	}

	for _, service := range monitStatus.ServicesInGroup("vcap") {
		process := Process{
			Name:  service.Name,
			State: service.Status,
			Uptime: UptimeVitals{
				Secs: service.Uptime,
			},
			Memory: MemoryVitals{
				Kb:      service.MemoryKilobytesTotal,
				Percent: service.MemoryPercentTotal,
			},
			CPU: CPUVitals{
				Total: service.CPUPercentTotal,
			},
		}
		processes = append(processes, process)
	}

	return
}

func (m monitJobSupervisor) getIncarnation() (int, error) {
	monitStatus, err := m.client.Status()
	if err != nil {
		return -1, err
	}

	return monitStatus.GetIncarnation()
}

func (m monitJobSupervisor) AddJob(jobName string, jobIndex int, configPath string) error {
	targetFilename := fmt.Sprintf("%04d_%s.monitrc", jobIndex, jobName)
	targetConfigPath := path.Join(m.dirProvider.MonitJobsDir(), targetFilename)

	configContent, err := m.fs.ReadFile(configPath)
	if err != nil {
		return bosherr.WrapError(err, "Reading job config from file")
	}

	err = m.fs.WriteFile(targetConfigPath, configContent)
	if err != nil {
		return bosherr.WrapError(err, "Writing to job config file")
	}

	return nil
}

func (m monitJobSupervisor) RemoveAllJobs() error {
	return m.fs.RemoveAll(m.dirProvider.MonitJobsDir())
}

func (m monitJobSupervisor) MonitorJobFailures(handler JobFailureHandler) (err error) {
	alertHandler := func(smtpd.Connection, smtpd.MailAddress) (env smtpd.Envelope, err error) {
		env = &alertEnvelope{
			new(smtpd.BasicEnvelope),
			handler,
			new(boshalert.MonitAlert),
		}
		return
	}

	serv := &smtpd.Server{
		Addr:      fmt.Sprintf("127.0.0.1:%d", m.jobFailuresServerPort),
		OnNewMail: alertHandler,
	}

	err = serv.ListenAndServe()
	if err != nil {
		err = bosherr.WrapError(err, "Listen for SMTP")
	}
	return
}

func (m monitJobSupervisor) stoppedFilePath() string {
	return path.Join(m.dirProvider.MonitDir(), "stopped")
}
