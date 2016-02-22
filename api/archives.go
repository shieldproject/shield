package api

import (
	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/timestamp"
)

type ArchiveFilter struct {
	Target string
	Store  string
	Before string
	After  string
	Status string
	Limit  string
}

type Archive struct {
	UUID      string    `json:"uuid"`
	StoreKey  string    `json:"key"`
	TakenAt   Timestamp `json:"taken_at"`
	ExpiresAt Timestamp `json:"expires_at"`
	Status    string    `json:"status"`
	Notes     string    `json:"notes"`

	TargetUUID     string `json:"target_uuid"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StoreUUID      string `json:"store_uuid"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
}

func GetArchives(filter ArchiveFilter) ([]Archive, error) {
	uri := ShieldURI("/v1/archives")
	uri.MaybeAddParameter("target", filter.Target)
	uri.MaybeAddParameter("store", filter.Store)
	uri.MaybeAddParameter("before", filter.Before)
	uri.MaybeAddParameter("after", filter.After)
	uri.MaybeAddParameter("status", filter.Status)
	uri.MaybeAddParameter("limit", filter.Limit)

	var data []Archive
	return data, uri.Get(&data)
}

func GetArchive(id uuid.UUID) (Archive, error) {
	var data Archive
	return data, ShieldURI("/v1/archive/%s", id).Get(&data)
}

func RestoreArchive(id uuid.UUID, targetJSON string) error {
	return ShieldURI("/v1/archive/%s/restore", id).Post(nil, targetJSON)
}

func UpdateArchive(id uuid.UUID, contentJSON string) (Archive, error) {
	err := ShieldURI("/v1/archive/%s", id).Put(nil, contentJSON)
	if err == nil {
		return GetArchive(id)
	}
	return Archive{}, err
}

func DeleteArchive(id uuid.UUID) error {
	return ShieldURI("/v1/archive/%s", id).Delete(nil)
}
