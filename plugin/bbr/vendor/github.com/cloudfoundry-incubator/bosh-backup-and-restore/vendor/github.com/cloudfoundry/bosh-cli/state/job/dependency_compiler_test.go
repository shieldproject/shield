package job_test

import (
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshreljob "github.com/cloudfoundry/bosh-cli/release/job"
	boshrelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	. "github.com/cloudfoundry/bosh-cli/state/job"
	bistatepkg "github.com/cloudfoundry/bosh-cli/state/pkg"
	mock_state_package "github.com/cloudfoundry/bosh-cli/state/pkg/mocks"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
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
		stage              *fakeui.FakeStage

		pkg1 *boshrelpkg.Package
		pkg2 *boshrelpkg.Package

		job  *boshreljob.Job
		jobs []boshreljob.Job

		expectCompilePkg1 *gomock.Call
		expectCompilePkg2 *gomock.Call
	)

	BeforeEach(func() {
		mockPackageCompiler = mock_state_package.NewMockCompiler(mockCtrl)

		logger = boshlog.NewLogger(boshlog.LevelNone)
		dependencyCompiler = NewDependencyCompiler(mockPackageCompiler, logger)

		stage = fakeui.NewFakeStage()

		pkg1 = newPkg("pkg1-name", "pkg1-fp", nil)
		pkg2 = newPkg("pkg2-name", "pkg2-fp", []string{"pkg1-name"})
		pkg2.AttachDependencies([]*boshrelpkg.Package{pkg1})
		job = boshreljob.NewJob(NewResourceWithBuiltArchive("cpi", "job-fp", "path", "sha1"))
		job.PackageNames = []string{"pkg2-name"}
		job.AttachPackages([]*boshrelpkg.Package{pkg2})
		jobs = []boshreljob.Job{*job}
	})

	JustBeforeEach(func() {
		compiledPackageRecord1 := bistatepkg.CompiledPackageRecord{
			BlobID:   "fake-compiled-package-blobstore-id-1",
			BlobSHA1: "fake-compiled-package-sha1-1",
		}
		expectCompilePkg1 = mockPackageCompiler.EXPECT().Compile(pkg1).Return(compiledPackageRecord1, false, nil).AnyTimes()

		compiledPackageRecord2 := bistatepkg.CompiledPackageRecord{
			BlobID:   "fake-compiled-package-blobstore-id-2",
			BlobSHA1: "fake-compiled-package-sha1-2",
		}
		expectCompilePkg2 = mockPackageCompiler.EXPECT().Compile(pkg2).Return(compiledPackageRecord2, false, nil).AnyTimes()
	})

	It("compiles all the job dependencies (packages) such that no package is compiled before its dependencies", func() {
		gomock.InOrder(
			expectCompilePkg1.Times(1),
			expectCompilePkg2.Times(1),
		)

		_, err := dependencyCompiler.Compile(jobs, stage)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns references to the compiled packages", func() {
		compiledPackageRefs, err := dependencyCompiler.Compile(jobs, stage)
		Expect(err).ToNot(HaveOccurred())

		Expect(compiledPackageRefs).To(Equal([]CompiledPackageRef{
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

	It("logs compile stages", func() {
		_, err := dependencyCompiler.Compile(jobs, stage)
		Expect(err).ToNot(HaveOccurred())

		Expect(stage.PerformCalls).To(Equal([]*fakeui.PerformCall{
			{Name: "Compiling package 'pkg1-name/pkg1-fp'"},
			{Name: "Compiling package 'pkg2-name/pkg2-fp'"},
		}))
	})

	Context("when packages are in circular dependency", func() {
		var (
			pkg1, pkg2, pkg3 *boshrelpkg.Package
		)

		BeforeEach(func() {
			pkg1 = newPkg("pkg1-name", "pkg1-fp", []string{"pkg3-name"})
			pkg2 = newPkg("pkg2-name", "pkg2-fp", []string{"pkg1-name"})
			pkg3 = newPkg("pkg3-name", "pkg3-fp", []string{"pkg2-name"})
			pkg1.AttachDependencies([]*boshrelpkg.Package{pkg3})
			pkg2.AttachDependencies([]*boshrelpkg.Package{pkg1})
			pkg3.AttachDependencies([]*boshrelpkg.Package{pkg2})

			job = boshreljob.NewJob(NewResourceWithBuiltArchive("cpi", "job-fp", "path", "sha1"))
			job.PackageNames = []string{"pkg2-name"}
			job.AttachPackages([]*boshrelpkg.Package{pkg1, pkg2, pkg3})
			jobs = []boshreljob.Job{*job}
		})

		It("returns an error", func() {
			_, err := dependencyCompiler.Compile(jobs, stage)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when a compiled releases is provided", func() {
		BeforeEach(func() {
			compiledPackageRecord1 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-1",
				BlobSHA1: "fake-compiled-package-sha1-1",
			}
			expectCompilePkg1 = mockPackageCompiler.EXPECT().Compile(pkg1).Return(compiledPackageRecord1, true, nil).AnyTimes()

			compiledPackageRecord2 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-2",
				BlobSHA1: "fake-compiled-package-sha1-2",
			}
			expectCompilePkg2 = mockPackageCompiler.EXPECT().Compile(pkg2).Return(compiledPackageRecord2, true, nil).AnyTimes()
		})

		It("skips compiling the packages in the release", func() {
			_, err := dependencyCompiler.Compile(jobs, stage)
			Expect(err).ToNot(HaveOccurred())

			for _, call := range stage.PerformCalls {
				Expect(call.SkipError).To(HaveOccurred())
				Expect(call.SkipError.Error()).To(MatchRegexp("Package already compiled: Package 'pkg\\d-name' is already compiled. Skipped compilation"))
			}
		})
	})

	Context("when multiple jobs depend on the same package", func() {
		JustBeforeEach(func() {
			job2 := boshreljob.NewJob(NewResourceWithBuiltArchive("job2-name", "job2-fp", "", ""))
			job2.PackageNames = []string{"pkg2-name"}
			job2.AttachPackages([]*boshrelpkg.Package{pkg2})
			jobs = append(jobs, *job2)
		})

		It("only compiles each package once", func() {
			gomock.InOrder(
				expectCompilePkg1.Times(1),
				expectCompilePkg2.Times(1),
			)

			_, err := dependencyCompiler.Compile(jobs, stage)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when multiple packages depend on the same package", func() {
		var (
			pkg3              *boshrelpkg.Package
			expectCompilePkg3 *gomock.Call
		)

		BeforeEach(func() {
			pkg3 = newPkg("pkg3-name", "pkg3-fp", []string{"pkg1-name"})
			pkg3.AttachDependencies([]*boshrelpkg.Package{pkg1})

			job.PackageNames = append(job.PackageNames, pkg3.Name())
			job.AttachPackages([]*boshrelpkg.Package{pkg1, pkg2, pkg3})
		})

		JustBeforeEach(func() {
			compiledPackageRecord3 := bistatepkg.CompiledPackageRecord{
				BlobID:   "fake-compiled-package-blobstore-id-3",
				BlobSHA1: "fake-compiled-package-sha1-3",
			}
			expectCompilePkg3 = mockPackageCompiler.EXPECT().Compile(pkg3).Return(compiledPackageRecord3, false, nil).AnyTimes()
		})

		It("only compiles each package once", func() {
			expectCompilePkg1.Times(1)
			expectCompilePkg2.After(expectCompilePkg1)
			expectCompilePkg3.After(expectCompilePkg1)

			_, err := dependencyCompiler.Compile(jobs, stage)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
