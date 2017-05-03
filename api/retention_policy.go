package api

import (
	"github.com/pborman/uuid"
)

type RetentionPolicyFilter struct {
	Unused     YesNo
	Name       string
	ExactMatch YesNo
}

type RetentionPolicy struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires uint   `json:"expires"`
}

func GetRetentionPolicies(filter RetentionPolicyFilter) ([]RetentionPolicy, error) {
	uri, err := ShieldURI("/v1/retention")
	if err != nil {
		return []RetentionPolicy{}, err
	}
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("unused", filter.Unused)
	uri.MaybeAddParameter("exact", filter.ExactMatch)

	var data []RetentionPolicy
	return data, uri.Get(&data)
}

func GetRetentionPolicy(id uuid.UUID) (RetentionPolicy, error) {
	var data RetentionPolicy
	uri, err := ShieldURI("/v1/retention/%s", id)
	if err != nil {
		return RetentionPolicy{}, err
	}
	return data, uri.Get(&data)
}

func CreateRetentionPolicy(contentJSON string) (RetentionPolicy, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	uri, err := ShieldURI("/v1/retention")
	if err != nil {
		return RetentionPolicy{}, err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return RetentionPolicy{}, err
	}
	return GetRetentionPolicy(uuid.Parse(data.UUID))
}

func UpdateRetentionPolicy(id uuid.UUID, contentJSON string) (RetentionPolicy, error) {
	uri, err := ShieldURI("/v1/retention/%s", id)
	if err != nil {
		return RetentionPolicy{}, err
	}
	if err := uri.Put(nil, contentJSON); err != nil {
		return RetentionPolicy{}, err
	}
	return GetRetentionPolicy(id)
}

func DeleteRetentionPolicy(id uuid.UUID) error {
	uri, err := ShieldURI("/v1/retention/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}
