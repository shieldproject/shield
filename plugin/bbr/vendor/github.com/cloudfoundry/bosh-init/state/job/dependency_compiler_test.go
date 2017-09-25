package job_test

import (
	. "github.com/cloudfoundry/bosh-init/state/job"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	mock_state_package "github.com/cloudfoundry/bosh-init/state/pkg/mocks"
	"github.com/golang/mock/gomock"

	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bistatepkg "github.com/cloudfoundry/bosh-init/state/pkg"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("DependencyCompiler", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		mockPackageCompiler *mock_state_package.MockCompiler
		logger              boshlog.Logger

		dependencyCompiler DependencyCompiler

		releaseJobs []bireljob.Job
		fakeStage   *fakebiui.FakeStage

		releasePackage1 *birelpkg.Package
		releasePackage2 *birelpkg.Package

		releaseJob bireljob.Job

		expectCompilePkg1 *gomock.Call
		expectCompilePkg2 *gomock.Call
	)

	BeforeEach(func() {
		mockPackageCompiler = mock_state_package.NewMockCompiler(mockCtrl)

		logger = boshlog.NewLogger(boshlog.LevelNone)
		dependencyCompiler = NewDependencyCompiler(mockPackageCompiler, logger)

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
		releaseJobs = []bireljob.Job{releaseJob}
	})

	JustBeforeEach(func() {
		compiledPackageRecord1 := bistatepkg.CompiledPackageRecord{
			BlobID:   "fake-compiled-package-blobstore-id-1",
			BlobSHA1: "fake-compiled-package-sha1-1",
		}
		expectCompilePkg1 = mockPackageCompiler.EXPECT().Compile(releasePackage1).Return(compiledPackageRecord1, false, nil).AnyTimes()

		compiledPackageRecord2 := bistatepkg.CompiledPackageRecord{
			BlobID:   "fake-compiled-package-blobstore-id-2",
			BlobSHA1: "fake-compiled-package-sha1-2",
		}
		expectCompilePkg2 = mockPackageCompiler.EXPECT().Compile(releasePackage2).Return(compiledPackageRecord2, false, nil).AnyTimes()
	})

	It("compiles all the job dependencies (packages) such that no package is compiled before its dependencies", func() {
		gomock.InOrder(
			expectCompilePkg1.Times(1),
			expectCompilePkg2.Times(1),
		)

		_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns references to the compiled packages", func() {
		compiledPackageRefs, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
		Expect(err).ToNot(HaveOccurred())

		Expect(compiledPackageRefs).To(Equal([]CompiledPackageRef{
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

	It("logs compile stages", func() {
		_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
			{Name: "Compiling package 'fake-release-package-name-1/fake-release-package-fingerprint-1'"},
			{Name: "Compiling package 'fake-release-package-name-2/fake-release-package-fingerprint-2'"},
		}))
	})

	Context("Graph with circular dependency", func() {
		var (
			package1,
			package2,
			package3 *birelpkg.Package
		)
		BeforeEach(func() {
			package1 = &birelpkg.Package{
				Name:         "fake-package-name-1",
				Dependencies: []*birelpkg.Package{},
			}
			package2 = &birelpkg.Package{
				Name:         "fake-package-name-2",
				Dependencies: []*birelpkg.Package{package1},
			}
			package3 = &birelpkg.Package{
				Name:         "fake-package-name-3",
				Dependencies: []*birelpkg.Package{package2},
			}

			package1.Dependencies = append(package1.Dependencies, package3)

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
				Packages:     []*birelpkg.Package{package1, package2, package3},
				Properties:   map[string]bireljob.PropertyDefinition{},
			}

		})

		It("returns an error", func() {
			releaseJobs = []bireljob.Job{releaseJob}
			_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
			Expect(err).NotTo(BeNil())

		})
	})

	Context("when a compiled releases is provided", func() {

		BeforeEach(func() {
			compiledPackageRecord1 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-1",
				BlobSHA1: "fake-compiled-package-sha1-1",
			}
			expectCompilePkg1 = mockPackageCompiler.EXPECT().Compile(releasePackage1).Return(compiledPackageRecord1, true, nil).AnyTimes()

			compiledPackageRecord2 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-2",
				BlobSHA1: "fake-compiled-package-sha1-2",
			}
			expectCompilePkg2 = mockPackageCompiler.EXPECT().Compile(releasePackage2).Return(compiledPackageRecord2, true, nil).AnyTimes()
		})

		It("skips compiling the packages in the release", func() {
			_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			for _, call := range fakeStage.PerformCalls {
				Expect(call.SkipError.Error()).To(MatchRegexp("Package already compiled: Package 'fake-release-package-name-\\d' is already compiled. Skipped compilation"))
			}
		})
	})

	Context("when multiple jobs depend on the same package", func() {
		JustBeforeEach(func() {
			releaseJob2 := bireljob.Job{
				Name:         "fake-other-job",
				Fingerprint:  "fake-other-job-fingerprint",
				PackageNames: []string{releasePackage2.Name},
				Packages:     []*birelpkg.Package{releasePackage2},
			}
			releaseJobs = append(releaseJobs, releaseJob2)
		})

		It("only compiles each package once", func() {
			gomock.InOrder(
				expectCompilePkg1.Times(1),
				expectCompilePkg2.Times(1),
			)

			_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when multiple packages depend on the same package", func() {
		var (
			releasePackage3 *birelpkg.Package

			expectCompilePkg3 *gomock.Call
		)

		BeforeEach(func() {
			releasePackage3 = &birelpkg.Package{
				Name:          "fake-release-package-name-3",
				Fingerprint:   "fake-release-package-fingerprint-3",
				SHA1:          "fake-release-package-sha1-3",
				Dependencies:  []*birelpkg.Package{releasePackage1},
				ExtractedPath: "/extracted-release-path/extracted_packages/fake-release-package-name-3",
			}

			releaseJob.PackageNames = append(releaseJob.PackageNames, releasePackage3.Name)
			releaseJob.Packages = append(releaseJob.Packages, releasePackage3)
			releaseJobs = []bireljob.Job{releaseJob}
		})

		JustBeforeEach(func() {
			compiledPackageRecord3 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-3",
				BlobSHA1: "fake-compiled-package-sha1-3",
			}
			expectCompilePkg3 = mockPackageCompiler.EXPECT().Compile(releasePackage3).Return(compiledPackageRecord3, false, nil).AnyTimes()
		})

		It("only compiles each package once", func() {
			expectCompilePkg1.Times(1)
			expectCompilePkg2.After(expectCompilePkg1)
			expectCompilePkg3.After(expectCompilePkg1)

			_, err := dependencyCompiler.Compile(releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
