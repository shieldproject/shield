package core

import (
	"github.com/starkandwayne/shield/crypter"
)

func (core *Core) Initialize(master string) (bool, string, error) {
	if init, err := core.vault.IsInitialized(); init || err != nil {
		return init, "", err
	}

	fixedKey, err := core.vault.Init(core.vaultKeyfile, master)
	if err != nil {
		return false, "", err
	}

	if sealed, err := core.vault.IsSealed(); sealed || err != nil {
		return false, "", err
	}

	return false, fixedKey, nil
}

func (core *Core) Unlock(master string) (bool, error) {
	if init, err := core.vault.IsInitialized(); !init || err != nil {
		return init, err
	}

	creds, err := crypter.ReadConfig(core.vaultKeyfile, master)
	if err != nil {
		return true, err
	}

	core.vault.Token = creds.RootToken
	if err := core.vault.Unseal(creds.SealKey); err != nil {
		return true, err
	}

	if sealed, err := core.vault.IsSealed(); sealed == true || err != nil {
		return true, err
	}

	return true, nil
}

func (core *Core) Rekey(current, proposed string, rotateFixed bool) (string, error) {
	creds, err := crypter.ReadConfig(core.vaultKeyfile, current)
	if err != nil {
		return "", err
	}

	err = crypter.WriteConfig(core.vaultKeyfile, proposed, creds)
	if err != nil {
		return "", err
	}

	if rotateFixed {
		return core.vault.FixedKeygen()
	}

	return "", nil
}
