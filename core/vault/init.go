package vault

import (
	"fmt"
)

func (c *Client) Initialized() (bool, error) {
	var out struct {
		Initialized bool `json:"initialized"`
	}
	if _, err := c.Get("/v1/sys/init", &out); err != nil {
		return false, fmt.Errorf("failed to check vault init status: %s", err)
	}
	return out.Initialized, nil
}

/* START HERE <----- refactor */
func (c *Client) Initialize(crypt, master string) (string, error) {
	initialized, err := c.Initialized()
	if err != nil {
		return "", err
	}

	if initialized {
		creds, err := ReadCrypt(crypt, master)
		if err != nil {
			return "", err
		}
		c.Token = creds.RootToken
		return "", c.Unseal(creds.SealKey)
	}

	/* initialize the vault, with one seal key */
	in := struct {
		Shares    int `json:"secret_shares"`
		Threshold int `json:"secret_threshold"`
	}{1, 1}

	var out struct {
		Token string   `json:"root_token"`
		Keys  []string `json:"keys"`
	}

	if err := c.Put("/v1/sys/init", in, &out); err != nil {
		return "", fmt.Errorf("failed to initialize the vault: %s", err)
	}

	if out.Token == "" {
		return "", fmt.Errorf("failed to initialize the vault: no root token returned")
	}
	if len(out.Keys) != 1 {
		return "", fmt.Errorf("failed to initialize the vault: %d seal keys returned (wanted just one)", len(out.Keys))
	}

	c.Token = out.Token
	if err := WriteCrypt(crypt, master, &Credentials{
		SealKey:   out.Keys[0],
		RootToken: out.Token,
	}); err != nil {
		return "", err
	}
	if err := c.Unseal(out.Keys[0]); err != nil {
		return "", err
	}

	k, p, err := c.FixedParameters()
	if err != nil {
		return "", err
	}
	if err := c.StoreFixed(p); err != nil {
		return "", err
	}

	return k, nil
}
