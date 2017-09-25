package net_test

import (
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform/net"
)

var _ = Describe("DNSValidator", func() {
	var (
		dnsValidator DNSValidator
		fs           *fakesys.FakeFileSystem
	)

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		dnsValidator = NewDNSValidator(fs)
	})

	Context("when /etc/resolv.conf contains at least one dns server", func() {
		BeforeEach(func() {
			fs.WriteFileString("/etc/resolv.conf", `
				nameserver 8.8.8.8
				nameserver 9.9.9.9`)
		})

		It("returns nil", func() {
			err := dnsValidator.Validate([]string{"8.8.8.8", "10.10.10.10"})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when reading /etc/resolv.conf failed", func() {
		It("returns error", func() {
			err := dnsValidator.Validate([]string{"8.8.8.8", "9.9.9.9"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Reading /etc/resolv.conf"))
		})
	})

	Context("when /etc/resolv.conf does not contain any of the dns servers specified in the manifest", func() {
		BeforeEach(func() {
			fs.WriteFileString("/etc/resolv.conf", `nameserver 6.6.6.6`)
		})

		It("returns error", func() {
			err := dnsValidator.Validate([]string{"8.8.8.8", "9.9.9.9"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("None of the DNS servers that were specified in the manifest were found in /etc/resolv.conf."))
		})
	})
})
