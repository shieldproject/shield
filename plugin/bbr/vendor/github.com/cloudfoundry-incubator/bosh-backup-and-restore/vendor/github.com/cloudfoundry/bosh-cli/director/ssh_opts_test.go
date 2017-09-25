package director_test

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"

	fakeuuid "github.com/cloudfoundry/bosh-utils/uuid/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"

	. "github.com/cloudfoundry/bosh-cli/director"
)

var _ = Describe("NewSSHOpts", func() {
	const UUID = "2a4e8104-dc50-4ad7-939a-2efd53b029ae"
	const ExpUsername = "bosh_2a4e8104dc504ad"

	var (
		uuidGen *fakeuuid.FakeGenerator
	)

	BeforeEach(func() {
		uuidGen = &fakeuuid.FakeGenerator{
			GeneratedUUID: UUID,
		}
	})

	// Windows logon names cannot contain certain characters, and when
	// created through NET USER, which the BOSH Agent does, must be 20
	// characters or less (this is to maintain backwards compatibility
	// with Windows 2000).
	//
	// The invalid characters are:
	//   " / \ [ ] : ; | = , + * ? < >
	//
	// Reference: https://msdn.microsoft.com/en-us/library/bb726984.aspx
	//
	It("generates a username that is compatible with Windows", func() {
		const MaxLength = 20
		const InvalidChars = "\"/\\[]:|<>+=;?*"

		opts, _, err := NewSSHOpts(uuidGen)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(opts.Username)).To(BeNumerically("<=", MaxLength))

		switch n := strings.IndexAny(opts.Username, InvalidChars); {
		case n != -1:
			err = fmt.Errorf("invalid char (%c) in username: %s",
				opts.Username[n], opts.Username)
		case strings.HasSuffix(opts.Username, "."):
			err = fmt.Errorf("username may not end in a period '.': %s",
				opts.Username)
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns opts and private key", func() {
		opts, privKeyStr, err := NewSSHOpts(uuidGen)
		Expect(err).ToNot(HaveOccurred())

		Expect(opts.Username).To(Equal(ExpUsername))
		Expect(opts.PublicKey).ToNot(BeEmpty())
		Expect(privKeyStr).ToNot(BeEmpty())

		privKey, err := ssh.ParseRawPrivateKey([]byte(privKeyStr))
		Expect(err).ToNot(HaveOccurred())
		Expect(privKey).ToNot(BeNil())

		pubKey, err := ssh.NewPublicKey(privKey.(*rsa.PrivateKey).Public())
		Expect(err).ToNot(HaveOccurred())

		Expect(opts.PublicKey).To(Equal(string(ssh.MarshalAuthorizedKey(pubKey))))
	})

	It("generates 2048 bits private key", func() {
		_, privKeyStr, err := NewSSHOpts(uuidGen)
		Expect(err).ToNot(HaveOccurred())

		privKey, err := ssh.ParseRawPrivateKey([]byte(privKeyStr))
		Expect(err).ToNot(HaveOccurred())

		Expect(privKey.(*rsa.PrivateKey).D.BitLen()).To(BeNumerically("~", 2048, 20))
	})

	It("returns error if uuid cannot be generated", func() {
		uuidGen.GenerateError = errors.New("fake-err")

		_, _, err := NewSSHOpts(uuidGen)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fake-err"))
	})
})
