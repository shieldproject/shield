package crypter

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/twofish"
)

func Stream(enctype string, key, iv []byte) (cipher.Stream, cipher.Stream, error) {
	// cipher-mode combinations included so far are:
	// aes/twofish+ctr/orb/cfb
	if enctype == "" {
		return nil, nil, fmt.Errorf("No encryption type specified")
	}
	if !strings.Contains(enctype, "-") {
		return nil, nil, fmt.Errorf("Invalid encryption type '%s' specified", enctype)
	}
	cipherName := strings.Split(enctype, "-")[0]
	mode := strings.Split(enctype, "-")[1]

	var err error
	var block cipher.Block

	switch cipherName {
	//keysize determines aes128 vs 256.
	case "aes128", "aes256":
		block, err = aes.NewCipher(key)
	case "twofish":
		block, err = twofish.NewCipher(key)
	default:
		return nil, nil, fmt.Errorf("Invalid cipher '%s' specified", cipherName)
	}

	if err != nil {
		return nil, nil, err
	}

	switch mode {
	case "cfb":
		return cipher.NewCFBEncrypter(block, iv), cipher.NewCFBDecrypter(block, iv), nil
	case "ofb":
		return cipher.NewOFB(block, iv), cipher.NewOFB(block, iv), nil
	case "ctr":
		return cipher.NewCTR(block, iv), cipher.NewCTR(block, iv), nil
	default:
		return nil, nil, fmt.Errorf("Invalid encryption mode '%s' specified", cipherName)
	}
}

func Initialize() (string, string, error) {
	//Generate the random IV
	iv := make([]byte, twofish.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", "", err
	}
	return "", "", nil
}
