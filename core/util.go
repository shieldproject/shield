package core

import (
	"github.com/jhunt/go-log"

	"github.com/shieldproject/shield/db"
)

func IsValidTenantRole(role string) bool {
	return role == "admin" || role == "engineer" || role == "operator"
}

func IsValidSystemRole(role string) bool {
	return role == "admin" || role == "manager" || role == "engineer"
}

func (c *Core) DeltaIncrease(filter *db.ArchiveFilter) (int64, error) {
	delta_increase, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		ForStore:   filter.ForStore,
		ForTenant:  filter.ForTenant,
		Before:     filter.Before,
		After:      filter.After,
		WithStatus: []string{"valid"},
	})
	if err != nil {
		log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
		return -1, err
	}

	delta_purged, err := c.db.ArchiveStorageFootprint(&db.ArchiveFilter{
		ForStore:      filter.ForStore,
		ForTenant:     filter.ForTenant,
		ExpiresBefore: filter.ExpiresBefore,
		ExpiresAfter:  filter.ExpiresAfter,
		WithStatus:    []string{"purged"},
	})
	if err != nil {
		log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
		return -1, err
	}
	return (delta_increase - delta_purged), nil
}
