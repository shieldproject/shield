package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/goutils/timestamp"
)

type Archive struct {
	UUID           uuid.UUID `json:"uuid"`
	StoreKey       string    `json:"key"`
	TakenAt        Timestamp `json:"taken_at"`
	ExpiresAt      Timestamp `json:"expires_at"`
	Notes          string    `json:"notes"`
	Status         string    `json:"status"`
	PurgeReason    string    `json:"purge_reason"`
	TargetUUID     uuid.UUID `json:"target_uuid"`
	TargetName     string    `json:"target_name"`
	TargetPlugin   string    `json:"target_plugin"`
	TargetEndpoint string    `json:"target_endpoint"`
	StoreUUID      uuid.UUID `json:"store_uuid"`
	StoreName      string    `json:"store_name"`
	StorePlugin    string    `json:"store_plugin"`
	StoreEndpoint  string    `json:"store_endpoint"`
}

type ArchiveFilter struct {
	ForTarget     string
	ForStore      string
	Before        *time.Time
	After         *time.Time
	ExpiresBefore *time.Time
	WithStatus    []string
	WithOutStatus []string
	Limit         string
}

func (f *ArchiveFilter) Query() (string, []interface{}) {
	wheres := []string{"a.uuid = a.uuid"}
	var args []interface{}
	if f.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		wheres = append(wheres, "store_uuid = ?")
		args = append(args, f.ForStore)
	}
	if f.Before != nil {
		wheres = append(wheres, "taken_at <= ?")
		args = append(args, f.Before.Unix())
	}
	if f.After != nil {
		wheres = append(wheres, "taken_at >= ?")
		args = append(args, f.After.Unix())
	}
	if len(f.WithStatus) > 0 {
		var params []string
		for _, e := range f.WithStatus {
			params = append(params, "?")
			args = append(args, e)
		}
		wheres = append(wheres, fmt.Sprintf("status IN (%s)", strings.Join(params, ", ")))
	}
	if len(f.WithOutStatus) > 0 {
		var params []string
		for _, e := range f.WithOutStatus {
			params = append(params, "?")
			args = append(args, e)
		}
		wheres = append(wheres, fmt.Sprintf("status NOT IN (%s)", strings.Join(params, ", ")))
	}
	if f.ExpiresBefore != nil {
		wheres = append(wheres, "expires_at < ?")
		args = append(args, f.ExpiresBefore.Unix())
	}
	limit := ""
	if f.Limit != "" {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.name, t.plugin, t.endpoint,
		       s.uuid, s.name, s.plugin, s.endpoint,
		       a.status, a.purge_reason

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY a.taken_at DESC, a.uuid ASC
	` + limit, args
}

func (db *DB) GetAllArchives(filter *ArchiveFilter) ([]*Archive, error) {
	if filter == nil {
		filter = &ArchiveFilter{}
	}

	l := []*Archive{}
	if filter.Limit != "" {
		if lim, err := strconv.Atoi(filter.Limit); err != nil || lim < 0 {
			return l, fmt.Errorf("Invalid limit given: '%s'", filter.Limit)
		}
	}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Archive{}

		var takenAt, expiresAt *int64
		var targetName, storeName *string
		var this, target, store NullUUID
		if err = r.Scan(
			&this, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
			&target, &targetName, &ann.TargetPlugin, &ann.TargetEndpoint,
			&store, &storeName, &ann.StorePlugin, &ann.StoreEndpoint,
			&ann.Status, &ann.PurgeReason); err != nil {

			return l, err
		}
		ann.UUID = this.UUID
		ann.TargetUUID = target.UUID
		ann.StoreUUID = store.UUID
		if takenAt != nil {
			ann.TakenAt = parseEpochTime(*takenAt)
		}
		if expiresAt != nil {
			ann.ExpiresAt = parseEpochTime(*expiresAt)
		}
		if targetName != nil {
			ann.TargetName = *targetName
		}
		if storeName != nil {
			ann.StoreName = *storeName
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetArchive(id uuid.UUID) (*Archive, error) {
	r, err := db.Query(`
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.name, t.plugin, t.endpoint,
		       s.uuid, s.name, s.plugin, s.endpoint, a.status, a.purge_reason

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE a.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	ann := &Archive{}

	var takenAt, expiresAt *int64
	var targetName, storeName *string
	var this, target, store NullUUID
	if err = r.Scan(
		&this, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
		&target, &targetName, &ann.TargetPlugin, &ann.TargetEndpoint,
		&store, &storeName, &ann.StorePlugin, &ann.StoreEndpoint,
		&ann.Status, &ann.PurgeReason); err != nil {

		return nil, err
	}
	ann.UUID = this.UUID
	ann.TargetUUID = target.UUID
	ann.StoreUUID = store.UUID
	if takenAt != nil {
		ann.TakenAt = parseEpochTime(*takenAt)
	}
	if expiresAt != nil {
		ann.ExpiresAt = parseEpochTime(*expiresAt)
	}
	if targetName != nil {
		ann.TargetName = *targetName
	}
	if storeName != nil {
		ann.StoreName = *storeName
	}

	return ann, nil
}

func (db *DB) AnnotateArchive(id uuid.UUID, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ?`,
		notes, id.String(),
	)
}

func (db *DB) GetArchivesNeedingPurge() ([]*Archive, error) {
	filter := &ArchiveFilter{
		WithOutStatus: []string{"purged", "valid"},
	}
	return db.GetAllArchives(filter)
}

func (db *DB) GetExpiredArchives() ([]*Archive, error) {
	now := time.Now()
	filter := &ArchiveFilter{
		ExpiresBefore: &now,
		WithStatus:    []string{"valid"},
	}
	return db.GetAllArchives(filter)
}

func (db *DB) InvalidateArchive(id uuid.UUID) error {
	return db.Exec(`UPDATE archives SET status = 'invalid' WHERE uuid = ?`, id.String())
}

func (db *DB) PurgeArchive(id uuid.UUID) error {
	a, err := db.GetArchive(id)
	if err != nil {
		return err
	}

	if a.Status == "valid" {
		return fmt.Errorf("Invalid attempt to purge a 'valid' archive detected")
	}

	err = db.Exec(`UPDATE archives SET purge_reason = status WHERE uuid = ?`, id.String())
	if err != nil {
		return err
	}

	return db.Exec(`UPDATE archives SET status = 'purged' WHERE uuid = ?`, id.String())
}

func (db *DB) ExpireArchive(id uuid.UUID) error {
	return db.Exec(`UPDATE archives SET status = 'expired' WHERE uuid = ?`, id.String())
}

func (db *DB) DeleteArchive(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM archives WHERE uuid = ?`,
		id.String(),
	)
}
