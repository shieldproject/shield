package api_agent

import (
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

func CreateStore(contentJSON string) (*db.AnnotatedStore, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/stores").Post(&data, contentJSON)
	if err == nil {
		return GetStore(data.UUID)
	}
	return nil, err
}

func UpdateStore(uuid string, contentJSON string) (*db.AnnotatedStore, error) {
	err := ShieldURI("/v1/store/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetStore(uuid)
	}
	return nil, err
}

func DeleteStore(uuid string) error {
	return ShieldURI("/v1/store/%s", uuid).Delete(nil)
}
