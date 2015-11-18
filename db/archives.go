package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

type AnnotatedArchive struct {
	UUID      string `json:"uuid"`
	StoreKey  string `json:"key"`
	TakenAt   string `json:"taken_at"`
	ExpiresAt string `json:"expires_at"`
	Notes     string `json:"notes"`

	TargetUUID     string `json:"target_uuid"`
	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StoreUUID      string `json:"store_uuid"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
}

type ArchiveFilter struct {
	ForTarget string
	ForStore  string
	Before    *time.Time
	After     *time.Time
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
	return `
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.uuid, t.plugin, t.endpoint,
		       s.uuid, s.plugin, s.endpoint

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY a.taken_at DESC, a.uuid ASC
	`
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
		args = append(args, f.Before)
	}
	if f.After != nil {
		args = append(args, f.After)
	}
	return args
}

func (db *DB) GetAllAnnotatedArchives(filter *ArchiveFilter) ([]*AnnotatedArchive, error) {
	l := []*AnnotatedArchive{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedArchive{}

		if err = r.Scan(
			&ann.UUID, &ann.StoreKey, &ann.TakenAt, &ann.ExpiresAt, &ann.Notes,
			&ann.TargetUUID, &ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.StoreUUID, &ann.StorePlugin, &ann.StoreEndpoint); err != nil {

			return l, err
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
		       s.uuid, s.plugin, s.endpoint

		FROM archives a
			INNER JOIN targets t   ON t.uuid = a.target_uuid
			INNER JOIN stores  s   ON s.uuid = a.store_uuid

		WHERE a.uuid == $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	ann := &AnnotatedArchive{}

	if err = r.Scan(
		&ann.UUID, &ann.StoreKey, &ann.TakenAt, &ann.ExpiresAt, &ann.Notes,
		&ann.TargetUUID, &ann.TargetPlugin, &ann.TargetEndpoint,
		&ann.StoreUUID, &ann.StorePlugin, &ann.StoreEndpoint); err != nil {

		return nil, err
	}

	return ann, nil
}

func (db *DB) AnnotateArchive(id uuid.UUID, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = $1 WHERE uuid = $2`,
		notes, id.String(),
	)
}

func (db *DB) DeleteArchive(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM archives WHERE uuid = $1`,
		id.String(),
	)
}
