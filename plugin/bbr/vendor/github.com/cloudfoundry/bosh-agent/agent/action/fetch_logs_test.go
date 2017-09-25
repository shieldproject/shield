package action_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
	fakeblobstore "github.com/cloudfoundry/bosh-utils/blobstore/fakes"
	boshcrypto "github.com/cloudfoundry/bosh-utils/crypto"
	fakecmd "github.com/cloudfoundry/bosh-utils/fileutil/fakes"
)

var _ = Describe("FetchLogsAction", func() {
	var (
		compressor  *fakecmd.FakeCompressor
		copier      *fakecmd.FakeCopier
		blobstore   *fakeblobstore.FakeDigestBlobstore
		dirProvider boshdirs.Provider
		action      FetchLogsAction
	)

	BeforeEach(func() {
		compressor = fakecmd.NewFakeCompressor()
		blobstore = &fakeblobstore.FakeDigestBlobstore{}
		dirProvider = boshdirs.NewProvider("/fake/dir")
		copier = fakecmd.NewFakeCopier()
		action = NewFetchLogs(compressor, copier, blobstore, dirProvider)
	})

	AssertActionIsAsynchronous(action)
	AssertActionIsNotPersistent(action)
	AssertActionIsLoggable(action)

	AssertActionIsNotResumable(action)
	AssertActionIsNotCancelable(action)

	Describe("Run", func() {
		testLogs := func(logType string, filters []string, expectedFilters []string) {
			copier.FilteredCopyToTempTempDir = "/fake-temp-dir"
			compressor.CompressFilesInDirTarballPath = "logs_test.tar"
			multidigestSha := boshcrypto.MustNewMultipleDigest(boshcrypto.NewDigest(boshcrypto.DigestAlgorithmSHA1, "sec_dep_sha1"))
			sha1 := multidigestSha.String()
			blobstore.CreateStub = func(fileName string) (blobID string, digest boshcrypto.MultipleDigest, err error) {
				return "my-blob-id", multidigestSha, nil
			}

			logs, err := action.Run(logType, filters)
			Expect(err).ToNot(HaveOccurred())

			var expectedPath string
			switch logType {
			case "job":
				expectedPath = filepath.Join("/fake", "dir", "sys", "log")
			case "agent":
				expectedPath = filepath.Join("/fake", "dir", "bosh", "log")
			}

			Expect(copier.FilteredCopyToTempDir).To(boshassert.MatchPath(expectedPath))
			Expect(copier.FilteredCopyToTempFilters).To(Equal(expectedFilters))

			Expect(copier.FilteredCopyToTempTempDir).To(Equal(compressor.CompressFilesInDirDir))
			Expect(copier.CleanUpTempDir).To(Equal(compressor.CompressFilesInDirDir))

			Expect(compressor.CompressFilesInDirTarballPath).To(Equal(blobstore.CreateArgsForCall(0)))

			boshassert.MatchesJSONString(GinkgoT(), logs, `{"blobstore_id":"my-blob-id","sha1":"`+sha1+`"}`)
		}

		It("logs errs if given invalid log type", func() {
			_, err := action.Run("other-logs", []string{})
			Expect(err).To(HaveOccurred())
		})

		It("agent logs with filters", func() {
			filters := []string{"**/*.stdout.log", "**/*.stderr.log"}
			expectedFilters := []string{"**/*.stdout.log", "**/*.stderr.log"}
			testLogs("agent", filters, expectedFilters)
		})

		It("agent logs without filters", func() {
			filters := []string{}
			expectedFilters := []string{"**/*"}
			testLogs("agent", filters, expectedFilters)
		})

		It("job logs without filters", func() {
			filters := []string{}
			expectedFilters := []string{"**/*"}
			testLogs("job", filters, expectedFilters)
		})

		It("job logs with filters", func() {
			filters := []string{"**/*.stdout.log", "**/*.stderr.log"}
			expectedFilters := []string{"**/*.stdout.log", "**/*.stderr.log"}
			testLogs("job", filters, expectedFilters)
		})

		It("cleans up compressed package after uploading it to blobstore", func() {
			var beforeCleanUpTarballPath, afterCleanUpTarballPath string

			compressor.CompressFilesInDirTarballPath = "/fake-compressed-logs.tar"

			blobstore.CreateStub = func(fileName string) (blobID string, digest boshcrypto.MultipleDigest, err error) {
				beforeCleanUpTarballPath = compressor.CleanUpTarballPath

				return "my-blob-id", boshcrypto.MultipleDigest{}, nil
			}

			_, err := action.Run("job", []string{})
			Expect(err).ToNot(HaveOccurred())

			// Logs are not cleaned up before blobstore upload
			Expect(beforeCleanUpTarballPath).To(Equal(""))

			// Deleted after it was uploaded
			afterCleanUpTarballPath = compressor.CleanUpTarballPath
			Expect(afterCleanUpTarballPath).To(Equal("/fake-compressed-logs.tar"))
		})
	})
})
