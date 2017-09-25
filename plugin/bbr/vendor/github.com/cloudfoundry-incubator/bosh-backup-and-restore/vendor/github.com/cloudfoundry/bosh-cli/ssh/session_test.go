package ssh_test

import (
	"errors"

	boshsys "github.com/cloudfoundry/bosh-utils/system"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	. "github.com/cloudfoundry/bosh-cli/ssh"
)

var _ = Describe("SessionImpl", func() {
	var (
		connOpts       ConnectionOpts
		sessOpts       SessionImplOpts
		result         boshdir.SSHResult
		privKeyFile    *fakesys.FakeFile
		knownHostsFile *fakesys.FakeFile
		fs             *fakesys.FakeFileSystem
		session        *SessionImpl
	)

	BeforeEach(func() {
		connOpts = ConnectionOpts{}
		sessOpts = SessionImplOpts{}
		result = boshdir.SSHResult{}
		fs = fakesys.NewFakeFileSystem()
		privKeyFile = fakesys.NewFakeFile("/tmp/priv-key", fs)
		knownHostsFile = fakesys.NewFakeFile("/tmp/known-hosts", fs)
		fs.ReturnTempFilesByPrefix = map[string]boshsys.File{
			"ssh-priv-key":    privKeyFile,
			"ssh-known-hosts": knownHostsFile,
		}
		session = NewSessionImpl(connOpts, sessOpts, result, fs)
	})

	Describe("Start", func() {
		act := func() *SessionImpl { return NewSessionImpl(connOpts, sessOpts, result, fs) }

		It("writes out private key", func() {
			connOpts.PrivateKey = "priv-key"

			_, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.ReadFileString("/tmp/priv-key")).To(Equal("priv-key"))
		})

		It("returns error if cannot create private key temp file", func() {
			fs.TempFileErrorsByPrefix = map[string]error{
				"ssh-priv-key": errors.New("fake-err"),
			}

			_, err := act().Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("returns error if writing public key failed", func() {
			privKeyFile.WriteErr = errors.New("fake-err")

			_, err := act().Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
		})

		It("writes out all known hosts", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1", HostPublicKey: "pub-key1"},
				{Host: "127.0.0.2", HostPublicKey: "pub-key2"},
			}

			_, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.ReadFileString("/tmp/known-hosts")).To(Equal(
				"127.0.0.1 pub-key1\n127.0.0.2 pub-key2\n"))
		})

		It("returns error if cannot create known hosts temp file and deletes private key", func() {
			fs.TempFileErrorsByPrefix = map[string]error{
				"ssh-known-hosts": errors.New("fake-err"),
			}

			_, err := act().Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(fs.FileExists("/tmp/priv-key")).To(BeFalse())
		})

		It("returns error if writing known hosts failed and deletes private key", func() {
			result.Hosts = []boshdir.Host{
				{Host: "127.0.0.1", HostPublicKey: "pub-key1"},
			}
			knownHostsFile.WriteErr = errors.New("fake-err")

			_, err := act().Start()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))

			Expect(fs.FileExists("/tmp/priv-key")).To(BeFalse())
		})

		It("returns ssh options with correct paths to private key and known hosts", func() {
			cmdOpts, err := session.Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
			}))
		})

		It("returns ssh options with forced tty option if requested", func() {
			sessOpts.ForceTTY = true

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-tt",
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
			}))
		})

		It("returns ssh options with custom raw options specified", func() {
			connOpts.RawOpts = []string{"raw1", "raw2"}

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
				"raw1", "raw2",
			}))
		})

		It("returns ssh options with gateway settings returned from the Director", func() {
			result.GatewayUsername = "gw-user"
			result.GatewayHost = "gw-host"

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
				"-o", "ProxyCommand=ssh -tt -W %h:%p -l gw-user gw-host -o ServerAliveInterval=30 -o ForwardAgent=no -o ClearAllForwardings=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			}))
		})

		It("returns ssh options with gateway settings returned from the Director and private key set by user", func() {
			connOpts.GatewayPrivateKeyPath = "/tmp/gw-priv-key"

			result.GatewayUsername = "gw-user"
			result.GatewayHost = "gw-host"

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
				"-o", "ProxyCommand=ssh -tt -W %h:%p -l gw-user gw-host -o ServerAliveInterval=30 -o ForwardAgent=no -o ClearAllForwardings=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o PasswordAuthentication=no -o IdentitiesOnly=yes -o IdentityFile=/tmp/gw-priv-key",
			}))
		})

		It("returns ssh options with gateway settings overridden by user even if the Director specifies some", func() {
			connOpts.GatewayUsername = "user-gw-user"
			connOpts.GatewayHost = "user-gw-host"

			result.GatewayUsername = "gw-user"
			result.GatewayHost = "gw-host"

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
				"-o", "ProxyCommand=ssh -tt -W %h:%p -l user-gw-user user-gw-host -o ServerAliveInterval=30 -o ForwardAgent=no -o ClearAllForwardings=yes -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null",
			}))
		})

		It("returns ssh options without gateway settings if disabled even if user or the Director specifies some", func() {
			connOpts.GatewayDisable = true
			connOpts.GatewayUsername = "user-gw-user"
			connOpts.GatewayHost = "user-gw-host"

			result.GatewayUsername = "gw-user"
			result.GatewayHost = "gw-host"

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
			}))
		})

		It("returns ssh options without socks5 settings if SOCKS5Proxy is set", func() {
			connOpts.GatewayDisable = true
			connOpts.GatewayUsername = "user-gw-user"
			connOpts.GatewayHost = "user-gw-host"
			connOpts.SOCKS5Proxy = "socks5://some-proxy"

			result.GatewayUsername = "gw-user"
			result.GatewayHost = "gw-host"

			cmdOpts, err := act().Start()
			Expect(err).ToNot(HaveOccurred())
			Expect(cmdOpts).To(Equal([]string{
				"-o", "ServerAliveInterval=30",
				"-o", "ForwardAgent=no",
				"-o", "PasswordAuthentication=no",
				"-o", "IdentitiesOnly=yes",
				"-o", "IdentityFile=/tmp/priv-key",
				"-o", "StrictHostKeyChecking=yes",
				"-o", "UserKnownHostsFile=/tmp/known-hosts",
				"-o", "ProxyCommand=nc -X 5 -x some-proxy %h %p",
			}))
		})
	})

	Describe("Finish", func() {
		BeforeEach(func() {
			_, err := session.Start()
			Expect(err).ToNot(HaveOccurred())
		})

		It("removes private key and known hosts files", func() {
			err := session.Finish()
			Expect(err).ToNot(HaveOccurred())
			Expect(fs.FileExists("/tmp/priv-key")).To(BeFalse())
			Expect(fs.FileExists("/tmp/known-hosts")).To(BeFalse())
		})

		It("returns error if deleting private key file fails but still deletes known hosts file", func() {
			fs.RemoveAllStub = func(path string) error {
				if path == "/tmp/priv-key" {
					return errors.New("fake-err")
				}
				return nil
			}
			err := session.Finish()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
			Expect(fs.FileExists("/tmp/known-hosts")).To(BeFalse())
		})

		It("returns error if deleting known hosts file fails but still deletes private key file", func() {
			fs.RemoveAllStub = func(path string) error {
				if path == "/tmp/known-hosts" {
					return errors.New("fake-err")
				}
				return nil
			}
			err := session.Finish()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fake-err"))
			Expect(fs.FileExists("/tmp/priv-key")).To(BeFalse())
		})
	})
})
