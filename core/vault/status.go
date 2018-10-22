package vault

import (
	"fmt"
)

func (vault *Client) Status() (string, error) {
	seal, err := vault.Sealed()
	if err != nil {
		return "", err
	}

	init, err := vault.Initialized()
	if err != nil {
		return "", err
	}

	if init {
		if seal {
			return "locked", nil
		}
		return "unlocked", nil
	}
	return "uninitialized", nil
}

func (c *Client) Initialized() (bool, error) {
	var out struct {
		Initialized bool `json:"initialized"`
	}
	if _, err := c.Get("/v1/sys/init", &out); err != nil {
		return false, fmt.Errorf("failed to check vault init status: %s", err)
	}
	return out.Initialized, nil
}

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
