package integration_test

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
)

var _ = Describe("SystemMounts", func() {
	var (
		registrySettings boshsettings.Settings
	)

	Context("mounting /tmp", func() {

		BeforeEach(func() {
			err := testEnvironment.StopAgent()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.CleanupDataDir()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.CleanupLogFile()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.SetupConfigDrive()
			Expect(err).ToNot(HaveOccurred())

			err = testEnvironment.UpdateAgentConfig("config-drive-agent-no-default-tmp-dir.json")
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

		Context("when ephemeral disk exists", func() {
			BeforeEach(func() {
				err := testEnvironment.AttachDevice("/dev/sdh", 128, 2)
				Expect(err).ToNot(HaveOccurred())

				registrySettings.Disks = boshsettings.Disks{
					Ephemeral: "/dev/sdh",
				}
			})

			AfterEach(func() {
				err := testEnvironment.DetachDevice("/dev/sdh")
				Expect(err).ToNot(HaveOccurred())

				_, err = testEnvironment.RunCommand("! mount | grep -q ' on /tmp ' || sudo umount /tmp")
				Expect(err).ToNot(HaveOccurred())

				_, err = testEnvironment.RunCommand("! mount | grep -q ' on /var/tmp ' || sudo umount /var/tmp")
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when agent is first started", func() {
				It("binds /var/vcap/data/root_tmp on /tmp", func() {
					Eventually(func() string {
						result, _ := testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /tmp'")
						return strings.TrimSpace(result)
					}, 2*time.Minute, 1*time.Second).Should(Equal("1"))

					result, err := testEnvironment.RunCommand("stat -c %a /tmp")
					Expect(err).ToNot(HaveOccurred())
					Expect(strings.TrimSpace(result)).To(Equal("770"))
				})

				It("binds /var/vcap/data/root_tmp on /var/tmp", func() {
					Eventually(func() string {
						result, _ := testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /var/tmp'")
						return strings.TrimSpace(result)
					}, 2*time.Minute, 1*time.Second).Should(Equal("1"))

					result, err := testEnvironment.RunCommand("stat -c %a /var/tmp")
					Expect(err).ToNot(HaveOccurred())
					Expect(strings.TrimSpace(result)).To(Equal("770"))
				})
			})

			Context("when agent is restarted", func() {
				It("does not change mounts and permissions", func() {
					waitForAgentAndExpectMounts := func() {
						Eventually(func() bool {
							return testEnvironment.LogFileContains("sv start monit")
						}, 2*time.Minute, 1*time.Second).Should(BeTrue())

						result, _ := testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /tmp'")
						Expect(strings.TrimSpace(result)).To(Equal("1"))

						result, _ = testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /var/tmp'")
						Expect(strings.TrimSpace(result)).To(Equal("1"))

						result, err := testEnvironment.RunCommand("stat -c %a /tmp")
						Expect(err).ToNot(HaveOccurred())
						Expect(strings.TrimSpace(result)).To(Equal("770"))

						result, err = testEnvironment.RunCommand("stat -c %a /var/tmp")
						Expect(err).ToNot(HaveOccurred())
						Expect(strings.TrimSpace(result)).To(Equal("770"))
					}

					waitForAgentAndExpectMounts()

					err := testEnvironment.CleanupLogFile()
					Expect(err).ToNot(HaveOccurred())

					err = testEnvironment.RestartAgent()
					Expect(err).ToNot(HaveOccurred())

					waitForAgentAndExpectMounts()
				})
			})

			Context("when the bind-mounts are removed", func() {
				It("has permission 770 on /tmp", func() {
					Eventually(func() string {
						result, _ := testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /tmp'")
						return strings.TrimSpace(result)
					}, 2*time.Minute, 1*time.Second).Should(Equal("1"))

					_, err := testEnvironment.RunCommand("sudo umount /tmp")
					Expect(err).ToNot(HaveOccurred())

					result, err := testEnvironment.RunCommand("stat -c %a /tmp")
					Expect(err).ToNot(HaveOccurred())
					Expect(strings.TrimSpace(result)).To(Equal("770"))
				})

				It("has permission 770 on /var/tmp", func() {
					Eventually(func() string {
						result, _ := testEnvironment.RunCommand("sudo mount | grep -c '/var/vcap/data/root_tmp on /var/tmp'")
						return strings.TrimSpace(result)
					}, 2*time.Minute, 1*time.Second).Should(Equal("1"))

					_, err := testEnvironment.RunCommand("sudo umount /var/tmp")
					Expect(err).ToNot(HaveOccurred())

					result, err := testEnvironment.RunCommand("stat -c %a /var/tmp")
					Expect(err).ToNot(HaveOccurred())
					Expect(strings.TrimSpace(result)).To(Equal("770"))
				})
			})
		})
	})
})
