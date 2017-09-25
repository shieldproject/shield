package installation_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/installation"
	boshrel "github.com/cloudfoundry/bosh-cli/release"
	fakerel "github.com/cloudfoundry/bosh-cli/release/releasefakes"
)

var _ = Describe("ReleaseManager", func() {
	var (
		releaseA       *fakerel.FakeRelease
		releaseB       *fakerel.FakeRelease
		releaseManager ReleaseManager
	)

	BeforeEach(func() {
		releaseA = &fakerel.FakeRelease{
			NameStub:    func() string { return "release-a" },
			VersionStub: func() string { return "version-a" },
		}
		releaseB = &fakerel.FakeRelease{
			NameStub:    func() string { return "release-b" },
			VersionStub: func() string { return "version-b" },
		}
		releaseManager = NewReleaseManager(boshlog.NewLogger(boshlog.LevelNone))
	})

	Describe("List", func() {
		It("returns all releases that have been added", func() {
			releaseManager.Add(releaseA)
			releaseManager.Add(releaseB)
			Expect(releaseManager.List()).To(Equal([]boshrel.Release{releaseA, releaseB}))
		})
	})

	Describe("Find", func() {
		It("returns false when no releases have been added", func() {
			_, found := releaseManager.Find("release-a")
			Expect(found).To(BeFalse())
		})

		Context("when releases have been added", func() {
			It("returns true and the release with the requested name", func() {
				releaseManager.Add(releaseA)
				releaseManager.Add(releaseB)

				releaseAFound, found := releaseManager.Find("release-a")
				Expect(found).To(BeTrue())
				Expect(releaseAFound).To(Equal(releaseA))

				releaseBFound, found := releaseManager.Find("release-b")
				Expect(found).To(BeTrue())
				Expect(releaseBFound).To(Equal(releaseB))
			})

			It("returns false when the requested release has not been added", func() {
				releaseManager.Add(releaseA)

				_, found := releaseManager.Find("release-c")
				Expect(found).To(BeFalse())
			})
		})
	})

	Describe("DeleteAll", func() {
		It("deletes all added releases", func() {
			releaseManager.Add(releaseA)
			releaseManager.Add(releaseB)

			err := releaseManager.DeleteAll()
			Expect(err).ToNot(HaveOccurred())

			Expect(releaseA.CleanUpCallCount()).To(Equal(1))
			Expect(releaseB.CleanUpCallCount()).To(Equal(1))
			Expect(releaseManager.List()).To(BeEmpty())
		})
	})
})
