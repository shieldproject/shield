package installation_test

import (
	fakeboshblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakeboshcmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	biproperty "github.com/cloudfoundry/bosh-utils/property"
	fakeboshsys "github.com/cloudfoundry/bosh-utils/system/fakes"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/bosh-cli/installation"
	biinstallmanifest "github.com/cloudfoundry/bosh-cli/installation/manifest"
	bireljob "github.com/cloudfoundry/bosh-cli/release/job"
	birelpkg "github.com/cloudfoundry/bosh-cli/release/pkg"
	. "github.com/cloudfoundry/bosh-cli/release/resource"
	bitemplate "github.com/cloudfoundry/bosh-cli/templatescompiler"
	mock_template "github.com/cloudfoundry/bosh-cli/templatescompiler/mocks"
	fakebiui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
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
		fakeBlobstore       *fakeboshblob.FakeDigestBlobstore

		fs *fakeboshsys.FakeFileSystem

		logger boshlog.Logger

		renderer installation.JobRenderer

		releaseJob  bireljob.Job
		releaseJobs []bireljob.Job

		manifest  biinstallmanifest.Manifest
		fakeStage *fakebiui.FakeStage

		renderedJobList bitemplate.RenderedJobList
	)

	BeforeEach(func() {
		mockJobListRenderer = mock_template.NewMockJobListRenderer(mockCtrl)
		fakeCompressor = fakeboshcmd.NewFakeCompressor()
		fakeBlobstore = &fakeboshblob.FakeDigestBlobstore{}

		fs = fakeboshsys.NewFakeFileSystem()

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

		pkg1 := birelpkg.NewPackage(NewResource("pkg1-name", "pkg1-fp", nil), nil)
		pkg2 := birelpkg.NewPackage(NewResource("pkg2-name", "pkg2-fp", nil), []string{"pkg1-name"})
		pkg2.AttachDependencies([]*birelpkg.Package{pkg1})

		job := bireljob.NewJob(NewResource("cpi", "fake-release-job-fingerprint", nil))
		job.PackageNames = []string{"pkg2-name"}
		job.AttachPackages([]*birelpkg.Package{pkg2})

		releaseJob = *job
		releaseJobs = []bireljob.Job{*job}
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
		renderedJobList.Add(bitemplate.NewRenderedJob(releaseJob, "/fake-rendered-job-cpi", fs, logger))

		mockJobListRenderer.EXPECT().Render(releaseJobs, releaseJobProperties, jobProperties, globalProperties, deploymentName, address).Return(renderedJobList, nil).AnyTimes()

		fakeCompressor.CompressFilesInDirTarballPath = "/fake-rendered-job-tarball-cpi.tgz"
		multiDigest := boshcrypto.MustParseMultipleDigest("fakerenderedjobtarballsha1cpi")
		fakeBlobstore.CreateReturns("fake-rendered-job-tarball-blobstore-id-cpi", multiDigest, nil)
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
			Expect(fakeBlobstore.CreateArgsForCall(0)).To(Equal("/fake-rendered-job-tarball-cpi.tgz"))
			Expect(fakeCompressor.CleanUpTarballPath).To(Equal("/fake-rendered-job-tarball-cpi.tgz"))
		})

		It("returns rendered job refs", func() {
			jobs, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(jobs).To(Equal([]installation.RenderedJobRef{
				installation.NewRenderedJobRef("cpi", "fake-release-job-fingerprint", "fake-rendered-job-tarball-blobstore-id-cpi", "fakerenderedjobtarballsha1cpi"),
			}))
		})

		It("cleans up the rendered jobs from the installation directory", func() {
			_, err := renderer.RenderAndUploadFrom(manifest, releaseJobs, fakeStage)
			Expect(err).ToNot(HaveOccurred())

			Expect(renderedJobList.All()).To(BeEmpty())
		})
	})
})
