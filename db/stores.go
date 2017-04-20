package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type Store struct {
	UUID     uuid.UUID `json:"uuid"`
	Name     string    `json:"name"`
	Summary  string    `json:"summary"`
	Plugin   string    `json:"plugin"`
	Endpoint string    `json:"endpoint"`
}

type StoreFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ExactMatch bool
}

func (f *StoreFilter) Query() (string, []interface{}) {
	wheres := []string{"s.uuid = s.uuid"}
	args := []interface{}{}
	if f.SearchName != "" {
		comparator := "LIKE"
		toAdd := Pattern(f.SearchName)
		if f.ExactMatch {
			comparator = "="
			toAdd = f.SearchName
		}
		wheres = append(wheres, fmt.Sprintf("s.name %s ?", comparator))
		args = append(args, toAdd)
	}
	if f.ForPlugin != "" {
		wheres = append(wheres, "s.plugin = ?")
		args = append(args, f.ForPlugin)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint, -1 AS n
				FROM stores s
				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY s.name, s.uuid ASC
		`, args
	}

	// by default, show stores with no attached jobs (unused)
	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		// otherwise, only show stores that have attached jobs
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
		SELECT DISTINCT s.uuid, s.name, s.summary, s.plugin, s.endpoint, COUNT(j.uuid) AS n
			FROM stores s
				LEFT JOIN jobs j
					ON j.store_uuid = s.uuid
			WHERE ` + strings.Join(wheres, " AND ") + `
			GROUP BY s.uuid
			` + having + `
			ORDER BY s.name, s.uuid ASC
	`, args
}

func (db *DB) GetAllStores(filter *StoreFilter) ([]*Store, error) {
	if filter == nil {
		filter = &StoreFilter{}
	}

	l := []*Store{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Store{}
		var n int
		var this NullUUID
		if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}
		ann.UUID = this.UUID

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetStore(id uuid.UUID) (*Store, error) {
	r, err := db.Query(`
		SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint
			FROM stores s
				LEFT JOIN jobs j
					ON j.store_uuid = s.uuid
			WHERE s.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &Store{}
	var this NullUUID
	if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint); err != nil {
		return nil, err
	}
	ann.UUID = this.UUID

	return ann, nil
}

func (db *DB) AnnotateStore(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE stores SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateStore(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO stores (uuid, plugin, endpoint) VALUES (?, ?, ?)`,
		id.String(), plugin, endpoint,
	)
}

func (db *DB) UpdateStore(id uuid.UUID, plugin string, endpoint interface{}) error {
	return db.Exec(
		`UPDATE stores SET plugin = ?, endpoint = ? WHERE uuid = ?`,
		plugin, endpoint, id.String(),
	)
}

func (db *DB) DeleteStore(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = ?`,
		id.String(),
	)
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Store %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM stores WHERE uuid = ?`,
		id.String(),
	)
}
