package types_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"

	. "github.com/cloudfoundry/config-server/types"
)

var _ = Describe("RSAKeyGenerator", func() {
	var generator ValueGenerator

	BeforeEach(func() {
		generator = NewRSAKeyGenerator()
	})

	Context("Generate", func() {
		It("generates an rsa key with 2048 bits", func() {
			rsaKey, err := generator.Generate(nil)
			Expect(err).ToNot(HaveOccurred())

			typedRSAKey := rsaKey.(RSAKey)
			Expect(typedRSAKey.PrivateKey).To(ContainSubstring("PRIVATE KEY"))
			Expect(typedRSAKey.PublicKey).To(ContainSubstring("PUBLIC KEY"))

			privBlock, _ := pem.Decode([]byte(typedRSAKey.PrivateKey))

			privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
			Expect(err).ToNot(HaveOccurred())

			pubBlock, _ := pem.Decode([]byte(typedRSAKey.PublicKey))

			pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
			Expect(err).ToNot(HaveOccurred())

			Expect(privKey.Public()).To(Equal(pubKey))
		})

		It("serializes nicely in json/yaml", func() {
			rsaKey, err := generator.Generate(nil)
			Expect(err).ToNot(HaveOccurred())

			bytes, err := yaml.Marshal(rsaKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(bytes)).To(ContainSubstring("private_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key"))

			bytes, err = json.Marshal(rsaKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(bytes)).To(ContainSubstring("private_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key"))
		})
	})
})
