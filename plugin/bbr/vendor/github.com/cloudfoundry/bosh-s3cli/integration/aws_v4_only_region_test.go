package integration_test

import (
	"os"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing in any AWS region that only supports v4 signature version", func() {
	Context("with AWS V4 ONLY REGION (static creds) configurations", func() {
		It("fails with a config that specifies signature version 2", func() {
			accessKeyID := os.Getenv("ACCESS_KEY_ID")
			Expect(accessKeyID).ToNot(BeEmpty(), "ACCESS_KEY_ID must be set")

			secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
			Expect(secretAccessKey).ToNot(BeEmpty(), "SECRET_ACCESS_KEY must be set")

			bucketName := os.Getenv("BUCKET_NAME")
			Expect(bucketName).ToNot(BeEmpty(), "BUCKET_NAME must be set")

			region := os.Getenv("REGION")
			Expect(region).ToNot(BeEmpty(), "REGION must be set")

			cfg := &config.S3Cli{
				SignatureVersion: 2,
				AccessKeyID:      accessKeyID,
				SecretAccessKey:  secretAccessKey,
				BucketName:       bucketName,
				Region:           region,
			}
			s3Filename := integration.GenerateRandomString()

			configPath := integration.MakeConfigFile(cfg)
			defer func() { _ = os.Remove(configPath) }()

			contentFile := integration.MakeContentFile("test")
			defer func() { _ = os.Remove(contentFile) }()

			s3CLISession, err := integration.RunS3CLI(s3CLIPath, configPath, "put", contentFile, s3Filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(s3CLISession.ExitCode()).ToNot(BeZero())

			s3CLISession, err = integration.RunS3CLI(s3CLIPath, configPath, "delete", s3Filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(s3CLISession.ExitCode()).ToNot(BeZero())
		})
	})
})
