package cmd_test

import (
	"errors"

	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshreldir "github.com/cloudfoundry/bosh-cli/releasedir"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("AddBlobCmd", func() {
	var (
		blobsDir *fakereldir.FakeBlobsDir
		fs       *fakesys.FakeFileSystem
		ui       *fakeui.FakeUI
		command  AddBlobCmd
	)

	BeforeEach(func() {
		blobsDir = &fakereldir.FakeBlobsDir{}
		fs = fakesys.NewFakeFileSystem()
		ui = &fakeui.FakeUI{}
		command = NewAddBlobCmd(blobsDir, fs, ui)
	})

	Describe("Run", func() {
		var (
			opts AddBlobOpts
		)

		BeforeEach(func() {
			fs.WriteFileString("/path/to/blob.tgz", "blob")
			opts = AddBlobOpts{
				Args: AddBlobArgs{
					Path:      "/path/to/blob.tgz",
					BlobsPath: "my-blob.tgz",
				},
			}
		})

		act := func() error { return command.Run(opts) }

		It("starts tracking blob", func() {
			blobsDir.TrackBlobReturns(boshreldir.Blob{Path: "my-blob.tgz"}, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.TrackBlobCallCount()).To(Equal(1))

			blobsPath, src := blobsDir.TrackBlobArgsForCall(0)
			Expect(blobsPath).To(Equal("my-blob.tgz"))

			file := src.(*fakesys.FakeFile)
			Expect(file.Name()).To(Equal("/path/to/blob.tgz"))
			Expect(file.Stats.Open).To(BeFalse())

			Expect(ui.Said).To(Equal([]string{"Added blob 'my-blob.tgz'"}))
		})

		It("returns error if tracking fails", func() {
			blobsDir.TrackBlobReturns(boshreldir.Blob{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(ui.Said).To(BeEmpty())
		})

		It("returns error if file cannot be open", func() {
			fs.OpenFileErr = errors.New("fake-err")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.TrackBlobCallCount()).To(Equal(0))

			Expect(ui.Said).To(BeEmpty())
		})
	})
})
