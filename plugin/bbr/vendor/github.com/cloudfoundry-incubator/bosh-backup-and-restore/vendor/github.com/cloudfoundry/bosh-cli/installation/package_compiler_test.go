package installation_test

import (
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/installation"
	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	bistatejob "github.com/cloudfoundry/bosh-cli/state/job"
	mock_state_job "github.com/cloudfoundry/bosh-cli/state/job/mocks"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("PackageCompiler", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		mockDependencyCompiler *mock_state_job.MockDependencyCompiler

		fs       *fakeboshsys.FakeFileSystem
		compiler installation.PackageCompiler

		releaseJob    bireljob.Job
		releaseJobs   []bireljob.Job
		stage         *fakebiui.FakeStage
		expectCompile *gomock.Call
	)

	BeforeEach(func() {
		mockDependencyCompiler = mock_state_job.NewMockDependencyCompiler(mockCtrl)
		fs = fakeboshsys.NewFakeFileSystem()
		stage = fakebiui.NewFakeStage()

		job := bireljob.NewJob(NewResource("cpi", "fake-release-job-fingerprint", nil))
		releaseJob = *job
	})

	JustBeforeEach(func() {
		compiler = installation.NewPackageCompiler(mockDependencyCompiler, fs)

		releaseJobs = []bireljob.Job{releaseJob}
		compiledPackageRefs := []bistatejob.CompiledPackageRef{
			{
				Name:        "pkg1-name",
				Version:     "pkg1-fp",
				BlobstoreID: "fake-compiled-package-blobstore-id-1",
				SHA1:        "fake-compiled-package-sha1-1",
			},
			{
				Name:        "pkg2-name",
				Version:     "pkg2-fp",
				BlobstoreID: "fake-compiled-package-blobstore-id-2",
				SHA1:        "fake-compiled-package-sha1-2",
			},
		}
		expectCompile = mockDependencyCompiler.EXPECT().Compile(releaseJobs, stage).Return(compiledPackageRefs, nil).AnyTimes()
	})

	Describe("From", func() {
		It("returns compiled packages and release jobs", func() {
			packages, err := compiler.For(releaseJobs, stage)
			Expect(err).ToNot(HaveOccurred())

			Expect(packages).To(ConsistOf([]installation.CompiledPackageRef{
				{
					Name:        "pkg1-name",
					Version:     "pkg1-fp",
					BlobstoreID: "fake-compiled-package-blobstore-id-1",
					SHA1:        "fake-compiled-package-sha1-1",
				},
				{
					Name:        "pkg2-name",
					Version:     "pkg2-fp",
					BlobstoreID: "fake-compiled-package-blobstore-id-2",
					SHA1:        "fake-compiled-package-sha1-2",
				},
			}))
		})

		Context("when package compilation fails", func() {
			JustBeforeEach(func() {
				expectCompile.Return([]bistatejob.CompiledPackageRef{}, bosherr.Error("fake-compile-package-2-error")).Times(1)
			})

			It("returns an error", func() {
				_, err := compiler.For(releaseJobs, stage)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-compile-package-2-error"))
			})
		})
	})
})
