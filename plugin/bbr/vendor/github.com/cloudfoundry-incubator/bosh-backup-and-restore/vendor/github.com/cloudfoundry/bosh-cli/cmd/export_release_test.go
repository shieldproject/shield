package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("ExportReleaseCmd", func() {
	var (
		deployment *fakedir.FakeDeployment
		downloader *fakecmd.FakeDownloader
		command    ExportReleaseCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		downloader = &fakecmd.FakeDownloader{}
		command = NewExportReleaseCmd(deployment, downloader)
	})

	Describe("Run", func() {
		var (
			opts ExportReleaseOpts
		)

		BeforeEach(func() {
			opts = ExportReleaseOpts{
				Args: ExportReleaseArgs{
					ReleaseSlug:   boshdir.NewReleaseSlug("rel", "rel-ver"),
					OSVersionSlug: boshdir.NewOSVersionSlug("os", "os-ver"),
				},

				Directory: DirOrCWDArg{Path: "/fake-dir"},
			}
		})

		act := func() error { return command.Run(opts) }

		It("fetches exported release", func() {
			result := boshdir.ExportReleaseResult{
				BlobstoreID: "blob-id",
				SHA1:        "sha1",
			}

			deployment.ExportReleaseReturns(result, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(deployment.ExportReleaseCallCount()).To(Equal(1))

			rel, os := deployment.ExportReleaseArgsForCall(0)
			Expect(rel).To(Equal(boshdir.NewReleaseSlug("rel", "rel-ver")))
			Expect(os).To(Equal(boshdir.NewOSVersionSlug("os", "os-ver")))

			Expect(downloader.DownloadCallCount()).To(Equal(1))

			blobID, sha1, prefix, dstDirPath := downloader.DownloadArgsForCall(0)
			Expect(blobID).To(Equal("blob-id"))
			Expect(sha1).To(Equal("sha1"))
			Expect(prefix).To(Equal("rel-rel-ver-os-os-ver"))
			Expect(dstDirPath).To(Equal("/fake-dir"))
		})

		It("returns error if exporting release failed", func() {
			deployment.ExportReleaseReturns(boshdir.ExportReleaseResult{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if downloading release failed", func() {
			downloader.DownloadReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
