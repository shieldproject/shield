package core

import (
	"fmt"
	"time"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
)

func (core *Core) DailyStoreStorageAnalytics() error {
	core.DailyStoreIncrease()
	core.DailyStoreStats()
	return nil
}

// DailyStoreIncrease calculates the delta in storage space over the past 24 hours.
// It stores the number of bytes increased/decreased in the past 24 hours in the stores table.
// Calculation is preformed by taking (total new archives created - any archives newly purged)
func (core *Core) DailyStoreIncrease() error {
	base := time.Now()
	threshold := base.Add(0 - time.Duration(24)*time.Hour)

	stores, err := core.DB.GetAllStores(nil)
	if err != nil {
		log.Errorf("Failed to get stores for daily storage statistics: %s", err)
		return err
	}

	for _, store := range stores {
		delta_increase, err := core.DB.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForStore:   store.UUID.String(),
				Before:     &base,
				After:      &threshold,
				WithStatus: []string{"valid"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}

		delta_purged, err := core.DB.ArchiveStorageFootprint(
			&db.ArchiveFilter{
				ForStore:      store.UUID.String(),
				ExpiresBefore: &base,
				ExpiresAfter:  &threshold,
				WithStatus:    []string{"purged"},
			},
		)
		if err != nil {
			log.Errorf("Failed to get archive stats for daily storage statistics: %s", err)
			return err
		}
		fmt.Printf("Increase %d/Purged %d\n", delta_increase, delta_purged)
		store.DailyIncrease = (delta_increase - delta_purged)
		err = core.DB.UpdateStore(store)
		if err != nil {
			log.Errorf("Failed to update stores with daily storage statistics: %s", err)
			return err
		}
	}
	return nil
}

// DailyStoreStats batches updates of daily archive storage space statistics.
// It stores the number total archives corresponding to each store, and the total size of those archives
func (core *Core) DailyStoreStats() error {
	stores, err := core.DB.GetAllStores(nil)
	if err != nil {
		log.Errorf("Failed to get stores for daily storage statistics: %s", err)
		return err
	}

	for _, store := range stores {
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

		store.StorageUsed = total_size
		store.ArchiveCount = int64(total_count)
		err = core.DB.UpdateStore(store)
		if err != nil {
			log.Errorf("Failed to update stores with daily storage statistics: %s", err)
			return err
		}
	}
	return nil
}
