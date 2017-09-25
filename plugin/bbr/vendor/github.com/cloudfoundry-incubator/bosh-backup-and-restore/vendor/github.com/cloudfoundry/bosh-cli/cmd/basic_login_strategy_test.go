package cmd_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	cmdconf "github.com/cloudfoundry/bosh-cli/cmd/config"
	fakecmdconf "github.com/cloudfoundry/bosh-cli/cmd/config/configfakes"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("BasicLoginStrategy", func() {
	var (
		sessions map[cmdconf.Config]*fakecmd.FakeSession
		config   *fakecmdconf.FakeConfig
		ui       *fakeui.FakeUI
		strategy BasicLoginStrategy
	)

	BeforeEach(func() {
		sessions = map[cmdconf.Config]*fakecmd.FakeSession{}
		sessionFactory := func(config cmdconf.Config) Session {
			return sessions[config]
		}
		config = &fakecmdconf.FakeConfig{}
		ui = &fakeui.FakeUI{}
		strategy = NewBasicLoginStrategy(sessionFactory, config, ui)
	})

	Describe("Try", func() {
		var (
			initialSession *fakecmd.FakeSession
			updatedSession *fakecmd.FakeSession
			updatedConfig  *fakecmdconf.FakeConfig
			director       *fakedir.FakeDirector
		)

		BeforeEach(func() {
			initialSession = &fakecmd.FakeSession{}
			sessions[config] = initialSession

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

			director = &fakedir.FakeDirector{}
			updatedSession.DirectorReturns(director, nil)
		})

		act := func() error { return strategy.Try() }

		itLogsInOrErrs := func(expectedEnvironment, expectedUsername, expectedPassword string) {
			Context("when credentials are correct", func() {
				BeforeEach(func() {
					director.IsAuthenticatedReturns(true, nil)
				})

				It("successfully logs in", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(ui.Said).To(Equal([]string{fmt.Sprintf("Logged in to '%s'", expectedEnvironment)}))
				})

				It("saves the config with new credentials", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(updatedConfig.SaveCallCount()).To(Equal(1))
					Expect(updatedConfig.Credentials(expectedEnvironment)).To(Equal(cmdconf.Creds{
						Client:       expectedUsername,
						ClientSecret: expectedPassword,
					}))
				})
			})

			Context("when cannot check credentials correctness", func() {
				BeforeEach(func() {
					director.IsAuthenticatedReturns(false, errors.New("fake-err"))
				})

				It("returns an error and does not save config", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))

					Expect(updatedConfig.SaveCallCount()).To(Equal(0))
				})
			})
		}

		itKeepsAsking := func(expectedEnvironment, expectedUsername, expectedPassword string) {
			Context("when credentials are not correct", func() {
				BeforeEach(func() {
					tries := []bool{false, false, true}

					director.IsAuthenticatedStub = func() (bool, error) {
						result := tries[0]
						tries = tries[1:]
						return result, nil
					}
				})

				It("keeps on asking for new username and password until success", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(ui.Errors).To(Equal([]string{
						"Failed to login to 'environment'",
						"Failed to login to 'environment'",
					}))

					Expect(ui.Said).To(Equal([]string{"Logged in to 'environment'"}))
				})

				It("only saves config upon successful log in", func() {
					err := act()
					Expect(err).ToNot(HaveOccurred())

					Expect(updatedConfig.SaveCallCount()).To(Equal(1))
					Expect(updatedConfig.Credentials(expectedEnvironment)).To(Equal(cmdconf.Creds{
						Client:       expectedUsername,
						ClientSecret: expectedPassword,
					}))
				})
			})
		}

		itErrsWithoutAsking := func() {
			Context("when credentials are not correct", func() {
				BeforeEach(func() {
					director.IsAuthenticatedReturns(false, nil)
				})

				It("returns an error without asking for username or password", func() {
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Invalid credentials"))

					Expect(ui.Errors).To(Equal([]string{"Failed to login to 'environment'"}))
				})

				It("does not save config with new credentials", func() {
					err := act()
					Expect(err).To(HaveOccurred())

					Expect(updatedConfig.SaveCallCount()).To(Equal(0))
				})
			})
		}

		Context("when no global flags or config values are set", func() {
			BeforeEach(func() {
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

			itLogsInOrErrs("environment", "asked-username1", "asked-password1")
			itKeepsAsking("environment", "asked-username3", "asked-password3")
		})

		Context("when global flags or config values are set", func() {
			BeforeEach(func() {
				initialSession.CredentialsStub = func() cmdconf.Creds {
					return cmdconf.Creds{
						Client:       "global-username",
						ClientSecret: "global-password",
					}
				}
			})

			itLogsInOrErrs("environment", "global-username", "global-password")
			itErrsWithoutAsking()
		})
	})
})
