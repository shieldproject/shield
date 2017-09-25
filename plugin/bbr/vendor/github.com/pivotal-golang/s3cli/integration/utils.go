package integration

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os/exec"
	"time"

	"github.com/cloudfoundry/bosh-s3cli/config"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const alphanum = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// GenerateRandomString generates a random string of desired length (default: 25)
func GenerateRandomString(params ...int) string {
	size := 25
	if len(params) == 1 {
		size = params[0]
	}

	randBytes := make([]byte, size)
	for i := range randBytes {
		randBytes[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(randBytes)
}

// MakeConfigFile creates a config file from a S3Cli config struct
func MakeConfigFile(cfg *config.S3Cli) string {
	cfgBytes, err := json.Marshal(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	tmpFile, err := ioutil.TempFile("", "s3cli-test")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	_, err = tmpFile.Write(cfgBytes)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = tmpFile.Close()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return tmpFile.Name()
}

// MakeContentFile creates a temporary file with content to upload to S3
func MakeContentFile(content string) string {
	tmpFile, err := ioutil.TempFile("", "s3cli-test-content")
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	_, err = tmpFile.Write([]byte(content))
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	err = tmpFile.Close()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return tmpFile.Name()
}

// RunS3CLI runs the s3cli and outputs the session after waiting for it to finish
func RunS3CLI(s3CLIPath string, configPath string, subcommand string, args ...string) (*gexec.Session, error) {
	cmdArgs := []string{
		"-c",
		configPath,
		subcommand,
	}
	cmdArgs = append(cmdArgs, args...)
	command := exec.Command(s3CLIPath, cmdArgs...)
	gexecSession, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return nil, err
	}
	gexecSession.Wait(1 * time.Minute)
	return gexecSession, nil
}
