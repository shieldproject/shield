package ip_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	fakeip "github.com/cloudfoundry/bosh-agent/platform/net/ip/fakes"
)

var _ = Describe("InterfaceAddressesValidator", func() {
	var (
		interfaceAddrsProvider  *fakeip.FakeInterfaceAddressesProvider
		interfaceAddrsValidator boship.InterfaceAddressesValidator
	)

	BeforeEach(func() {
		interfaceAddrsProvider = &fakeip.FakeInterfaceAddressesProvider{}
		interfaceAddrsValidator = boship.NewInterfaceAddressesValidator(interfaceAddrsProvider)
	})

	Context("when networks match", func() {
		BeforeEach(func() {
			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
				boship.NewSimpleInterfaceAddress("eth1", "5.6.7.8"),
			}
		})

		It("returns nil", func() {
			err := interfaceAddrsValidator.Validate([]boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
				boship.NewSimpleInterfaceAddress("eth1", "5.6.7.8"),
			})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when desired networks do not match actual network IP address", func() {
		BeforeEach(func() {
			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.5"),
			}
		})

		It("fails", func() {
			err := interfaceAddrsValidator.Validate([]boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Validating network interface 'eth0' IP addresses, expected: '1.2.3.4', actual: '1.2.3.5'"))
		})
	})

	Context("when validating manual networks fails", func() {
		BeforeEach(func() {
			interfaceAddrsProvider.GetErr = errors.New("interface-error")
		})

		It("fails", func() {
			err := interfaceAddrsValidator.Validate([]boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("interface-error"))
		})
	})

	Context("when interface is not configured", func() {
		BeforeEach(func() {
			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("another-ethstatic", "1.2.3.5"),
			}
		})

		It("fails", func() {
			err := interfaceAddrsValidator.Validate([]boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Validating network interface 'eth0' IP addresses, no interface configured with that name"))
		})
	})

	Context("when resolv.conf has valid dns configurations", func() {
		It("fails", func() {

		})

	})

	Context("when resolv.conf has invalid dns configurations", func() {

	})
})
