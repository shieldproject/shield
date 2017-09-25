package release

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

type Manager interface {
	Add(Release)
	List() []Release
	Find(name string) (releases Release, found bool)
	DeleteAll() error
}

type manager struct {
	logger boshlog.Logger
	logTag string

	releases []Release
}

func NewManager(
	logger boshlog.Logger,
) Manager {
	return &manager{
		logger:   logger,
		logTag:   "releaseManager",
		releases: []Release{},
	}
}

func (m *manager) Add(release Release) {
	m.logger.Info(m.logTag, "Adding extracted release '%s-%s'", release.Name(), release.Version())
	m.releases = append(m.releases, release)
}

func (m *manager) List() []Release {
	return append([]Release(nil), m.releases...)
}

func (m *manager) Find(name string) (Release, bool) {
	for _, release := range m.releases {
		if release.Name() == name {
			return release, true
		}
	}
	return nil, false
}

func (m *manager) DeleteAll() error {
	for _, release := range m.releases {
		deleteErr := release.Delete()
		if deleteErr != nil {
			return bosherr.Errorf("Failed to delete extracted release '%s': %s", release.Name(), deleteErr.Error())
		}
	}
	m.releases = []Release{}
	return nil
}
