package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

type Archive struct {
	UUID           uuid.UUID `json:"uuid"`
	StoreKey       string    `json:"key"`
	TakenAt        int64     `json:"taken_at"`
	ExpiresAt      int64     `json:"expires_at"`
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
	Job            string    `json:"job"`
	EncryptionType string    `json:"encryption_type"`
	Compression    string    `json:"compression"`
	TenantUUID     uuid.UUID `json:"tenant_uuid"`
	Size           int64     `json:"size"`
}

type ArchiveFilter struct {
	ForTarget     string
	ForStore      string
	Before        *time.Time
	After         *time.Time
	ExpiresBefore *time.Time
	ExpiresAfter  *time.Time
	WithStatus    []string
	WithOutStatus []string
	ForTenant     string
	Limit         int
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

	if f.ForTenant != "" {
		wheres = append(wheres, "a.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}
	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.name, t.plugin, t.endpoint,
		       s.uuid, s.name, s.plugin, s.endpoint,
		       a.status, a.purge_reason, a.job, a.encryption_type,
		       a.compression, a.tenant_uuid, a.size

		FROM archives a
		   LEFT JOIN targets t   ON t.uuid = a.target_uuid
		   INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY a.taken_at DESC, a.uuid ASC
	` + limit, args
}

func (db *DB) CountArchives(filter *ArchiveFilter) (int, error) {
	if filter == nil {
		filter = &ArchiveFilter{}
	}

	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	i := 0
	for r.Next() {
		i++
	}
	return i, nil
}

func (db *DB) GetAllArchives(filter *ArchiveFilter) ([]*Archive, error) {
	if filter == nil {
		filter = &ArchiveFilter{}
	}

	l := []*Archive{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Archive{}

		var takenAt, expiresAt, size *int64
		var targetName, storeName *string
		var this, target, store, tenant NullUUID
		if err = r.Scan(
			&this, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
			&target, &targetName, &ann.TargetPlugin, &ann.TargetEndpoint,
			&store, &storeName, &ann.StorePlugin, &ann.StoreEndpoint,
			&ann.Status, &ann.PurgeReason, &ann.Job, &ann.EncryptionType,
			&ann.Compression, &tenant, &size); err != nil {

			return l, err
		}
		ann.UUID = this.UUID
		ann.TargetUUID = target.UUID
		ann.StoreUUID = store.UUID
		ann.TenantUUID = tenant.UUID
		if takenAt != nil {
			ann.TakenAt = *takenAt
		}
		if expiresAt != nil {
			ann.ExpiresAt = *expiresAt
		}
		if targetName != nil {
			ann.TargetName = *targetName
		}
		if storeName != nil {
			ann.StoreName = *storeName
		}
		if size != nil {
			ann.Size = *size
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
		       s.uuid, s.name, s.plugin, s.endpoint, a.status,
		       a.purge_reason, a.job, a.encryption_type,
		       a.compression, a.tenant_uuid, a.size

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

	var takenAt, expiresAt, size *int64
	var targetName, storeName *string
	var this, target, store, tenant NullUUID
	if err = r.Scan(
		&this, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
		&target, &targetName, &ann.TargetPlugin, &ann.TargetEndpoint,
		&store, &storeName, &ann.StorePlugin, &ann.StoreEndpoint,
		&ann.Status, &ann.PurgeReason, &ann.Job, &ann.EncryptionType,
		&ann.Compression, &tenant, &size); err != nil {

		return nil, err
	}
	ann.UUID = this.UUID
	ann.TargetUUID = target.UUID
	ann.StoreUUID = store.UUID
	ann.TenantUUID = tenant.UUID
	if takenAt != nil {
		ann.TakenAt = *takenAt
	}
	if expiresAt != nil {
		ann.ExpiresAt = *expiresAt
	}
	if targetName != nil {
		ann.TargetName = *targetName
	}
	if storeName != nil {
		ann.StoreName = *storeName
	}
	if size != nil {
		ann.Size = *size
	}

	return ann, nil
}

func (db *DB) UpdateArchive(update *Archive) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ?`,
		update.Notes, update.UUID.String(),
	)
}

func (db *DB) AnnotateTargetArchive(target uuid.UUID, id string, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ? AND target_uuid = ?`,
		notes, id, target.String(),
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

func (db *DB) GetTennantlessArchives() ([]*Archive, error) {
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

func (db *DB) ArchiveStorageFootprint(filter *ArchiveFilter) (int64, error) {
	var i int64

	if filter == nil {
		filter = &ArchiveFilter{}
	}

	wheres := []string{"a.uuid = a.uuid"}
	var args []interface{}
	if filter.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
		args = append(args, filter.ForTarget)
	}
	if filter.ForStore != "" {
		wheres = append(wheres, "store_uuid = ?")
		args = append(args, filter.ForStore)
	}
	if filter.Before != nil {
		wheres = append(wheres, "taken_at <= ?")
		args = append(args, filter.Before.Unix())
	}
	if filter.After != nil {
		wheres = append(wheres, "taken_at >= ?")
		args = append(args, filter.After.Unix())
	}
	if len(filter.WithStatus) > 0 {
		var params []string
		for _, e := range filter.WithStatus {
			params = append(params, "?")
			args = append(args, e)
		}
		wheres = append(wheres, fmt.Sprintf("status IN (%s)", strings.Join(params, ", ")))
	}
	if len(filter.WithOutStatus) > 0 {
		var params []string
		for _, e := range filter.WithOutStatus {
			params = append(params, "?")
			args = append(args, e)
		}
		wheres = append(wheres, fmt.Sprintf("status NOT IN (%s)", strings.Join(params, ", ")))
	}
	if filter.ExpiresBefore != nil {
		wheres = append(wheres, "expires_at <= ?")
		args = append(args, filter.ExpiresBefore.Unix())
	}
	if filter.ExpiresAfter != nil {
		wheres = append(wheres, "expires_at >= ?")
		args = append(args, filter.ExpiresAfter.Unix())
	}
	if filter.ForTenant != "" {
		wheres = append(wheres, "a.tenant_uuid = ?")
		args = append(args, filter.ForTenant)
	}
	limit := ""
	if filter.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, filter.Limit)
	}

	r, err := db.Query(`
		SELECT SUM(a.size)
		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid
		WHERE `+strings.Join(wheres, " AND ")+limit, args...)
	if err != nil {
		return i, err
	}
	defer r.Close()

	var p *int64
	for r.Next() {
		if err = r.Scan(&p); err != nil {
			return i, err
		}
		if p != nil {
			i = *p
		}
		return i, nil
	}

	return i, fmt.Errorf("no results from SUM(size) query...")
}
