package action

import (
	"errors"
	"time"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
)

type PrepareNetworkChangeAction struct {
	fs                      boshsys.FileSystem
	settingsService         boshsettings.Service
	waitToKillAgentInterval time.Duration
	agentKiller             Killer
}

func NewPrepareNetworkChange(
	fs boshsys.FileSystem,
	settingsService boshsettings.Service,
	agentKiller Killer,
) (prepareAction PrepareNetworkChangeAction) {
	prepareAction.fs = fs
	prepareAction.settingsService = settingsService
	prepareAction.waitToKillAgentInterval = 1 * time.Second
	prepareAction.agentKiller = agentKiller
	return
}

func (a PrepareNetworkChangeAction) IsAsynchronous() bool {
	return false
}

func (a PrepareNetworkChangeAction) IsPersistent() bool {
	return false
}

func (a PrepareNetworkChangeAction) Run() (interface{}, error) {

	err := a.settingsService.InvalidateSettings()
	if err != nil {
		return nil, bosherr.WrapError(err, "Invalidating settings")
	}

	err = a.fs.RemoveAll("/etc/udev/rules.d/70-persistent-net.rules")
	if err != nil {
		return nil, bosherr.WrapError(err, "Removing network rules file")
	}

	go a.agentKiller.KillAgent(a.waitToKillAgentInterval)

	return "ok", nil
}

func (a PrepareNetworkChangeAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a PrepareNetworkChangeAction) Cancel() error {
	return errors.New("not supported")
}
