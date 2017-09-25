package cmd_test

import (
	semver "github.com/cppforlife/go-semi-semantic/version"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("VersionArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg VersionArg
		)

		BeforeEach(func() {
			arg = VersionArg{}
		})

		It("returns parsed version", func() {
			err := (&arg).UnmarshalFlag("1.1")
			Expect(err).ToNot(HaveOccurred())
			Expect(arg).To(Equal(VersionArg(semver.MustNewVersionFromString("1.1"))))
		})

		It("returns error if it cannot be parsed", func() {
			err := (&arg).UnmarshalFlag("1.1~ver")
			Expect(err).To(HaveOccurred())
		})
	})
})
