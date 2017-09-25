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

var _ = Describe("SCPCmd", func() {
	const UUID = "8c5ff117-9572-45c5-8564-8bcf076ecafa"
	const ExpUsername = "bosh_8c5ff117957245c"

	var (
		deployment *fakedir.FakeDeployment
		uuidGen    *fakeuuid.FakeGenerator
		scpRunner  *fakessh.FakeSCPRunner
		ui         *fakeui.FakeUI
		command    SCPCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{}
		uuidGen = &fakeuuid.FakeGenerator{}
		scpRunner = &fakessh.FakeSCPRunner{}
		ui = &fakeui.FakeUI{}
		command = NewSCPCmd(deployment, uuidGen, scpRunner, ui)
	})

	Describe("Run", func() {
		var (
			opts SCPOpts
		)

		BeforeEach(func() {
			opts = SCPOpts{
				GatewayFlags: GatewayFlags{
					UUIDGen: uuidGen,
				},
			}
			uuidGen.GeneratedUUID = UUID
		})

		act := func() error { return command.Run(opts) }

		Context("when valid SCP args are provided", func() {
			BeforeEach(func() {
				opts.Args.Paths = []string{"from:file", "/something"}
			})

			It("sets up SSH access, runs SSH command and later cleans up SSH access", func() {
				scpRunner.RunStub = func(boshssh.ConnectionOpts, boshdir.SSHResult, boshssh.SCPArgs) error {
					Expect(deployment.CleanUpSSHCallCount()).To(Equal(0))
					return nil
				}
				Expect(act()).ToNot(HaveOccurred())

				Expect(deployment.SetUpSSHCallCount()).To(Equal(1))
				Expect(scpRunner.RunCallCount()).To(Equal(1))
				Expect(deployment.CleanUpSSHCallCount()).To(Equal(1))

				setupSlug, setupSSHOpts := deployment.SetUpSSHArgsForCall(0)
				Expect(setupSlug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("from", "")))
				Expect(setupSSHOpts.Username).To(Equal(ExpUsername))
				Expect(setupSSHOpts.PublicKey).To(ContainSubstring("ssh-rsa AAAA"))

				slug, sshOpts := deployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(setupSlug))
				Expect(sshOpts).To(Equal(setupSSHOpts))
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

			It("runs SCP with flags, and command", func() {
				result := boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}
				deployment.SetUpSSHReturns(result, nil)

				opts.GatewayFlags.Disable = true
				opts.GatewayFlags.Username = "gw-username"
				opts.GatewayFlags.Host = "gw-host"
				opts.GatewayFlags.PrivateKeyPath = "gw-private-key"
				opts.GatewayFlags.SOCKS5Proxy = "some-proxy"

				Expect(act()).ToNot(HaveOccurred())

				Expect(scpRunner.RunCallCount()).To(Equal(1))

				runConnOpts, runResult, runCommand := scpRunner.RunArgsForCall(0)
				Expect(runConnOpts.PrivateKey).To(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
				Expect(runConnOpts.GatewayDisable).To(Equal(true))
				Expect(runConnOpts.GatewayUsername).To(Equal("gw-username"))
				Expect(runConnOpts.GatewayHost).To(Equal("gw-host"))
				Expect(runConnOpts.GatewayPrivateKeyPath).To(Equal("gw-private-key"))
				Expect(runConnOpts.SOCKS5Proxy).To(Equal("some-proxy"))
				Expect(runResult).To(Equal(boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}))
				Expect(runCommand).To(Equal(boshssh.NewSCPArgs([]string{"from:file", "/something"}, false)))
			})

			It("sets up SCP to be recursive if recursive flag is set", func() {
				opts.Recursive = true
				Expect(act()).ToNot(HaveOccurred())
				Expect(scpRunner.RunCallCount()).To(Equal(1))

				_, _, runCommand := scpRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal(boshssh.NewSCPArgs([]string{"from:file", "/something"}, true)))
			})

			It("returns error if SCP errors", func() {
				scpRunner.RunReturns(errors.New("fake-err"))
				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})
		})

		Context("when valid SCP args are not provided", func() {
			BeforeEach(func() {
				opts.Args.Paths = []string{"invalid-arg"}
			})

			It("returns an error", func() {
				Expect(act()).To(Equal(errors.New(
					"Missing remote host information in source/destination arguments")))
			})
		})
	})
})
