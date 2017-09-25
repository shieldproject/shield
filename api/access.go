package api

import (
	"fmt"
)

func Unlock(master string) error {
	uri, err := ShieldURI("/v2/unlock")
	if err != nil {
		return err
	}

	respMap := make(map[string]string)
	contentJSON := fmt.Sprintf("{\"master_password\": \"%s\"}", master)
	if err := uri.Post(&respMap, contentJSON); err != nil {
		return err
	}

	return nil
}

func Init(master string) error {
	uri, err := ShieldURI("/v2/init")
	if err != nil {
		return err
	}

	respMap := make(map[string]string)
	contentJSON := fmt.Sprintf("{\"master_password\": \"%s\"}", master)
	if err := uri.Post(&respMap, contentJSON); err != nil {
		return err
	}

	return nil
}
