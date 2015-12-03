package api_agent

import (
	"github.com/starkandwayne/shield/db"
)

type RetentionPoliciesFilter struct {
	All    bool
	Unused bool
}

func GetRetentionPolicies(filter RetentionPoliciesFilter) (*[]db.AnnotatedRetentionPolicy, error) {
	uri := ShieldURI("/v1/retention")
	if !filter.All {
		uri.AddParameter("unused", filter.Unused)
	}

	data := &[]db.AnnotatedRetentionPolicy{}
	return data, uri.Get(&data)
}

func GetAllRetentionPolicies() (*[]db.AnnotatedRetentionPolicy, error) {
	return GetRetentionPolicies(RetentionPoliciesFilter{All: true})
}
func GetUnusedRetentionPolicies() (*[]db.AnnotatedRetentionPolicy, error) {
	return GetRetentionPolicies(RetentionPoliciesFilter{Unused: true})
}
func GetUsedRetentionPolicies() (*[]db.AnnotatedRetentionPolicy, error) {
	return GetRetentionPolicies(RetentionPoliciesFilter{Unused: false})
}

func GetRetentionPolicy(uuid string) (*db.AnnotatedRetentionPolicy, error) {
	data := &db.AnnotatedRetentionPolicy{}
	return data, ShieldURI("v1/retention/%s", uuid).Get(&data)
}

func CreateRetentionPolicy(contentJSON string) (*db.AnnotatedRetentionPolicy, error) {
	data := struct {
		UUID string `json:"uuid"`
	}{}
	err := ShieldURI("/v2/retention").Post(&data, contentJSON)
	if err == nil {
		return GetRetentionPolicy(data.UUID)
	}
	return nil, err
}

func UpdateRetentionPolicy(uuid string, contentJSON string) (*db.AnnotatedRetentionPolicy, error) {
	err := ShieldURI("/v1/retention/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetRetentionPolicy(uuid)
	}
	return nil, err
}

func DeleteRetentionPolicy(uuid string) error {
	return ShieldURI("/v1/retention/%s", uuid).Delete(nil)
}
