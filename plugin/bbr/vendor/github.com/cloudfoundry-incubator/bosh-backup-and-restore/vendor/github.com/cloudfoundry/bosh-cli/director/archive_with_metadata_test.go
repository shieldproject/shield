package director_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"

	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewFSReleaseArchive", func() {
	It("returns release archive", func() {
		fs := fakesys.NewFakeFileSystem()
		Expect(NewFSReleaseArchive("/path", fs)).To(
			Equal(NewFSArchiveWithMetadata("/path", "release.MF", fs)))
	})
})

var _ = Describe("NewFSStemcellArchive", func() {
	It("returns stemcell archive", func() {
		fs := fakesys.NewFakeFileSystem()
		Expect(NewFSStemcellArchive("/path", fs)).To(
			Equal(NewFSArchiveWithMetadata("/path", "stemcell.MF", fs)))
	})
})

var _ = Describe("FSArchiveWithMetadata", func() {
	var (
		fs      *fakesys.FakeFileSystem
		archive ReleaseArchive
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		archive = NewFSArchiveWithMetadata("/path", "metadata.MF", fs)
	})

	Describe("Info", func() {
		validContent := "---\nname: name\nversion: ver\n"

		validReleaseTgzBytes := func(fileName, content string) []byte {
			fileBytes := &bytes.Buffer{}
			gzipWriter := gzip.NewWriter(fileBytes)
			tarWriter := tar.NewWriter(gzipWriter)

			{
				otherContents := []byte("other-content")
				otherHeader := &tar.Header{
					Name: "other-file",
					Size: int64(len(otherContents)),
				}

				err := tarWriter.WriteHeader(otherHeader)
				Expect(err).ToNot(HaveOccurred())

				_, err = tarWriter.Write(otherContents)
				Expect(err).ToNot(HaveOccurred())
			}

			{
				mfContents := []byte(content)
				mfHeader := &tar.Header{
					Name: fileName,
					Size: int64(len(mfContents)),
				}

				err := tarWriter.WriteHeader(mfHeader)
				Expect(err).ToNot(HaveOccurred())

				_, err = tarWriter.Write(mfContents)
				Expect(err).ToNot(HaveOccurred())
			}

			{
				otherContents := []byte("other-content-after")
				otherHeader := &tar.Header{
					Name: "other-file-after",
					Size: int64(len(otherContents)),
				}

				err := tarWriter.WriteHeader(otherHeader)
				Expect(err).ToNot(HaveOccurred())

				_, err = tarWriter.Write(otherContents)
				Expect(err).ToNot(HaveOccurred())
			}

			err := tarWriter.Close()
			Expect(err).ToNot(HaveOccurred())

			err = gzipWriter.Close()
			Expect(err).ToNot(HaveOccurred())

			return fileBytes.Bytes()
		}

		It("returns release name and version from metadata file", func() {
			err := fs.WriteFile("/path", validReleaseTgzBytes("metadata.MF", validContent))
			Expect(err).ToNot(HaveOccurred())

			name, version, err := archive.Info()
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("name"))
			Expect(version).To(Equal("ver"))
		})

		It("returns release name and version from dotted metadata file", func() {
			err := fs.WriteFile("/path", validReleaseTgzBytes("./metadata.MF", validContent))
			Expect(err).ToNot(HaveOccurred())

			name, version, err := archive.Info()
			Expect(err).ToNot(HaveOccurred())
			Expect(name).To(Equal("name"))
			Expect(version).To(Equal("ver"))
		})

		It("returns error if cannot read gzip", func() {
			err := fs.WriteFileString("/path", "invalid-gzip")
			Expect(err).ToNot(HaveOccurred())

			_, _, err = archive.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("gzip: invalid header"))
		})

		It("returns error if cannot read tar", func() {
			fileBytes := &bytes.Buffer{}
			gzipWriter := gzip.NewWriter(fileBytes)

			_, err := gzipWriter.Write([]byte("invalid-tar"))
			Expect(err).ToNot(HaveOccurred())

			err = gzipWriter.Close()
			Expect(err).ToNot(HaveOccurred())

			err = fs.WriteFile("/path", fileBytes.Bytes())
			Expect(err).ToNot(HaveOccurred())

			_, _, err = archive.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Reading next tar entry"))
		})

		It("returns error if cannot find manifest file", func() {
			err := fs.WriteFile("/path", validReleaseTgzBytes("./wrong.MF", ""))
			Expect(err).ToNot(HaveOccurred())

			_, _, err = archive.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Missing 'metadata.MF'"))
		})

		It("returns error if cannot parse manifest file", func() {
			err := fs.WriteFile("/path", validReleaseTgzBytes("./metadata.MF", "-"))
			Expect(err).ToNot(HaveOccurred())

			_, _, err = archive.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshalling 'metadata.MF'"))
		})

		It("returns error if cannot open archive", func() {
			fs.OpenFileErr = errors.New("fake-err")

			_, _, err := archive.Info()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})

	Describe("File", func() {
		It("returns file and keeps it open", func() {
			err := fs.WriteFileString("/path", "content")
			Expect(err).ToNot(HaveOccurred())

			file, err := archive.File()
			Expect(err).ToNot(HaveOccurred())

			fakeFile := file.(*fakesys.FakeFile)
			Expect(fakeFile.Contents).To(Equal([]byte("content")))
			Expect(fakeFile.Stats.Open).To(BeTrue())

			// has a way to close it
			Expect(file.Close()).To(BeNil())
		})

		It("returns error if cannot open archive", func() {
			fs.OpenFileErr = errors.New("fake-err")

			_, err := archive.File()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})
	})
})
