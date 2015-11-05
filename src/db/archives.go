package db

import (
	"strings"

	"github.com/pborman/uuid"
)

type AnnotatedArchive struct {
	UUID string `json:"uuid"`
	/* FIXME: need other fields for target / store / key */
	StoreKey  string `json:"key"`
	TakenAt   string `json:"taken_at"`
	ExpiresAt string `json:"expires_at"`
	Notes     string `json:"notes"`

	TargetPlugin   string `json:"target_plugin"`
	TargetEndpoint string `json:"target_endpoint"`
	StorePlugin    string `json:"store_plugin"`
	StoreEndpoint  string `json:"store_endpoint"`
}

type ArchiveFilter struct {
	ForTarget string
	ForStore  string
}

func (f *ArchiveFilter) Query() string {
	var wheres []string = []string{"1"}
	if f.ForTarget != "" {
		wheres = append(wheres, "target_uuid = ?")
	}
	if f.ForStore != "" {
		wheres = append(wheres, "store_uuid = ?")
	}
	return `
		SELECT a.uuid, a.store_key,
		       a.taken_at, a.expires_at, a.notes,
		       t.plugin, t.endpoint,
		       s.plugin, s.endpoint

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
	return args
}

func (db *DB) GetAllAnnotatedArchives(filter *ArchiveFilter) ([]*AnnotatedArchive, error) {
	l := []*AnnotatedArchive{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedArchive{}

		if err = r.Scan(
			&ann.UUID, &ann.StoreKey, &ann.TakenAt, &ann.ExpiresAt, &ann.Notes,
			&ann.TargetPlugin, &ann.TargetEndpoint,
			&ann.StorePlugin, &ann.StoreEndpoint); err != nil {

			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetAnnotatedArchive(id uuid.UUID) (*AnnotatedArchive, error) {
	return nil, nil
}

func (db *DB) AnnotateArchive(id uuid.UUID, notes string) error {
	return db.Exec(
		`UPDATE archives SET notes = ? WHERE uuid = ?`,
		notes, id.String(),
	)
}

func (db *DB) DeleteArchive(id uuid.UUID) (bool, error) {
	return true, db.Exec(
		`DELETE FROM archives WHERE uuid = ?`,
		id.String(),
	)
}
