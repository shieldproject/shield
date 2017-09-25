package types

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

const (
	rsaKeyGeneratorKeyBits          = 2048
	rsaKeyGeneratorHeaderPrivateKey = "RSA PRIVATE KEY"
	rsaKeyGeneratorHeaderPublicKey  = "PUBLIC KEY"
)

type RSAKeyGenerator struct{}

func NewRSAKeyGenerator() RSAKeyGenerator {
	return RSAKeyGenerator{}
}

type RSAKey struct {
	PrivateKey string `json:"private_key" yaml:"private_key"`
	PublicKey  string `json:"public_key" yaml:"public_key"`
}

func (g RSAKeyGenerator) Generate(parameters interface{}) (interface{}, error) {
	priv, pub, err := g.generateRSAKeyPair()
	if err != nil {
		return nil, fmt.Errorf("Generating RSA key pair: %s", err)
	}

	pubKeyStr, err := g.publicKeyToPEM(pub)
	if err != nil {
		return nil, fmt.Errorf("Generating RSA public key pair: %s", err)
	}

	return RSAKey{
		PrivateKey: g.privateKeyToPEM(priv),
		PublicKey:  pubKeyStr,
	}, nil
}

func (g RSAKeyGenerator) encodePEM(keyBytes []byte, keyType string) string {
	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	return string(pem.EncodeToMemory(block))
}

func (g RSAKeyGenerator) generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, rsaKeyGeneratorKeyBits)
	if err != nil {
		return nil, nil, err
	}

	public := private.Public().(*rsa.PublicKey)

	return private, public, nil
}

func (g RSAKeyGenerator) privateKeyToPEM(privateKey *rsa.PrivateKey) string {
	return g.encodePEM(x509.MarshalPKCS1PrivateKey(privateKey), rsaKeyGeneratorHeaderPrivateKey)
}

func (g RSAKeyGenerator) publicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	return g.encodePEM(keyBytes, rsaKeyGeneratorHeaderPublicKey), nil
}
