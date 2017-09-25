package infrastructure_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/infrastructure"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("DigDNSResolver", func() {
	var (
		resolver DigDNSResolver
		runner   *fakesys.FakeCmdRunner
	)

	BeforeEach(func() {
		logger := boshlog.NewLogger(boshlog.LevelNone)
		runner = fakesys.NewFakeCmdRunner()
		resolver = NewDigDNSResolver(runner, logger)
	})

	Describe("LookupHost", func() {
		Context("when host is an ip", func() {
			It("lookup host with an ip", func() {
				ip, err := resolver.LookupHost([]string{"8.8.8.8"}, "74.125.239.101")
				Expect(err).ToNot(HaveOccurred())
				Expect(runner.RunCommands).To(BeEmpty())
				Expect(ip).To(Equal("74.125.239.101"))
			})
		})

		Context("when host is not an ip", func() {
			It("returns 127.0.0.1 for 'localhost'", func() {
				ip, err := resolver.LookupHost([]string{"8.8.8.8"}, "localhost")
				Expect(err).ToNot(HaveOccurred())
				Expect(ip).To(Equal("127.0.0.1"))
			})

			It("returns ip for resolved host", func() {
				digResult := fakesys.FakeCmdResult{
					Stdout: "74.125.19.99",
				}
				runner.AddCmdResult("dig @8.8.8.8 google.com. +short +time=1", digResult)
				ip, err := resolver.LookupHost([]string{"8.8.8.8"}, "google.com.")
				Expect(err).ToNot(HaveOccurred())
				Expect(ip).To(Equal("74.125.19.99"))
			})

			It("returns ip for resolved host after failing and then succeeding", func() {
				digResult := fakesys.FakeCmdResult{
					Stdout: "74.125.19.99",
				}
				runner.AddCmdResult("dig @8.8.8.8 google.com. +short +time=1", digResult)
				ip, err := resolver.LookupHost([]string{"127.0.0.127", "8.8.8.8"}, "google.com.")
				Expect(err).ToNot(HaveOccurred())
				Expect(ip).To(Equal("74.125.19.99"))
			})

			It("returns error if there are 0 dns servers", func() {
				ip, err := resolver.LookupHost([]string{}, "google.com.")
				Expect(err).To(MatchError("No DNS servers provided"))
				Expect(ip).To(BeEmpty())
			})

			It("returns error if all dns servers cannot resolve it", func() {
				ip, err := resolver.LookupHost([]string{"8.8.8.8"}, "google.com.local.")
				Expect(err).To(HaveOccurred())
				Expect(ip).To(BeEmpty())
			})
		})
	})
})
