package types_test

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"

	. "github.com/cloudfoundry/config-server/types"
)

var _ = Describe("SSHKeyGenerator", func() {
	var generator ValueGenerator

	BeforeEach(func() {
		generator = NewSSHKeyGenerator()
	})

	Context("Generate", func() {
		It("generates an ssh key with 2048 bits", func() {
			sshKey, err := generator.Generate(nil)
			Expect(err).ToNot(HaveOccurred())

			typedSSHKey := sshKey.(SSHKey)
			Expect(typedSSHKey.PrivateKey).To(ContainSubstring("PRIVATE KEY"))
			Expect(len(typedSSHKey.PublicKeyFingerprint)).To(Equal(47))
			Expect(typedSSHKey.PublicKey).To(ContainSubstring("ssh-rsa "))

			privBlock, _ := pem.Decode([]byte(typedSSHKey.PrivateKey))

			privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
			Expect(err).ToNot(HaveOccurred())

			pubKey, err := ssh.NewPublicKey(privKey.Public())
			Expect(err).ToNot(HaveOccurred())

			expectedPubKey := ssh.MarshalAuthorizedKey(pubKey)
			Expect(expectedPubKey).To(Equal([]byte(typedSSHKey.PublicKey)))
		})

		It("serializes nicely in json/yaml", func() {
			sshKey, err := generator.Generate(nil)
			Expect(err).ToNot(HaveOccurred())

			bytes, err := yaml.Marshal(sshKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(bytes)).To(ContainSubstring("private_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key_fingerprint"))

			bytes, err = json.Marshal(sshKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(bytes)).To(ContainSubstring("private_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key"))
			Expect(string(bytes)).To(ContainSubstring("public_key_fingerprint"))
		})
	})
})
