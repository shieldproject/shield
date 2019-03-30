package vault

import (
	"fmt"
)

func (c *Client) Initialize(crypt, master string) (string, error) {
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

	if err := c.unseal(out.Keys[0]); err != nil {
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
