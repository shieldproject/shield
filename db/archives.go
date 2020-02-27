package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Archive struct {
	UUID           string `json:"uuid"            mbus:"uuid"`
	TenantUUID     string `json:"tenant_uuid"     mbus:"tenant_uuid"`
	TargetUUID     string `json:"target_uuid"     mbus:"target_uuid"`
	StoreUUID      string `json:"store_uuid"      mbus:"store_uuid"`
	StoreKey       string `json:"key"             mbus:"key"`
	TakenAt        int64  `json:"taken_at"        mbus:"taken_at"`
	ExpiresAt      int64  `json:"expires_at"      mbus:"expires_at"`
	Notes          string `json:"notes"           mbus:"notes"`
	Status         string `json:"status"          mbus:"status"`
	PurgeReason    string `json:"purge_reason"    mbus:"purge_reason"`
	EncryptionType string `json:"encryption_type" mbus:"encryption_type"`
	Compression    string `json:"compression"     mbus:"compression"`
	Size           int64  `json:"size"            mbus:"size"`

	TargetName     string `json:"target_name"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StoreName      string `json:"store_name"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
	StoreAgent     string `json:"store_agent"`
	Job            string `json:"job"`
}

type ArchiveFilter struct {
	UUID          string
	ExactMatch    bool
	ForTarget     string
	ForStore      string
	Before        *time.Time
	After         *time.Time
	ExpiresBefore *time.Time
	ExpiresAfter  *time.Time
	WithStatus    []string
	WithOutStatus []string
	ForTenant     string
	ForStoreKey   string
	Limit         int
}

func (f *ArchiveFilter) Query() (string, []interface{}) {
	wheres := []string{"a.uuid = a.uuid"}
	var args []interface{}
	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "a.uuid = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "a.uuid LIKE ? ESCAPE '/'")
			args = append(args, PatternPrefix(f.UUID))
		}
	}
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

	if f.ForStoreKey != "" {
		wheres = append(wheres, "a.store_key = ?")
		args = append(args, f.ForStoreKey)
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
               s.uuid, s.name, s.plugin, s.endpoint, s.agent,
               a.status, a.purge_reason, a.job, a.encryption_type,
               a.compression, a.tenant_uuid, a.size

        FROM archives a
           LEFT  JOIN targets t   ON t.uuid = a.target_uuid
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
	db.exclusive.Lock()
	uintRet, err := db.count(query, args...)
	db.exclusive.Unlock()
	ret := int(uintRet)
	return ret, err
}

func (db *DB) GetAllArchives(filter *ArchiveFilter) ([]*Archive, error) {
	if filter == nil {
		filter = &ArchiveFilter{}
	}

	l := []*Archive{}
	query, args := filter.Query()

	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		a := &Archive{}

		var takenAt, expiresAt, size *int64
		var targetUUID, targetPlugin, targetEndpoint, targetName, storeName sql.NullString
		if err = r.Scan(
			&a.UUID, &a.StoreKey, &takenAt, &expiresAt, &a.Notes,
			&targetUUID, &targetName, &targetPlugin, &targetEndpoint,
			&a.StoreUUID, &storeName, &a.StorePlugin, &a.StoreEndpoint, &a.StoreAgent,
			&a.Status, &a.PurgeReason, &a.Job, &a.EncryptionType,
			&a.Compression, &a.TenantUUID, &size); err != nil {

			return l, err
		}
		if takenAt != nil {
			a.TakenAt = *takenAt
		}
		if expiresAt != nil {
			a.ExpiresAt = *expiresAt
		}
		if targetName.Valid {
			a.TargetName = targetName.String
		}
		if targetUUID.Valid {
			a.TargetUUID = targetUUID.String
		}
		if targetPlugin.Valid {
			a.TargetPlugin = targetPlugin.String
		}
		if targetEndpoint.Valid {
			a.TargetPlugin = targetEndpoint.String
		}
		if storeName.Valid {
			a.StoreName = storeName.String
		}
		if size != nil {
			a.Size = *size
		}

		l = append(l, a)
	}

	return l, nil
}

