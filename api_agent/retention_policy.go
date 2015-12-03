package api_agent

import (
	"bytes"
	"fmt"
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

func CreateRetentionPolicy(content string) (*db.AnnotatedRetentionPolicy, error) {

	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}

	buf := bytes.NewBufferString(content)

	err := makeApiCall(&data, `POST`, `v1/retention`, buf)

	if err == nil {
		return GetRetentionPolicy(data.UUID)
	}
	return nil, err
}

func UpdateRetentionPolicy(uuid string, content string) (*db.AnnotatedRetentionPolicy, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/retention/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetRetentionPolicy(uuid)
	}
	return nil, err
}

func DeleteRetentionPolicy(uuid string) error {

	uri := fmt.Sprintf("v1/retention/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
