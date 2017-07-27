package cmd_test

import (
	"errors"

	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
	fakecmd "github.com/cloudfoundry/bosh-cli/cmd/cmdfakes"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	fakedir "github.com/cloudfoundry/bosh-cli/director/directorfakes"
	boshssh "github.com/cloudfoundry/bosh-cli/ssh"
	fakessh "github.com/cloudfoundry/bosh-cli/ssh/sshfakes"
)

var _ = Describe("LogsCmd", func() {
	const UUID = "8c5ff117-9572-45c5-8564-8bcf076ecafa"
	const ExpUsername = "bosh_8c5ff117957245c"

	var (
		deployment      *fakedir.FakeDeployment
		downloader      *fakecmd.FakeDownloader
		uuidGen         *fakeuuid.FakeGenerator
		nonIntSSHRunner *fakessh.FakeRunner
		command         LogsCmd
	)

	BeforeEach(func() {
		deployment = &fakedir.FakeDeployment{
			NameStub: func() string { return "dep" },
		}
		downloader = &fakecmd.FakeDownloader{}
		uuidGen = &fakeuuid.FakeGenerator{}
		nonIntSSHRunner = &fakessh.FakeRunner{}
		command = NewLogsCmd(deployment, downloader, uuidGen, nonIntSSHRunner)
	})

	Describe("Run", func() {

		var (
			opts LogsOpts
		)

		BeforeEach(func() {
			opts = LogsOpts{
				Args: AllOrInstanceGroupOrInstanceSlugArgs{
					Slug: boshdir.NewAllOrInstanceGroupOrInstanceSlug("job", "index"),
				},

				Directory: DirOrCWDArg{Path: "/fake-dir"},
			}
		})

		act := func() error { return command.Run(opts) }

		Context("when fetching logs (not tailing)", func() {
			It("fetches logs for a given instance", func() {
				result := boshdir.LogsResult{BlobstoreID: "blob-id", SHA1: "sha1"}
				deployment.FetchLogsReturns(result, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.FetchLogsCallCount()).To(Equal(1))

				slug, filters, agent := deployment.FetchLogsArgsForCall(0)
				Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("job", "index")))
				Expect(filters).To(BeEmpty())
				Expect(agent).To(BeFalse())

				Expect(downloader.DownloadCallCount()).To(Equal(1))

				blobID, sha1, prefix, dstDirPath := downloader.DownloadArgsForCall(0)
				Expect(blobID).To(Equal("blob-id"))
				Expect(sha1).To(Equal("sha1"))
				Expect(prefix).To(Equal("dep.job.index"))
				Expect(dstDirPath).To(Equal("/fake-dir"))
			})

			It("fetches agent logs and allows custom filters", func() {
				opts.Filters = []string{"filter1", "filter2"}
				opts.Agent = true

				deployment.FetchLogsReturns(boshdir.LogsResult{}, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.FetchLogsCallCount()).To(Equal(1))

				slug, filters, agent := deployment.FetchLogsArgsForCall(0)
				Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("job", "index")))
				Expect(filters).To(Equal([]string{"filter1", "filter2"}))
				Expect(agent).To(BeTrue())
			})

			It("fetches logs for more than one instance", func() {
				opts.Args.Slug = boshdir.NewAllOrInstanceGroupOrInstanceSlug("", "")

				result := boshdir.LogsResult{BlobstoreID: "blob-id", SHA1: "sha1"}
				deployment.FetchLogsReturns(result, nil)

				err := act()
				Expect(err).ToNot(HaveOccurred())

				Expect(deployment.FetchLogsCallCount()).To(Equal(1))

				Expect(downloader.DownloadCallCount()).To(Equal(1))

				blobID, sha1, prefix, dstDirPath := downloader.DownloadArgsForCall(0)
				Expect(blobID).To(Equal("blob-id"))
				Expect(sha1).To(Equal("sha1"))
				Expect(prefix).To(Equal("dep"))
				Expect(dstDirPath).To(Equal("/fake-dir"))
			})

			It("returns error if fetching logs failed", func() {
				deployment.FetchLogsReturns(boshdir.LogsResult{}, errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns error if downloading release failed", func() {
				downloader.DownloadReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("does not try to tail logs", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(nonIntSSHRunner.RunCallCount()).To(Equal(0))
			})
		})

		Context("when tailing logs (or specifying number of lines)", func() {

			BeforeEach(func() {
				opts.Follow = true
				opts.GatewayFlags.UUIDGen = uuidGen
				uuidGen.GeneratedUUID = UUID
			})

			It("sets up SSH access, runs SSH command and later cleans up SSH access", func() {
				nonIntSSHRunner.RunStub = func(boshssh.ConnectionOpts, boshdir.SSHResult, []string) error {
					Expect(deployment.CleanUpSSHCallCount()).To(Equal(0))
					return nil
				}
				Expect(act()).ToNot(HaveOccurred())

				Expect(deployment.SetUpSSHCallCount()).To(Equal(1))
				Expect(nonIntSSHRunner.RunCallCount()).To(Equal(1))
				Expect(deployment.CleanUpSSHCallCount()).To(Equal(1))

				setupSlug, setupSSHOpts := deployment.SetUpSSHArgsForCall(0)
				Expect(setupSlug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("job", "index")))
				Expect(setupSSHOpts.Username).To(Equal(ExpUsername))
				Expect(setupSSHOpts.PublicKey).To(ContainSubstring("ssh-rsa AAAA"))

				slug, sshOpts := deployment.CleanUpSSHArgsForCall(0)
				Expect(slug).To(Equal(setupSlug))
				Expect(sshOpts).To(Equal(setupSSHOpts))
			})

			It("sets up SSH access for more than one instance", func() {
				opts.Args.Slug = boshdir.NewAllOrInstanceGroupOrInstanceSlug("", "")

				Expect(act()).ToNot(HaveOccurred())

				setupSlug, _ := deployment.SetUpSSHArgsForCall(0)
				Expect(setupSlug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("", "")))
			})

			It("runs non-interactive SSH", func() {
				Expect(act()).ToNot(HaveOccurred())
				Expect(nonIntSSHRunner.RunCallCount()).To(Equal(1))
			})

			It("returns an error if generating SSH options fails", func() {
				uuidGen.GenerateError = errors.New("fake-err")
				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("returns an error if setting up SSH access fails", func() {
				deployment.SetUpSSHReturns(boshdir.SSHResult{}, errors.New("fake-err"))
				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("runs non-interactive SSH session with flags, and basic tail -f command that tails all logs", func() {
				result := boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}
				deployment.SetUpSSHReturns(result, nil)

				opts.GatewayFlags.Disable = true
				opts.GatewayFlags.Username = "gw-username"
				opts.GatewayFlags.Host = "gw-host"
				opts.GatewayFlags.PrivateKeyPath = "gw-private-key"
				opts.GatewayFlags.SOCKS5Proxy = "some-proxy"

				Expect(act()).ToNot(HaveOccurred())

				Expect(nonIntSSHRunner.RunCallCount()).To(Equal(1))

				runConnOpts, runResult, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runConnOpts.PrivateKey).To(ContainSubstring("-----BEGIN RSA PRIVATE KEY-----"))
				Expect(runConnOpts.GatewayDisable).To(Equal(true))
				Expect(runConnOpts.GatewayUsername).To(Equal("gw-username"))
				Expect(runConnOpts.GatewayHost).To(Equal("gw-host"))
				Expect(runConnOpts.GatewayPrivateKeyPath).To(Equal("gw-private-key"))
				Expect(runConnOpts.SOCKS5Proxy).To(Equal("some-proxy"))
				Expect(runResult).To(Equal(boshdir.SSHResult{Hosts: []boshdir.Host{{Host: "ip1"}}}))
				Expect(runCommand).To(Equal([]string{"sudo", "tail", "-F", "/var/vcap/sys/log/{**/,}*.log"}))
			})

			It("runs tail command with specified number of lines and quiet option", func() {
				opts.Num = 10
				opts.Quiet = true

				deployment.SetUpSSHReturns(boshdir.SSHResult{}, nil)
				Expect(act()).ToNot(HaveOccurred())

				_, _, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal([]string{
					"sudo", "tail", "-F", "-n", "10", "-q", "/var/vcap/sys/log/{**/,}*.log"}))
			})

			It("runs tail command with specified number of lines even if following is not requested", func() {
				opts.Follow = false
				opts.Num = 10

				deployment.SetUpSSHReturns(boshdir.SSHResult{}, nil)
				Expect(act()).ToNot(HaveOccurred())

				_, _, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal([]string{
					"sudo", "tail", "-n", "10", "/var/vcap/sys/log/{**/,}*.log"}))
			})

			It("runs tail command for the agent log if agent is specified", func() {
				opts.Agent = true

				deployment.SetUpSSHReturns(boshdir.SSHResult{}, nil)
				Expect(act()).ToNot(HaveOccurred())

				_, _, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal([]string{
					"sudo", "tail", "-F", "/var/vcap/bosh/log/{**/,}*.log"}))
			})

			It("runs tail command with jobs filters if specified", func() {
				opts.Jobs = []string{"job1", "job2"}

				deployment.SetUpSSHReturns(boshdir.SSHResult{}, nil)
				Expect(act()).ToNot(HaveOccurred())

				_, _, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal([]string{
					"sudo", "tail", "-F", "/var/vcap/sys/log/job1/*.log", "/var/vcap/sys/log/job2/*.log"}))
			})

			It("runs tail command with custom filters if specified", func() {
				opts.Filters = []string{"other/*.log", "**/*.log"}

				deployment.SetUpSSHReturns(boshdir.SSHResult{}, nil)
				Expect(act()).ToNot(HaveOccurred())

				_, _, runCommand := nonIntSSHRunner.RunArgsForCall(0)
				Expect(runCommand).To(Equal([]string{
					"sudo", "tail", "-F", "/var/vcap/sys/log/other/*.log", "/var/vcap/sys/log/**/*.log"}))
			})

			It("returns error if non-interactive SSH session errors", func() {
				nonIntSSHRunner.RunReturns(errors.New("fake-err"))

				err := act()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fake-err"))
			})

			It("does not try to fetch logs", func() {
				err := act()
				Expect(err).ToNot(HaveOccurred())
				Expect(deployment.FetchLogsCallCount()).To(Equal(0))
			})
		})
	})
})