func (db *DB) getArchive(id string) (*Archive, error) {
	r, err := db.query(`
        SELECT a.uuid, a.store_key,
               a.taken_at, a.expires_at, a.notes,
               t.uuid, t.name, t.plugin, t.endpoint,
               s.uuid, s.name, s.plugin, s.endpoint, s.agent,
               a.status, a.purge_reason, a.job, a.encryption_type,
               a.compression, a.tenant_uuid, a.size

        FROM archives a
           LEFT  JOIN targets t   ON t.uuid = a.target_uuid
           INNER JOIN stores  s   ON s.uuid = a.store_uuid

        WHERE a.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	a := &Archive{}

	var takenAt, expiresAt, size *int64
	var targetUUID, targetName, targetPlugin, targetEndpoint, storeName sql.NullString
	if err = r.Scan(
		&a.UUID, &a.StoreKey, &takenAt, &expiresAt, &a.Notes,
		&targetUUID, &targetName, &targetPlugin, &targetEndpoint,
		&a.StoreUUID, &storeName, &a.StorePlugin, &a.StoreEndpoint, &a.StoreAgent,
		&a.Status, &a.PurgeReason, &a.Job, &a.EncryptionType,
		&a.Compression, &a.TenantUUID, &size); err != nil {

		return nil, err
	}
	if takenAt != nil {
		a.TakenAt = *takenAt
	}
	if expiresAt != nil {
		a.ExpiresAt = *expiresAt
	}
	if targetUUID.Valid {
		a.TargetUUID = targetUUID.String
	}
	if targetName.Valid {
		a.TargetName = targetName.String
	}
	if targetPlugin.Valid {
		a.TargetPlugin = targetPlugin.String
	}
	if targetEndpoint.Valid {
		a.TargetEndpoint = targetEndpoint.String
	}
	if storeName.Valid {
		a.StoreName = storeName.String
	}
	if size != nil {
		a.Size = *size
	}

	return a, nil
}

func (db *DB) GetArchive(id string) (*Archive, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.getArchive(id)
}

func (db *DB) createArchiveFromTask(task_uuid string, archive Archive) (*Archive, error) {
	err := db.exec(`
              INSERT INTO archives
                (uuid, target_uuid, store_uuid, store_key, taken_at,
                 expires_at, notes, status, purge_reason, job,
                 compression, encryption_type, size, tenant_uuid)

                  SELECT ?, t.uuid, s.uuid, ?, ?,
                         ?, '', 'valid', '', j.Name,
                         ?, ?, ?, ?
                  FROM tasks
                     INNER JOIN jobs    j     ON j.uuid = tasks.job_uuid
                     INNER JOIN targets t     ON t.uuid = j.target_uuid
                     INNER JOIN stores  s     ON s.uuid = j.store_uuid
                  WHERE tasks.uuid = ?`,
		archive.UUID, archive.StoreKey, archive.TakenAt,
		archive.ExpiresAt,
		archive.Compression, archive.EncryptionType, archive.Size, archive.TenantUUID,
		task_uuid)
	if err != nil {
		return nil, err
	}

	return db.getArchive(archive.UUID)
}

func (db *DB) UpdateArchive(update *Archive) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ?`,
		update.Notes, update.UUID,
	)
}

func (db *DB) AnnotateTargetArchive(target, id, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ? AND target_uuid = ?`,
		notes, id, target,
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

func (db *DB) InvalidateArchive(id string) error {
	return db.Exec(`UPDATE archives SET status = 'invalid' WHERE uuid = ?`, id)
}

func (db *DB) PurgeArchive(id string) error {
	a, err := db.GetArchive(id)
	if err != nil {
		return fmt.Errorf("unable to retrieve archive [%s]: %s", id, err)
	}
	if a == nil {
		return fmt.Errorf("unable to retrieve archive [%s]: not found in database.", id)
	}

	if a.Status == "valid" {
		return fmt.Errorf("invalid attempt to purge a 'valid' archive detected")
	}

	err = db.Exec(`UPDATE archives SET purge_reason = status WHERE uuid = ?`, id)
	if err != nil {
		return err
	}

	return db.Exec(`UPDATE archives SET status = 'purged', expires_at = ? WHERE uuid = ?`, time.Now().Unix(), id)
}

func (db *DB) ExpireArchive(id string) error {
	return db.Exec(`UPDATE archives SET status = 'expired' WHERE uuid = ?`, id)
}

func (db *DB) ManuallyPurgeArchive(id string) error {
	return db.exclusively(func() error {
		err := db.exec(`UPDATE archives SET status = 'manually purged', expires_at = ? WHERE uuid = ?`, time.Now().Unix(), id)
		if err != nil {
			return err
		}

		archive, err := db.getArchive(id)
		if err != nil {
			return fmt.Errorf("unable to retrieve archive [%s]: %s", id, err)
		}
		db.sendUpdateObjectEvent(archive, "tenant:"+archive.TenantUUID)
		return nil
	})
}

func (db *DB) DeleteArchive(id string) (bool, error) {
	return true, db.Exec(`DELETE FROM archives WHERE uuid = ?`, id)
}

func (db *DB) CleanupArchives(age int) error {
	return db.Exec(`
       DELETE FROM archives
             WHERE status IN ('purged', 'manually purged')
               AND expires_at < ?`,
		(int)(time.Now().Unix())-age)
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

	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	r, err := db.query(`
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
	if !r.Next() {
		return 0, fmt.Errorf("no results from SUM(size) query...")
	}

	if err = r.Scan(&p); err != nil {
		return 0, err
	}
	if p != nil {
		i = *p
	}
	return i, nil
}

func (db *DB) CleanArchives() error {
	return db.Exec(`
       UPDATE archives
          SET status = "expired"
        WHERE uuid IN (SELECT a.uuid
                         FROM archives a
                    LEFT JOIN tenants  t
                           ON t.uuid = a.tenant_uuid
                        WHERE t.uuid IS NULL)`)
}
