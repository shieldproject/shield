package types_test

import (
	. "github.com/cloudfoundry/config-server/types"
	"github.com/cloudfoundry/config-server/types/typesfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValueGeneratorFactoryConcrete", func() {
	var valueGeneratorFactory ValueGeneratorFactory

	Context("GetGenerator", func() {
		BeforeEach(func() {

			valueGeneratorFactory = NewValueGeneratorConcrete(&typesfakes.FakeCertsLoader{})
		})

		It("throws an error for unsupported value types", func() {
			_, err := valueGeneratorFactory.GetGenerator("bad_type")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Unsupported value type: bad_type"))
		})

		It("supports the password type", func() {
			generator, err := valueGeneratorFactory.GetGenerator("password")
			Expect(err).ToNot(HaveOccurred())
			Expect(generator).ToNot(BeNil())
		})

		It("supports the certificate type", func() {
			generator, err := valueGeneratorFactory.GetGenerator("certificate")
			Expect(err).ToNot(HaveOccurred())
			Expect(generator).ToNot(BeNil())
		})

		It("supports the ssh type", func() {
			generator, err := valueGeneratorFactory.GetGenerator("ssh")
			Expect(err).ToNot(HaveOccurred())
			Expect(generator).ToNot(BeNil())
		})

		It("supports the rsa type", func() {
			generator, err := valueGeneratorFactory.GetGenerator("rsa")
			Expect(err).ToNot(HaveOccurred())
			Expect(generator).ToNot(BeNil())
		})
	})
})
