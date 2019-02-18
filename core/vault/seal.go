package vault

import (
	"fmt"
)

func (c *Client) unseal(key string) error {
	in := struct {
		Key string `json:"key"`
	}{
		Key: key,
	}
	var out struct {
		Sealed bool `json:"sealed"`
	}
	if err := c.Post("/v1/sys/unseal", in, &out); err != nil {
		return fmt.Errorf("failed to unseal vault: %s", err)
	}

	if out.Sealed {
		return fmt.Errorf("unseal attempt failed to unseal vault")
	}
	return nil
}

func (c *Client) Unseal(crypt, master string) error {
	/* retrieve our seal keys from the crypt file */
	creds, err := ReadCrypt(crypt, master)
	if err != nil {
		return err
	}
	c.Token = creds.RootToken

	/* unseal the vault */
	return c.unseal(creds.SealKey)
}

func (c *Client) Seal() error {
	if err := c.Put("/v1/sys/seal", nil, nil); err != nil {
		return fmt.Errorf("failed to seal vault: %s", err)
	}
	return nil
}
