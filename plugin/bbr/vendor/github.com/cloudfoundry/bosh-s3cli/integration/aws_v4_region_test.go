package integration_test

import (
	"os"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing in any AWS region that supports v4 signature version", func() {
	Context("with AWS V4 REGION (static creds) configurations", func() {
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
				SignatureVersion:  4,
				CredentialsSource: "static",
				AccessKeyID:       accessKeyID,
				SecretAccessKey:   secretAccessKey,
				BucketName:        bucketName,
				Region:            region,
			}),
			Entry("with host and without region", &config.S3Cli{
				SignatureVersion:  4,
				CredentialsSource: "static",
				AccessKeyID:       accessKeyID,
				SecretAccessKey:   secretAccessKey,
				BucketName:        bucketName,
				Host:              s3Host,
			}),
			Entry("with maximal config", &config.S3Cli{
				SignatureVersion:  4,
				CredentialsSource: "static",
				AccessKeyID:       accessKeyID,
				SecretAccessKey:   secretAccessKey,
				BucketName:        bucketName,
				Host:              s3Host,
				Port:              443,
				UseSSL:            true,
				SSLVerifyPeer:     true,
				Region:            region,
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
	})
})
