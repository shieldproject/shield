package core

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/starkandwayne/goutils/log"
)

type Vault struct {
	URL            string
	Token          string
	EncryptionType string
	Insecure       bool
	HTTP           *http.Client
}

type VaultCreds struct {
	SealKey        string `json:"seal_key"`
	RootToken      string `json:"root_token"`
	EncryptionType string `json:"encryption_type"`
}

var status struct {
	Sealed bool `json:"sealed"`
}

func (vault *Vault) Init(store string) error {
	initialized, err := vault.IsInitialized()
	if err != nil {
		return err
	}

	if initialized {
		log.Infof("vault is already initialized")

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter Master Password: ")
		master, _ := reader.ReadString('\n')
		master = strings.TrimSpace(master)

		creds, err := vault.ReadConfig(store, master)
		if err != nil {
			return err
		}
		vault.Token = creds.RootToken
		vault.EncryptionType = creds.EncryptionType
		os.Setenv("VAULT_TOKEN", vault.Token)
		return vault.Unseal(creds.SealKey)
	}

	//////////////////////////////////////////

	log.Infof("initializing the vault with 1/1 keys")
	res, err := vault.Do("PUT", "/v1/sys/init", map[string]int{
		"secret_shares":    1,
		"secret_threshold": 1,
	})
	if err != nil {
		log.Errorf("failed to initialize the vault: %s", err)
		return err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read response from the vault, concerning our initialization attempt: %s", err)
		return err
	}

	var keys struct {
		RootToken string   `json:"root_token"`
		Keys      []string `json:"keys"`
	}
	if err = json.Unmarshal(b, &keys); err != nil {
		log.Errorf("failed to parse response from the vault, concerning our initialization attempt: %s", err)
		return err
	}
	if keys.RootToken == "" || len(keys.Keys) != 1 {
		if keys.RootToken == "" {
			log.Errorf("failed to initialize vault: root token was blank")
		}
		if len(keys.Keys) != 1 {
			log.Errorf("failed to initialize vault: incorrect number of seal keys (%d) returned", len(keys.Keys))
		}
		err = fmt.Errorf("invalid response from vault: token '%s' and %d keys", keys.RootToken, len(keys.Keys))
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Encryption Cipher (aes-128, aes-256, twofish): ")
	encryptionCipher, _ := reader.ReadString('\n')
	encryptionCipher = strings.TrimSpace(encryptionCipher)

	fmt.Print("Enter Encryption Mode (cfb, ofb, ctr): ")
	encryptionMode, _ := reader.ReadString('\n')
	encryptionType := strings.TrimSpace(encryptionCipher) + "-" + strings.TrimSpace(encryptionMode)
	encryptionType = strings.TrimSpace(encryptionType)

	fmt.Print("Enter Master Password: ")
	master, _ := reader.ReadString('\n')
	master = strings.TrimSpace(master)

	creds := VaultCreds{
		SealKey:        keys.Keys[0],
		RootToken:      keys.RootToken,
		EncryptionType: encryptionType,
	}

	vault.WriteConfig(store, master, creds)

	vault.Token = creds.RootToken
	vault.EncryptionType = encryptionType
	return vault.Unseal(creds.SealKey)
}

func (vault *Vault) Unseal(key string) error {

	sealed, err := vault.IsSealed()
	if err != nil {
		return err
	}

	if !sealed {
		log.Infof("vault is already unsealed")
		return nil
	}

	//////////////////////////////////////////

	log.Infof("vault is sealed; unsealing it")
	res, err := vault.Do("POST", "/v1/sys/unseal", map[string]string{
		"key": key,
	})
	if err != nil {
		log.Errorf("failed to unseal vault: %s", err)
		return err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read response from the vault, concerning our unseal attempt: %s", err)
		return err
	}

	err = json.Unmarshal(b, &status)
	if err != nil {
		log.Errorf("failed to parse response from the vault, concerning our unseal attempt: %s", err)
		return err
	}

	if status.Sealed {
		err = fmt.Errorf("vault is still sealed after unseal attempt")
		log.Errorf("%s", err)
		return err
	}

	log.Infof("unsealed the vault")
	return nil
}

func (vault *Vault) NewRequest(method, url string, data interface{}) (*http.Request, error) {
	if data == nil {
		return http.NewRequest(method, url, nil)
	}
	cooked, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return http.NewRequest(method, url, strings.NewReader(string(cooked)))
}

func (vault *Vault) Do(method, url string, data interface{}) (*http.Response, error) {
	req, err := vault.NewRequest(method, fmt.Sprintf("%s%s", vault.URL, url), data)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Vault-Token", vault.Token)
	return vault.HTTP.Do(req)
}

func (vault *Vault) Get(path string) (map[string]interface{}, bool, error) {
	exists := false

	res, err := vault.Do("GET", fmt.Sprintf("/v1/secret/%s", path), nil)
	if err != nil {
		return nil, exists, err
	}
	if res.StatusCode == 404 {
		return nil, exists, nil
	}
	if res.StatusCode != 200 && res.StatusCode != 204 {
		return nil, exists, fmt.Errorf("API %s", res.Status)
	}

	exists = true
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, exists, err
	}

	var raw map[string]interface{}
	if err = json.Unmarshal(b, &raw); err != nil {
		return nil, exists, err
	}

	if x, ok := raw["data"]; ok {
		return x.(map[string]interface{}), exists, nil
	}

	return nil, exists, fmt.Errorf("Malformed response from Vault")
}

