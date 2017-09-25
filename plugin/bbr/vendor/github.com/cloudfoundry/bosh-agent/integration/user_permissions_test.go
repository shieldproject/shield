package integration_test

import (
	"github.com/cloudfoundry/bosh-agent/agent/action"
	"github.com/cloudfoundry/bosh-agent/integration/integrationagentclient"
	"github.com/cloudfoundry/bosh-agent/settings"

	"strings"

	"github.com/cloudfoundry/bosh-agent/integration"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Instance Info", func() {
	var (
		agentClient      *integrationagentclient.IntegrationAgentClient
		registrySettings settings.Settings
	)

	BeforeEach(func() {
		err := testEnvironment.StopAgent()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupDataDir()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupLogFile()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupSSH()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.SetupConfigDrive()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.UpdateAgentConfig("config-drive-agent.json")
		Expect(err).ToNot(HaveOccurred())

		networks, err := testEnvironment.GetVMNetworks()
		Expect(err).ToNot(HaveOccurred())

		registrySettings = settings.Settings{
			AgentID: "fake-agent-id",

			// note that this SETS the username and password for HTTP message bus access
			Mbus: "https://mbus-user:mbus-pass@127.0.0.1:6868",

			Blobstore: settings.Blobstore{
				Type: "local",
				Options: map[string]interface{}{
					"blobstore_path": "/var/vcap/data",
				},
			},

			Disks: settings.Disks{
				Ephemeral: "/dev/sdh",
			},
			Networks: networks,
		}

		err = testEnvironment.AttachDevice("/dev/sdh", 128, 2)
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.StartRegistry(registrySettings)
		Expect(err).ToNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		err := testEnvironment.StartAgent()
		Expect(err).ToNot(HaveOccurred())

		agentClient, err = testEnvironment.StartAgentTunnel("mbus-user", "mbus-pass", 6868)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := testEnvironment.StopAgentTunnel()
		Expect(err).NotTo(HaveOccurred())

		err = testEnvironment.StopAgent()
		Expect(err).NotTo(HaveOccurred())

		err = testEnvironment.DetachDevice("/dev/sdh")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("on ubuntu when a new user is created", func() {
		BeforeEach(func() {
			testEnvironment.RunCommand("sudo groupadd bosh_sudoers")
			testEnvironment.RunCommand("sudo groupadd bosh_sshers")
			testEnvironment.RunCommand("sudo userdel -r username")
		})

		AfterEach(func() {
			testEnvironment.RunCommand("sudo userdel -r username")
		})

		It("should contain the correct home directory permissions", func() {
			err := agentClient.SSH("setup", action.SSHParams{
				User:      "username",
				PublicKey: "public-key",
			})

			Expect(err).ToNot(HaveOccurred())

			verifyFilePerm("755", "/var/vcap/bosh_ssh", testEnvironment)
			verifyFilePerm("700", "/var/vcap/bosh_ssh/username", testEnvironment)
		})
	})
})

func verifyFilePerm(perm string, filePath string, testEnvironment *integration.TestEnvironment) {
	filePerms, err := testEnvironment.RunCommand("sudo stat -c '%a %n' " + filePath + " | cut -d' ' -f 1")
	Expect(err).NotTo(HaveOccurred())

	Expect(strings.Trim(filePerms, "\n")).To(Equal(perm))
}
