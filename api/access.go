package api

import (
	"encoding/json"
	"errors"
)

func Unlock(master string) error {
	uri, err := ShieldURI("/v2/unlock")
	if err != nil {
		return err
	}

	creds := struct {
		Master string `json:"master_password"`
	}{
		Master: master,
	}
	contentJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	respMap := make(map[string]string)
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		if init_error, present := respMap["error"]; present {
			return errors.New(init_error)
		}
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
	creds := struct {
		Master string `json:"master_password"`
	}{
		Master: master,
	}
	contentJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		if init_error, present := respMap["error"]; present {
			return errors.New(init_error)
		}
		return err
	}

	return nil
}

func Rekey(current, proposed string) error {
	uri, err := ShieldURI("/v2/rekey")
	if err != nil {
		return err
	}

	respMap := make(map[string]string)
	creds := struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}{
		Current: current,
		New:     proposed,
	}
	b, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err := uri.Post(&respMap, string(b)); err != nil {
		if rekey_error, present := respMap["error"]; present {
			return errors.New(rekey_error)
		}
		return err
	}

	return nil
}
