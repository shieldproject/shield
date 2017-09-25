package blobstore_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	. "github.com/cloudfoundry/bosh-utils/blobstore"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshsys "github.com/cloudfoundry/bosh-utils/system"
	boshsysfake "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Blob Manager", func() {
	var (
		fs       boshsys.FileSystem
		logger   boshlog.Logger
		basePath string
		blobPath string
		blobId   string
		toWrite  io.Reader
	)

	BeforeEach(func() {
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = boshsys.NewOsFileSystem(logger)
		blobId = "105d33ae-655c-493d-bf9f-1df5cf3ca847"
		basePath = os.TempDir()
		blobPath = filepath.Join(basePath, blobId)
		toWrite = bytes.NewReader([]byte("new data"))
	})

	readFile := func(fileIO boshsys.File) []byte {
		fileStat, _ := fileIO.Stat()
		fileBytes := make([]byte, fileStat.Size())
		fileIO.Read(fileBytes)
		return fileBytes
	}

	It("fetches", func() {
		blobManager := NewBlobManager(fs, basePath)
		fs.WriteFileString(blobPath, "some data")

		readOnlyFile, err, _ := blobManager.Fetch(blobId)
		defer fs.RemoveAll(readOnlyFile.Name())

		Expect(err).ToNot(HaveOccurred())
		fileBytes := readFile(readOnlyFile)

		Expect(string(fileBytes)).To(Equal("some data"))
	})

	It("writes", func() {
		blobManager := NewBlobManager(fs, basePath)
		fs.WriteFileString(blobPath, "some data")
		defer fs.RemoveAll(blobPath)

		err := blobManager.Write(blobId, toWrite)
		Expect(err).ToNot(HaveOccurred())

		contents, err := fs.ReadFileString(blobPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(contents).To(Equal("new data"))
	})

	Context("when it writes", func() {
		BeforeEach(func() {
			basePath = filepath.ToSlash(basePath)
			blobPath = filepath.ToSlash(blobPath)
		})

		It("creates and closes the file", func() {
			fs_ := boshsysfake.NewFakeFileSystem()
			blobManager := NewBlobManager(fs_, basePath)
			err := blobManager.Write(blobId, toWrite)
			Expect(err).ToNot(HaveOccurred())
			fileStats, err := fs_.FindFileStats(blobPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileStats.Open).To(BeFalse())
		})

		It("creates file with correct permissions", func() {
			fs_ := boshsysfake.NewFakeFileSystem()
			blobManager := NewBlobManager(fs_, basePath)
			err := blobManager.Write(blobId, toWrite)
			fileStats, err := fs_.FindFileStats(blobPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(fileStats.FileMode).To(Equal(os.FileMode(0666)))
			Expect(fileStats.Flags).To(Equal(os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
		})
	})

})
