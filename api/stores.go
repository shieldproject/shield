package api

import (
	"github.com/pborman/uuid"
)

type Store struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

type StoreFilter struct {
	Name   string
	Plugin string
	Unused YesNo
}

func GetStores(filter StoreFilter) ([]Store, error) {
	uri := ShieldURI("/v1/stores")
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)
	var data []Store
	return data, uri.Get(&data)
}

func GetStore(id uuid.UUID) (Store, error) {
	var data Store
	return data, ShieldURI("/v1/store/%s", id).Get(&data)
}

func CreateStore(contentJSON string) (Store, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/stores").Post(&data, contentJSON)
	if err == nil {
		return GetStore(uuid.Parse(data.UUID))
	}
	return Store{}, err
}

func UpdateStore(id uuid.UUID, contentJSON string) (Store, error) {
	err := ShieldURI("/v1/store/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetStore(id)
	}
	return Store{}, err
}

func DeleteStore(id uuid.UUID) error {
	return ShieldURI("/v1/store/%s", id).Delete(nil)
}
