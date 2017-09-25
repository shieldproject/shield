package action_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"errors"

	. "github.com/cloudfoundry/bosh-agent/agent/action"
	fakeplatform "github.com/cloudfoundry/bosh-agent/platform/fakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshdirs "github.com/cloudfoundry/bosh-agent/settings/directories"
	fakesettings "github.com/cloudfoundry/bosh-agent/settings/fakes"
	boshassert "github.com/cloudfoundry/bosh-utils/assert"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
)

func buildSSHAction(settingsService boshsettings.Service) (*fakeplatform.FakePlatform, SSHAction) {
	platform := fakeplatform.NewFakePlatform()
	dirProvider := boshdirs.NewProvider("/foo")
	logger := boshlog.NewLogger(boshlog.LevelNone)
	action := NewSSH(settingsService, platform, dirProvider, logger)
	return platform, action
}

var _ = Describe("SSHAction", func() {
	var (
		platform        *fakeplatform.FakePlatform
		settingsService boshsettings.Service
		action          SSHAction
	)

	Context("Action setup", func() {
		BeforeEach(func() {
			settingsService = &fakesettings.FakeSettingsService{}
			platform, action = buildSSHAction(settingsService)
		})

		AssertActionIsNotAsynchronous(action)
		AssertActionIsNotPersistent(action)
		AssertActionIsLoggable(action)

		AssertActionIsNotResumable(action)
		AssertActionIsNotCancelable(action)
	})

	Describe("Run", func() {
		Context("setupSSH", func() {
			var (
				response SSHResult
				params   SSHParams
				err      error

				defaultIP string

				platformPublicKeyValue string
				platformPublicKeyErr   error
			)

			BeforeEach(func() {
				defaultIP = "ww.xx.yy.zz"

				platformPublicKeyValue = ""
				platformPublicKeyErr = nil
			})

			JustBeforeEach(func() {
				settingsService := &fakesettings.FakeSettingsService{}
				settingsService.Settings.Networks = boshsettings.Networks{
					"fake-net": boshsettings.Network{IP: defaultIP},
				}

				platform, action = buildSSHAction(settingsService)

				platform.GetHostPublicKeyValue = platformPublicKeyValue
				platform.GetHostPublicKeyError = platformPublicKeyErr

				params = SSHParams{
					User:      "fake-user",
					PublicKey: "fake-public-key",
				}

				response, err = action.Run("setup", params)
			})

			Context("without default ip", func() {
				BeforeEach(func() {
					defaultIP = ""
				})

				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("No default ip"))
				})
			})

			Context("with an empty password", func() {
				It("should create user with an empty password", func() {
					Expect(platform.CreateUserUsername).To(Equal("fake-user"))
					Expect(platform.CreateUserBasePath).To(boshassert.MatchPath("/foo/bosh_ssh"))
					Expect(platform.AddUserToGroupsGroups["fake-user"]).To(Equal(
						[]string{boshsettings.VCAPUsername, boshsettings.AdminGroup, boshsettings.SudoersGroup, boshsettings.SshersGroup},
					))
					Expect(platform.SetupSSHPublicKeys["fake-user"]).To(ConsistOf("fake-public-key"))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("with a host public key available", func() {
				It("should return SSH Result with HostPublicKey", func() {
					hostPublicKey, _ := platform.GetHostPublicKey()
					Expect(response).To(Equal(SSHResult{
						Command:       "setup",
						Status:        "success",
						IP:            defaultIP,
						HostPublicKey: hostPublicKey,
					}))
					Expect(err).To(BeNil())
				})
			})

			Context("without a host public key available", func() {
				BeforeEach(func() {
					platformPublicKeyErr = errors.New("Get Host Public Key Failure")
				})

				It("should return an error", func() {
					Expect(response).To(Equal(SSHResult{}))
					Expect(err).ToNot(BeNil())
				})
			})
		})

		Context("cleanupSSH", func() {
			It("should delete ephemeral user", func() {
				response, err := action.Run("cleanup", SSHParams{UserRegex: "^foobar.*"})
				Expect(err).ToNot(HaveOccurred())
				Expect(platform.DeleteEphemeralUsersMatchingRegex).To(Equal("^foobar.*"))

				// Make sure empty ip field is not included in the response
				boshassert.MatchesJSONMap(GinkgoT(), response, map[string]interface{}{
					"command": "cleanup",
					"status":  "success",
				})
			})
		})
	})
})
