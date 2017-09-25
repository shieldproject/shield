package integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("version flag", func() {
	It("returns the version string and exits clean", func() {
		stdout, stderr, status, err := testEnvironment.RunCommand3("sudo /var/vcap/bosh/bin/bosh-agent -v")
		Expect(err).ToNot(HaveOccurred())

		Expect(stdout).To(ContainSubstring("[DEV BUILD]"))
		Expect(stderr).To(BeEmpty())
		Expect(status).To(Equal(0))
	})
})
