package integration_test

import (
	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CertManager", func() {
	var (
		agentClient      agentclient.AgentClient
		registrySettings settings.Settings
	)

	BeforeEach(func() {
		err := testEnvironment.StopAgent()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupDataDir()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupLogFile()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.SetupConfigDrive()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.UpdateAgentConfig("config-drive-agent.json")
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

	Context("on ubuntu", func() {
		It("adds and registers new certs on a fresh machine", func() {
			var cert string = `This certificate is the first one. It's more awesome than the other one.
-----BEGIN CERTIFICATE-----
MIIEJDCCAwygAwIBAgIJAO+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV
aWRnaXRzIFB0eSBMdGQxIjAgBgNVBAMTGWR4MTkwLnRvci5waXZvdGFsbGFicy5j
DtmvI8bXKxU=
-----END CERTIFICATE-----
Junk between the certs!
-----BEGIN CERTIFICATE-----
MIIEJDCCaWRnaXRzIFB0eSBMdGQxIjAgBgNVBAMTGWR4MTkwLnRvci5waXZvdGFs
b20wHhcNMTUwNTEzMTM1NjA2WhcNMjUwNTEwMTM1NjA2WjBpMQswCQYDVQQGEwJD
QTETMBEGA1U=
-----END CERTIFICATE-----`
			settings := settings.Settings{TrustedCerts: cert}

			err := agentClient.UpdateSettings(settings)

			Expect(err).NotTo(HaveOccurred())

			individualCerts, err := testEnvironment.RunCommand("ls /usr/local/share/ca-certificates/")
			Expect(err).NotTo(HaveOccurred())
			Expect(individualCerts).To(Equal("bosh-trusted-cert-1.crt\nbosh-trusted-cert-2.crt\n"))

			processedCerts, err := testEnvironment.RunCommand("grep MIIEJDCCAwygAwIBAgIJAO\\+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV /etc/ssl/certs/ca-certificates.crt")
			Expect(processedCerts).To(Equal("MIIEJDCCAwygAwIBAgIJAO+CqgiJnCgpMA0GCSqGSIb3DQEBBQUAMGkxCzAJBgNV\n"))
		})
	})
})
