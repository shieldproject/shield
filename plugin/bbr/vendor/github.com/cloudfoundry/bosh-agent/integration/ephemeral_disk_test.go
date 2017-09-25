package integration_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("EphemeralDisk", func() {
	var (
		registrySettings boshsettings.Settings
	)

	Context("mounted on /var/vcap/data", func() {

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
				Mbus:    "https://mbus-user:mbus-pass@127.0.0.1:6868",
				Blobstore: boshsettings.Blobstore{
					Type: "local",
					Options: map[string]interface{}{
						"blobstore_path": "/var/vcap/data",
					},
				},
				Networks: networks,
			}
		})

		JustBeforeEach(func() {
			err := testEnvironment.StartRegistry(registrySettings)
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.StartAgent()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when ephemeral disk is provided in settings", func() {
			BeforeEach(func() {
				registrySettings.Disks = boshsettings.Disks{
					Ephemeral: "/dev/sdh",
				}
			})

			Context("when ephemeral disk exists", func() {
				BeforeEach(func() {
					err := testEnvironment.AttachDevice("/dev/sdh", 128, 2)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err := testEnvironment.DetachDevice("/dev/sdh")
					Expect(err).ToNot(HaveOccurred())
				})

				It("agent is running", func() {
					Eventually(func() error {
						_, err := testEnvironment.RunCommand("netcat -z -v 127.0.0.1 6868")
						return err
					}, 2*time.Minute, 1*time.Second).ShouldNot(HaveOccurred())
				})

				It("it is being mounted", func() {
					Eventually(func() string {
						result, _ := testEnvironment.RunCommand("sudo mount | grep /dev/sdh | grep -c /var/vcap/data")
						return strings.TrimSpace(result)
					}, 2*time.Minute, 1*time.Second).Should(Equal("1"))
				})

				Context("when bind mount /var/vcap/data/root_tmp on /tmp", func() {
					BeforeEach(func() {
						err := testEnvironment.UpdateAgentConfig("bind-mount-agent.json")
						Expect(err).ToNot(HaveOccurred())
					})

					JustBeforeEach(func() {
						Eventually(func() string {
							result, _ := testEnvironment.RunCommand("sudo mount | grep /tmp | grep -c /var/vcap/data/root_tmp")
							return strings.TrimSpace(result)
						}, 2*time.Minute, 1*time.Second).Should(Equal("2"))
					})

					It("does not execute executables", func() {
						_, err := testEnvironment.RunCommand("cp /bin/echo /tmp/echo")
						Expect(err).ToNot(HaveOccurred())

						_, err = testEnvironment.RunCommand("/tmp/echo hello")
						Expect(err).To(HaveOccurred())
					})

					It("does not allow device files", func() {
						_, err := testEnvironment.RunCommand("sudo mknod /tmp/blockDevice b 7 98")
						Expect(err).ToNot(HaveOccurred())

						_, err = testEnvironment.RunCommand("sudo dd if=/tmp/blockDevice bs=1M count=10")
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("when ephemeral disk does not exist", func() {
				BeforeEach(func() {
					testEnvironment.DetachDevice("/dev/sdh")
				})

				It("agent fails with error", func() {
					Eventually(func() bool {
						return testEnvironment.LogFileContains("ERROR .* App setup .* No ephemeral disk found")
					}, 2*time.Minute, 1*time.Second).Should(BeTrue())
				})
			})
		})

		Context("when ephemeral disk is not provided in settings", func() {
			Context("when root disk can be used as ephemeral", func() {
				var (
					oldRootDevice string
				)

				BeforeEach(func() {
					err := testEnvironment.UpdateAgentConfig("root-partition-agent.json")
					Expect(err).ToNot(HaveOccurred())

					oldRootDevice, err = testEnvironment.AttachPartitionedRootDevice("/dev/sdz", 2048, 128)
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					err := testEnvironment.DetachPartitionedRootDevice(oldRootDevice, "/dev/sdz")
					Expect(err).ToNot(HaveOccurred())
				})

				It("partitions root disk", func() {
					Eventually(func() string {
						ephemeralDataDevice, err := testEnvironment.RunCommand(`sudo mount | grep "on /var/vcap/data " | cut -d' ' -f1`)
						Expect(err).ToNot(HaveOccurred())

						return strings.TrimSpace(ephemeralDataDevice)
					}, 2*time.Minute, 1*time.Second).Should(Equal("/dev/sdz3"))

					partitionTable, err := testEnvironment.RunCommand("sudo sfdisk -d /dev/sdz")
					Expect(err).ToNot(HaveOccurred())

					Expect(partitionTable).To(ContainSubstring("/dev/sdz1 : start=        1, size=   273104, Id=83"))
					Expect(partitionTable).To(ContainSubstring("/dev/sdz2 : start=   274432, size=  1960600, Id=83"))
					Expect(partitionTable).To(ContainSubstring("/dev/sdz3 : start=  2236416, size=  1957888, Id=83"))
				})
			})

			Context("when root disk can not be used as ephemeral", func() {
				It("agent fails with error", func() {
					Eventually(func() bool {
						return testEnvironment.LogFileContains("ERROR .* App setup .* No ephemeral disk found")
					}, 2*time.Minute, 1*time.Second).Should(BeTrue())
				})
			})
		})
	})
})
