package cmd_test

import (
	"errors"

	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshssh "github.com/cloudfoundry/bosh-cli/ssh"
	fakessh "github.com/cloudfoundry/bosh-cli/ssh/sshfakes"
	fakeui "github.com/cloudfoundry/bosh-cli/ui/fakes"
)

var _ = Describe("SSHCmd", func() {
	var (
		deployment       *fakedir.FakeDeployment
		uuidGen          *fakeuuid.FakeGenerator
		intSSHRunner     *fakessh.FakeRunner
		nonIntSSHRunner  *fakessh.FakeRunner
		resultsSSHRunner *fakessh.FakeRunner
		ui               *fakeui.FakeUI
		command          SSHCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		uuidGen = &fakeuuid.FakeGenerator{}
		intSSHRunner = &fakessh.FakeRunner{}
		nonIntSSHRunner = &fakessh.FakeRunner{}
		resultsSSHRunner = &fakessh.FakeRunner{}
		ui = &fakeui.FakeUI{}
		command = NewSSHCmd(
			deployment, uuidGen, intSSHRunner, nonIntSSHRunner, resultsSSHRunner, ui)
	})

	Describe("Run", func() {
		const UUID = "8c5ff117-9572-45c5-8564-8bcf076ecafa"
		const ExpUsername = "bosh_8c5ff117957245c"

		var (
			opts SSHOpts
		)

		BeforeEach(func() {
			opts = SSHOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("job-name", ""),
				},

				GatewayFlags: GatewayFlags{
					UUIDGen: uuidGen,
				},
			}

			uuidGen.GeneratedUUID = UUID
		})

		act := func() error { return command.Run(opts) }

		itRunsNonInteractiveSSHWhenCommandIsGiven := func(runner **fakessh.FakeRunner) {
			Context("when commmand is provided", func() {
				BeforeEach(func() {
					opts.Command = []string{"cmd", "arg1"}
				})

				It("sets up SSH access, runs SSH command and later cleans up SSH access", func() {
					(*runner).RunStub = func(boshssh.ConnectionOpts, boshdir.SSHResult, []string) error {
						Expect(deployment.CleanUpSSHCallCount()).To(Equal(0))
						return nil
					}
					Expect(act()).ToNot(HaveOccurred())

					Expect(deployment.SetUpSSHCallCount()).To(Equal(1))
					Expect((*runner).RunCallCount()).To(Equal(1))
					Expect(deployment.CleanUpSSHCallCount()).To(Equal(1))

					setupSlug, setupSSHOpts := deployment.SetUpSSHArgsForCall(0)
					Expect(setupSlug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("job-name", "")))
					Expect(setupSSHOpts.Username).To(Equal(ExpUsername))
					Expect(setupSSHOpts.PublicKey).To(ContainSubstring("ssh-rsa AAAA"))

					slug, sshOpts := deployment.CleanUpSSHArgsForCall(0)
					Expect(slug).To(Equal(setupSlug))
					Expect(sshOpts).To(Equal(setupSSHOpts))
				})

				It("runs non-interactive SSH", func() {
					Expect(act()).ToNot(HaveOccurred())
					Expect((*runner).RunCallCount()).To(Equal(1))
					Expect(intSSHRunner.RunCallCount()).To(Equal(0))
				})

				It("returns an error if setting up SSH access fails", func() {
					deployment.SetUpSSHReturns(boshdir.SSHResult{}, errors.New("fake-err"))
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})

				It("returns an error if generating SSH options fails", func() {
					uuidGen.GenerateError = errors.New("fake-err")
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})

				It("runs non-interactive SSH session with flags, and command", func() {
					result := boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}
					deployment.SetUpSSHReturns(result, nil)

					opts.RawOpts = TrimmedSpaceArgs([]string{"raw1", "raw2"})
					opts.GatewayFlags.Disable = true
					opts.GatewayFlags.Username = "gw-username"
					opts.GatewayFlags.Host = "gw-host"
					opts.GatewayFlags.PrivateKeyPath = "gw-private-key"
					opts.GatewayFlags.SOCKS5Proxy = "socks5"

					Expect(act()).ToNot(HaveOccurred())

					Expect((*runner).RunCallCount()).To(Equal(1))

					runConnOpts, runResult, runCommand := (*runner).RunArgsForCall(0)
					Expect(runConnOpts.RawOpts).To(Equal([]string{"raw1", "raw2"}))
					Expect(runConnOpts.PrivateKey).To(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
					Expect(runConnOpts.GatewayDisable).To(Equal(true))
					Expect(runConnOpts.GatewayUsername).To(Equal("gw-username"))
					Expect(runConnOpts.GatewayHost).To(Equal("gw-host"))
					Expect(runConnOpts.GatewayPrivateKeyPath).To(Equal("gw-private-key"))
					Expect(runConnOpts.SOCKS5Proxy).To(Equal("socks5"))
					Expect(runResult).To(Equal(boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}))
					Expect(runCommand).To(Equal([]string{"cmd", "arg1"}))
				})

				It("returns error if non-interactive SSH session errors", func() {
					(*runner).RunReturns(errors.New("fake-err"))
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})
			})
		}

		Context("when ui is interactive", func() {
			BeforeEach(func() {
				ui.Interactive = true
			})

			itRunsNonInteractiveSSHWhenCommandIsGiven(&nonIntSSHRunner)

			Context("when command is not provided", func() {
				It("sets up SSH access, runs SSH command and later cleans up SSH access", func() {
					intSSHRunner.RunStub = func(boshssh.ConnectionOpts, boshdir.SSHResult, []string) error {
						Expect(deployment.CleanUpSSHCallCount()).To(Equal(0))
						return nil
					}
					Expect(act()).ToNot(HaveOccurred())

					Expect(deployment.SetUpSSHCallCount()).To(Equal(1))
					Expect(intSSHRunner.RunCallCount()).To(Equal(1))
					Expect(deployment.CleanUpSSHCallCount()).To(Equal(1))

					setupSlug, setupSSHOpts := deployment.SetUpSSHArgsForCall(0)
					Expect(setupSlug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("job-name", "")))
					Expect(setupSSHOpts.Username).To(Equal(ExpUsername))
					Expect(setupSSHOpts.PublicKey).To(ContainSubstring("ssh-rsa AAAA"))

					slug, sshOpts := deployment.CleanUpSSHArgsForCall(0)
					Expect(slug).To(Equal(setupSlug))
					Expect(sshOpts).To(Equal(setupSSHOpts))
				})

				It("runs only interactive SSH", func() {
					Expect(act()).ToNot(HaveOccurred())
					Expect(nonIntSSHRunner.RunCallCount()).To(Equal(0))
					Expect(intSSHRunner.RunCallCount()).To(Equal(1))
					Expect(resultsSSHRunner.RunCallCount()).To(Equal(0))
				})

				It("returns an error if setting up SSH access fails", func() {
					deployment.SetUpSSHReturns(boshdir.SSHResult{}, errors.New("fake-err"))
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})

				It("runs interactive SSH session with flags, but without command", func() {
					result := boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}
					deployment.SetUpSSHReturns(result, nil)

					opts.RawOpts = TrimmedSpaceArgs([]string{"raw1", "raw2"})
					opts.GatewayFlags.Disable = true
					opts.GatewayFlags.Username = "gw-username"
					opts.GatewayFlags.Host = "gw-host"
					opts.GatewayFlags.PrivateKeyPath = "gw-private-key"

					Expect(act()).ToNot(HaveOccurred())

					Expect(intSSHRunner.RunCallCount()).To(Equal(1))

					runConnOpts, runResult, runCommand := intSSHRunner.RunArgsForCall(0)
					Expect(runConnOpts.RawOpts).To(Equal([]string{"raw1", "raw2"}))
					Expect(runConnOpts.PrivateKey).To(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
					Expect(runConnOpts.GatewayDisable).To(Equal(true))
					Expect(runConnOpts.GatewayUsername).To(Equal("gw-username"))
					Expect(runConnOpts.GatewayHost).To(Equal("gw-host"))
					Expect(runConnOpts.GatewayPrivateKeyPath).To(Equal("gw-private-key"))
					Expect(runResult).To(Equal(boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}))
					Expect(runCommand).To(BeNil())
				})

				It("returns error if interactive SSH session errors", func() {
					intSSHRunner.RunReturns(errors.New("fake-err"))
					err := act()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("fake-err"))
				})
			})
		})

		Context("when ui is not interactive", func() {
			BeforeEach(func() {
				ui.Interactive = false
			})

			itRunsNonInteractiveSSHWhenCommandIsGiven(&nonIntSSHRunner)

			Context("when command is not provided", func() {
				It("returns an error since command is required", func() {
					Expect(act()).To(Equal(errors.New("Non-interactive SSH requires non-empty command")))
				})

				It("does not try to run any SSH sessions", func() {
					Expect(act()).To(HaveOccurred())
					Expect(intSSHRunner.RunCallCount()).To(Equal(0))
					Expect(nonIntSSHRunner.RunCallCount()).To(Equal(0))
					Expect(resultsSSHRunner.RunCallCount()).To(Equal(0))
				})
			})
		})

		Context("when results are requested", func() {
			BeforeEach(func() {
				ui.Interactive = true
				opts.Results = true
			})

			itRunsNonInteractiveSSHWhenCommandIsGiven(&resultsSSHRunner)

			Context("when command is not provided", func() {
				It("returns an error since command is required", func() {
					Expect(act()).To(Equal(errors.New("Non-interactive SSH requires non-empty command")))
				})

				It("does not try to run any SSH sessions", func() {
					Expect(act()).To(HaveOccurred())
					Expect(intSSHRunner.RunCallCount()).To(Equal(0))
					Expect(nonIntSSHRunner.RunCallCount()).To(Equal(0))
					Expect(resultsSSHRunner.RunCallCount()).To(Equal(0))
				})
			})
		})
	})
})
