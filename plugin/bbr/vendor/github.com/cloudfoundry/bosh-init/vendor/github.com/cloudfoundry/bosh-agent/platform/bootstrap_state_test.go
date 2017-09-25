package platform_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	platform "github.com/cloudfoundry/bosh-agent/platform"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("State", func() {
	var (
		fs   *fakesys.FakeFileSystem
		path string
		s    *platform.BootstrapState
		err  error
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		path = "/agent_state.json"
		s, err = platform.NewBootstrapState(fs, path)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("SaveState", func() {
		It("saves the state file with the appropriate properties", func() {
			s.Linux = platform.LinuxState{HostsConfigured: true}
			s.SaveState()

			contents, readerr := fs.ReadFile(path)

			Expect(readerr).ToNot(HaveOccurred())
			Expect(string(contents)).To(Equal(`{"Linux":{"hosts_configured":true}}`))
		})

		It("saves the state file with the properties passed in", func() {
			s.Linux = platform.LinuxState{HostsConfigured: true}
			s.SaveState()

			contents, readerr := fs.ReadFile(path)

			Expect(readerr).ToNot(HaveOccurred())
			Expect(string(contents)).To(Equal(`{"Linux":{"hosts_configured":true}}`))
		})

		It("returns an error when it can't write the file", func() {
			s.Linux = platform.LinuxState{HostsConfigured: true}
			fs.WriteFileError = errors.New("ENXIO: disk failed")
			saveerr := s.SaveState()

			Expect(saveerr.Error()).To(ContainSubstring("disk failed"))
		})

		It("does not return an error when it tries to save an empty object", func() {
			s.Linux = platform.LinuxState{}
			saveerr := s.SaveState()

			Expect(saveerr).ToNot(HaveOccurred())
		})
	})

	Describe("NewState", func() {
		Context("When the agent's state file cannot be found", func() {
			It("returns state object with false properties", func() {
				path = "/non-existent/agent_state.json"
				s, err = platform.NewBootstrapState(fs, path)

				Expect(s.Linux.HostsConfigured).To(BeFalse())
			})
		})

		Context("When the path to the agent's state is ''", func() {
			It("returns an error and a state object with false properties", func() {
				s, err = platform.NewBootstrapState(fs, "")

				Expect(s.Linux.HostsConfigured).To(BeFalse())
			})
		})

		Context("When the agent cannot read the state file due to a failed disk", func() {
			It("returns an error and a state object with false properties", func() {
				fs.WriteFileString(path, `{
					"hosts_configured": true,
					"hostname_configured": true
				}`)

				fs.RegisterReadFileError(path, errors.New("ENXIO: disk failed"))

				_, readerr := platform.NewBootstrapState(fs, path)

				Expect(readerr.Error()).To(ContainSubstring("disk failed"))
			})
		})

		Context("When the agent cannot parse the state file due to malformed JSON", func() {
			It("returns an error and a state object with false properties", func() {
				fs.WriteFileString(path, "malformed-JSON")

				_, readerr := platform.NewBootstrapState(fs, path)

				Expect(readerr).To(HaveOccurred())
			})
		})
	})
})
