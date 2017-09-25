package action

import (
	"encoding/json"
	"errors"
	"fmt"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"

	boshplat "github.com/cloudfoundry/bosh-agent/platform"
	boshblob "github.com/cloudfoundry/bosh-utils/blobstore"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type SyncDNS struct {
	blobstore       boshblob.Blobstore
	settingsService boshsettings.Service
	platform        boshplat.Platform
	logger          boshlog.Logger
	logTag          string
}

func NewSyncDNS(blobstore boshblob.Blobstore, settingsService boshsettings.Service, platform boshplat.Platform, logger boshlog.Logger) SyncDNS {
	return SyncDNS{
		blobstore:       blobstore,
		settingsService: settingsService,
		platform:        platform,
		logger:          logger,
		logTag:          "Sync DNS action",
	}
}

func (a SyncDNS) IsAsynchronous() bool {
	return false
}

func (a SyncDNS) IsPersistent() bool {
	return false
}

func (a SyncDNS) Resume() (interface{}, error) {
	return nil, errors.New("Not supported")
}

func (a SyncDNS) Cancel() error {
	return errors.New("Not supported")
}

func (a SyncDNS) Run(blobID, sha1 string) (string, error) {
	fileName, err := a.blobstore.Get(blobID, sha1)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Getting %s from blobstore", blobID)
	}

	fs := a.platform.GetFs()

	contents, err := fs.ReadFile(fileName)
	if err != nil {
		return "", bosherr.WrapErrorf(err, "Reading fileName %s from blobstore", fileName)
	}

	err = fs.RemoveAll(fileName)
	if err != nil {
		a.logger.Info(a.logTag, fmt.Sprintf("Failed to remove dns blob file at path '%s'", fileName))
	}

	dnsRecords := boshsettings.DNSRecords{}
	err = json.Unmarshal(contents, &dnsRecords)
	if err != nil {
		return "", bosherr.WrapError(err, "Unmarshalling DNS records")
	}

	err = a.platform.SaveDNSRecords(dnsRecords, a.settingsService.GetSettings().AgentID)
	if err != nil {
		return "", bosherr.WrapError(err, "Saving DNS records in platform")
	}

	return "synced", nil
}
