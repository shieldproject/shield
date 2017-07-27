package releasedir_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	fakecrypto "github.com/cloudfoundry/bosh-cli/crypto/fakes"
	fakeblob "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	fakelogger "github.com/cloudfoundry/bosh-utils/logger/loggerfakes"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	. "github.com/cloudfoundry/bosh-cli/releasedir"
	fakereldir "github.com/cloudfoundry/bosh-cli/releasedir/releasedirfakes"
)

var _ = Describe("FSBlobsDir", func() {
	var (
		fs               *fakesys.FakeFileSystem
		reporter         *fakereldir.FakeBlobsDirReporter
		blobstore        *fakeblob.FakeDigestBlobstore
		digestCalculator *fakecrypto.FakeDigestCalculator
		blobsDir         FSBlobsDir
		logger           *fakelogger.FakeLogger
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		reporter = &fakereldir.FakeBlobsDirReporter{}
		blobstore = &fakeblob.FakeDigestBlobstore{}
		digestCalculator = fakecrypto.NewFakeDigestCalculator()
		logger = &fakelogger.FakeLogger{}
		blobsDir = NewFSBlobsDir(filepath.Join("/", "dir"), reporter, blobstore, digestCalculator, fs, logger)
	})

	Describe("Blobs", func() {
		act := func() ([]Blob, error) {
			return blobsDir.Blobs()
		}

		It("returns no blobs if blobs.yml is empty", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), "")

			blobs, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(BeEmpty())
		})

		It("returns parsed blobs", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
bosh-116.tgz:
  size: 133959511
  sha: 13ebc5850fcbde216ec32ab4354df53df76e4745
`+filepath.Join("dir", "file.tgz")+`:
  size: 133959000
  object_id: ea50bf88-52ca-4230-4ef3-ff22c3975d04
  sha: 2b86b5850fcbde216ec565b4354df53df76e4745
file2.tgz:
  size: 245959511
  object_id: dc21b23e-1e32-40f4-61fb-5c9db26f7375
  sha: 3456b5850fcbde216ec32ab4354df53395607042
`)

			blobs, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(blobs).To(Equal([]Blob{
				{
					Path: "bosh-116.tgz",
					Size: 133959511,
					SHA1: "13ebc5850fcbde216ec32ab4354df53df76e4745",
				},
				{
					Path:        filepath.Join("dir", "file.tgz"),
					Size:        133959000,
					BlobstoreID: "ea50bf88-52ca-4230-4ef3-ff22c3975d04",
					SHA1:        "2b86b5850fcbde216ec565b4354df53df76e4745",
				},
				{
					Path:        "file2.tgz",
					Size:        245959511,
					BlobstoreID: "dc21b23e-1e32-40f4-61fb-5c9db26f7375",
					SHA1:        "3456b5850fcbde216ec32ab4354df53395607042",
				},
			}))
		})

		It("returns error if blobs.yml is not found so that user initializes it explicitly", func() {
			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Reading blobs index"))
		})

		It("returns error if blobs.yml is not parseable", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), "-")

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unmarshalling blobs index"))
		})
	})

	Describe("SyncBlobs", func() {
		act := func(numOfParallelWorkers int) error {
			return blobsDir.SyncBlobs(numOfParallelWorkers)
		}

		BeforeEach(func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), filepath.Join("dir", "file-in-directory.tgz")+":"+`
  object_id: blob1
  size: 133
  sha: blob1sha
non-uploaded.tgz:
  size: 245
  sha: 345
file-in-root.tgz:
  object_id: blob2
  size: 245
  sha: blob2sha
already-downloaded.tgz:
  object_id: blob3
  size: 245
  sha: 1da283030f72f285fa9e05d597a528f08780c992
`)

			fs.WriteFileString(filepath.Join("/", "blob1-tmp"), "blob1-content")
			fs.WriteFileString(filepath.Join("/", "blob2-tmp"), "blob2-content")
			fs.WriteFileString(filepath.Join("/", "dir", "blobs", "already-downloaded.tgz"), "blob3-content")

			times := 0
			blobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (string, error) {
				defer func() { times += 1 }()
				return []string{filepath.Join("/", "blob1-tmp"), filepath.Join("/", "blob2-tmp")}[times], nil
			}
		})

		Context("Multiple workers used to download blobs", func() {
			It("downloads all blobs without local blob copy, skipping non-uploaded blobs", func() {
				blobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (fileName string, err error) {
					if blobID == "blob1" && digest.String() == "blob1sha" {
						return filepath.Join("/", "blob1-tmp"), nil
					} else if blobID == "blob2" && digest.String() == "blob2sha" {
						return filepath.Join("/", "blob2-tmp"), nil
					} else {
						panic("Received non-matching blobstore.Get call")
					}
				}

				blobsDir = NewFSBlobsDir(filepath.Join("/", "dir"), reporter, blobstore, digestCalculator, fs, logger)

				err := act(4)
				Expect(err).ToNot(HaveOccurred())

				Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeTrue())
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(Equal("blob1-content"))
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(Equal("blob2-content"))
			})
		})

		Context("A single worker to download blobs", func() {
			It("downloads all blobs without local blob copy, skipping non-uploaded blobs", func() {
				err := act(1)
				Expect(err).ToNot(HaveOccurred())

				id1, digest1 := blobstore.GetArgsForCall(0)
				Expect(id1).To(Equal("blob1"))
				Expect(digest1).To(Equal(boshcrypto.MustParseMultipleDigest("blob1sha")))

				id2, digest2 := blobstore.GetArgsForCall(1)
				Expect(id2).To(Equal("blob2"))
				Expect(digest2).To(Equal(boshcrypto.MustParseMultipleDigest("blob2sha")))

				Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeTrue())
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(Equal("blob1-content"))
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(Equal("blob2-content"))
			})

			It("reports downloaded blobs skipping already existing ones", func() {
				err := act(1)
				Expect(err).ToNot(HaveOccurred())

				{
					Expect(reporter.BlobDownloadStartedCallCount()).To(Equal(2))

					path, size, blobID, sha1 := reporter.BlobDownloadStartedArgsForCall(0)
					Expect(path).To(Equal(filepath.Join("dir", "file-in-directory.tgz")))
					Expect(size).To(Equal(int64(133)))
					Expect(blobID).To(Equal("blob1"))
					Expect(sha1).To(Equal("blob1sha"))

					path, size, blobID, sha1 = reporter.BlobDownloadStartedArgsForCall(1)
					Expect(path).To(Equal("file-in-root.tgz"))
					Expect(size).To(Equal(int64(245)))
					Expect(blobID).To(Equal("blob2"))
					Expect(sha1).To(Equal("blob2sha"))
				}

				{
					Expect(reporter.BlobDownloadFinishedCallCount()).To(Equal(2))

					path, blobID, err := reporter.BlobDownloadFinishedArgsForCall(0)
					Expect(path).To(Equal(filepath.Join("dir", "file-in-directory.tgz")))
					Expect(blobID).To(Equal("blob1"))
					Expect(err).ToNot(HaveOccurred())

					path, blobID, err = reporter.BlobDownloadFinishedArgsForCall(1)
					Expect(path).To(Equal("file-in-root.tgz"))
					Expect(blobID).To(Equal("blob2"))
					Expect(err).ToNot(HaveOccurred())
				}
			})
		})

		Context("downloading fails", func() {
			It("reports error", func() {
				blobstore.GetReturns("", errors.New("fake-err"))

				err := act(1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Getting blob 'blob1' for path '" + filepath.Join("dir", "file-in-directory.tgz") + "': fake-err"))

				Expect(reporter.BlobDownloadStartedCallCount()).To(Equal(2))
				Expect(reporter.BlobDownloadFinishedCallCount()).To(Equal(2))
			})

			Context("when more than one blob fails to download", func() {
				It("reports error", func() {
					times := 0
					blobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (string, error) {
						defer func() { times += 1 }()
						return []string{filepath.Join("/", "blob1-tmp"), filepath.Join("/", "blob2-tmp")}[times], []error{errors.New("fake-err1"), errors.New("fake-err2")}[times]
					}

					err := act(1)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Getting blob 'blob1' for path '" + filepath.Join("dir", "file-in-directory.tgz") + "': fake-err1"))
					Expect(err.Error()).To(ContainSubstring("Getting blob 'blob2' for path 'file-in-root.tgz': fake-err2"))

					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeFalse())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(BeFalse())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(BeFalse())

				})
			})

			Context("without creating any blob sub-dirs", func() {
				It("returns error", func() {
					blobstore.GetReturns("", errors.New("fake-err"))

					err := act(1)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeFalse())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(BeFalse())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(BeFalse())
				})
			})

			Context("without placing any local blobs", func() {
				It("returns error", func() {
					times := 0
					blobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (string, error) {
						defer func() { times += 1 }()
						path := []string{filepath.Join("/", "blob1-tmp"), filepath.Join("/", "blob2-tmp")}[times]
						err := []error{nil, errors.New("fake-err")}[times]
						return path, err
					}

					err := act(1)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeTrue())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(BeTrue())
					Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(BeFalse())
				})
			})
		})

		Context("parsing digest string for sha fails", func() {
			BeforeEach(func() {
				fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
bad-sha-blob.tgz:
  object_id: blob3
  size: 245
  sha: ''
`)
			})

			It("returns descriptive error", func() {
				err := act(1)
				Expect(err).To(MatchError(ContainSubstring("No digest algorithm found. Supported algorithms: sha1, sha256, sha512")))
			})
		})

		Context("when blobs already on disk have different sha than in index", func() {
			BeforeEach(func() {
				fs.WriteFileString(filepath.Join("/", "blob3-tmp"), "blob3-content")
				fs.WriteFileString(filepath.Join("/", "dir", "blobs", "already-downloaded.tgz"), "incorrect-blob3-content")

				times := 0
				blobstore.GetStub = func(blobID string, digest boshcrypto.Digest) (string, error) {
					defer func() { times += 1 }()
					return []string{filepath.Join("/", "blob3-tmp"), filepath.Join("/", "blob1-tmp"), filepath.Join("/", "blob2-tmp")}[times], nil
				}
			})

			It("downloads new copy from blobstore and logs an error", func() {
				err := act(1)
				Expect(err).ToNot(HaveOccurred())

				id3, digest3 := blobstore.GetArgsForCall(0)
				Expect(id3).To(Equal("blob3"))
				Expect(digest3).To(Equal(boshcrypto.MustParseMultipleDigest("1da283030f72f285fa9e05d597a528f08780c992")))

				id1, digest1 := blobstore.GetArgsForCall(1)
				Expect(id1).To(Equal("blob1"))
				Expect(digest1).To(Equal(boshcrypto.MustParseMultipleDigest("blob1sha")))

				id2, digest2 := blobstore.GetArgsForCall(2)
				Expect(id2).To(Equal("blob2"))
				Expect(digest2).To(Equal(boshcrypto.MustParseMultipleDigest("blob2sha")))

				Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir"))).To(BeTrue())
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file-in-directory.tgz"))).To(Equal("blob1-content"))
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "file-in-root.tgz"))).To(Equal("blob2-content"))
				Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "already-downloaded.tgz"))).To(Equal("blob3-content"))

				tag, message, _ := logger.ErrorArgsForCall(0)
				Expect(tag).To(Equal("releasedir.FSBlobsDir"))
				Expect(message).To(Equal("Incorrect SHA sum for blob at '" + filepath.Join("/", "dir", "blobs", "already-downloaded.tgz") + "'. Re-downloading from blobstore."))
			})
		})

		Context("when creating blob sub-dir fails", func() {
			It("returns error", func() {
				fs.MkdirAllError = errors.New("fake-err")

				err := act(1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when moving temp blob file across devices into its final destination", func() {
			BeforeEach(func() {
				fs.RenameError = &os.LinkError{
					Err: syscall.Errno(0x12),
				}
			})

			It("downloads all blobs without local blob copy", func() {
				err := act(1)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when copying blobs across devices fails", func() {
				It("returns error", func() {
					fs.CopyFileError = errors.New("failed to copy")

					err := act(1)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("failed to copy"))
				})
			})
		})

		Context("when moving temp blob file into its final destination fails for an uncaught reason", func() {
			It("returns error", func() {
				fs.RenameError = errors.New("fake-err")

				err := act(1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when blobs exist on the file system which are not in the blobs.yml", func() {
			BeforeEach(func() {
				fs.SetGlob(filepath.Join("/", "dir", "blobs", "**", "*"), []string{filepath.Join("/", "dir", "blobs", "dir"), filepath.Join("/", "dir", "blobs", "already-downloaded.tgz"), filepath.Join("/", "dir", "blobs", "extra-blob.tgz")})
				fs.MkdirAll(filepath.Join("/", "dir", "blobs", "dir"), os.ModeDir)
				fs.WriteFileString(filepath.Join("/", "dir", "blobs", "extra-blob.tgz"), "I don't belong here")
			})

			It("deletes the blobs in the blob dir, logging a warning for each file deleted, and leaving correct blobs and directories", func() {
				err := act(1)
				Expect(err).ToNot(HaveOccurred())
				Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "extra-blob.tgz"))).To(BeFalse())
				Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "already-downloaded.tgz"))).To(BeTrue())

				tag, message, _ := logger.InfoArgsForCall(0)
				Expect(tag).To(Equal("releasedir.FSBlobsDir"))
				Expect(message).To(Equal("Deleting blob at '" + filepath.Join("/", "dir", "blobs", "extra-blob.tgz") + "' that is not in the blob index."))
			})

			It("returns an error when the glob fails", func() {
				fs.GlobStub = func(string) ([]string, error) {
					return []string{}, errors.New("failed to glob")
				}
				err := act(1)
				Expect(err).To(MatchError("Syncing blobs: Checking for unknown blobs: failed to glob"))
			})

			It("returns an error when the unknown blob removal fails", func() {
				fs.RemoveAllStub = func(filename string) error {
					return fmt.Errorf("failed to remove %s", filename)
				}
				err := act(1)
				Expect(err).To(MatchError("Syncing blobs: Removing unknown blob: failed to remove " + filepath.Join("/", "dir", "blobs", "extra-blob.tgz")))
			})
		})
	})

	Describe("TrackBlob", func() {
		act := func() (Blob, error) {
			content := ioutil.NopCloser(strings.NewReader(string("content")))
			return blobsDir.TrackBlob(filepath.Join("dir", "file.tgz"), content)
		}

		BeforeEach(func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), "")

			fs.ReturnTempFile = fakesys.NewFakeFile(filepath.Join("/", "tmp-file"), fs)

			digestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
				filepath.Join("/", "tmp-file"): fakecrypto.CalculateInput{DigestStr: "contentsha1"},
			})
		})

		It("adds a blob to the list if it's not already tracked", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
file2.tgz:
  size: 245
  sha: 345
`)

			blob, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(blob).To(Equal(Blob{Path: filepath.Join("dir", "file.tgz"), Size: 7, SHA1: "contentsha1"}))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: filepath.Join("dir", "file.tgz"), Size: 7, SHA1: "contentsha1"},
				{Path: "file2.tgz", Size: 245, SHA1: "345"},
			}))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"))).To(Equal("content"))
		})

		It("updates blob record if it's already tracked", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), filepath.Join("dir", "file.tgz")+`:
  size: 133
  sha: 13e
file2.tgz:
  size: 245
  sha: 345
`)

			blob, err := act()
			Expect(err).ToNot(HaveOccurred())
			Expect(blob).To(Equal(Blob{Path: filepath.Join("dir", "file.tgz"), Size: 7, SHA1: "contentsha1"}))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: filepath.Join("dir", "file.tgz"), Size: 7, SHA1: "contentsha1"},
				{Path: "file2.tgz", Size: 245, SHA1: "345"},
			}))

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"))).To(Equal("content"))
		})

		It("overrides existing local blob copy", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"), "prev-content")

			_, err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.ReadFileString(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"))).To(Equal("content"))
		})

		It("returns error and does not update blobs.yml if temp file cannot be opened", func() {
			fs.TempFileError = errors.New("fake-err")

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(BeEmpty())
		})

		It("returns error and does not update blobs.yml if copying from src fails", func() {
			file := fakesys.NewFakeFile(filepath.Join("/", "tmp-file"), fs)
			file.WriteErr = errors.New("fake-err")
			fs.ReturnTempFile = file

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(BeEmpty())
		})

		It("returns error and does not update blobs.yml if cannot determine size", func() {
			file := fakesys.NewFakeFile(filepath.Join("/", "tmp-file"), fs)
			file.StatErr = errors.New("fake-err")
			fs.ReturnTempFile = file

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(BeEmpty())
		})

		It("returns error and does not update blobs.yml if calculating sha1 fails", func() {
			digestCalculator.SetCalculateBehavior(map[string]fakecrypto.CalculateInput{
				filepath.Join("/", "tmp-file"): fakecrypto.CalculateInput{Err: errors.New("fake-err")},
			})

			_, err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(BeEmpty())
		})
	})

	Describe("UntrackBlob", func() {
		act := func() error {
			return blobsDir.UntrackBlob(filepath.Join("dir", "file.tgz"))
		}

		It("removes reference from list of blobs (first)", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), filepath.Join("dir", "file.tgz")+`:
  size: 133
  sha: 13e
file2.tgz:
  size: 245
  sha: 345
`)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "file2.tgz", Size: 245, SHA1: "345"},
			}))
		})

		It("removes reference from list of blobs (middle)", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
bosh-116.tgz:
  size: 133
  sha: 13e
`+filepath.Join("dir", "file.tgz")+`:
  size: 133
  sha: 2b8
file2.tgz:
  size: 245
  sha: 345
`)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "bosh-116.tgz", Size: 133, SHA1: "13e"},
				{Path: "file2.tgz", Size: 245, SHA1: "345"},
			}))
		})

		It("removes reference from list of blobs (last)", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
bosh-116.tgz:
  size: 133
  sha: 13e
`+filepath.Join("dir", "file.tgz")+`:
  size: 245
  sha: 345
`)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "bosh-116.tgz", Size: 133, SHA1: "13e"},
			}))
		})

		It("succeeds even if record is not found", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), `
bosh-116.tgz:
  size: 133
  sha: 13e
`)

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "bosh-116.tgz", Size: 133, SHA1: "13e"},
			}))
		})

		It("removes local blob copy", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), "")
			fs.WriteFileString(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"), "blob")

			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(fs.FileExists(filepath.Join("/", "dir", "blobs", "dir", "file.tgz"))).To(BeFalse())
		})

		It("returns error if removing local blob copy fails", func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), filepath.Join("dir", "file.tgz:")+`
  size: 133
  sha: 13e
`)

			fs.RemoveAllStub = func(_ string) error {
				return errors.New("fake-err")
			}

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: filepath.Join("dir", "file.tgz"), Size: 133, SHA1: "13e"},
			}))
		})
	})

	Describe("UploadBlobs", func() {
		act := func() error {
			return blobsDir.UploadBlobs()
		}

		BeforeEach(func() {
			fs.WriteFileString(filepath.Join("/", "dir", "config", "blobs.yml"), filepath.Join("dir", "file-in-directory.tgz")+`:
  object_id: blob1
  size: 133
  sha: blob1sha
non-uploaded.tgz:
  size: 243
  sha: blob2sha
file-in-root.tgz:
  object_id: blob3
  size: 245
  sha: blob3sha
already-downloaded.tgz:
  object_id: blob4
  size: 245
  sha: blob4sha
non-uploaded2.tgz:
  size: 245
  sha: blob5sha
`)

			times := 0
			blobstore.CreateStub = func(fileName string) (string, boshcrypto.MultipleDigest, error) {
				defer func() { times += 1 }()
				multiDigest := boshcrypto.MustNewMultipleDigest(
					boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "whatever"),
				)
				return []string{"blob2", "blob5"}[times], multiDigest, nil
			}
		})

		It("uploads non-uploaded blobs", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			Expect(blobstore.CreateArgsForCall(0)).To(Equal(filepath.Join("/", "dir", "blobs", "non-uploaded.tgz")))
			Expect(blobstore.CreateArgsForCall(1)).To(Equal(filepath.Join("/", "dir", "blobs", "non-uploaded2.tgz")))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "already-downloaded.tgz", Size: 245, BlobstoreID: "blob4", SHA1: "blob4sha"},
				{Path: filepath.Join("dir", "file-in-directory.tgz"), Size: 133, BlobstoreID: "blob1", SHA1: "blob1sha"},
				{Path: "file-in-root.tgz", Size: 245, BlobstoreID: "blob3", SHA1: "blob3sha"},
				{Path: "non-uploaded.tgz", Size: 243, BlobstoreID: "blob2", SHA1: "blob2sha"},
				{Path: "non-uploaded2.tgz", Size: 245, BlobstoreID: "blob5", SHA1: "blob5sha"},
			}))
		})

		It("reports uploaded blobs skipping already existing ones", func() {
			err := act()
			Expect(err).ToNot(HaveOccurred())

			{
				Expect(reporter.BlobUploadStartedCallCount()).To(Equal(2))

				path, size, sha1 := reporter.BlobUploadStartedArgsForCall(0)
				Expect(path).To(Equal("non-uploaded.tgz"))
				Expect(size).To(Equal(int64(243)))
				Expect(sha1).To(Equal("blob2sha"))

				path, size, sha1 = reporter.BlobUploadStartedArgsForCall(1)
				Expect(path).To(Equal("non-uploaded2.tgz"))
				Expect(size).To(Equal(int64(245)))
				Expect(sha1).To(Equal("blob5sha"))
			}

			{
				Expect(reporter.BlobUploadFinishedCallCount()).To(Equal(2))

				path, blobID, err := reporter.BlobUploadFinishedArgsForCall(0)
				Expect(path).To(Equal("non-uploaded.tgz"))
				Expect(blobID).To(Equal("blob2"))
				Expect(err).ToNot(HaveOccurred())

				path, blobID, err = reporter.BlobUploadFinishedArgsForCall(1)
				Expect(path).To(Equal("non-uploaded2.tgz"))
				Expect(blobID).To(Equal("blob5"))
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("returns error if uploading fails and does not change blobs.yml", func() {
			blobstore.CreateReturns("", boshcrypto.MultipleDigest{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "already-downloaded.tgz", Size: 245, BlobstoreID: "blob4", SHA1: "blob4sha"},
				{Path: filepath.Join("dir", "file-in-directory.tgz"), Size: 133, BlobstoreID: "blob1", SHA1: "blob1sha"},
				{Path: "file-in-root.tgz", Size: 245, BlobstoreID: "blob3", SHA1: "blob3sha"},
				{Path: "non-uploaded.tgz", Size: 243, SHA1: "blob2sha"},
				{Path: "non-uploaded2.tgz", Size: 245, SHA1: "blob5sha"},
			}))
		})

		It("reports error if uploading fails", func() {
			blobstore.CreateReturns("", boshcrypto.MultipleDigest{}, errors.New("fake-err"))

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(reporter.BlobUploadStartedCallCount()).To(Equal(1))
			Expect(reporter.BlobUploadFinishedCallCount()).To(Equal(1))

			path, size, sha1 := reporter.BlobUploadStartedArgsForCall(0)
			Expect(path).To(Equal("non-uploaded.tgz"))
			Expect(size).To(Equal(int64(243)))
			Expect(sha1).To(Equal("blob2sha"))

			path, blobID, err := reporter.BlobUploadFinishedArgsForCall(0)
			Expect(path).To(Equal("non-uploaded.tgz"))
			Expect(blobID).To(Equal(""))
			Expect(err).To(HaveOccurred())
		})

		It("returns if saving blobstore id fails and does not continue to upload other blobs", func() {
			fs.WriteFileError = errors.New("fake-err")

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			// Include blobstore id in error message for cleanup purposes
			Expect(err.Error()).To(ContainSubstring("Saving newly created blob 'blob2'"))

			Expect(reporter.BlobUploadStartedCallCount()).To(Equal(1))
		})

		It("returns error if uploading fails and saves blob id for successfully uploaded blobs", func() {
			times := 0
			blobstore.CreateStub = func(fileName string) (string, boshcrypto.MultipleDigest, error) {
				defer func() { times += 1 }()
				blobID := []string{"blob2", "blob5"}[times]
				err := []error{nil, errors.New("fake-err")}[times]
				return blobID, boshcrypto.MultipleDigest{}, err
			}

			err := act()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(blobsDir.Blobs()).To(Equal([]Blob{
				{Path: "already-downloaded.tgz", Size: 245, BlobstoreID: "blob4", SHA1: "blob4sha"},
				{Path: filepath.Join("dir", "file-in-directory.tgz"), Size: 133, BlobstoreID: "blob1", SHA1: "blob1sha"},
				{Path: "file-in-root.tgz", Size: 245, BlobstoreID: "blob3", SHA1: "blob3sha"},
				{Path: "non-uploaded.tgz", Size: 243, BlobstoreID: "blob2", SHA1: "blob2sha"},
				{Path: "non-uploaded2.tgz", Size: 245, SHA1: "blob5sha"},
			}))
		})
	})
})
