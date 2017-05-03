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
	Name       string
	Plugin     string
	Unused     YesNo
	ExactMatch YesNo
}

func GetTargets(filter TargetFilter) ([]Target, error) {
	uri, err := ShieldURI("/v1/targets")
	if err != nil {
		return []Target{}, err
	}
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)
	uri.MaybeAddParameter("exact", filter.ExactMatch)

	var data []Target
	return data, uri.Get(&data)
}

func GetTarget(id uuid.UUID) (Target, error) {
	var data Target
	uri, err := ShieldURI("/v1/target/%s", id)
	if err != nil {
		return Target{}, err
	}
	return data, uri.Get(&data)
}

func CreateTarget(contentJSON string) (Target, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	uri, err := ShieldURI("/v1/targets")
	if err != nil {
		return Target{}, err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return Target{}, err
	}
	return GetTarget(uuid.Parse(data.UUID))
}

func UpdateTarget(id uuid.UUID, contentJSON string) (Target, error) {
	uri, err := ShieldURI("/v1/target/%s", id)
	if err != nil {
		return Target{}, err
	}
	if err := uri.Put(nil, contentJSON); err != nil {
		return Target{}, err
	}
	return GetTarget(id)
}

func DeleteTarget(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/target/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}
