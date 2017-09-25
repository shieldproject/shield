package release_test

import (
	. "github.com/cloudfoundry/bosh-init/release"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	fake_release "github.com/cloudfoundry/bosh-init/release/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

var _ = Describe("Manager", func() {

	var (
		releaseManager Manager

		releaseA = fake_release.New("release-a", "version-a")
		releaseB = fake_release.New("release-b", "version-b")
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)

		releaseManager = NewManager(logger)
	})

	Describe("List", func() {
		It("returns all releases that have been added", func() {
			releaseManager.Add(releaseA)
			releaseManager.Add(releaseB)

			Expect(releaseManager.List()).To(Equal([]Release{releaseA, releaseB}))
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

			Expect(releaseManager.List()).To(BeEmpty())
			Expect(releaseA.Exists()).To(BeFalse())
			Expect(releaseB.Exists()).To(BeFalse())
		})
	})
})
