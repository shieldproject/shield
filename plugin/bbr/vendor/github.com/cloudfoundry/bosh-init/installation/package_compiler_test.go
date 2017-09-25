package installation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-init/installation"

	mock_state_job "github.com/cloudfoundry/bosh-init/state/job/mocks"
	"github.com/golang/mock/gomock"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatejob "github.com/cloudfoundry/bosh-init/state/job"
	bosherr "github.com/cloudfoundry/bosh-utils/errors"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
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

		fakeFS   *fakeboshsys.FakeFileSystem
		compiler installation.PackageCompiler

		releaseJob  bireljob.Job
		releaseJobs []bireljob.Job

		fakeStage *fakebiui.FakeStage

		releasePackage1 *birelpkg.Package
		releasePackage2 *birelpkg.Package

		expectCompile *gomock.Call
	)

	BeforeEach(func() {
		mockDependencyCompiler = mock_state_job.NewMockDependencyCompiler(mockCtrl)

		fakeFS = fakeboshsys.NewFakeFileSystem()

		fakeStage = fakebiui.NewFakeStage()

		releasePackage1 = &birelpkg.Package{
			Name:          "fake-release-package-name-1",
			Fingerprint:   "fake-release-package-fingerprint-1",
			SHA1:          "fake-release-package-sha1-1",
			Dependencies:  []*birelpkg.Package{},
			ExtractedPath: "/extracted-release-path/extracted_packages/fake-release-package-name-1",
		}

		releasePackage2 = &birelpkg.Package{
			Name:          "fake-release-package-name-2",
			Fingerprint:   "fake-release-package-fingerprint-2",
			SHA1:          "fake-release-package-sha1-2",
			Dependencies:  []*birelpkg.Package{releasePackage1},
			ExtractedPath: "/extracted-release-path/extracted_packages/fake-release-package-name-2",
		}

		releaseJob = bireljob.Job{
			Name:          "cpi",
			Fingerprint:   "fake-release-job-fingerprint",
			SHA1:          "fake-release-job-sha1",
			ExtractedPath: "/extracted-release-path/extracted_jobs/cpi",
			Templates: map[string]string{
				"cpi.erb":     "bin/cpi",
				"cpi.yml.erb": "config/cpi.yml",
			},
			PackageNames: []string{releasePackage2.Name},
			Packages:     []*birelpkg.Package{releasePackage2},
			Properties:   map[string]bireljob.PropertyDefinition{},
		}
	})

	JustBeforeEach(func() {
		compiler = installation.NewPackageCompiler(
			mockDependencyCompiler,
			fakeFS,
		)

		releaseJobs = []bireljob.Job{releaseJob}
		compiledPackageRefs := []bistatejob.CompiledPackageRef{
			{
				Name:        "fake-release-package-name-1",
				Version:     "fake-release-package-fingerprint-1",
				BlobstoreID: "fake-compiled-package-blobstore-id-1",
				SHA1:        "fake-compiled-package-sha1-1",
			},
			{
				Name:        "fake-release-package-name-2",
				Version:     "fake-release-package-fingerprint-2",
				BlobstoreID: "fake-compiled-package-blobstore-id-2",
				SHA1:        "fake-compiled-package-sha1-2",
			},
		}
		expectCompile = mockDependencyCompiler.EXPECT().Compile(releaseJobs, fakeStage).Return(compiledPackageRefs, nil).AnyTimes()
	})

	Describe("From", func() {
		It("returns compiled packages and release jobs", func() {
			packages, err := compiler.For(releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(packages).To(ConsistOf([]installation.CompiledPackageRef{
				{
					Name:        "fake-release-package-name-1",
					Version:     "fake-release-package-fingerprint-1",
					BlobstoreID: "fake-compiled-package-blobstore-id-1",
					SHA1:        "fake-compiled-package-sha1-1",
				},
				{
					Name:        "fake-release-package-name-2",
					Version:     "fake-release-package-fingerprint-2",
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
				_, err := compiler.For(releaseJobs, fakeStage)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-compile-package-2-error"))
			})
		})
	})
})
