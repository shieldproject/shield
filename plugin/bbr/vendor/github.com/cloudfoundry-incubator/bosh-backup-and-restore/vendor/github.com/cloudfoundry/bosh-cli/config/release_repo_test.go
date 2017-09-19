package config_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/config"
	"github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
)

var _ = Describe("ReleaseRepo", func() {
	var (
		repo                   ReleaseRepo
		deploymentStateService DeploymentStateService
		fs                     *fakesys.FakeFileSystem
		fakeUUIDGenerator      *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = &fakeuuid.FakeGenerator{}
		fakeUUIDGenerator.GeneratedUUID = "fake-uuid"
		deploymentStateService = NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		deploymentStateService.Load()
		repo = NewReleaseRepo(deploymentStateService, fakeUUIDGenerator)
	})

	Describe("List", func() {
		Context("when a current release exists", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "fake-guid-a", Name: "fake-name-a", Version: "fake-version-a"},
					ReleaseRecord{ID: "fake-guid-b", Name: "fake-name-b", Version: "fake-version-b"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns existing release", func() {
				records, err := repo.List()
				Expect(err).ToNot(HaveOccurred())
				Expect(records).To(Equal([]ReleaseRecord{
					{
						ID:      "fake-guid-a",
						Name:    "fake-name-a",
						Version: "fake-version-a",
					},
					{
						ID:      "fake-guid-b",
						Name:    "fake-name-b",
						Version: "fake-version-b",
					},
				}))
			})
		})

		Context("when there are no releases recorded", func() {
			It("returns not found", func() {
				records, err := repo.List()
				Expect(err).ToNot(HaveOccurred())
				Expect(records).To(HaveLen(0))
			})
		})

		Context("when the config service fails to load", func() {
			BeforeEach(func() {
				fs.ReadFileError = errors.New("kaboom")
			})

			It("returns an error", func() {
				_, err := repo.List()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Loading existing config"))
			})
		})
	})

	Describe("Update", func() {
		Context("when there are no existing releases", func() {
			It("saves the provided releases to the config file", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).ToNot(HaveOccurred())
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.Releases).To(ConsistOf(
					ReleaseRecord{ID: "fake-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "fake-uuid", Name: "name2", Version: "2"},
				))
			})
		})

		Context("when the existing releases exactly match the provided releases", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "old-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "old-uuid", Name: "name2", Version: "2"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when existing versions differ from the provided release versions", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "old-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "old-uuid", Name: "name2", Version: "3"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the provided releases to the config file", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).ToNot(HaveOccurred())
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.Releases).To(ConsistOf(
					ReleaseRecord{ID: "fake-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "fake-uuid", Name: "name2", Version: "2"},
				))
			})
		})

		Context("when existing names differ from the provided release names", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "old-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "old-uuid", Name: "other-name", Version: "2"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the provided releases to the config file", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).ToNot(HaveOccurred())
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.Releases).To(ConsistOf(
					ReleaseRecord{ID: "fake-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "fake-uuid", Name: "name2", Version: "2"},
				))
			})
		})

		Context("when a release is removed", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "old-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "old-uuid", Name: "name2", Version: "2"},
					ReleaseRecord{ID: "old-uuid", Name: "name3", Version: "3"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the provided releases to the config file", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).ToNot(HaveOccurred())
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.Releases).To(ConsistOf(
					ReleaseRecord{ID: "fake-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "fake-uuid", Name: "name2", Version: "2"},
				))
			})
		})

		Context("when a release is added", func() {
			BeforeEach(func() {
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				conf.Releases = []ReleaseRecord{
					ReleaseRecord{ID: "old-uuid", Name: "name1", Version: "1"},
				}
				err = deploymentStateService.Save(conf)
				Expect(err).ToNot(HaveOccurred())
			})

			It("saves the provided releases to the config file", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).ToNot(HaveOccurred())
				conf, err := deploymentStateService.Load()
				Expect(err).ToNot(HaveOccurred())
				Expect(conf.Releases).To(ConsistOf(
					ReleaseRecord{ID: "fake-uuid", Name: "name1", Version: "1"},
					ReleaseRecord{ID: "fake-uuid", Name: "name2", Version: "2"},
				))
			})
		})

		Context("when the config service fails to save", func() {
			BeforeEach(func() {
				fs.WriteFileError = errors.New("kaboom")
			})

			It("returns an error", func() {
				err := repo.Update([]release.Release{
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name1" },
						VersionStub: func() string { return "1" },
					},
					&fakerel.FakeRelease{
						NameStub:    func() string { return "name2" },
						VersionStub: func() string { return "2" },
					},
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("kaboom"))
			})
		})
	})
})
