package main

import (
	"encoding/json"
	"fmt"
)

func RawJSON(raw interface{}) error {
	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", string(b))
	return nil
}

func RawUUID(uuid string) error {
	fmt.Printf("%s\n", uuid)
	return nil
}
