package crypter

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/jhunt/go-log"
	"golang.org/x/crypto/pbkdf2"
)

type Vault struct {
	URL   string
	Token string
	HTTP  *http.Client
}

type VaultCreds struct {
	SealKey   string `json:"seal_key"`
	RootToken string `json:"root_token"`
}

var status struct {
	Sealed bool `json:"sealed"`
}

func NewVault(url, cacert string) (Vault, error) {
	pool := x509.NewCertPool()
	if cacert != "" {
		if ok := pool.AppendCertsFromPEM([]byte(cacert)); !ok {
			return Vault{}, fmt.Errorf("Invalid or malformed CA Certificate")
		}
	}

	return Vault{
		URL:   url,
		Token: "",
		HTTP: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
				DisableKeepAlives: true,
			},
			Timeout: 30 * time.Second,
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

func (vault *Vault) Init(store string, master string) (string, error) {
	initialized, err := vault.IsInitialized()
	if err != nil {
		return "", err
	}

	if initialized {
		log.Infof("vault is already initialized")

		creds, err := ReadConfig(store, master)
		if err != nil {
			return "", err
		}
		vault.Token = creds.RootToken
		return "", vault.Unseal(creds.SealKey)
	}

	//////////////////////////////////////////

	log.Infof("initializing the vault with 1/1 keys")
	res, err := vault.Do("PUT", "/v1/sys/init", map[string]int{
		"secret_shares":    1,
		"secret_threshold": 1,
	})
	if err != nil {
		log.Errorf("failed to initialize the vault: %s", err)
		return "", err
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Errorf("failed to read response from the vault, concerning our initialization attempt: %s", err)
		return "", err
	}

	var keys struct {
		RootToken string   `json:"root_token"`
		Keys      []string `json:"keys"`
	}
	if err = json.Unmarshal(b, &keys); err != nil {
		log.Errorf("failed to parse response from the vault, concerning our initialization attempt: %s", err)
		return "", err
	}
	if keys.RootToken == "" || len(keys.Keys) != 1 {
		if keys.RootToken == "" {
			log.Errorf("failed to initialize vault: root token was blank")
		}
		if len(keys.Keys) != 1 {
			log.Errorf("failed to initialize vault: incorrect number of seal keys (%d) returned", len(keys.Keys))
		}
		err = fmt.Errorf("invalid response from vault: token '%s' and %d keys", keys.RootToken, len(keys.Keys))
		return "", err
	}

	creds := VaultCreds{
		SealKey:   keys.Keys[0],
		RootToken: keys.RootToken,
	}

	err = WriteConfig(store, master, creds)
	if err != nil {
		return "", err
	}

	vault.Token = creds.RootToken

	//////////////////////////////////////////
	// Unseal and Initialize DR Keys

	err = vault.Unseal(creds.SealKey)
	if err != nil {
		return "", err
	}

	return vault.FixedKeygen()
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

func (vault *Vault) Delete(path string) error {
	res, err := vault.Do("DELETE", fmt.Sprintf("/v1/secret/%s", path), nil)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("API %s", res.Status)
	}
	return nil
}

func (vault *Vault) List(path string) ([]string, error) {
	res, err := vault.Do("LIST", fmt.Sprintf("/v1/secret/%s", path), nil)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("API %s", res.Status)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}
	if err = json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return raw.Data.Keys, nil
}

// CreateBackupEncryptionConfig creats random keys and corresponding iv's for a given cipher
// It returns both a key and iv (hex format)
func (vault *Vault) CreateBackupEncryptionConfig(enctype string) (string, string, error) {
	//Keys/IVs are twice as long as they are treated as hex encoded for OpenSSL compatibility
	//Passing in full enctype in case modes determine key or iv sizes
	cipher := strings.Split(enctype, "-")[0]
	switch cipher {
	case "aes128":
		key, err := keygen(32)
		if err != nil {
			return "", "", err
		}

		iv, err := keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil

	case "aes256":
		key, err := keygen(64)
		if err != nil {
			return "", "", err
		}

		iv, err := keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil
	case "twofish":
		key, err := keygen(64)
		if err != nil {
			return "", "", err
		}

		iv, err := keygen(32)
		if err != nil {
			return "", "", err
		}
		return key, iv, nil
	default:
		return "", "", fmt.Errorf("Invalid cipher '%s' specified for key/iv generation", cipher)
	}
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

func (vault *Vault) FixedKeygen() (string, error) {
	fixedKey, err := keygen(512)
	if err != nil {
		return "", err
	}

	generatedMaterial := pbkdf2.Key([]byte(fixedKey[32:]), []byte(fixedKey[:32]), 4096, 48, sha256.New)

	err = vault.Put("secret/archives/fixed_key", map[string]interface{}{
		"key":  ASCIIHexEncode(hex.EncodeToString(generatedMaterial[:32]), 4),
		"iv":   ASCIIHexEncode(hex.EncodeToString(generatedMaterial[32:]), 4),
		"type": "aes256-ctr",
		"uuid": "fixed-key",
	})

	return fixedKey, err
}
