package integration_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("RawEphemeralDisk", func() {
	var (
		registrySettings boshsettings.Settings
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

		networks, err := testEnvironment.GetVMNetworks()
		Expect(err).ToNot(HaveOccurred())

		registrySettings = boshsettings.Settings{
			AgentID: "fake-agent-id",
			Mbus:    "https://127.0.0.1:6868",
			Blobstore: boshsettings.Blobstore{
				Type: "local",
				Options: map[string]interface{}{
					"blobstore_path": "/var/vcap/data",
				},
			},
			Networks: networks,
		}
	})

	Context("when raw ephemeral disk is provided in settings", func() {
		BeforeEach(func() {
			err := testEnvironment.AttachDevice("/dev/sdh", 8, 2)
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.AttachDevice("/dev/xvdb", 8, 1)
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.AttachDevice("/dev/xvdc", 8, 1)
			Expect(err).ToNot(HaveOccurred())

			registrySettings.Disks = boshsettings.Disks{
				Ephemeral:    "/dev/sdh",
				RawEphemeral: []boshsettings.DiskSettings{{ID: "1", Path: "/dev/xvdb"}, {ID: "2", Path: "/dev/xvdc"}},
			}

			err = testEnvironment.StartRegistry(registrySettings)
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.StartAgent()
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			err := testEnvironment.StopAgent()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.DetachDevice("/dev/sdh")
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.DetachDevice("/dev/xvdb")
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.DetachDevice("/dev/xvdc")
			Expect(err).ToNot(HaveOccurred())
		})

		It("labels the raw ephemeral paths for unpartitioned disks", func() {
			Eventually(func() string {
				stdout, _ := testEnvironment.RunCommand("find /dev/disk/by-partlabel | sort")

				return stdout
			}, 5*time.Minute, 1*time.Second).Should(ContainSubstring(`/dev/disk/by-partlabel/raw-ephemeral-0
/dev/disk/by-partlabel/raw-ephemeral-1`))
		})
	})

})
