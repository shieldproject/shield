package api_agent

import (
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

func RestoreArchive(uuid, targetJSON string) error {
	return ShieldURI("/v1/archive/%s/restore", uuid).Post(nil, targetJSON)
}

func UpdateArchive(uuid string, contentJSON string) (*db.AnnotatedArchive, error) {
	err := ShieldURI("/v1/archive/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetArchive(uuid)
	}
	return nil, err
}

func DeleteArchive(uuid string) error {
	return ShieldURI("/v1/archive/%s", uuid).Delete(nil)
}
