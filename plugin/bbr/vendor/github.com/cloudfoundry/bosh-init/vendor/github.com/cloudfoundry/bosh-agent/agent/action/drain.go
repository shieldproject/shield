package action

import (
	"errors"

	boshas "github.com/cloudfoundry/bosh-agent/agent/applier/applyspec"
	boshscript "github.com/cloudfoundry/bosh-agent/agent/script"
	boshdrain "github.com/cloudfoundry/bosh-agent/agent/script/drain"
	boshjobsuper "github.com/cloudfoundry/bosh-agent/jobsupervisor"
	boshnotif "github.com/cloudfoundry/bosh-agent/notification"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type DrainAction struct {
	jobScriptProvider boshscript.JobScriptProvider
	notifier          boshnotif.Notifier
	specService       boshas.V1Service
	jobSupervisor     boshjobsuper.JobSupervisor

	logTag   string
	logger   boshlog.Logger
	cancelCh chan struct{}
}

type DrainType string

const (
	DrainTypeUpdate   DrainType = "update"
	DrainTypeStatus   DrainType = "status"
	DrainTypeShutdown DrainType = "shutdown"
)

func NewDrain(
	notifier boshnotif.Notifier,
	specService boshas.V1Service,
	jobScriptProvider boshscript.JobScriptProvider,
	jobSupervisor boshjobsuper.JobSupervisor,
	logger boshlog.Logger,
) DrainAction {
	return DrainAction{
		notifier:          notifier,
		specService:       specService,
		jobScriptProvider: jobScriptProvider,
		jobSupervisor:     jobSupervisor,

		logTag:   "Drain Action",
		logger:   logger,
		cancelCh: make(chan struct{}, 1),
	}
}

func (a DrainAction) IsAsynchronous() bool {
	return true
}

func (a DrainAction) IsPersistent() bool {
	return false
}

func (a DrainAction) Run(drainType DrainType, newSpecs ...boshas.V1ApplySpec) (int, error) {
	currentSpec, err := a.specService.Get()
	if err != nil {
		return 0, bosherr.WrapError(err, "Getting current spec")
	}

	params, err := a.determineParams(drainType, currentSpec, newSpecs)
	if err != nil {
		return 0, err
	}

	a.logger.Debug(a.logTag, "Unmonitoring")

	err = a.jobSupervisor.Unmonitor()
	if err != nil {
		return 0, bosherr.WrapError(err, "Unmonitoring services")
	}

	var scripts []boshscript.Script

	for _, job := range currentSpec.Jobs() {
		script := a.jobScriptProvider.NewDrainScript(job.BundleName(), params)
		scripts = append(scripts, script)
	}

	script := a.jobScriptProvider.NewParallelScript("drain", scripts)

	resultsCh := make(chan error, 1)
	go func() { resultsCh <- script.Run() }()
	select {
	case result := <-resultsCh:
		a.logger.Debug(a.logTag, "Got a result")
		return 0, result
	case <-a.cancelCh:
		a.logger.Debug(a.logTag, "Got a cancel request")
		return 0, script.Cancel()
	}
}

func (a DrainAction) determineParams(drainType DrainType, currentSpec boshas.V1ApplySpec, newSpecs []boshas.V1ApplySpec) (boshdrain.ScriptParams, error) {
	var newSpec *boshas.V1ApplySpec
	var params boshdrain.ScriptParams

	if len(newSpecs) > 0 {
		newSpec = &newSpecs[0]
	}

	switch drainType {
	case DrainTypeStatus:
		// Status was used in the past when dynamic drain was implemented in the Director.
		// Now that we implement it in the agent, we should never get a call for this type.
		return params, bosherr.Error("Unexpected call with drain type 'status'")

	case DrainTypeUpdate:
		if newSpec == nil {
			return params, bosherr.Error("Drain update requires new spec")
		}

		params = boshdrain.NewUpdateParams(currentSpec, *newSpec)

	case DrainTypeShutdown:
		err := a.notifier.NotifyShutdown()
		if err != nil {
			return params, bosherr.WrapError(err, "Notifying shutdown")
		}

		params = boshdrain.NewShutdownParams(currentSpec, newSpec)
	}

	return params, nil
}

func (a DrainAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a DrainAction) Cancel() error {
	a.logger.Debug(a.logTag, "Cancelling drain action")
	select {
	case a.cancelCh <- struct{}{}:
	default:
	}
	return nil
}
