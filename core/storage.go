package core

import (
	"fmt"
	"time"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
)

func (core *Core) DailyStorageAnalytics() error {
	core.DailyStoreStats()
	core.DailyTenantStats()
	return nil
}

// DeltaIncrease calculates the delta in storage space over the period specified.
// It stores the number of bytes increased/decreased in the period specified in the stores table.
// Calculation is preformed by taking (total new archives created - any archives newly purged)
func (core *Core) DeltaIncrease(filter *db.ArchiveFilter) (int64, error) {
	delta_increase, err := core.DB.ArchiveStorageFootprint(&db.ArchiveFilter{
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

	delta_purged, err := core.DB.ArchiveStorageFootprint(&db.ArchiveFilter{
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
	fmt.Printf("Increase %d/Purged %d\n", delta_increase, delta_purged)
	return (delta_increase - delta_purged), nil
}

// DailyStoreStats batches updates of daily archive storage space statistics.
// It stores the number total archives corresponding to each store, and the total size of those archives
func (core *Core) DailyStoreStats() error {
	base := time.Now()
	threshold := base.Add(0 - time.Duration(24)*time.Hour)

	stores, err := core.DB.GetAllStores(nil)
	if err != nil {
		log.Errorf("Failed to get stores for daily storage statistics: %s", err)
		return err
	}

	for _, store := range stores {
		delta, err := core.DeltaIncrease(
			&db.ArchiveFilter{
				ForStore:      store.UUID.String(),
				Before:        &base,
				After:         &threshold,
				ExpiresBefore: &base,
				ExpiresAfter:  &threshold,
			},
		)
		if err != nil {
			return err
		}

		total_size, err := core.DB.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForStore:   store.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}

		total_count, err := core.DB.CountArchives(
			&db.ArchiveFilter{
				ForStore:   store.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}

		store.DailyIncrease = delta
		store.StorageUsed = total_size
		store.ArchiveCount = int64(total_count)
		fmt.Printf("Updating %s:%s with Used:%d, Count:%d, Increase:%d\n", store.Name, store.UUID.String(), store.StorageUsed, store.ArchiveCount, store.DailyIncrease)
		err = core.DB.UpdateStore(store)
		if err != nil {
			log.Errorf("Failed to update stores with daily storage statistics: %s", err)
			return err
		}
	}
	return nil
}

// DailyTenantStats batches updates of daily archive storage space statistics.
// It stores the number total archives corresponding to each tenant, and the total size of those archives
// It also agregates the daily increase over all the stores belonging to the tenant
func (core *Core) DailyTenantStats() error {
	base := time.Now()
	threshold := base.Add(0 - time.Duration(24)*time.Hour)
	tenants, err := core.DB.GetAllTenants(nil)
	if err != nil {
		log.Errorf("Failed to get tenants for daily storage statistics: %s", err)
		return err
	}

	for _, tenant := range tenants {
		delta, err := core.DeltaIncrease(
			&db.ArchiveFilter{
				ForTenant:     tenant.UUID.String(),
				Before:        &base,
				After:         &threshold,
				ExpiresBefore: &base,
				ExpiresAfter:  &threshold,
			},
		)
		if err != nil {
			return err
		}

		total_size, err := core.DB.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForTenant:  tenant.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}

		total_count, err := core.DB.CountArchives(
			&db.ArchiveFilter{
				ForTenant:  tenant.UUID.String(),
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}

		tenant.StorageUsed = total_size
		tenant.ArchiveCount = int64(total_count)
		tenant.DailyIncrease = delta

		fmt.Printf("Updating %s:%s with Used:%d, Count:%d, Increase:%d\n", tenant.Name, tenant.UUID.String(), tenant.StorageUsed, tenant.ArchiveCount, tenant.DailyIncrease)
		_, err = core.DB.UpdateTenant(tenant)
		if err != nil {
			log.Errorf("Failed to update tenant with daily storage statistics: %s", err)
			return err
		}
	}
	return nil
}
