package cmd_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/cmd"
)

var _ = Describe("BoolArg", func() {
	Describe("UnmarshalFlag", func() {
		var (
			arg BoolArg
		)

		BeforeEach(func() {
			arg = false
		})

		for _, v := range []string{"true", "yes", "on", "enable"} {
			It(fmt.Sprintf("populates with true for '%s'", v), func() {
				err := (&arg).UnmarshalFlag(v)
				Expect(err).ToNot(HaveOccurred())
				Expect(arg).To(Equal(BoolArg(true)))
			})
		}

		for _, v := range []string{"false", "no", "off", "disable"} {
			It(fmt.Sprintf("populates with false for '%s'", v), func() {
				err := (&arg).UnmarshalFlag(v)
				Expect(err).ToNot(HaveOccurred())
				Expect(arg).To(Equal(BoolArg(false)))
			})
		}

		It("returns error for unknown values", func() {
			err := (&arg).UnmarshalFlag("val")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Expected boolean variable 'val' to be either 'true' or 'false'"))
		})
	})
})
