package db

import (
	"fmt"

	"github.com/pborman/uuid"
)

type AnnotatedTarget struct {
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Summary  string `json:"summary"`
	Plugin   string `json:"plugin"`
	Endpoint string `json:"endpoint"`
}

type TargetFilter struct {
	SkipUsed bool
	SkipUnused bool
	ForPlugin string
}

func (f *TargetFilter) Args() []interface{} {
	args := []interface{}{}
	if f.ForPlugin != "" {
		args = append(args, f.ForPlugin)
	}
	return args
}

func (f *TargetFilter) Query() string {
	where := ""
	if f.ForPlugin != "" {
		where = "WHERE plugin = ?"
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, plugin, endpoint, -1 AS n
				FROM targets ` + where + `
				ORDER BY name, uuid ASC
		`
	}

	// by default, show targets with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show targets that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			` + where + ` GROUP BY t.uuid
			` + having + `
			ORDER BY t.name, t.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedTargets(filter *TargetFilter) ([]*AnnotatedTarget, error) {
	l := []*AnnotatedTarget{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedTarget{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) AnnotateTarget(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE targets SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateTarget(plugin string, endpoint interface{}) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO targets (uuid, plugin, endpoint) VALUES (?, ?, ?)`,
		id.String(), plugin, endpoint,
	)
}

func (db *DB) UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}) error {
	return db.Exec(
		`UPDATE targets SET plugin = ?, endpoint = ? WHERE uuid = ?`,
		plugin, endpoint, id.String(),
	)
}

func (db *DB) DeleteTarget(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = ?`,
		id.String(),
	)
	if err != nil {
		return false, err
	}
	defer r.Close()

	// already deleted?
	if !r.Next() {
		return true, nil
	}

	var numJobs int
	if err = r.Scan(&numJobs); err != nil {
		return false, err
	}

	if numJobs < 0 {
		return false, fmt.Errorf("Target %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM targets WHERE uuid = ?`,
		id.String(),
	)
}
