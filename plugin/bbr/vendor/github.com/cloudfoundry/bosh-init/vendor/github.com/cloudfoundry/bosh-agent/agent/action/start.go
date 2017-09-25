package action

import (
	"errors"

	boshappl "github.com/cloudfoundry/bosh-agent/agent/applier"
	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type StartAction struct {
	jobSupervisor boshjobsuper.JobSupervisor
	applier       boshappl.Applier
	specService   boshas.V1Service
}

func NewStart(jobSupervisor boshjobsuper.JobSupervisor, applier boshappl.Applier, specService boshas.V1Service) (start StartAction) {
	start = StartAction{
		jobSupervisor: jobSupervisor,
		specService:   specService,
		applier:       applier,
	}
	return
}

func (a StartAction) IsAsynchronous() bool {
	return false
}

func (a StartAction) IsPersistent() bool {
	return false
}

func (a StartAction) Run() (value string, err error) {
	desiredApplySpec, err := a.specService.Get()
	if err != nil {
		err = bosherr.WrapError(err, "Getting apply spec")
		return
	}

	err = a.applier.ConfigureJobs(desiredApplySpec)
	if err != nil {
		err = bosherr.WrapErrorf(err, "Configuring jobs")
		return
	}

	err = a.jobSupervisor.Start()
	if err != nil {
		err = bosherr.WrapError(err, "Starting Monitored Services")
		return
	}

	value = "started"
	return
}

func (a StartAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a StartAction) Cancel() error {
	return errors.New("not supported")
}
