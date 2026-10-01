package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateNonce generates a random base64 encoded string suitable for OpenID nonce
func GenerateNonce() (string, error) {
	nonceBytes := make([]byte, 32)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", fmt.Errorf("could not generate nonce")
	}

	return base64.URLEncoding.EncodeToString(nonceBytes), nil
}
