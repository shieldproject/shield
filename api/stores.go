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
	Name       string
	Plugin     string
	Unused     YesNo
	ExactMatch YesNo
}

func GetStores(filter StoreFilter) ([]Store, error) {
	uri, err := ShieldURI("/v1/stores")
	if err != nil {
		return []Store{}, err
	}
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)
	uri.MaybeAddParameter("exact", filter.ExactMatch)
	var data []Store
	return data, uri.Get(&data)
}

func GetStore(id uuid.UUID) (Store, error) {
	var data Store
	uri, err := ShieldURI("/v1/store/%s", id)
	if err != nil {
		return Store{}, err
	}
	return data, uri.Get(&data)
}

func CreateStore(contentJSON string) (Store, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	uri, err := ShieldURI("/v1/stores")
	if err != nil {
		return Store{}, err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return Store{}, err
	}
	return GetStore(uuid.Parse(data.UUID))
}

func UpdateStore(id uuid.UUID, contentJSON string) (Store, error) {
	uri, err := ShieldURI("/v1/store/%s", id)
	if err != nil {
		return Store{}, err
	}
	if err := uri.Put(nil, contentJSON); err != nil {
		return Store{}, err
	}
	return GetStore(id)
}

func DeleteStore(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/store/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}
