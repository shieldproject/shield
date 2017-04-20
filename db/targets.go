package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type Target struct {
	UUID     uuid.UUID `json:"uuid"`
	Name     string    `json:"name"`
	Summary  string    `json:"summary"`
	Plugin   string    `json:"plugin"`
	Endpoint string    `json:"endpoint"`
	Agent    string    `json:"agent"`
}

type TargetFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ExactMatch bool
}

func (f *TargetFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	args := []interface{}{}

	if f.SearchName != "" {
		comparator := "LIKE"
		toAdd := Pattern(f.SearchName)
		if f.ExactMatch {
			comparator = "="
			toAdd = f.SearchName
		}
		wheres = append(wheres, fmt.Sprintf("t.name %s ?", comparator))
		args = append(args, toAdd)
	}

	if f.ForPlugin != "" {
		wheres = append(wheres, "t.plugin LIKE ?")
		args = append(args, f.ForPlugin)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT t.uuid, t.name, t.summary, t.plugin, t.endpoint, t.agent, -1 AS n
				FROM targets t
				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY t.name, t.uuid ASC
		`, args
	}

	// by default, show targets with no attached jobs (unused)
	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		// otherwise, only show targets that have attached jobs
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
		SELECT DISTINCT t.uuid, t.name, t.summary, t.plugin, t.endpoint, t.agent, COUNT(j.uuid) AS n
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			WHERE ` + strings.Join(wheres, " AND ") + `
			GROUP BY t.uuid
			` + having + `
			ORDER BY t.name, t.uuid ASC
	`, args
}

func (db *DB) GetAllTargets(filter *TargetFilter) ([]*Target, error) {
	if filter == nil {
		filter = &TargetFilter{}
	}

	l := []*Target{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &Target{}
		var n int
		var this NullUUID
		if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &ann.Agent, &n); err != nil {
			return l, err
		}
		ann.UUID = this.UUID

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetTarget(id uuid.UUID) (*Target, error) {
	r, err := db.Query(`
		SELECT t.uuid, t.name, t.summary, t.plugin, t.endpoint, t.agent
			FROM targets t
				LEFT JOIN jobs j
					ON j.target_uuid = t.uuid
			WHERE t.uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &Target{}
	var this NullUUID
	if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &ann.Agent); err != nil {
		return nil, err
	}
	ann.UUID = this.UUID

	return ann, nil
}

func (db *DB) AnnotateTarget(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE targets SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateTarget(plugin string, endpoint interface{}, agent string) (uuid.UUID, error) {
	id := uuid.NewRandom()

	return id, db.Exec(
		`INSERT INTO targets (uuid, plugin, endpoint, agent) VALUES (?, ?, ?, ?)`,
		id.String(), plugin, endpoint, agent,
	)
}

func (db *DB) UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}, agent string) error {
	return db.Exec(
		`UPDATE targets SET plugin = ?, endpoint = ?, agent = ? WHERE uuid = ?`,
		plugin, endpoint, agent, id.String(),
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

	// already deleted
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
