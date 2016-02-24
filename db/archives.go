package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"

	. "github.com/starkandwayne/shield/timestamp"
)

type AnnotatedArchive struct {
	UUID        string    `json:"uuid"`
	StoreKey    string    `json:"key"`
	TakenAt     Timestamp `json:"taken_at"`
	ExpiresAt   Timestamp `json:"expires_at"`
	Notes       string    `json:"notes"`
	Status      string    `json:"status"`
	PurgeReason string    `json:"purge_reason"`

	TargetUUID     string `json:"target_uuid"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StoreUUID      string `json:"store_uuid"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
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

func (f *ArchiveFilter) Query() string {
	var wheres []string = []string{"a.uuid = a.uuid"}
	n := 1
	if f.ForTarget != "" {
		wheres = append(wheres, fmt.Sprintf("target_uuid = $%d", n))
		n++
	}
	if f.ForStore != "" {
		wheres = append(wheres, fmt.Sprintf("store_uuid = $%d", n))
		n++
	}
	if f.Before != nil {
		wheres = append(wheres, fmt.Sprintf("taken_at <= $%d", n))
		n++
	}
	if f.After != nil {
		wheres = append(wheres, fmt.Sprintf("taken_at >= $%d", n))
		n++
	}
	if len(f.WithStatus) > 0 {
		var params []string
		for range f.WithStatus {
			params = append(params, fmt.Sprintf("$%d", n))
			n++
		}
		wheres = append(wheres, fmt.Sprintf("status IN (%s)", strings.Join(params, ", ")))
	}
	if len(f.WithOutStatus) > 0 {
		var params []string
		for range f.WithOutStatus {
			params = append(params, fmt.Sprintf("$%d", n))
			n++
		}
		wheres = append(wheres, fmt.Sprintf("status NOT IN (%s)", strings.Join(params, ", ")))
	}
	if f.ExpiresBefore != nil {
		wheres = append(wheres, fmt.Sprintf("expires_at < $%d", n))
		n++
	}
	limit := ""
	if f.Limit != "" {
		limit = fmt.Sprintf(" LIMIT $%d", n)
		n++
	}

	return `
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.plugin, t.endpoint,
		       s.uuid, s.plugin, s.endpoint,
			   a.status, a.purge_reason

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY a.taken_at DESC, a.uuid ASC
	` + limit
}

func (f *ArchiveFilter) Args() []interface{} {
	var args []interface{}
	if f.ForTarget != "" {
		args = append(args, f.ForTarget)
	}
	if f.ForStore != "" {
		args = append(args, f.ForStore)
	}
	if f.Before != nil {
		args = append(args, f.Before.Unix())
	}
	if f.After != nil {
		args = append(args, f.After.Unix())
	}
	if len(f.WithStatus) > 0 {
		for _, e := range f.WithStatus {
			args = append(args, e)
		}
	}
	if len(f.WithOutStatus) > 0 {
		for _, e := range f.WithOutStatus {
			args = append(args, e)
		}
	}
	if f.ExpiresBefore != nil {
		args = append(args, f.ExpiresBefore.Unix())
	}
	if f.Limit != "" {
		args = append(args, f.Limit)
	}
	return args
}

func (db *DB) GetAllAnnotatedArchives(filter *ArchiveFilter) ([]*AnnotatedArchive, error) {
	l := []*AnnotatedArchive{}
	if filter.Limit != "" {
		if lim, err := strconv.Atoi(filter.Limit); err != nil || lim < 0 {
			return l, fmt.Errorf("Invalid limit given: '%s'", filter.Limit)
		}
	}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedArchive{}

		var takenAt, expiresAt *int64
		if err = r.Scan(
			&ann.UUID, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
			&ann.TargetUUID, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.StoreUUID, &ann.StorePlugin, &ann.StoreEndpoint,
			&ann.Status, &ann.PurgeReason); err != nil {

			return l, err
		}

		if takenAt != nil {
			ann.TakenAt = parseEpochTime(*takenAt)
		}

		if expiresAt != nil {
			ann.ExpiresAt = parseEpochTime(*expiresAt)
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetAnnotatedArchive(id uuid.UUID) (*AnnotatedArchive, error) {
	r, err := db.Query(`
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.plugin, t.endpoint,
		       s.uuid, s.plugin, s.endpoint, a.status, a.purge_reason

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE a.uuid = $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	ann := &AnnotatedArchive{}

	var takenAt, expiresAt *int64
	if err = r.Scan(
		&ann.UUID, &ann.StoreKey, &takenAt, &expiresAt, &ann.Notes,
		&ann.TargetUUID, &ann.TargetPlugin, &ann.TargetEndpoint,
		&ann.StoreUUID, &ann.StorePlugin, &ann.StoreEndpoint,
		&ann.Status, &ann.PurgeReason); err != nil {

		return nil, err
	}

	if takenAt != nil {
		ann.TakenAt = parseEpochTime(*takenAt)
	}

	if expiresAt != nil {
		ann.ExpiresAt = parseEpochTime(*expiresAt)
	}

	return ann, nil
}

func (db *DB) AnnotateArchive(id uuid.UUID, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = $1 WHERE uuid = $2`,
		notes, id.String(),
	)
}

func (db *DB) GetArchivesNeedingPurge() ([]*AnnotatedArchive, error) {
	filter := &ArchiveFilter{
		WithOutStatus: []string{"purged", "valid"},
	}
	return db.GetAllAnnotatedArchives(filter)
}

func (db *DB) GetExpiredArchives() ([]*AnnotatedArchive, error) {
	now := time.Now()
	filter := &ArchiveFilter{
		ExpiresBefore: &now,
		WithStatus:    []string{"valid"},
	}
	return db.GetAllAnnotatedArchives(filter)
}

func (db *DB) InvalidateArchive(id uuid.UUID) error {
	return db.Exec(`UPDATE archives SET status = 'invalid' WHERE uuid = $1`, id.String())
}

func (db *DB) PurgeArchive(id uuid.UUID) error {
	a, err := db.GetAnnotatedArchive(id)
	if err != nil {
		return err
	}

	if a.Status == "valid" {
		return fmt.Errorf("Invalid attempt to purge a 'valid' archive detected")
	}

	err = db.Exec(`UPDATE archives SET purge_reason =
                       (SELECT status FROM archives WHERE uuid = $1)
                       WHERE uuid = $1
                  `, id.String())
	if err != nil {
		return err
	}

	return db.Exec(`UPDATE archives SET status = 'purged' WHERE uuid = $1`, id.String())
}

func (db *DB) ExpireArchive(id uuid.UUID) error {
	return db.Exec(`UPDATE archives SET status = 'expired' WHERE uuid = $1`, id.String())
}

func (db *DB) DeleteArchive(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM archives WHERE uuid = $1`,
		id.String(),
	)
}
