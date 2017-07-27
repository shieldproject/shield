package integration_test

import (
	"io/ioutil"
	"os"
	"testing"

	bitestutils "github.com/cloudfoundry/bosh-cli/testutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	originalHome string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	var (
		homePath string
	)

	BeforeEach(func() {
		originalHome = os.Getenv("HOME")

		var err error
		homePath, err = ioutil.TempDir("", "bosh-init-cli-integration")
		Expect(err).NotTo(HaveOccurred())

		err = os.Setenv("HOME", homePath)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.Setenv("HOME", originalHome)
		Expect(err).NotTo(HaveOccurred())

		err = os.RemoveAll(homePath)
		Expect(err).NotTo(HaveOccurred())
	})

	RunSpecs(t, "integration")
}

var _ = BeforeSuite(func() {
	err := bitestutils.BuildExecutable()
	Expect(err).NotTo(HaveOccurred())
})
