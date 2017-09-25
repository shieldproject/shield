package integration_test

import (
	"errors"
	"regexp"
	"strings"

	"github.com/cloudfoundry/bosh-agent/agentclient"
	"github.com/cloudfoundry/bosh-agent/settings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteARPEntries", func() {
	const (
		emptyMacAddress string = "<incomplete>"
		testMacAddress  string = "52:54:00:12:35:aa"
	)

	var (
		agentClient      agentclient.AgentClient
		registrySettings settings.Settings
		testIP           string
	)

	var getValidIP = func(gatewayIP string) string {
		ipParts := strings.Split(gatewayIP, ".")
		ipParts[3] = "100"
		return strings.Join(ipParts, ".")
	}

	var parseARPCacheIntoMap = func() (map[string]string, error) {
		ARPCache := map[string]string{}
		ARPResultsRegex := regexp.MustCompile(`.*\((.*)\)\ at\ (\S+).*`)
		lines, err := testEnvironment.RunCommand("arp -a")
		if err != nil {
			return ARPCache, err
		}

		for _, item := range ARPResultsRegex.FindAllStringSubmatch(lines, -1) {
			ip := item[1]
			mac := item[2]
			ARPCache[ip] = mac
		}

		return ARPCache, nil
	}

	var getGatewayIP = func() (string, error) {
		ARPCache, err := parseARPCacheIntoMap()
		if err != nil {
			return "", err
		}

		for key := range ARPCache {
			return key, nil
		}

		return "", errors.New("Unable to find gateway ip")
	}

	BeforeEach(func() {
		err := testEnvironment.StopAgent()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.SetupConfigDrive()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupDataDir()
		Expect(err).ToNot(HaveOccurred())

		err = testEnvironment.CleanupLogFile()
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

		gatewayIP, err := getGatewayIP()
		Expect(err).ToNot(HaveOccurred())

		testIP = getValidIP(gatewayIP)
		testEnvironment.RunCommand("sudo arp -s " + testIP + " " + testMacAddress)

		ARPCache, _ := parseARPCacheIntoMap()
		macOfTestIP := ARPCache[testIP]
		Expect(macOfTestIP).To(Equal(testMacAddress))
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
		It("deletes ARP entries from the cache", func() {
			err := agentClient.DeleteARPEntries([]string{testIP})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				ARPCache, _ := parseARPCacheIntoMap()
				return ARPCache[testIP]
			}).Should(Equal(emptyMacAddress))
		})
	})
})
