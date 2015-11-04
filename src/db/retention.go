package db

import (
	"fmt"

	"github.com/pborman/uuid"
)

type AnnotatedRetentionPolicy struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Expires uint   `json:"expires"`
}

type RetentionFilter struct {
	SkipUsed bool
	SkipUnused bool
}

func (f *RetentionFilter) Query() string {
	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, expiry, -1 AS n
				FROM retention
				ORDER BY name, uuid ASC
		`
	}

	// by default, show retention policies with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show retention policies that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT r.uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
			FROM retention r
				LEFT JOIN jobs j
					ON j.retention_uuid = r.uuid
			GROUP BY r.uuid
			` + having + `
			ORDER BY r.name, r.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedRetentionPolicies(filter *RetentionFilter) ([]*AnnotatedRetentionPolicy, error) {
	l := []*AnnotatedRetentionPolicy{}
	r, err := db.Query(filter.Query())
	if err != nil {
		return l, err
	}

	for r.Next() {
		ann := &AnnotatedRetentionPolicy{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.Expires, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
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

	// already deleted?
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
