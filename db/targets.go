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
}

func (f *TargetFilter) Query() (string, []interface{}) {
	var wheres []string = []string{"t.uuid = t.uuid"}
	args := []interface{}{}
	n := 1

	if f.SearchName != "" {
		wheres = append(wheres, fmt.Sprintf("t.name LIKE $%d", n))
		args = append(args, Pattern(f.SearchName))
		n++
	}
	if f.ForPlugin != "" {
		wheres = append(wheres, fmt.Sprintf("t.plugin LIKE $%d", n))
		args = append(args, f.ForPlugin)
		n++
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
			WHERE t.uuid = $1`, id.String())
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
		`UPDATE targets SET name = $1, summary = $2 WHERE uuid = $3`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateTarget(plugin string, endpoint interface{}, agent string) (uuid.UUID, error) {
	id := uuid.NewRandom()

	return id, db.Exec(
		`INSERT INTO targets (uuid, plugin, endpoint, agent) VALUES ($1, $2, $3, $4)`,
		id.String(), plugin, endpoint, agent,
	)
}

func (db *DB) UpdateTarget(id uuid.UUID, plugin string, endpoint interface{}, agent string) error {
	return db.Exec(
		`UPDATE targets SET plugin = $1, endpoint = $2, agent = $3 WHERE uuid = $4`,
		plugin, endpoint, agent, id.String(),
	)
}

func (db *DB) DeleteTarget(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.target_uuid = $1`,
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
		`DELETE FROM targets WHERE uuid = $1`,
		id.String(),
	)
}
