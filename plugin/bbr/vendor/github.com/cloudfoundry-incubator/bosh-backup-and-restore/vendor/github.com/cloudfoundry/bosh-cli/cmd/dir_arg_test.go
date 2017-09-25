package cmd_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("DirOrCWDArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg DirOrCWDArg
		)

		BeforeEach(func() {
			arg = DirOrCWDArg{}
		})

		It("returns with path set", func() {
			err := (&arg).UnmarshalFlag("/some/path")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Path).To(Equal("/some/path"))
		})

		It("returns with cwd path set when it's empty", func() {
			err := (&arg).UnmarshalFlag("")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Path).ToNot(BeEmpty())
		})

		It("returns with cwd path set when it's '.'", func() {
			err := (&arg).UnmarshalFlag(".")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg.Path).ToNot(BeEmpty())
		})
	})
})
