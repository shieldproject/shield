package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchTargetsList(plugin, unused string) (*[]db.AnnotatedTarget, error) {
	uri := ShieldURI("/v1/targets")
	if plugin != "" {
		uri.AddParameter("plugin", plugin)
	}
	if unused != "" {
		uri.AddParameter("unused", unused)
	}

	data := &[]db.AnnotatedTarget{}
	return data, uri.Get(&data)
}

func GetTarget(uuid string) (*db.AnnotatedTarget, error) {
	data := &db.AnnotatedTarget{}
	return data, ShieldURI("/v1/targets/%s", uuid).Get(&data)
}

func CreateTarget(content string) (*db.AnnotatedTarget, error) {

	data := struct {
		Status string `json:"ok"`
		UUID   string `json:"uuid"`
	}{}

	buf := bytes.NewBufferString(content)

	err := makeApiCall(&data, `POST`, `v1/targets`, buf)

	if err == nil {
		return GetTarget(data.UUID)
	}
	return nil, err
}

func UpdateTarget(uuid string, content string) (*db.AnnotatedTarget, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/target/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetTarget(uuid)
	}
	return nil, err
}

func DeleteTarget(uuid string) error {

	uri := fmt.Sprintf("v1/target/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
