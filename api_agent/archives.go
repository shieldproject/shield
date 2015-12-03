package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListArchives(plugin, unused string) (*[]db.AnnotatedArchive, error) {
	uri := ShieldURI("/v1/archives")
	if plugin != "" {
		uri.AddParameter("plugin", plugin)
	}
	if unused != "" {
		uri.AddParameter("unused", unused)
	}

	data := &[]db.AnnotatedArchive{}
	return data, uri.Get(&data)
}

func GetArchive(uuid string) (*db.AnnotatedArchive, error) {
	data := &db.AnnotatedArchive{}
	return data, ShieldURI("/v1/archive/%s", uuid).Get(&data)
}

func RestoreArchive(uuid, target string) error {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(target)

	uri := fmt.Sprintf("v1/archive/%s/restore", uuid)

	err := makeApiCall(&data, `POST`, uri, buf)

	return err
}

func UpdateArchive(uuid string, content string) (*db.AnnotatedArchive, error) {

	data := struct {
		Status string `json:"ok"`
	}{}

	buf := bytes.NewBufferString(content)

	uri := fmt.Sprintf("v1/archive/%s", uuid)

	err := makeApiCall(&data, `PUT`, uri, buf)

	if err == nil {
		return GetArchive(uuid)
	}
	return nil, err
}

func DeleteArchive(uuid string) error {

	uri := fmt.Sprintf("v1/archive/%s", uuid)

	data := struct {
		Status string `json:"ok"`
	}{}

	err := makeApiCall(&data, `DELETE`, uri, nil)

	return err
}
