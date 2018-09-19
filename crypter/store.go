package crypter

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
)

func ReadConfig(path string, master string) (VaultCreds, error) {
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		return VaultCreds{}, fmt.Errorf("master password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return VaultCreds{}, fmt.Errorf("failed to read %s: %s", path, err)
	}

	raw, err := decrypt(key[:], b)
	if err != nil {
		return VaultCreds{}, fmt.Errorf("failed to decrypt %s: %s", path, err)
	}

	creds := VaultCreds{}
	err = json.Unmarshal([]byte(raw), &creds)
	if err != nil {
		return VaultCreds{}, fmt.Errorf("failed to decrypt %s: incorrect master password", path)
	}

	return creds, nil
}

func WriteConfig(path, master string, creds VaultCreds) error {
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		return fmt.Errorf("master password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	b, err := json.Marshal(VaultCreds{
		SealKey:   creds.SealKey,
		RootToken: creds.RootToken,
	})
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
