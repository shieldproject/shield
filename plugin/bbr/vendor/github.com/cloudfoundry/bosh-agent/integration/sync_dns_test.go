package integration_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"fmt"
	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/settings"
	"strings"
)

var _ = Describe("sync_dns", func() {
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

	It("sends a sync_dns message to agent", func() {
		oldEtcHosts, err := testEnvironment.RunCommand("sudo cat /etc/hosts")
		Expect(err).NotTo(HaveOccurred())

		newDNSRecords := settings.DNSRecords{
			Records: [][2]string{
				{"216.58.194.206", "google.com"},
				{"54.164.223.71", "pivotal.io"},
			},
		}
		contents, err := json.Marshal(newDNSRecords)
		Expect(err).NotTo(HaveOccurred())
		Expect(contents).NotTo(BeNil())

		_, err = testEnvironment.RunCommand("sudo mkdir -p /var/vcap/data")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo touch /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo ls -la /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		recordsJSON := `{
		  "version": 1,
			"records":[["216.58.194.206","google.com"],["54.164.223.71","pivotal.io"]],
			"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
			"record_infos": [
				["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
			]
		}`
		_, err = testEnvironment.RunCommand(fmt.Sprintf("sudo echo '%s' > /tmp/new-dns-records", recordsJSON))
		Expect(err).NotTo(HaveOccurred())

		blobDigest, err := testEnvironment.RunCommand("sudo shasum /tmp/new-dns-records | cut -f 1 -d ' '")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo mv /tmp/new-dns-records /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = agentClient.SyncDNS("new-dns-records", strings.TrimSpace(blobDigest), 1)
		Expect(err).NotTo(HaveOccurred())

		newEtcHosts, err := testEnvironment.RunCommand("sudo cat /etc/hosts")
		Expect(err).NotTo(HaveOccurred())

		Expect(newEtcHosts).To(MatchRegexp("216.58.194.206\\s+google.com"))
		Expect(newEtcHosts).To(MatchRegexp("54.164.223.71\\s+pivotal.io"))
		Expect(newEtcHosts).To(ContainSubstring(oldEtcHosts))

		instanceDNSRecords, err := testEnvironment.RunCommand("sudo cat /var/vcap/instance/dns/records.json")
		Expect(err).NotTo(HaveOccurred())
		Expect(instanceDNSRecords).To(MatchJSON(`{
			"version": 1,
			"records":[["216.58.194.206","google.com"],["54.164.223.71","pivotal.io"]],
			"record_keys": ["id", "instance_group", "az", "network", "deployment", "ip"],
			"record_infos": [
				["id-1", "instance-group-1", "az1", "network1", "deployment1", "ip1"]
			]
		}`))

		filePerms, err := testEnvironment.RunCommand("ls -l /var/vcap/instance/dns/records.json | cut -d ' ' -f 1,3,4")
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.TrimSpace(filePerms)).To(Equal("-rw-r----- root vcap"))
	})

	It("does not skip verification if no checksum is sent", func() {
		newDNSRecords := settings.DNSRecords{
			Records: [][2]string{
				{"216.58.194.206", "google.com"},
				{"54.164.223.71", "pivotal.io"},
			},
		}
		contents, err := json.Marshal(newDNSRecords)
		Expect(err).NotTo(HaveOccurred())
		Expect(contents).NotTo(BeNil())

		_, err = testEnvironment.RunCommand("sudo mkdir -p /var/vcap/data")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo touch /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo ls -la /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo echo '{\"records\":[[\"216.58.194.206\",\"google.com\"],[\"54.164.223.71\",\"pivotal.io\"]]}' > /tmp/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = testEnvironment.RunCommand("sudo mv /tmp/new-dns-records /var/vcap/data/new-dns-records")
		Expect(err).NotTo(HaveOccurred())

		_, err = agentClient.SyncDNS("new-dns-records", "", 1)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("No digest algorithm found. Supported algorithms: sha1, sha256, sha512"))
	})
})
