package cmd_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshreldir "github.com/cloudfoundry/bosh-cli/releasedir"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
	boshtbl "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("BlobsCmd", func() {
	var (
		blobsDir *fakereldir.FakeBlobsDir
		ui       *fakeui.FakeUI
		command  BlobsCmd
	)

	BeforeEach(func() {
		blobsDir = &fakereldir.FakeBlobsDir{}
		ui = &fakeui.FakeUI{}
		command = NewBlobsCmd(blobsDir, ui)
	})

	Describe("Run", func() {
		act := func() error { return command.Run() }

		It("lists blobs", func() {
			blobs := []boshreldir.Blob{
				boshreldir.Blob{
					Path: "fake-path",
					Size: 100,

					BlobstoreID: "fake-blob-id",
					SHA1:        "fake-sha1",
				},
				boshreldir.Blob{
					Path: "dir/fake-path",
					Size: 1000,

					BlobstoreID: "",
					SHA1:        "fake-sha2",
				},
			}

			blobsDir.BlobsReturns(blobs, nil)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(ui.Table).To(Equal(boshtbl.Table{
				Content: "blobs",

				Header: []boshtbl.Header{
					boshtbl.NewHeader("Path"),
					boshtbl.NewHeader("Size"),
					boshtbl.NewHeader("Blobstore ID"),
					boshtbl.NewHeader("Digest"),
				},

				SortBy: []boshtbl.ColumnSort{{Column: 0, Asc: true}},

				Rows: [][]boshtbl.Value{
					{
						boshtbl.NewValueString("fake-path"),
						boshtbl.NewValueBytes(100),
						boshtbl.NewValueString("fake-blob-id"),
						boshtbl.NewValueString("fake-sha1"),
					},
					{
						boshtbl.NewValueString("dir/fake-path"),
						boshtbl.NewValueBytes(1000),
						boshtbl.NewValueString("(local)"),
						boshtbl.NewValueString("fake-sha2"),
					},
				},
			}))
		})

		It("returns error if blobs cannot be retrieved", func() {
			blobsDir.BlobsReturns(nil, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
