package config_test

import (
	. "github.com/cloudfoundry/bosh-cli/config"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeploymentRepo", func() {
	var (
		repo                   DeploymentRepo
		deploymentStateService DeploymentStateService
		fs                     *fakesys.FakeFileSystem
		fakeUUIDGenerator      *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		fakeUUIDGenerator = fakeuuid.NewFakeGenerator()
		deploymentStateService = NewFileSystemDeploymentStateService(fs, fakeUUIDGenerator, logger, "/fake/path")
		repo = NewDeploymentRepo(deploymentStateService)
	})

	Describe("UpdateCurrent", func() {
		It("updates deployment manifest sha1", func() {
			err := repo.UpdateCurrent("fake-manifest-sha1")
			Expect(err).ToNot(HaveOccurred())

			deploymentState, err := deploymentStateService.Load()
			Expect(err).ToNot(HaveOccurred())

			expectedConfig := DeploymentState{
				DirectorID:         "fake-uuid-0",
				CurrentManifestSHA: "fake-manifest-sha1",
			}
			Expect(deploymentState).To(Equal(expectedConfig))
		})
	})

	Describe("FindCurrent", func() {
		Context("when a current manifest sha1 is set", func() {
			BeforeEach(func() {
				err := repo.UpdateCurrent("fake-manifest-sha1")
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns current manifest sha1", func() {
				record, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())
				Expect(record).To(Equal("fake-manifest-sha1"))
			})
		})

		Context("when a current manifest sha1 is not set", func() {
			It("returns false", func() {
				_, found, err := repo.FindCurrent()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeFalse())
			})
		})
	})
})
