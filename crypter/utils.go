package crypter

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"strings"
)

func encrypt(key, text []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(key, text []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(text) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	text, err = base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return "", err
	}

	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	return string(text), nil
}

func keygen(length int) (string, error) {
	chars := "0123456789ABCDEF"
	var buffer bytes.Buffer

	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		indexInt := index.Int64()
		buffer.WriteString(string(chars[indexInt]))
	}
	return buffer.String(), nil
}

func ASCIIHexEncode(s string, n int) string {
	var buffer bytes.Buffer
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == (n-1) && i != (len(s)-1) {
			buffer.WriteRune('-')
		}
	}
	return buffer.String()
}

func ASCIIHexDecode(s string) string {
	return strings.Replace(s, "-", "", -1)
}
