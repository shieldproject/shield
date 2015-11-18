package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchRetentionPoliciesList(unused string) (*[]db.AnnotatedRetentionPolicy, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedRetentionPolicy{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/retention")
	joiner := "?"
	if unused != "" {
		uri = fmt.Sprintf("%s%sunused=%s", uri, joiner, unused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetRetentionPolicy(uuid string) (*db.AnnotatedRetentionPolicy, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedRetentionPolicy{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/retention/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
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
