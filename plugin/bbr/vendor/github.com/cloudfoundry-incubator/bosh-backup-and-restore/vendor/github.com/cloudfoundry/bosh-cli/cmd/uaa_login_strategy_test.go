package cmd_test

import (
	"errors"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakecmdconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	fakeuaa "github.com/cloudfoundry/bosh-cli/uaa/uaafakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("UAALoginStrategy", func() {
	var (
		sessions map[cmdconf.Config]*fakecmd.FakeSession
		config   *fakecmdconf.FakeConfig
		ui       *fakeui.FakeUI
		strategy UAALoginStrategy
	)

	BeforeEach(func() {
		sessions = map[cmdconf.Config]*fakecmd.FakeSession{}
		sessionFactory := func(config cmdconf.Config) Session {
			return sessions[config]
		}
		config = &fakecmdconf.FakeConfig{}
		ui = &fakeui.FakeUI{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		strategy = NewUAALoginStrategy(sessionFactory, config, ui, logger)
	})

	Describe("Try", func() {
		var (
			initialSession *fakecmd.FakeSession
			updatedSession *fakecmd.FakeSession
			updatedConfig  *fakecmdconf.FakeConfig
			uaa            *fakeuaa.FakeUAA
		)

		BeforeEach(func() {
			initialSession = &fakecmd.FakeSession{}
			sessions[config] = initialSession

			uaa = &fakeuaa.FakeUAA{}
			initialSession.UAAReturns(uaa, nil)

			initialSession.EnvironmentReturns("environment")

			updatedConfig = &fakecmdconf.FakeConfig{}
			config.SetCredentialsStub = func(environment string, creds cmdconf.Creds) cmdconf.Config {
				updatedConfig.CredentialsStub = func(t string) cmdconf.Creds {
					return map[string]cmdconf.Creds{environment: creds}[t]
				}
				return updatedConfig
			}

			updatedSession = &fakecmd.FakeSession{}
			sessions[updatedConfig] = updatedSession

			updatedSession.UAAReturns(uaa, nil)
		})

		act := func() error { return strategy.Try() }

		Context("when session credentials are not set for UAA client (implies user login)", func() {
			var (
				accessToken *fakeuaa.FakeAccessToken
			)

			BeforeEach(func() {
				refreshToken := &fakeuaa.FakeToken{}
				refreshToken.ValueReturns("refresh-token")

				accessToken = &fakeuaa.FakeAccessToken{}
				accessToken.RefreshTokenReturns(refreshToken)

				ui.AskedText = []fakeui.Answer{
					{Text: "asked-username1"},
					{Text: "asked-username2"},
					{Text: "asked-username3"},
				}

				ui.AskedPasswords = []fakeui.Answer{
					{Text: "asked-password1"},
					{Text: "asked-password2"},
					{Text: "asked-password3"},
				}
			})

			Context("when UAA returns prompts", func() {
				BeforeEach(func() {
					uaa.PromptsReturns([]boshuaa.Prompt{
						{Key: "username", Type: "text", Label: "username-label"},
						{Key: "password", Type: "password", Label: "password-label"},
					}, nil)
				})

				Context("when credentials are correct", func() {
					BeforeEach(func() {
						uaa.OwnerPasswordCredentialsGrantReturns(accessToken, nil)
					})

					It("asks correct prompts and uses answers to retrieve token", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						answers := uaa.OwnerPasswordCredentialsGrantArgsForCall(0)
						Expect(answers).To(Equal([]boshuaa.PromptAnswer{
							{Key: "username", Value: "asked-username1"},
							{Key: "password", Value: "asked-password1"},
						}))

						Expect(ui.AskedTextLabels).To(Equal([]string{"username-label"}))
						Expect(ui.AskedPasswordLabels).To(Equal([]string{"password-label"}))
					})

					It("does not use empty answers to retrieve token", func() {
						ui.AskedText = []fakeui.Answer{
							{Text: ""},
							{Text: "asked-username2"},
							{Text: "asked-username3"},
						}

						err := act()
						Expect(err).ToNot(HaveOccurred())

						answers := uaa.OwnerPasswordCredentialsGrantArgsForCall(0)
						Expect(answers).To(Equal([]boshuaa.PromptAnswer{
							{Key: "password", Value: "asked-password1"},
						}))
					})

					It("successfully logs in", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Said).To(Equal([]string{"Successfully authenticated with UAA"}))
					})

					It("saves the config with a refresh token", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(updatedConfig.SaveCallCount()).To(Equal(1))
						Expect(updatedConfig.Credentials("environment")).To(
							Equal(cmdconf.Creds{RefreshToken: "refresh-token"}))
					})
				})

				Context("when cannot check credentials or they are not correct", func() {
					BeforeEach(func() {
						tokens := []*fakeuaa.FakeAccessToken{nil, nil, accessToken}
						errs := []error{errors.New("fail"), errors.New("fail"), nil}

						grantFunc := func([]boshuaa.PromptAnswer) (boshuaa.AccessToken, error) {
							token := tokens[0]
							tokens = tokens[1:]
							err := errs[0]
							errs = errs[1:]
							return token, err
						}

						uaa.OwnerPasswordCredentialsGrantStub = grantFunc
					})

					It("keeps on asking with prompts until success", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(ui.Errors).To(Equal([]string{
							"Failed to authenticate with UAA",
							"Failed to authenticate with UAA",
						}))

						Expect(ui.Said).To(Equal([]string{"Successfully authenticated with UAA"}))
					})

					It("only saves config upon successful log in", func() {
						err := act()
						Expect(err).ToNot(HaveOccurred())

						Expect(updatedConfig.SaveCallCount()).To(Equal(1))
						Expect(updatedConfig.Credentials("environment")).To(Equal(cmdconf.Creds{
							RefreshToken: "refresh-token",
						}))
					})
				})
			})

			It("returns an error when cannot get UAA prompts", func() {
				uaa.PromptsReturns(nil, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when session credentials are set for UAA client", func() {
			BeforeEach(func() {
				initialSession.CredentialsStub = func() cmdconf.Creds {
					return cmdconf.Creds{
						Client:       "uaa-client",
						ClientSecret: "uaa-client-secret",
					}
				}
			})

			Context("when credentials are correct", func() {
				BeforeEach(func() {
					uaa.ClientCredentialsGrantReturns(&fakeuaa.FakeToken{}, nil)
				})

				It("successfully logs in", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(ui.Said).To(Equal([]string{"Successfully authenticated with UAA"}))
				})

				It("does not save config", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(updatedConfig.SaveCallCount()).To(Equal(0))
				})
			})

			Context("when cannot check credentials or they are not correct", func() {
				BeforeEach(func() {
					uaa.ClientCredentialsGrantReturns(nil, errors.New("fake-err"))
				})

				It("returns an error without asking for anything", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(ui.Errors).To(Equal([]string{"Failed to authenticate with UAA"}))
				})

				It("does not save config", func() {
					err := act()
					Expect(err).To(HaveOccurred())

					Expect(updatedConfig.SaveCallCount()).To(Equal(0))
				})
			})
		})
	})
})
