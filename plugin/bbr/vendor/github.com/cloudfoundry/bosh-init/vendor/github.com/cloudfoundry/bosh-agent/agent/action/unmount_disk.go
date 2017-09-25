package action

import (
	"errors"
	"fmt"

	boshplatform "github.com/cloudfoundry/bosh-agent/platform"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type UnmountDiskAction struct {
	settingsService boshsettings.Service
	platform        boshplatform.Platform
}

func NewUnmountDisk(
	settingsService boshsettings.Service,
	platform boshplatform.Platform,
) (unmountDisk UnmountDiskAction) {
	unmountDisk.settingsService = settingsService
	unmountDisk.platform = platform
	return
}

func (a UnmountDiskAction) IsAsynchronous() bool {
	return true
}

func (a UnmountDiskAction) IsPersistent() bool {
	return false
}

func (a UnmountDiskAction) Run(diskID string) (value interface{}, err error) {
	settings := a.settingsService.GetSettings()

	diskSettings, found := settings.PersistentDiskSettings(diskID)
	if !found {
		err = bosherr.Errorf("Persistent disk with volume id '%s' could not be found", diskID)
		return
	}

	didUnmount, err := a.platform.UnmountPersistentDisk(diskSettings)
	if err != nil {
		err = bosherr.WrapError(err, "Unmounting persistent disk")
		return
	}

	msg := fmt.Sprintf("Partition of %+v is not mounted", diskSettings)

	if didUnmount {
		msg = fmt.Sprintf("Unmounted partition of %+v", diskSettings)
	}

	type valueType struct {
		Message string `json:"message"`
	}

	value = valueType{Message: msg}
	return
}

func (a UnmountDiskAction) Resume() (interface{}, error) {
	return nil, errors.New("not supported")
}

func (a UnmountDiskAction) Cancel() error {
	return errors.New("not supported")
}
