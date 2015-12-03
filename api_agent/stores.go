package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchStoresList(plugin, unused string) (*[]db.AnnotatedStore, error) {
	uri := ShieldURI("/v1/stores")
	if plugin != "" {
		uri.AddParameter("plugin", plugin)
	}
	if unused != "" {
		uri.AddParameter("unused", unused)
	}
	data := &[]db.AnnotatedStore{}
	return data, uri.Get(&data)
}

func GetStore(uuid string) (*db.AnnotatedStore, error) {
	data := &db.AnnotatedStore{}
	return data, ShieldURI("/v1/store/%s", uuid).Get(&data)
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
