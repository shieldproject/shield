package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type AnnotatedStore struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

type StoreFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
}

func (f *StoreFilter) Args() []interface{} {
	args := []interface{}{}
	if f.SearchName != "" {
		args = append(args, Pattern(f.SearchName))
	}
	if f.ForPlugin != "" {
		args = append(args, f.ForPlugin)
	}
	return args
}

func (f *StoreFilter) Query() string {
	var wheres []string = []string{"s.uuid = s.uuid"}
	n := 1
	if f.SearchName != "" {
		wheres = append(wheres, fmt.Sprintf("s.name LIKE $%d", n))
		n++
	}
	if f.ForPlugin != "" {
		wheres = append(wheres, fmt.Sprintf("s.plugin = $%d", n))
		n++
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint, -1 AS n
				FROM stores s
				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY s.name, s.uuid ASC
		`
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
	`
}

func (db *DB) GetAllAnnotatedStores(filter *StoreFilter) ([]*AnnotatedStore, error) {
	l := []*AnnotatedStore{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedStore{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetAnnotatedStore(id uuid.UUID) (*AnnotatedStore, error) {
	r, err := db.Query(`
		SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint
			FROM stores s
				LEFT JOIN jobs j
					ON j.store_uuid = s.uuid
			WHERE s.uuid = $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &AnnotatedStore{}

	if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint); err != nil {
		return nil, err
	}

	return ann, nil
}

func (db *DB) AnnotateStore(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE stores SET name = $1, summary = $2 WHERE uuid = $3`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateStore(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO stores (uuid, plugin, endpoint) VALUES ($1, $2, $3)`,
		id.String(), plugin, endpoint,
	)
}

func (db *DB) UpdateStore(id uuid.UUID, plugin string, endpoint interface{}) error {
	return db.Exec(
		`UPDATE stores SET plugin = $1, endpoint = $2 WHERE uuid = $3`,
		plugin, endpoint, id.String(),
	)
}

func (db *DB) DeleteStore(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.store_uuid = $1`,
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
		`DELETE FROM stores WHERE uuid = $1`,
		id.String(),
	)
}
