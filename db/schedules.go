package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"

	"github.com/starkandwayne/shield/timespec"
)

type AnnotatedSchedule struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	When    string `json:"when"`
}

type ScheduleFilter struct {
	SkipUsed   bool
	SkipUnused bool

	SearchName string
}

func (f *ScheduleFilter) Args() []interface{} {
	var args []interface{}
	if f.SearchName != "" {
		args = append(args, Pattern(f.SearchName))
	}
	return args
}

func (f *ScheduleFilter) Query() string {
	var wheres []string = []string{"s.uuid = s.uuid"}
	n := 1

	if f.SearchName != "" {
		wheres = append(wheres, fmt.Sprintf("s.name LIKE $%d", n))
		n++
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT s.uuid, s.name, s.summary, s.timespec, -1 AS n
				FROM schedules s
				WHERE ` + strings.Join(wheres, " AND ") + `
				ORDER BY s.name, s.uuid ASC
		`
	}

	// by default, show schedules with no attached jobs (unused)
	having := `HAVING COUNT(j.uuid) = 0`
	if f.SkipUnused {
		// otherwise, only show schedules that have attached jobs
		having = `HAVING COUNT(j.uuid) > 0`
	}

	return `
		SELECT DISTINCT s.uuid, s.name, s.summary, s.timespec, COUNT(j.uuid) AS n
			FROM schedules s
				LEFT JOIN jobs j
					ON j.schedule_uuid = s.uuid
			WHERE ` + strings.Join(wheres, " AND ") + `
			GROUP BY s.uuid
			` + having + `
			ORDER BY s.name, s.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedSchedules(filter *ScheduleFilter) ([]*AnnotatedSchedule, error) {
	l := []*AnnotatedSchedule{}
	r, err := db.Query(filter.Query(), filter.Args()...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		ann := &AnnotatedSchedule{}
		var n int

		if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.When, &n); err != nil {
			return l, err
		}

		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetAnnotatedSchedule(id uuid.UUID) (*AnnotatedSchedule, error) {
	r, err := db.Query(`
		SELECT uuid, name, summary, timespec
			FROM schedules WHERE uuid = $1`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	ann := &AnnotatedSchedule{}

	if err = r.Scan(&ann.UUID, &ann.Name, &ann.Summary, &ann.When); err != nil {
		return nil, err
	}

	return ann, nil
}

func (db *DB) AnnotateSchedule(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE schedules SET name = $1, summary = $2 WHERE uuid = $3`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateSchedule(ts string) (uuid.UUID, error) {
	id := uuid.NewRandom()

	_, err := timespec.Parse(ts)
	if err != nil {
		return id, err
	}
	return id, db.Exec(
		`INSERT INTO schedules (uuid, timespec) VALUES ($1, $2)`,
		id.String(), ts,
	)
}

func (db *DB) UpdateSchedule(id uuid.UUID, ts string) error {
	_, err := timespec.Parse(ts)
	if err != nil {
		return err
	}
	return db.Exec(
		`UPDATE schedules SET timespec = $1 WHERE uuid = $2`,
		ts, id.String(),
	)
}

func (db *DB) DeleteSchedule(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.schedule_uuid = $1`,
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
		return false, fmt.Errorf("Schedule %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM schedules WHERE uuid = $1`,
		id.String(),
	)
}
