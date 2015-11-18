package api_agent

import (
	"bytes"
	"fmt"
	"github.com/starkandwayne/shield/db"
)

func FetchListArchives(plugin, unused string) (*[]db.AnnotatedArchive, error) {

	// Data to be returned of proper type
	data := &[]db.AnnotatedArchive{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/archives")
	joiner := "?"
	if plugin != "" {
		uri = fmt.Sprintf("%s%splugin=%s", uri, joiner, plugin)
		joiner = "&"
	}
	if unused != "" {
		uri = fmt.Sprintf("%s%sunused=%s", uri, joiner, unused)
		joiner = "&"
	}

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
}

func GetArchive(uuid string) (*db.AnnotatedArchive, error) {

	// Data to be returned of proper type
	data := &db.AnnotatedArchive{}

	// Make uri based on options
	uri := fmt.Sprintf("v1/archive/%s", uuid)

	// Call generic API request
	err := makeApiCall(data, `GET`, uri, nil)
	return data, err
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
