package integration_test

import (
	"os"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing inside an AWS compute resource with an IAM role", func() {
	Context("with AWS STANDARD IAM ROLE (env_or_profile creds) configurations", func() {
		bucketName := os.Getenv("BUCKET_NAME")
		region := os.Getenv("REGION")
		s3Host := os.Getenv("S3_HOST")

		BeforeEach(func() {
			Expect(bucketName).ToNot(BeEmpty(), "BUCKET_NAME must be set")
			Expect(region).ToNot(BeEmpty(), "REGION must be set")
			Expect(s3Host).ToNot(BeEmpty(), "S3_HOST must be set")
		})

		configurations := []TableEntry{
			Entry("with minimal config", &config.S3Cli{
				CredentialsSource: "env_or_profile",
				BucketName:        bucketName,
			}),
			Entry("with region and without host, signature version 4", &config.S3Cli{
				SignatureVersion:  4,
				CredentialsSource: "env_or_profile",
				BucketName:        bucketName,
				Region:            region,
			}),
			Entry("with maximal config, signature version 4", &config.S3Cli{
				SignatureVersion:  4,
				CredentialsSource: "env_or_profile",
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
