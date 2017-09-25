package cmd_test

import (
	"errors"

	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("RemoveBlobCmd", func() {
	var (
		blobsDir *fakereldir.FakeBlobsDir
		fs       *fakesys.FakeFileSystem
		ui       *fakeui.FakeUI
		command  RemoveBlobCmd
	)

	BeforeEach(func() {
		blobsDir = &fakereldir.FakeBlobsDir{}
		fs = fakesys.NewFakeFileSystem()
		ui = &fakeui.FakeUI{}
		command = NewRemoveBlobCmd(blobsDir, ui)
	})

	Describe("Run", func() {
		var (
			opts RemoveBlobOpts
		)

		BeforeEach(func() {
			fs.WriteFileString("/path/to/blob.tgz", "blob")
			opts = RemoveBlobOpts{
				Args: RemoveBlobArgs{BlobsPath: "/path/to/blob.tgz"},
			}
		})

		act := func() error { return command.Run(opts) }

		It("untracks blob", func() {
			blobsDir.UntrackBlobReturns(nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.UntrackBlobCallCount()).To(Equal(1))
			Expect(blobsDir.UntrackBlobArgsForCall(0)).To(Equal("/path/to/blob.tgz"))

			Expect(ui.Said).To(Equal([]string{"Removed blob '/path/to/blob.tgz'"}))
		})

		It("returns error if untracking fails", func() {
			blobsDir.UntrackBlobReturns(errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(ui.Said).To(BeEmpty())
		})
	})
})
