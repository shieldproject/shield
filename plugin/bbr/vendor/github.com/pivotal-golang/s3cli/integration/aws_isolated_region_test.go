package integration_test

import (
	"os"

	"github.com/cloudfoundry/bosh-s3cli/config"
	"github.com/cloudfoundry/bosh-s3cli/integration"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing in any AWS region isolated from the US standard regions (i.e., cn-north-1)", func() {
	Context("with AWS ISOLATED REGION (static creds) configurations", func() {
		It("fails with a config that specifies a valid region but invalid host", func() {
			accessKeyID := os.Getenv("ACCESS_KEY_ID")
			Expect(accessKeyID).ToNot(BeEmpty(), "ACCESS_KEY_ID must be set")

			secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
			Expect(secretAccessKey).ToNot(BeEmpty(), "SECRET_ACCESS_KEY must be set")

			bucketName := os.Getenv("BUCKET_NAME")
			Expect(bucketName).ToNot(BeEmpty(), "BUCKET_NAME must be set")

			region := os.Getenv("REGION")
			Expect(region).ToNot(BeEmpty(), "REGION must be set")

			cfg := &config.S3Cli{
				SignatureVersion:  4,
				CredentialsSource: "static",
				AccessKeyID:       accessKeyID,
				SecretAccessKey:   secretAccessKey,
				BucketName:        bucketName,
				Region:            region,
				Host:              "s3-external-1.amazonaws.com",
			}
			s3Filename := integration.GenerateRandomString()

			configPath := integration.MakeConfigFile(cfg)
			defer func() { _ = os.Remove(configPath) }()

			contentFile := integration.MakeContentFile("test")
			defer func() { _ = os.Remove(contentFile) }()

			s3CLISession, err := integration.RunS3CLI(s3CLIPath, configPath, "put", contentFile, s3Filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(s3CLISession.ExitCode()).ToNot(BeZero())
			Expect(s3CLISession.Err.Contents()).To(ContainSubstring("AuthorizationHeaderMalformed"))

			s3CLISession, err = integration.RunS3CLI(s3CLIPath, configPath, "delete", s3Filename)
			Expect(err).ToNot(HaveOccurred())
			Expect(s3CLISession.ExitCode()).ToNot(BeZero())
			Expect(s3CLISession.Err.Contents()).To(ContainSubstring("AuthorizationHeaderMalformed"))
		})
	})
})
