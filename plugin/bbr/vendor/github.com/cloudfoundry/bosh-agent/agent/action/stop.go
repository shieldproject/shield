package action

import (
	"errors"

	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type StopAction struct {
	jobSupervisor boshjobsuper.JobSupervisor
}

func NewStop(jobSupervisor boshjobsuper.JobSupervisor) (stop StopAction) {
	stop = StopAction{
		jobSupervisor: jobSupervisor,
	}
	return
}

func (a StopAction) IsAsynchronous(_ ProtocolVersion) bool {
	return true
}

func (a StopAction) IsPersistent() bool {
	return false
}

func (a StopAction) IsLoggable() bool {
	return true
}

func (a StopAction) Run(protocolVersion ProtocolVersion) (value string, err error) {
	if protocolVersion > 2 {
		err = a.jobSupervisor.StopAndWait()
	} else {
		err = a.jobSupervisor.Stop()
	}

	if err != nil {
		err = bosherr.WrapError(err, "Stopping Monitored Services")
		return
	}

	value = "stopped"
	return
}

func (a StopAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a StopAction) Cancel() error {
	return errors.New("not supported")
}
