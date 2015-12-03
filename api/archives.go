package api

import (
	. "github.com/starkandwayne/shield/timestamp"
)

type ArchiveFilter struct {
	Plugin string
	Unused YesNo
}

type Archive struct {
	UUID      string    `json:"uuid"`
	StoreKey  string    `json:"key"`
	TakenAt   Timestamp `json:"taken_at"`
	ExpiresAt Timestamp `json:"expires_at"`
	Notes     string    `json:"notes"`

	TargetUUID     string `json:"target_uuid"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StoreUUID      string `json:"store_uuid"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
}

func FetchListArchives(plugin, unused string) ([]Archive, error) {
	return GetArchives(ArchiveFilter{
		Plugin: plugin,
		Unused: MaybeString(unused),
	})
}

func GetArchives(filter ArchiveFilter) ([]Archive, error) {
	uri := ShieldURI("/v1/archives")
	uri.MaybeAddParameter("plugin", filter.Plugin)
	uri.MaybeAddParameter("unused", filter.Unused)

	var data []Archive
	return data, uri.Get(&data)
}

func GetArchive(uuid string) (Archive, error) {
	var data Archive
	return data, ShieldURI("/v1/archive/%s", uuid).Get(&data)
}

func RestoreArchive(uuid, targetJSON string) error {
	return ShieldURI("/v1/archive/%s/restore", uuid).Post(nil, targetJSON)
}

func UpdateArchive(uuid string, contentJSON string) (Archive, error) {
	err := ShieldURI("/v1/archive/%s", uuid).Put(nil, contentJSON)
	if err == nil {
		return GetArchive(uuid)
	}
	return Archive{}, err
}

func DeleteArchive(uuid string) error {
	return ShieldURI("/v1/archive/%s", uuid).Delete(nil)
}
