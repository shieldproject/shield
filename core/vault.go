package core

import (
	"github.com/starkandwayne/shield/core/vault"
)

func (core *Core) Initialize(master string) (bool, string, error) {
	if init, err := core.vault.Initialized(); init || err != nil {
		return init, "", err
	}

	fixed, err := core.vault.Initialize(core.vaultKeyfile, master)
	if err != nil {
		return false, "", err
	}

	if sealed, err := core.vault.Sealed(); sealed || err != nil {
		return false, "", err
	}

	return false, fixed, nil
}

func (core *Core) Unlock(master string) (bool, error) {
	if init, err := core.vault.Initialized(); !init || err != nil {
		return init, err
	}

	creds, err := vault.ReadCrypt(core.vaultKeyfile, master)
	if err != nil {
		return true, err
	}

	core.vault.Token = creds.RootToken
	if err := core.vault.Unseal(creds.SealKey); err != nil {
		return true, err
	}

	if sealed, err := core.vault.Sealed(); sealed == true || err != nil {
		return true, err
	}

	return true, nil
}
