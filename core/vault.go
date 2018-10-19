package core

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

	return true, core.vault.Unseal(core.vaultKeyfile, master)
}
