package blobextract_test

import (
	"errors"
	"os"

	. "github.com/cloudfoundry/bosh-init/installation/blobextract"
	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Extractor", func() {
	var (
		extractor  Extractor
		blobstore  *fakeblobstore.FakeBlobstore
		targetDir  string
		compressor *fakecmd.FakeCompressor
		logger     boshlog.Logger
		fs         *fakesys.FakeFileSystem

		blobID    string
		blobSHA1  string
		fileName  string
		fakeError error
	)

	BeforeEach(func() {
		blobstore = fakeblobstore.NewFakeBlobstore()
		targetDir = "fake-target-dir"
		compressor = fakecmd.NewFakeCompressor()
		logger = boshlog.NewLogger(boshlog.LevelNone)
		fs = fakesys.NewFakeFileSystem()
		blobID = "fake-blob-id"
		blobSHA1 = "fake-sha1"
		fileName = "tarball.tgz"
		blobstore.GetFileName = fileName
		fakeError = errors.New("Initial error")

		extractor = NewExtractor(fs, compressor, blobstore, logger)
	})

	Describe("Cleanup", func() {
		BeforeEach(func() {
			err := extractor.Extract(blobID, blobSHA1, targetDir)
			Expect(err).ToNot(HaveOccurred())
		})

		It("deletes the extracted temp file", func() {
			Expect(fs.FileExists(targetDir)).To(BeTrue())
			err := extractor.Cleanup(blobID, targetDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.FileExists(targetDir)).To(BeFalse())
		})

		It("deletes the stored blob", func() {
			err := extractor.Cleanup(blobID, targetDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(blobstore.DeleteBlobID).To(Equal(blobID))
		})
	})

	Describe("Extract", func() {
		Context("when the specified blobID exists in the blobstore", func() {
			It("creates the installed package dir if it does not exist", func() {
				Expect(fs.FileExists(targetDir)).To(BeFalse())
				err := extractor.Extract(blobID, blobSHA1, targetDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(fs.FileExists(targetDir)).To(BeTrue())
			})

			It("decompresses the blob into the target dir", func() {
				err := extractor.Extract(blobID, blobSHA1, targetDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(compressor.DecompressFileToDirTarballPaths).To(ContainElement(fileName))
				Expect(compressor.DecompressFileToDirDirs).To(ContainElement(targetDir))
			})

			It("cleans up the extracted blob file", func() {
				err := extractor.Extract(blobID, blobSHA1, targetDir)
				Expect(err).ToNot(HaveOccurred())
				Expect(blobstore.CleanUpFileName).To(Equal(fileName))
			})

			Context("when the installed package dir already exists", func() {
				BeforeEach(func() {
					fs.MkdirAll(targetDir, os.ModePerm)
				})

				It("decompresses the blob into the target dir", func() {
					Expect(fs.FileExists(targetDir)).To(BeTrue())
					Expect(compressor.DecompressFileToDirTarballPaths).ToNot(ContainElement(fileName))

					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).ToNot(HaveOccurred())
					Expect(fs.FileExists(targetDir)).To(BeTrue())
					Expect(compressor.DecompressFileToDirTarballPaths).To(ContainElement(fileName))
				})

				It("does not re-create the target package dir", func() {
					fs.MkdirAllError = fakeError
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).ToNot(HaveOccurred())
				})

				Context("and decompressing the blob fails", func() {
					It("returns an error and doesn't remove the target dir", func() {
						compressor.DecompressFileToDirErr = fakeError
						Expect(fs.FileExists(targetDir)).To(BeTrue())
						err := extractor.Extract(blobID, blobSHA1, targetDir)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(Equal("Decompressing compiled package: BlobID: 'fake-blob-id', BlobSHA1: 'fake-sha1': Initial error"))
						Expect(fs.FileExists(targetDir)).To(BeTrue())
					})
				})
			})

			Context("when getting the blob from the blobstore errors", func() {
				BeforeEach(func() {
					blobstore.GetError = fakeError
				})

				It("returns an error", func() {
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Getting object from blobstore: fake-blob-id: Initial error"))
				})
			})

			Context("when creating the target dir fails", func() {
				It("return an error", func() {
					fs.MkdirAllError = fakeError
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Creating target dir: fake-target-dir: Initial error"))
				})

				It("cleans up the blob file", func() {
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).ToNot(HaveOccurred())
					Expect(blobstore.CleanUpFileName).To(Equal(fileName))
				})
			})

			Context("when decompressing the blob fails", func() {
				BeforeEach(func() {
					compressor.DecompressFileToDirErr = fakeError
				})

				It("returns an error", func() {
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Decompressing compiled package: BlobID: 'fake-blob-id', BlobSHA1: 'fake-sha1'"))
				})

				It("cleans up the target dir if it was created by this extractor", func() {
					Expect(fs.FileExists(targetDir)).To(BeFalse())
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).To(HaveOccurred())
					Expect(fs.FileExists(targetDir)).To(BeFalse())
				})
			})

			Context("when cleaning up the downloaded blob errors", func() {
				BeforeEach(func() {
					blobstore.CleanUpErr = fakeError
				})

				It("does not return the error", func() {
					err := extractor.Extract(blobID, blobSHA1, targetDir)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
	})

	Describe("ChmodExecutables", func() {
		var (
			binGlob  string
			filePath string
		)

		BeforeEach(func() {
			binGlob = "fake-glob/*"
			filePath = "fake-glob/file"
			fs.SetGlob("fake-glob/*", []string{filePath})
			fs.WriteFileString(filePath, "content")
		})

		It("fetches the files", func() {
			fileMode := fs.GetFileTestStat(filePath).FileMode
			Expect(fileMode).To(Equal(os.FileMode(0)))

			err := extractor.ChmodExecutables(binGlob)
			Expect(err).ToNot(HaveOccurred())

			fileMode = fs.GetFileTestStat(filePath).FileMode
			Expect(fileMode).To(Equal(os.FileMode(0755)))
		})
	})
})
