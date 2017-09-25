package cmd_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/clock/fakeclock"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
)

var _ = Describe("UIDownloader", func() {
	var (
		director    *fakedir.FakeDirector
		fs          *fakesys.FakeFileSystem
		timeService clock.Clock
		ui          *fakeui.FakeUI
		downloader  UIDownloader
	)

	BeforeEach(func() {
		director = &fakedir.FakeDirector{}
		timeService = fakeclock.NewFakeClock(time.Date(2009, time.November, 10, 23, 1, 2, 333, time.UTC))
		fs = fakesys.NewFakeFileSystem()
		ui = &fakeui.FakeUI{}
		downloader = NewUIDownloader(director, timeService, fs, ui)
	})

	Describe("Download", func() {
		var expectedPath string

		BeforeEach(func() {
			expectedPath = filepath.Join("/", "fake-dst-dir", "prefix-20091110-230102-000000333.tgz")

			err := fs.MkdirAll("/fake-dst-dir", os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
		})

		itReturnsErrs := func(act func() error) {
			It("returns error if downloading resource fails", func() {
				err := fs.MkdirAll("/fake-dst-dir", os.ModePerm)
				Expect(err).ToNot(HaveOccurred())

				fs.ReturnTempFile = fakesys.NewFakeFile("/some-tmp-file", fs)

				director.DownloadResourceUncheckedStub = func(_ string, _ io.Writer) error {
					return errors.New("fake-err")
				}

				err = act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(fs.FileExists("/some-tmp-file")).To(BeFalse())
				Expect(fs.FileExists(expectedPath)).To(BeFalse())
			})

			It("returns error if temp file cannot be created", func() {
				fs.TempFileError = errors.New("fake-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(director.DownloadResourceUncheckedCallCount()).To(Equal(0))
				Expect(fs.FileExists(expectedPath)).To(BeFalse())
			})
		}

		Context("when SHA1 is provided", func() {
			act := func() error {
				return downloader.Download("fake-blob-id", "a2511842a89119b9da922f9528307b7f8f55b798", "prefix", "/fake-dst-dir")
			}

			It("downloads specified blob to a specific destination", func() {
				fakeFile := fakesys.NewFakeFile("/some-tmp-file", fs)
				fakeFile.Write([]byte("file-contents"))
				fs.ReturnTempFile = fakeFile

				director.DownloadResourceUncheckedStub = func(_ string, out io.Writer) error {
					out.Write([]byte("file-contents"))
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("/some-tmp-file")).To(BeFalse())
				Expect(fs.FileExists(expectedPath)).To(BeTrue())
				Expect(fs.ReadFileString(expectedPath)).To(Equal("file-contents"))

				blobID, _ := director.DownloadResourceUncheckedArgsForCall(0)
				Expect(blobID).To(Equal("fake-blob-id"))

				Expect(ui.Said).To(Equal([]string{
					fmt.Sprintf("Downloading resource 'fake-blob-id' to '%s'...", expectedPath)}))
			})

			It("returns error if sha1 does not match expected sha1", func() {
				fakeFile := fakesys.NewFakeFile("/some-tmp-file", fs)
				fakeFile.Write([]byte("file-contents-that-were-corrupted"))
				fs.ReturnTempFile = fakeFile

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Expected stream to have digest 'a2511842a89119b9da922f9528307b7f8f55b798' but was '93135ede4065c7d5958ab7e328d501f8d4d9e2aa'"))

				Expect(fs.FileExists(expectedPath)).To(BeFalse())
			})

			It("returns error if sha1 check fails", func() {
				fakeFile := fakesys.NewFakeFile("/some-tmp-file", fs)
				fakeFile.Write([]byte("file-contents-that-were-corrupted"))
				fs.ReturnTempFile = fakeFile
				fs.OpenFileErr = errors.New("fake-err")

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))

				Expect(fs.FileExists(expectedPath)).To(BeFalse())
			})

			itReturnsErrs(act)
		})

		Context("when SHA1 is not provided", func() {
			act := func() error { return downloader.Download("fake-blob-id", "", "prefix", "/fake-dst-dir") }

			It("downloads specified blob to a specific destination without checking SHA1", func() {
				fs.ReturnTempFile = fakesys.NewFakeFile("/some-tmp-file", fs)

				director.DownloadResourceUncheckedStub = func(_ string, out io.Writer) error {
					out.Write([]byte("content"))
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("/some-tmp-file")).To(BeFalse())
				Expect(fs.FileExists(expectedPath)).To(BeTrue())
				Expect(fs.ReadFileString(expectedPath)).To(Equal("content"))

				blobID, _ := director.DownloadResourceUncheckedArgsForCall(0)
				Expect(blobID).To(Equal("fake-blob-id"))

				Expect(ui.Said).To(Equal([]string{
					fmt.Sprintf("Downloading resource 'fake-blob-id' to '%s'...", expectedPath)}))
			})

			itReturnsErrs(act)
		})

		Context("when downloading across devices", func() {
			BeforeEach(func() {
				fs.RenameError = &os.LinkError{
					Err: syscall.Errno(0x12),
				}
			})

			act := func() error { return downloader.Download("fake-blob-id", "", "prefix", "/fake-dst-dir") }

			It("downloads specified blob to a specific destination without checking SHA1", func() {
				fs.ReturnTempFile = fakesys.NewFakeFile("/some-tmp-file", fs)

				director.DownloadResourceUncheckedStub = func(_ string, out io.Writer) error {
					out.Write([]byte("content"))
					return nil
				}

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists("/some-tmp-file")).To(BeFalse())
				Expect(fs.FileExists(expectedPath)).To(BeTrue())
				Expect(fs.ReadFileString(expectedPath)).To(Equal("content"))

				blobID, _ := director.DownloadResourceUncheckedArgsForCall(0)
				Expect(blobID).To(Equal("fake-blob-id"))

				Expect(ui.Said).To(Equal([]string{
					fmt.Sprintf("Downloading resource 'fake-blob-id' to '%s'...", expectedPath)}))
			})

			itReturnsErrs(act)
		})
	})
})
