package server_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/config-server/server"
)

var _ = Describe("JwtTokenValidator", func() {
	var jwtTokenValidator JwtTokenValidator
	var privateKey *rsa.PrivateKey

	BeforeEach(func() {
		var publicKey *rsa.PublicKey
		privateKey, publicKey = generateRSAKeyPair()
		jwtTokenValidator = NewJWTTokenValidatorWithKey(publicKey)
	})

	Describe("Validate", func() {
		Context("a valid token", func() {
			It("succeeds", func() {
				token := jwt.NewWithClaims(
					jwt.SigningMethodRS256,
					jwt.MapClaims{
						"scope": []string{"config_server.admin"},
					},
				)

				signedToken, err := token.SignedString(privateKey)
				Expect(err).ToNot(HaveOccurred())

				err = jwtTokenValidator.Validate(signedToken)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error if non-rsa alg is used", func() {
				token := jwt.NewWithClaims(
					jwt.SigningMethodHS256,
					jwt.MapClaims{
						"scope": []string{"config_server.admin"},
					},
				)

				signedToken, err := token.SignedString([]byte("secret"))
				Expect(err).ToNot(HaveOccurred())

				err = jwtTokenValidator.Validate(signedToken)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Invalid signing method"))
			})

			It("returns error if token does not have config_server.admin scope", func() {
				token := jwt.NewWithClaims(
					jwt.SigningMethodRS256,
					jwt.MapClaims{
						"scope": []string{"unknown", "config_server.foo"},
					},
				)

				signedToken, err := token.SignedString(privateKey)
				Expect(err).ToNot(HaveOccurred())

				err = jwtTokenValidator.Validate(signedToken)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Missing required scope: config_server.admin"))
			})
		})

		Context("an invalid token", func() {
			It("returns an error", func() {
				token := jwt.NewWithClaims(
					jwt.SigningMethodRS256,
					jwt.MapClaims{
						"scope": []string{"config_server.admin"},
					},
				)
				differentPrivateKey, _ := generateRSAKeyPair()

				signedToken, err := token.SignedString(differentPrivateKey)
				Expect(err).ToNot(HaveOccurred())

				err = jwtTokenValidator.Validate(signedToken)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Validating token: crypto/rsa: verification error"))
			})
		})
	})

	XIt("prepare assets for integration test", func() {
		token := jwt.NewWithClaims(
			jwt.SigningMethodRS256,
			jwt.MapClaims{
				"scope": []string{"config_server.admin"},
			},
		)

		signedToken, err := token.SignedString(privateKey)
		Expect(err).ToNot(HaveOccurred())

		fmt.Printf("Token:\n%s\n", signedToken)

		pubDER, err := x509.MarshalPKIXPublicKey(privateKey.Public())
		Expect(err).ToNot(HaveOccurred(), "Failed to get der format for PublicKey")

		pubBlk := pem.Block{
			Type:    "PUBLIC KEY",
			Headers: nil,
			Bytes:   pubDER,
		}
		pubPEM := string(pem.EncodeToMemory(&pubBlk))

		fmt.Printf("Private key:\n%s\n", pubPEM)
	})
})

func generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	Expect(err).ToNot(HaveOccurred())

	return private, private.Public().(*rsa.PublicKey)
}
