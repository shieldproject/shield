package crypter

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"regexp"
	"strings"

	"github.com/starkandwayne/goutils/log"
)

type Vault struct {
	URL      string
	Token    string
	Insecure bool
	HTTP     *http.Client
}

type VaultCreds struct {
	SealKey   string `json:"seal_key"`
	RootToken string `json:"root_token"`
}

var status struct {
	Sealed bool `json:"sealed"`
}

func NewVault() (Vault, error) {
	return Vault{
		URL:      "http://127.0.0.1:8200",
		Token:    "",
		Insecure: true,
		HTTP: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				req.Header.Add("X-Vault-Token", "")
				return nil
			},
		},
	}, nil
}

func (vault *Vault) Init(store string, master string) error {
	initialized, err := vault.IsInitialized()
	if err != nil {
		return err
	}

	if initialized {
		log.Infof("vault is already initialized")

		creds, err := vault.ReadConfig(store, master)
		if err != nil {
			return err
		}
		vault.Token = creds.RootToken
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

	creds := VaultCreds{
		SealKey:   keys.Keys[0],
		RootToken: keys.RootToken,
	}

	err = vault.WriteConfig(store, master, creds)
	if err != nil {
		return err
	}

	vault.Token = creds.RootToken
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
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		log.Errorf("Failed to decrypt vault credentials: Master Password must contain only printable chars")
		return VaultCreds{}, errors.New("Failed to decrypt vault credentials: Master Password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	log.Debugf("reading credentials files from %s", store)
	b, err := ioutil.ReadFile(store)
	if err != nil {
		log.Errorf("failed to read vault credentials from %s: %s", store, err)
		return VaultCreds{}, err
	}

	jsonCreds, err := vault.decrypt(key[:], b)
	if err != nil {
		log.Errorf("Failed to decrypt sealkey for longterm storage: %s", err)
		return VaultCreds{}, err
	}

	plainCreds := VaultCreds{}
	err = json.Unmarshal([]byte(jsonCreds), &plainCreds)
	if err != nil {
		log.Errorf("Failed to decrypt vault credentials: Incorrect Password")
		return VaultCreds{}, errors.New("Failed to decrypt vault credentials: Incorrect Password")
	}

	return plainCreds, err
}

func (vault *Vault) WriteConfig(store string, master string, creds VaultCreds) error {
	if !regexp.MustCompile(`^[\x20-\x7e]+$`).Match([]byte(master)) {
		log.Errorf("Failed to encrypt vault credentials: Master Password must contain only printable chars")
		return errors.New("Failed to encrypt vault credentials: Master Password must contain only printable chars")
	}

	key := sha256.Sum256([]byte(master))
	encryptedCreds := VaultCreds{
		SealKey:   creds.SealKey,
		RootToken: creds.RootToken,
	}

	log.Debugf("marshaling credentials for longterm storage")
	jsonCreds, err := json.Marshal(encryptedCreds)
	if err != nil {
		log.Errorf("failed to marshal vault root token / seal key for longterm storage: %s", err)
		return err
	}

	encCreds, err := vault.encrypt(key[:], jsonCreds)
	if err != nil {
		log.Errorf("Failed to encrypt vault credentials for longterm storage: %s", err)
		return err
	}

	log.Debugf("storing credentials at %s (mode 0600)", store)
	err = ioutil.WriteFile(store, []byte(encCreds), 0600)
	if err != nil {
		log.Errorf("failed to write credentials to longterm storage file %s: %s", store, err)
		return err
	}
	return nil
}

// CreateBackupEncryptionConfig creats random keys and corresponding iv's for a given cipher
// It returns both a key and iv (hex format)
func (vault *Vault) CreateBackupEncryptionConfig(enctype string) (string, string, error) {
	//Keys/IVs are twice as long as they are treated as hex encoded for OpenSSL compatibility
	//Passing in full enctype in case modes determine key or iv sizes
	cipher := strings.Split(enctype, "-")[0]
	switch cipher {
	case "aes128":
		key, err := vault.Keygen(32)
		if err != nil {
			return "", "", err
		}

		iv, err := vault.Keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil

	case "aes256":
		key, err := vault.Keygen(64)
		if err != nil {
			return "", "", err
		}

		iv, err := vault.Keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil
	case "twofish":
		key, err := vault.Keygen(64)
		if err != nil {
			return "", "", err
		}

		iv, err := vault.Keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil
	default:
		return "", "", fmt.Errorf("Invalid cipher '%s' specified for key/iv generation", cipher)
	}
}

func (vault *Vault) Keygen(length int) (string, error) {
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

func (vault *Vault) ASCIIHexEncode(s string, n int) string {
	var buffer bytes.Buffer
	for i, rune := range s {
		buffer.WriteRune(rune)
		if i%n == (n-1) && i != (len(s)-1) {
			buffer.WriteRune('-')
		}
	}
	return buffer.String()
}

func (vault *Vault) IsSealed() (bool, error) {
	if init, err := vault.IsInitialized(); err != nil || init == false {
		return true, err
	}
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
	//Treat unauthorized/missing root token as sealed - token needs to be read from encrypted config
	seal_status := status.Sealed || vault.Token == ""

	return seal_status, err
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

func (vault *Vault) Status() (string, error) {
	vaultSealed, err := vault.IsSealed()
	if err != nil {
		return "", err
	}

	vaultInit, err := vault.IsInitialized()
	if err != nil {
		return "", err
	}

	if vaultInit {
		if vaultSealed {
			return "sealed", nil
		}
		return "unsealed", nil
	}
	return "uninitialized", nil
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
