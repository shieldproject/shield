package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type RetentionPolicy struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	Expires uint      `json:"expires"`
}

type RetentionFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ExactMatch bool
}

func (f *RetentionFilter) Query() (string, []interface{}) {
	wheres := []string{"r.uuid = r.uuid"}
	var args []interface{}

	if f.SearchName != "" {
		comparator := "LIKE"
		toAdd := Pattern(f.SearchName)
		if f.ExactMatch {
			comparator = "="
			toAdd = f.SearchName
		}
		wheres = append(wheres, fmt.Sprintf("r.name %s ?", comparator))
		args = append(args, toAdd)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT r.uuid, r.name, r.summary, r.expiry, -1 AS n
				FROM retention r
				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY r.name, r.uuid ASC
		`, args
	}

	// by default, show retention policies with no attached jobs (unused)
	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		// otherwise, only show retention policies that have attached jobs
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
		SELECT DISTINCT r.uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
			FROM retention r
				LEFT JOIN jobs j
					ON j.retention_uuid = r.uuid
			WHERE ` + strings.Join(wheres, " AND ") + `
			GROUP BY r.uuid
			` + having + `
			ORDER BY r.name, r.uuid ASC
	`, args
}

func (db *DB) GetAllRetentionPolicies(filter *RetentionFilter) ([]*RetentionPolicy, error) {
	if filter == nil {
		filter = &RetentionFilter{}
	}

	l := []*RetentionPolicy{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &RetentionPolicy{}
		var n int
		var this NullUUID

		if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Expires, &n); err != nil {
			return l, err
		}
		ann.UUID = this.UUID

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetRetentionPolicy(id uuid.UUID) (*RetentionPolicy, error) {
	r, err := db.Query(`
		SELECT uuid, name, summary, expiry
			FROM retention WHERE uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	ann := &RetentionPolicy{}
	var this NullUUID
	if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Expires); err != nil {
		return nil, err
	}
	ann.UUID = this.UUID

	return ann, nil
}

func (db *DB) AnnotateRetentionPolicy(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE retention SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateRetentionPolicy(expiry uint) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO retention (uuid, expiry) VALUES (?, ?)`,
		id.String(), expiry,
	)
}

func (db *DB) UpdateRetentionPolicy(id uuid.UUID, expiry uint) error {
	return db.Exec(
		`UPDATE retention SET expiry = ? WHERE uuid = ?`,
		expiry, id.String(),
	)
}

func (db *DB) DeleteRetentionPolicy(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.retention_uuid = ?`,
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
		return false, fmt.Errorf("Retention policy %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM retention WHERE uuid = ?`,
		id.String(),
	)
}
