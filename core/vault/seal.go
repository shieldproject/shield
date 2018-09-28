package vault

import (
	"fmt"
)

func (c *Client) Sealed() (bool, error) {
	if ok, err := c.Initialized(); !ok || err != nil {
		return true, err
	}

	/* treat a missing token (unauthenticated) as sealed. */
	if c.Token == "" {
		return true, nil
	}

	var out struct {
		Sealed bool `json:"sealed"`
	}
	if _, err := c.Get("/v1/sys/seal-status", &out); err != nil {
		return true, fmt.Errorf("failed to check current vault seal status: %s", err)
	}

	return out.Sealed, nil
}

func (c *Client) Unseal(key string) error {
	if sealed, err := c.Sealed(); err != nil {
		return err
	} else if !sealed {
		return nil
	}

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
