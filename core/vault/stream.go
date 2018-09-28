package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"strings"
)

func Stream(enctype string, key, iv []byte) (cipher.Stream, cipher.Stream, error) {
	if enctype == "" {
		return nil, nil, fmt.Errorf("No encryption type specified")
	}
	if !strings.Contains(enctype, "-") {
		return nil, nil, fmt.Errorf("Invalid encryption type '%s' specified", enctype)
	}
	typ := strings.Split(enctype, "-")[0]
	mode := strings.Split(enctype, "-")[1]

	var err error
	var block cipher.Block

	switch typ {
	case "aes128", "aes256":
		block, err = aes.NewCipher(key)

	default:
		return nil, nil, fmt.Errorf("Invalid cipher '%s' specified", typ)
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
		return nil, nil, fmt.Errorf("Invalid encryption mode '%s' specified", mode)
	}
}
