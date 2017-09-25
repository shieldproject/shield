// +build windows

package fileutil_test

import (
	"errors"
	"os"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/fileutil"

	"github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("Mover", func() {
	var mover Mover
	var fs *fakes.FakeFileSystem
	var oldLocation, newLocation string

	BeforeEach(func() {
		fs = fakes.NewFakeFileSystem()
		mover = NewFileMover(fs)

		oldLocation = "/path/to/old_file"
		newLocation = "/path/to/new_file"

		fs.WriteFileString(oldLocation, "some content")
	})

	Context("when Rename fails due to Win32 Error Code ERROR_NOT_SAME_DEVICE", func() {
		BeforeEach(func() {
			fs.RenameError = &os.LinkError{
				Err: syscall.Errno(0x11),
			}
		})

		It("moves the file", func() {
			Expect(fs.FileExists(oldLocation)).To(BeTrue())
			Expect(fs.FileExists(newLocation)).To(BeFalse())

			err := mover.Move(oldLocation, newLocation)
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists(oldLocation)).To(BeFalse())

			contents, err := fs.ReadFileString(newLocation)
			Expect(err).ToNot(HaveOccurred())
			Expect(contents).To(Equal("some content"))

			Expect(fs.CopyFileCallCount).To(Equal(1))
		})

		Context("when copying the file returns an error", func() {
			BeforeEach(func() {
				fs.CopyFileError = errors.New("copying error")
			})

			It("returns an error", func() {
				err := mover.Move(oldLocation, newLocation)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when deleting the old file returns an error", func() {
			BeforeEach(func() {
				fs.RemoveAllStub = func(_ string) error {
					return errors.New("error removing")
				}
			})

			It("returns an error", func() {
				err := mover.Move(oldLocation, newLocation)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
