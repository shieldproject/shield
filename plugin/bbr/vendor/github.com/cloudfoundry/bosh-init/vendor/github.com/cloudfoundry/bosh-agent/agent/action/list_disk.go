package action

import (
	"errors"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type ListDiskAction struct {
	settingsService boshsettings.Service
	platform        boshplatform.Platform
	logger          boshlog.Logger
}

func NewListDisk(
	settingsService boshsettings.Service,
	platform boshplatform.Platform,
	logger boshlog.Logger,
) (action ListDiskAction) {
	action.settingsService = settingsService
	action.platform = platform
	action.logger = logger
	return
}

func (a ListDiskAction) IsAsynchronous() bool {
	return false
}

func (a ListDiskAction) IsPersistent() bool {
	return false
}

func (a ListDiskAction) Run() (interface{}, error) {
	settings := a.settingsService.GetSettings()
	diskIDs := []string{}

	for diskID := range settings.Disks.Persistent {
		var isMounted bool

		diskSettings, _ := settings.PersistentDiskSettings(diskID)
		isMounted, err := a.platform.IsPersistentDiskMounted(diskSettings)
		if err != nil {
			return nil, bosherr.WrapErrorf(err, "Checking whether device %+v is mounted", diskSettings)
		}

		if isMounted {
			diskIDs = append(diskIDs, diskID)
		} else {
			a.logger.Debug("list-disk-action", "Volume '%s' not mounted", diskID)
		}
	}

	return diskIDs, nil
}

func (a ListDiskAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a ListDiskAction) Cancel() error {
	return errors.New("not supported")
}
