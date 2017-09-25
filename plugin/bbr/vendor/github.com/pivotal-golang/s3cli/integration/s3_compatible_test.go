package integration_test

import (
	"os"
	"strconv"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing in any non-AWS, S3 compatible storage service", func() {
	Context("with S3 COMPATIBLE (static creds) configurations", func() {
		accessKeyID := os.Getenv("ACCESS_KEY_ID")
		secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
		bucketName := os.Getenv("BUCKET_NAME")
		s3Host := os.Getenv("S3_HOST")
		s3PortString := os.Getenv("S3_PORT")
		s3Port, atoiErr := strconv.Atoi(s3PortString)

		BeforeEach(func() {
			Expect(accessKeyID).ToNot(BeEmpty(), "ACCESS_KEY_ID must be set")
			Expect(secretAccessKey).ToNot(BeEmpty(), "SECRET_ACCESS_KEY must be set")
			Expect(bucketName).ToNot(BeEmpty(), "BUCKET_NAME must be set")
			Expect(s3Host).ToNot(BeEmpty(), "S3_HOST must be set")
			Expect(s3PortString).ToNot(BeEmpty(), "S3_PORT must be set")
			Expect(atoiErr).ToNot(HaveOccurred())
		})

		configurations := []TableEntry{
			Entry("with the minimal configuration", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Host:            s3Host,
			}),
			Entry("with region specified", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Host:            s3Host,
				Region:          "invalid-region",
			}),
			Entry("with use_ssl set to false", &config.S3Cli{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				BucketName:      bucketName,
				Host:            s3Host,
				UseSSL:          false,
			}),
			Entry("with the maximal configuration", &config.S3Cli{
				SignatureVersion:  2,
				CredentialsSource: "static",
				AccessKeyID:       accessKeyID,
				SecretAccessKey:   secretAccessKey,
				BucketName:        bucketName,
				Host:              s3Host,
				Port:              s3Port,
				UseSSL:            true,
				SSLVerifyPeer:     true,
				Region:            "invalid-region",
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
		DescribeTable("Invoking `s3cli put` handling of mulitpart uploads",
			func(cfg *config.S3Cli) { integration.AssertOnMultipartUploads(s3CLIPath, cfg, largeContent) },
			configurations...,
		)
	})
})
