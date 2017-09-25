package installation_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-init/installation"

	mock_template "github.com/cloudfoundry/bosh-init/templatescompiler/mocks"
	"github.com/golang/mock/gomock"

	biinstallmanifest "github.com/cloudfoundry/bosh-init/installation/manifest"
	bireljob "github.com/cloudfoundry/bosh-init/release/job"
	birelpkg "github.com/cloudfoundry/bosh-init/release/pkg"
	bitemplate "github.com/cloudfoundry/bosh-init/templatescompiler"
	fakeboshblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakeboshcmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"

	fakebiui "github.com/cloudfoundry/bosh-init/ui/fakes"
)

var _ = Describe("JobRenderer", func() {
	var mockCtrl *gomock.Controller

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	var (
		mockJobListRenderer *mock_template.MockJobListRenderer
		fakeCompressor      *fakeboshcmd.FakeCompressor
		fakeBlobstore       *fakeboshblob.FakeBlobstore

		fakeFS *fakeboshsys.FakeFileSystem

		logger boshlog.Logger

		renderer installation.JobRenderer

		releaseJob  bireljob.Job
		releaseJobs []bireljob.Job

		manifest  biinstallmanifest.Manifest
		fakeStage *fakebiui.FakeStage

		releasePackage1 *birelpkg.Package
		releasePackage2 *birelpkg.Package

		renderedJobList bitemplate.RenderedJobList
	)

	BeforeEach(func() {
		mockJobListRenderer = mock_template.NewMockJobListRenderer(mockCtrl)
		fakeCompressor = fakeboshcmd.NewFakeCompressor()
		fakeBlobstore = fakeboshblob.NewFakeBlobstore()

		fakeFS = fakeboshsys.NewFakeFileSystem()

		logger = boshlog.NewLogger(boshlog.LevelNone)

		fakeStage = fakebiui.NewFakeStage()

		manifest = biinstallmanifest.Manifest{
			Name: "fake-installation-name",
			Template: biinstallmanifest.ReleaseJobRef{
				Name:    "fake-cpi-job-name",
				Release: "fake-cpi-release-name",
			},
			Properties: biproperty.Map{
				"fake-installation-property": "fake-installation-property-value",
			},
		}

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
		renderer = installation.NewJobRenderer(
			mockJobListRenderer,
			fakeCompressor,
			fakeBlobstore,
		)

		releaseJobs := []bireljob.Job{releaseJob}

		releaseJobProperties := map[string]*biproperty.Map{}
		jobProperties := biproperty.Map{
			"fake-installation-property": "fake-installation-property-value",
		}
		globalProperties := biproperty.Map{}
		deploymentName := "fake-installation-name"
		address := ""

		renderedJobList = bitemplate.NewRenderedJobList()
		renderedJobList.Add(bitemplate.NewRenderedJob(releaseJob, "/fake-rendered-job-cpi", fakeFS, logger))

		mockJobListRenderer.EXPECT().Render(releaseJobs, releaseJobProperties, jobProperties, globalProperties, deploymentName, address).Return(renderedJobList, nil).AnyTimes()

		fakeCompressor.CompressFilesInDirTarballPath = "/fake-rendered-job-tarball-cpi.tgz"

		fakeBlobstore.CreateBlobIDs = []string{"fake-rendered-job-tarball-blobstore-id-cpi"}
		fakeBlobstore.CreateFingerprints = []string{"fake-rendered-job-tarball-sha1-cpi"}
	})

	Describe("RenderAndUploadFrom", func() {
		It("logs compile & render stages", func() {
			_, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeStage.PerformCalls).To(Equal([]*fakebiui.PerformCall{
				// compile stages not produced by mockDependencyCompiler
				{Name: "Rendering job templates"},
			}))
		})

		It("compresses and uploads the rendered cpi job, deleting the local tarball afterward", func() {
			_, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCompressor.CompressFilesInDirDir).To(Equal("/fake-rendered-job-cpi"))
			Expect(fakeBlobstore.CreateFileNames).To(Equal([]string{"/fake-rendered-job-tarball-cpi.tgz"}))
			Expect(fakeCompressor.CleanUpTarballPath).To(Equal("/fake-rendered-job-tarball-cpi.tgz"))
		})

		It("returns rendered job refs", func() {
			jobs, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(jobs).To(Equal([]installation.RenderedJobRef{
				installation.NewRenderedJobRef("cpi", "fake-release-job-fingerprint", "fake-rendered-job-tarball-blobstore-id-cpi", "fake-rendered-job-tarball-sha1-cpi"),
			}))
		})

		It("cleans up the rendered jobs from the installation directory", func() {
			_, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(renderedJobList.All()).To(BeEmpty())
		})
	})
})
