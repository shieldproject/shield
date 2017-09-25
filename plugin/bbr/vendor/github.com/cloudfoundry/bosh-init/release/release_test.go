package release_test

import (
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-init/release"
)

var _ = Describe("Release", func() {
	var (
		release     Release
		expectedJob bireljob.Job
		fakeFS      *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		expectedJob = bireljob.Job{
			Name: "fake-job-name",
		}
		fakeFS = fakesys.NewFakeFileSystem()
		release = NewRelease(
			"fake-release-name",
			"fake-release-version",
			[]bireljob.Job{expectedJob},
			[]*birelpkg.Package{},
			"fake-extracted-path",
			fakeFS,
			false,
		)
	})

	Describe("FindJobByName", func() {
		Context("when the job exists", func() {
			It("returns the job and true", func() {
				actualJob, ok := release.FindJobByName("fake-job-name")
				Expect(actualJob).To(Equal(expectedJob))
				Expect(ok).To(BeTrue())
			})
		})

		Context("when the job does not exist", func() {
			It("returns nil and false", func() {
				_, ok := release.FindJobByName("fake-non-existent-job")
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			fakeFS.WriteFileString("fake-extracted-path", "")
		})

		It("deletes the extracted release path", func() {
			Expect(fakeFS.FileExists("fake-extracted-path")).To(BeTrue())
			err := release.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(fakeFS.FileExists("fake-extracted-path")).To(BeFalse())
		})
	})

	Describe("Exists", func() {
		BeforeEach(func() {
			fakeFS.WriteFileString("fake-extracted-path", "")
		})

		It("returns false after deletion", func() {
			Expect(release.Exists()).To(BeTrue())
			err := release.Delete()
			Expect(err).ToNot(HaveOccurred())
			Expect(release.Exists()).To(BeFalse())
		})
	})
})
