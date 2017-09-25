package disk_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdevutil "github.com/cloudfoundry/bosh-agent/platform/deviceutil"
	fakedisk "github.com/cloudfoundry/bosh-agent/platform/disk/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/cloudfoundry/bosh-agent/platform/disk"
)

var _ = Describe("Diskutil", func() {
	var (
		diskUtil      boshdevutil.DeviceUtil
		mounter       *fakedisk.FakeMounter
		fs            *fakesys.FakeFileSystem
		fakeCmdRunner *fakesys.FakeCmdRunner
	)

	BeforeEach(func() {
		mounter = &fakedisk.FakeMounter{}
		fs = fakesys.NewFakeFileSystem()
		logger := boshlog.NewLogger(boshlog.LevelNone)
		diskUtil = NewDiskUtil("fake-disk-path", fakeCmdRunner, mounter, fs, logger)
	})

	Describe("GetFileContents", func() {
		Context("when disk path does not exist", func() {
			It("returns an error if diskpath does not exist", func() {
				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("disk path 'fake-disk-path' does not exist"))
			})
		})

		Context("when disk path does not exist", func() {
			BeforeEach(func() {
				fs.MkdirAll("fake-disk-path", 0700)
				fs.TempDirDir = "fake-tempdir"
				fs.WriteFileString("fake-tempdir/fake-file-path-1", "fake-contents-1")
				fs.WriteFileString("fake-tempdir/fake-file-path-2", "fake-contents-2")
			})

			It("mounts disk path to temporary directory", func() {
				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).ToNot(HaveOccurred())

				Expect(mounter.MountPartitionPaths).To(ContainElement("fake-disk-path"))
				Expect(mounter.MountMountPoints).To(ContainElement("fake-tempdir"))
			})

			It("returns contents of files on a disk", func() {
				contents, err := diskUtil.GetFilesContents([]string{"fake-file-path-1", "fake-file-path-2"})
				Expect(err).ToNot(HaveOccurred())
				Expect(len(contents)).To(Equal(2))
				Expect(string(contents[0])).To(Equal("fake-contents-1"))
				Expect(string(contents[1])).To(Equal("fake-contents-2"))
			})

			It("unmount disk path", func() {
				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).ToNot(HaveOccurred())

				Expect(mounter.UnmountPartitionPathOrMountPoint).To(Equal("fake-tempdir"))
			})

			It("cleans up temporary directory after reading settings", func() {
				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("fake-tempdir")).To(BeFalse())
			})

			It("returns error if it fails to create temporary mount directory", func() {
				fs.TempDirError = errors.New("fake-tempdir-error")

				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-tempdir-error"))
			})

			It("returns error if it fails to mount disk path", func() {
				mounter.MountErr = errors.New("fake-mount-error")

				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-mount-error"))
			})

			It("returns an error if it fails to read the file", func() {
				fs.ReadFileError = errors.New("fake-read-error")
				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-read-error"))
			})

			It("returns error if it fails to unmount disk path", func() {
				mounter.UnmountErr = errors.New("fake-unmount-error")

				_, err := diskUtil.GetFilesContents([]string{"fake-file-path-1"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-unmount-error"))
			})
		})
	})
})
