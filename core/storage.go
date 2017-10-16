package core

import (
	"time"

	"github.com/starkandwayne/goutils/log"
	"github.com/starkandwayne/shield/db"
)

// DailyStorageIncrease batches updates of daily archive storage space statistics.
// It stores the number of bytes increased in the past 24 hours in the stores table.
// Calculation is preformed by taking (total archive size and subtracting total size from 24 hours previous)
func (core *Core) DailyStorageIncrease() error {
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
		store.DailyIncrease = (delta_increase - delta_purged)
		err = core.DB.UpdateStore(store)
		if err != nil {
			log.Errorf("Failed to update stores with daily storage statistics: %s", err)
			return err
		}
	}

	return nil
}
