package platform

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type monitRetryable struct {
	cmdRunner boshsys.CmdRunner
}

func NewMonitRetryable(cmdRunner boshsys.CmdRunner) boshretry.Retryable {
	return &monitRetryable{
		cmdRunner: cmdRunner,
	}
}

func (r *monitRetryable) Attempt() (bool, error) {
	_, _, _, err := r.cmdRunner.RunCommand("sv", "start", "monit")
	if err != nil {
		return true, bosherr.WrapError(err, "Starting monit")
	}

	return false, nil
}
