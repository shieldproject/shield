package api

import (
	"github.com/pborman/uuid"
)

type Target struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
	Agent    string `json:"agent"`
}

type TargetFilter struct {
	Name   string
	Plugin string
	Unused YesNo
}

func GetTargets(filter TargetFilter) ([]Target, error) {
	uri := ShieldURI("/v1/targets")
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []Target
	return data, uri.Get(&data)
}

func GetTarget(id uuid.UUID) (Target, error) {
	var data Target
	return data, ShieldURI("/v1/target/%s", id).Get(&data)
}

func CreateTarget(contentJSON string) (Target, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/targets").Post(&data, contentJSON)
	if err == nil {
		return GetTarget(uuid.Parse(data.UUID))
	}
	return Target{}, err
}

func UpdateTarget(id uuid.UUID, contentJSON string) (Target, error) {
	err := ShieldURI("/v1/target/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetTarget(id)
	}
	return Target{}, err
}

func DeleteTarget(id uuid.UUID) error {
	return ShieldURI("/v1/target/%s", id).Delete(nil)
}
