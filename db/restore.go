package db

import (
	"fmt"

	"github.com/pborman/uuid"
)

func (db *DB) GetRestoreTaskDetails(archive, target uuid.UUID, storePlugin, storeEndpoint, storeKey, targetPlugin, targetEndpoint, agent *string) error {
	// retrieve store plugin / endpoint / key
	r, err := db.Query(`
		SELECT s.plugin, s.endpoint, a.store_key
			FROM stores s INNER JOIN archives a  ON s.uuid = a.store_uuid
		WHERE a.uuid = $1`, archive.String())
	if err != nil {
		return err
	}
	defer r.Close()

	if !r.Next() {
		return fmt.Errorf("failed to determine task details for archive %s -> target %s", archive, target)
	}

	if err := r.Scan(storePlugin, storeEndpoint, storeKey); err != nil {
		return err
	}
	r.Close()

	// retrieve target plugin / endpoint
	r, err = db.Query(`
		SELECT t.plugin, t.endpoint, t.agent
			FROM targets t WHERE t.uuid = $1`, target.String())
	if err != nil {
		return err
	}
	defer r.Close()

	if !r.Next() {
		return fmt.Errorf("failed to determine task details for archive %s -> target %s", archive, target)
	}

	if err := r.Scan(targetPlugin, targetEndpoint, agent); err != nil {
		return err
	}
	r.Close()

	return nil
}
