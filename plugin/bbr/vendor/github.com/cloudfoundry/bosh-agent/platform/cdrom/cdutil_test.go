package cdrom_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshcdrom "github.com/cloudfoundry/bosh-agent/platform/cdrom"
	fakecdrom "github.com/cloudfoundry/bosh-agent/platform/cdrom/fakes"
	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Cdutil", func() {
	var (
		fs     *fakesys.FakeFileSystem
		cdrom  *fakecdrom.FakeCdrom
		cdutil boshdevutil.DeviceUtil
		logger boshlog.Logger
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		cdrom = fakecdrom.NewFakeCdrom(fs, "env", "fake env contents")
		logger = boshlog.NewLogger(boshlog.LevelNone)
	})

	JustBeforeEach(func() {
		cdutil = boshcdrom.NewCdUtil("/fake/settings/dir", fs, cdrom, logger)
	})

	It("gets file contents from CDROM", func() {
		contents, err := cdutil.GetFilesContents([]string{"env"})
		Expect(err).NotTo(HaveOccurred())

		Expect(cdrom.Mounted).To(Equal(false))
		Expect(cdrom.MediaAvailable).To(Equal(false))
		Expect(fs.FileExists("/fake/settings/dir")).To(Equal(true))
		Expect(cdrom.MountMountPath).To(Equal("/fake/settings/dir"))

		Expect(len(contents)).To(Equal(1))
		Expect(contents[0]).To(Equal([]byte("fake env contents")))
	})

})
