package types_test

import (
	. "github.com/cloudfoundry/config-server/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PasswordGenerator", func() {

	Describe("passwordGenerator", func() {
		var generator ValueGenerator

		BeforeEach(func() {
			generator = NewPasswordGenerator()
		})

		Context("Generate", func() {
			It("generates a 20 character password", func() {
				password, err := generator.Generate(nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(password.(string))).To(Equal(20))
			})

			It("generates unique passwords", func() {
				password1, err := generator.Generate(nil)
				Expect(err).ToNot(HaveOccurred())

				password2, err := generator.Generate(nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(password1).ToNot(Equal(password2))
			})

			It("only uses allowed characters", func() {
				for i := 0; i < 20; i++ { // arbitrary number
					password, err := generator.Generate(nil)
					Expect(err).ToNot(HaveOccurred())
					Expect(password).To(MatchRegexp("^[a-z0-9]{20}$"))
				}
			})
		})
	})
})
