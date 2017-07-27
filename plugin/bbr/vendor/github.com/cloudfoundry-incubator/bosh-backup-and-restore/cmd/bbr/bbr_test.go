package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr"
)

var _ = Describe("bbr", func() {
	Describe("ExtractNameFromAddress", func() {
		It("Returns IP address when that's all it's given", func() {
			Expect(ExtractNameFromAddress("10.5.26.522")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP and port", func() {
			Expect(ExtractNameFromAddress("10.5.26.522:53")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP, protocol and port", func() {
			Expect(ExtractNameFromAddress("https://10.5.26.522:53")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP and protocol", func() {
			Expect(ExtractNameFromAddress("https://10.5.26.522")).To(Equal("10.5.26.522"))
		})

		It("Returns hostname when that's all it's given", func() {
			Expect(ExtractNameFromAddress("my.bosh.com")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname and port", func() {
			Expect(ExtractNameFromAddress("my.bosh.com:42")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname, protocol and port", func() {
			Expect(ExtractNameFromAddress("http://my.bosh.com:42")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname and protocol", func() {
			Expect(ExtractNameFromAddress("http://my.bosh.com")).To(Equal("my.bosh.com"))
		})
	})
})
