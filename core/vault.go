package core

func (core *Core) Initialize(master string) (bool, string, error) {
	if init, err := core.vault.IsInitialized(); init || err != nil {
		return init, "", err
	}

	disasterKey, err := core.vault.Init(core.vaultKeyfile, master)
	if err != nil {
		return false, "", err
	}

	if sealed, err := core.vault.IsSealed(); sealed || err != nil {
		return false, "", err
	}

	return false, disasterKey, nil
}

func (core *Core) Unlock(master string) (bool, error) {
	if init, err := core.vault.IsInitialized(); !init || err != nil {
		return init, err
	}

	creds, err := core.vault.ReadConfig(core.vaultKeyfile, master)
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

func (core *Core) Rekey(current, proposed string, rekeyDR bool) (string, error) {
	creds, err := core.vault.ReadConfig(core.vaultKeyfile, current)
	if err != nil {
		return "", err
	}

	err = core.vault.WriteConfig(core.vaultKeyfile, proposed, creds)
	if err != nil {
		return "", err
	}

	if rekeyDR {
		return core.vault.DisasterKeygen()
	}

	return "", nil
}
