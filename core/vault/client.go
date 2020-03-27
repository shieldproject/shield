package vault

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudfoundry-community/vaultkv"
)

type Client struct {
	Prefix string

	vault  vaultkv.Client
	kv     *vaultkv.KV
}

type Credentials struct {
	SealKey   string `json:"seal_key"`
	RootToken string `json:"root_token"`
}

func Connect(uri, cacert string) (*Client, error) {
	pool := x509.NewCertPool()
	if cacert != "" {
		if ok := pool.AppendCertsFromPEM([]byte(cacert)); !ok {
			return nil, fmt.Errorf("Invalid or malformed CA Certificate")
		}
	}

	vaultURI, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("Invalid or malformed Vault URI '%s': %s", uri, err)
	}

	c := &Client{
		Prefix: "secret/secret",
		vault: vaultkv.Client{
			VaultURL: vaultURI,
			Client: &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: pool,
					},
				},
			},
		},
	}
	c.kv = c.vault.NewKV()

	return c, nil
}

type Status int

const (
	Unknown Status = iota
	Blank
	Locked
	Ready
)

func (c *Client) Status() (Status, error) {
	err := c.vault.Health(false)
	if err == nil {
		return Ready, nil
	}
	if vaultkv.IsUninitialized(err) {
		return Blank, nil
	}
	if vaultkv.IsSealed(err) {
		return Locked, nil
	}
	return Unknown, err
}
func (c *Client) StatusString() (string, error) {
	st, err := c.Status()
	if err != nil {
		return "unknown", err
	}
	switch st {
	case Blank:
		return "uninitialized", nil
	case Locked:
		return "locked", nil
	case Ready:
		return "unlocked", nil
	}
	return "unknown", nil
}

func (c *Client) Initialize(crypt, master string) (string, error) {
	/* initialize the vault, with one seal key */
	creds, err := c.vault.InitVault(vaultkv.InitConfig{
		Shares:    1,
		Threshold: 1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize the vault: %s", err)
	}

	if err := WriteCrypt(crypt, master, &Credentials{
		SealKey:   creds.Keys[0],
		RootToken: creds.RootToken,
	}); err != nil {
		return "", err
	}

	if err := creds.Unseal(); err != nil {
		return "", err
	}

	k, p, err := GenerateFixedParameters()
	if err != nil {
		return "", err
	}
	if err := c.StoreFixed(p); err != nil {
		return "", err
	}

	return k, nil
}

func (c *Client) Unseal(crypt, master string) error {
	/* retrieve our seal keys from the crypt file */
	creds, err := ReadCrypt(crypt, master)
	if err != nil {
		return err
	}

	/* unseal the vault */
	_, err = c.vault.Unseal(creds.SealKey)
	return err
}

func (c *Client) Seal() error {
	return c.vault.Seal()
}
