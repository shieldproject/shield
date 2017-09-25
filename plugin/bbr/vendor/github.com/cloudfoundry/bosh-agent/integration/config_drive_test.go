package integration_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("ConfigDrive", func() {
	Context("when vm is using config drive", func() {
		BeforeEach(func() {
			err := testEnvironment.StopAgent()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.SetupConfigDrive()
			Expect(err).ToNot(HaveOccurred())

			registrySettings := boshsettings.Settings{
				AgentID: "fake-agent-id",
			}

			err = testEnvironment.StartRegistry(registrySettings)
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.UpdateAgentConfig("config-drive-agent.json")
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.StartAgent()
			Expect(err).ToNot(HaveOccurred())
		})

		It("using config drive to get registry URL", func() {
			settingsJSON, err := testEnvironment.GetFileContents("/var/vcap/bosh/settings.json")
			Expect(err).ToNot(HaveOccurred())
			Expect(settingsJSON).To(ContainSubstring("fake-agent-id"))
		})

		It("config drive is being unmounted", func() {
			Eventually(func() string {
				result, _ := testEnvironment.RunCommand("sudo mount | grep -c /dev/loop2")
				return strings.TrimSpace(result)
			}, 5*time.Second, 1*time.Second).Should(Equal("0"))
		})
	})
})
