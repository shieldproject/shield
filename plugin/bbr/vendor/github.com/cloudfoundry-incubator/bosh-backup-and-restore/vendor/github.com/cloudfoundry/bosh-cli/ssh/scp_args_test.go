package ssh_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	. "github.com/cloudfoundry/bosh-cli/ssh"
)

var _ = Describe("SCPArgs", func() {
	var (
		host boshdir.Host
	)

	BeforeEach(func() {
		host = boshdir.Host{Username: "user", Host: "127.0.0.1"}
	})

	Describe("AllOrInstanceGroupOrInstanceSlug", func() {
		It("returns slug of the first named host", func() {
			scpArgs := NewSCPArgs([]string{"host:arg1"}, true)
			slug, err := scpArgs.AllOrInstanceGroupOrInstanceSlug()
			Expect(err).ToNot(HaveOccurred())
			Expect(slug).To(Equal(boshdir.NewAllOrInstanceGroupOrInstanceSlug("host", "")))
		})

		It("returns error if host cannot be parsed into a slug", func() {
			scpArgs := NewSCPArgs([]string{"/:arg1"}, true)
			_, err := scpArgs.AllOrInstanceGroupOrInstanceSlug()
			Expect(err).To(HaveOccurred())
		})

		It("returns error if host is not specified", func() {
			scpArgs := NewSCPArgs([]string{"arg1"}, true)
			_, err := scpArgs.AllOrInstanceGroupOrInstanceSlug()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(
				"Missing remote host information in source/destination arguments"))
		})
	})

	Describe("ForHost", func() {
		It("includes -r at the beginning when it's recursive", func() {
			scpArgs := NewSCPArgs([]string{"arg1"}, true)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"-r", "arg1"}))
		})

		It("does not include -r when it's not recursive", func() {
			scpArgs := NewSCPArgs([]string{"arg1"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"arg1"}))
		})

		It("replaces named host with resolved host info", func() {
			scpArgs := NewSCPArgs([]string{"host:arg1"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"user@127.0.0.1:arg1"}))

			scpArgs = NewSCPArgs([]string{"host:arg1", "arg2"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"user@127.0.0.1:arg1", "arg2"}))

			scpArgs = NewSCPArgs([]string{"host:arg1", "host:arg2"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"user@127.0.0.1:arg1", "user@127.0.0.1:arg2"}))
		})

		It("replaces named host keeping remaining :s", func() {
			scpArgs := NewSCPArgs([]string{"host:some:file"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"user@127.0.0.1:some:file"}))
		})

		It("returns as is if no host info is included", func() {
			scpArgs := NewSCPArgs([]string{"arg1", "arg2"}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{"arg1", "arg2"}))
		})

		It("returns empty when it's empty", func() {
			scpArgs := NewSCPArgs([]string{}, false)
			Expect(scpArgs.ForHost(host)).To(Equal([]string{}))
		})
	})
})