func (vault *Vault) Put(path string, data interface{}) error {
	res, err := vault.Do("POST", fmt.Sprintf("/v1/secret/%s", path), data)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 && res.StatusCode != 204 {
		return fmt.Errorf("API %s", res.Status)
	}
	return nil
}

func (vault *Vault) ReadConfig(store string, master string) (VaultCreds, error) {
	key := sha256.Sum256([]byte(master))
	log.Debugf("reading credentials files from %s", store)
	b, err := ioutil.ReadFile(store)
	if err != nil {
		log.Errorf("failed to read vault credentials from %s: %s", store, err)
		return VaultCreds{}, err
	}
	creds := VaultCreds{}
	err = json.Unmarshal(b, &creds)
	if err != nil {
		log.Errorf("failed to parse vault credentials from %s: %s", store, err)
		return VaultCreds{}, err
	}

	plainSeal, err := vault.decrypt(key[:], []byte(creds.SealKey))
	if err != nil {
		log.Errorf("Failed to encrypt sealkey for longterm storage: %s", err)
		return VaultCreds{}, err
	}

	plainRoot, err := vault.decrypt(key[:], []byte(creds.RootToken))
	if err != nil {
		log.Errorf("Failed to encrypt root token for longterm storage: %s", err)
		return VaultCreds{}, err
	}

	plainType, err := vault.decrypt(key[:], []byte(creds.EncryptionType))
	if err != nil {
		log.Errorf("Failed to encrypt type for longterm storage: %s", err)
		return VaultCreds{}, err
	}

	plainCreds := VaultCreds{
		SealKey:        plainSeal,
		RootToken:      plainRoot,
		EncryptionType: plainType,
	}

	return plainCreds, err
}

func (vault *Vault) WriteConfig(store string, master string, creds VaultCreds) error {

	key := sha256.Sum256([]byte(master))
	encSeal, err := vault.encrypt(key[:], []byte(creds.SealKey))
	if err != nil {
		log.Errorf("Failed to encrypt sealkey for longterm storage: %s", err)
		return err
	}

	encRoot, err := vault.encrypt(key[:], []byte(creds.RootToken))
	if err != nil {
		log.Errorf("Failed to encrypt root token for longterm storage: %s", err)
		return err
	}

	encType, err := vault.encrypt(key[:], []byte(creds.EncryptionType))
	if err != nil {
		log.Errorf("Failed to encrypt type for longterm storage: %s", err)
		return err
	}

	encryptedCreds := VaultCreds{
		SealKey:        encSeal,
		RootToken:      encRoot,
		EncryptionType: encType,
	}

	log.Debugf("marshaling credentials for longterm storage")
	b, err := json.Marshal(encryptedCreds)
	if err != nil {
		log.Errorf("failed to marshal vault root token / seal key for longterm storage: %s", err)
		return err
	}
	log.Debugf("storing credentials at %s (mode 0600)", store)
	err = ioutil.WriteFile(store, b, 0600)
	if err != nil {
		log.Errorf("failed to write credentials to longterm storage file %s: %s", store, err)
		return err
	}
	return nil
}

func (vault *Vault) Gen(length int) (string, error) {
	chars := "0123456789ABCDEF"
	var buffer bytes.Buffer

	for i := 0; i < length; i++ {
		if i > 0 && i%4 == 0 {
			buffer.WriteString("-")
		}
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		indexInt := index.Int64()
		buffer.WriteString(string(chars[indexInt]))
	}

	return buffer.String(), nil
}

func (vault *Vault) IsSealed() (bool, error) {
	res, err := vault.Do("GET", "/v1/sys/seal-status", nil)
	if err != nil {
		log.Errorf("failed to check current seal status of the vault: %s", err)
		return true, err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read response from the vault, concerning current seal status: %s", err)
		return true, err
	}

	err = json.Unmarshal(b, &status)
	if err != nil {
		log.Errorf("failed to parse response from the vault, concerning current seal status: %s", err)
		return true, err
	}

	return status.Sealed, err
}

func (vault *Vault) IsInitialized() (bool, error) {
	res, err := vault.Do("GET", "/v1/sys/init", nil)
	if err != nil {
		log.Errorf("failed to check initialization state of the vault: %s", err)
		return false, err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read response from the vault, concerning its initialization state: %s", err)
		return false, err
	}
	var init struct {
		Initialized bool `json:"initialized"`
	}
	if err = json.Unmarshal(b, &init); err != nil {
		log.Errorf("failed to parse response from the vault, concerning its initialization state: %s", err)
		return false, err
	}
	return init.Initialized, err
}

func (vault *Vault) encrypt(key, text []byte) (string, error) {
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

func (vault *Vault) decrypt(key, text []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	if len(text) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
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
