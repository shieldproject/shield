package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchStoresList(plugin, unused string) (*[]db.AnnotatedStore, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedStore{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/stores")
	joiner := "?"
	if plugin != "" {
		uri = fmt.Sprintf("%s%splugin=%s", uri, joiner, plugin)
		joiner = "&"
	}
	if unused != "" {
		uri = fmt.Sprintf("%s%sunused=%s", uri, joiner, unused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetStore(uuid string) (*db.AnnotatedStore, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedStore{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/store/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func CreateStore(content string) (*db.AnnotatedStore, error) {

	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}

	buf := bytes.NewBufferString(content)

	err := makeApiCall(&data, `POST`, `v1/stores`, buf)

	if err == nil {
		return GetStore(data.UUID)
	}
	return nil, err
}

func UpdateStore(uuid string, content string) (*db.AnnotatedStore, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/store/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetStore(uuid)
	}
	return nil, err
}

func DeleteStore(uuid string) error {

	uri := fmt.Sprintf("v1/store/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
