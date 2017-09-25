package types

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

const (
	sshKeyGeneratorKeyBits          = 2048
	sshKeyGeneratorHeaderPrivateKey = "RSA PRIVATE KEY"
)

type SSHKeyGenerator struct{}

func NewSSHKeyGenerator() SSHKeyGenerator {
	return SSHKeyGenerator{}
}

type SSHKey struct {
	PrivateKey           string `json:"private_key" yaml:"private_key"`
	PublicKey            string `json:"public_key" yaml:"public_key"`
	PublicKeyFingerprint string `json:"public_key_fingerprint" yaml:"public_key_fingerprint"`
}

func (g SSHKeyGenerator) Generate(parameters interface{}) (interface{}, error) {
	priv, pub, err := g.generateRSAKeyPair()
	if err != nil {
		return nil, fmt.Errorf("Generating RSA key pair: %s", err)
	}

	sshPubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}

	return SSHKey{
		PrivateKey:           g.privateKeyToPEM(priv),
		PublicKey:            string(ssh.MarshalAuthorizedKey(sshPubKey)),
		PublicKeyFingerprint: g.fingerprintMD5(sshPubKey),
	}, nil
}

func (g SSHKeyGenerator) encodePEM(keyBytes []byte, keyType string) string {
	block := &pem.Block{
		Type:  keyType,
		Bytes: keyBytes,
	}

	return string(pem.EncodeToMemory(block))
}

func (g SSHKeyGenerator) generateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	private, err := rsa.GenerateKey(rand.Reader, sshKeyGeneratorKeyBits)
	if err != nil {
		return nil, nil, err
	}
	public := private.Public().(*rsa.PublicKey)
	return private, public, nil
}

func (g SSHKeyGenerator) privateKeyToPEM(privateKey *rsa.PrivateKey) string {
	return g.encodePEM(x509.MarshalPKCS1PrivateKey(privateKey), sshKeyGeneratorHeaderPrivateKey)
}

func (g SSHKeyGenerator) fingerprintMD5(key ssh.PublicKey) string {
	hash := md5.Sum(key.Marshal())
	out := ""
	for i := 0; i < len(hash); i++ {
		if i > 0 {
			out += ":"
		}
		out += fmt.Sprintf("%02x", hash[i]) // don't forget the leading zeroes
	}
	return out
}
