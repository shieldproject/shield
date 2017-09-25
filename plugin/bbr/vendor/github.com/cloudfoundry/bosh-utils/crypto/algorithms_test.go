package crypto_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-utils/crypto"
)

var _ = Describe("Algorithms", func() {

	Context("digest from a single reader", func() {
		var reader io.Reader

		BeforeEach(func() {
			reader = bytes.NewReader([]byte("something different"))
		})

		Context("sha1", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA1.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("da7102c07515effc353226eac2be923c916c5c94"))
			})
		})

		Context("sha256", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA256.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha256:73af606b33433fa3a699134b39d5f6bce1ab4a6d9ca3263d3300f31fc5776b12"))
			})
		})

		Context("sha512", func() {
			It("computes digest from a reader", func() {
				digest, err := DigestAlgorithmSHA512.CreateDigest(reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(digest.String()).To(Equal("sha512:25b38e5cf4069979d4de934ed6cde40eceec1f7100fc2a5fc38d3569456ab2b7e191bbf5a78b533df94a77fcd48b8cb025a4b5db20720d1ac36ecd9af0c8989a"))
			})
		})

	})
})
