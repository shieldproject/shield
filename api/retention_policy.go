package api

import (
	"github.com/pborman/uuid"
)

type RetentionPolicyFilter struct {
	Unused YesNo
	Name   string
}

type RetentionPolicy struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires uint   `json:"expires"`
}

func GetRetentionPolicies(filter RetentionPolicyFilter) ([]RetentionPolicy, error) {
	uri := ShieldURI("/v1/retention")
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []RetentionPolicy
	return data, uri.Get(&data)
}

func GetRetentionPolicy(id uuid.UUID) (RetentionPolicy, error) {
	var data RetentionPolicy
	return data, ShieldURI("/v1/retention/%s", id).Get(&data)
}

func CreateRetentionPolicy(contentJSON string) (RetentionPolicy, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v1/retention").Post(&data, contentJSON)
	if err == nil {
		return GetRetentionPolicy(uuid.Parse(data.UUID))
	}
	return RetentionPolicy{}, err
}

func UpdateRetentionPolicy(id uuid.UUID, contentJSON string) (RetentionPolicy, error) {
	err := ShieldURI("/v1/retention/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetRetentionPolicy(id)
	}
	return RetentionPolicy{}, err
}

func DeleteRetentionPolicy(id uuid.UUID) error {
	return ShieldURI("/v1/retention/%s", id).Delete(nil)
}
