package integration_test

import (
	"os"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("General testing for all AWS regions", func() {
	Context("with GENERAL AWS (static creds) configurations", func() {
		accessKeyID := os.Getenv("ACCESS_KEY_ID")
		secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
		bucketName := os.Getenv("BUCKET_NAME")
		region := os.Getenv("REGION")
		s3Host := os.Getenv("S3_HOST")

		BeforeEach(func() {
			Expect(accessKeyID).ToNot(BeEmpty(), "ACCESS_KEY_ID must be set")
			Expect(secretAccessKey).ToNot(BeEmpty(), "SECRET_ACCESS_KEY must be set")
			Expect(bucketName).ToNot(BeEmpty(), "BUCKET_NAME must be set")
			Expect(region).ToNot(BeEmpty(), "REGION must be set")
			Expect(s3Host).ToNot(BeEmpty(), "S3_HOST must be set")
		})

		configurations := []TableEntry{
			Entry("with region and without host", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Region:          region,
			}),
			Entry("with host and without region", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Host:            s3Host,
			}),
			Entry("with folder", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				FolderName:      "test-folder/a-folder",
				Region:          region,
			}),
		}
		DescribeTable("Blobstore lifecycle works",
			func(cfg *config.S3Cli) { integration.AssertLifecycleWorks(s3CLIPath, cfg) },
			configurations...,
		)
		DescribeTable("Invoking `s3cli get` on a non-existent-key fails",
			func(cfg *config.S3Cli) { integration.AssertGetNonexistentFails(s3CLIPath, cfg) },
			configurations...,
		)
		DescribeTable("Invoking `s3cli delete` on a non-existent-key does not fail",
			func(cfg *config.S3Cli) { integration.AssertDeleteNonexistentWorks(s3CLIPath, cfg) },
			configurations...,
		)

		configurations = []TableEntry{
			Entry("with encryption", &config.S3Cli{
				AccessKeyID:          accessKeyID,
				SecretAccessKey:      secretAccessKey,
				BucketName:           bucketName,
				Region:               region,
				ServerSideEncryption: "AES256",
			}),
			Entry("without encryption", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Region:          region,
			}),
		}
		DescribeTable("Invoking `s3cli put` uploads with options",
			func(cfg *config.S3Cli) { integration.AssertPutOptionsApplied(s3CLIPath, cfg) },
			configurations...,
		)

		Describe("Invoking `s3cli put` with arbitrary upload failures", func() {
			It("returns the appropriate error message", func() {
				cfg := &config.S3Cli{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
					BucketName:      bucketName,
					Host:            "localhost",
				}
				msg := "upload failure"
				integration.AssertOnPutFailures(s3CLIPath, cfg, largeContent, msg)
			})
		})

		Describe("Invoking `s3cli put` with multipart upload failures", func() {
			It("returns the appropriate error message", func() {
				cfg := &config.S3Cli{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
					BucketName:      bucketName,
					Region:          region,
				}
				msg := "upload retry limit exceeded"
				integration.AssertOnPutFailures(s3CLIPath, cfg, largeContent, msg)
			})
		})
	})
})
