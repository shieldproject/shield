package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
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

func ReadCrypt(path, master string) (*Credentials, error) {
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		return nil, fmt.Errorf("master password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %s", path, err)
	}

	raw, err := decrypt(key[:], b)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt %s: %s", path, err)
	}

	creds := &Credentials{}
	if err = json.Unmarshal([]byte(raw), &creds); err != nil {
		return nil, fmt.Errorf("failed to decrypt %s: incorrect master password", path)
	}

	return creds, nil
}

func WriteCrypt(path, master string, creds *Credentials) error {
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		return fmt.Errorf("master password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	b, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal vault root token / seal key: %s", err)
	}

	enc, err := encrypt(key[:], b)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault root token / seal key: %s", err)
	}

	err = ioutil.WriteFile(path, []byte(enc), 0600)
	if err != nil {
		return fmt.Errorf("failed to write %s: %s", path, err)
	}

	return nil
}

func (c *Client) Rekey(path, current, proposed string, rotate bool) (string, error) {
	creds, err := ReadCrypt(path, current)
	if err != nil {
		return "", err
	}

	err = WriteCrypt(path, proposed, creds)
	if err != nil {
		return "", err
	}

	if rotate {
		k, p, err := GenerateFixedParameters()
		if err != nil {
			return "", err
		}
		return k, c.StoreFixed(p)
	}
	return "", nil
}
