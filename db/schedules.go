package db

import (
	"fmt"
	"github.com/pborman/uuid"
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
}

func (f *ScheduleFilter) Query() string {
	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT uuid, name, summary, timespec, -1 AS n
				FROM schedules
				ORDER BY name, uuid ASC
		`
	}

	// by default, show schedules with no attached jobs (unused)
	having := `HAVING n = 0`
	if f.SkipUnused {
		// otherwise, only show schedules that have attached jobs
		having = `HAVING n > 0`
	}

	return `
		SELECT DISTINCT s.uuid, s.name, s.summary, s.timespec, COUNT(j.uuid) AS n
			FROM schedules s
				LEFT JOIN jobs j
					ON j.schedule_uuid = s.uuid
			GROUP BY s.uuid
			` + having + `
			ORDER BY s.name, s.uuid ASC
	`
}

func (db *DB) GetAllAnnotatedSchedules(filter *ScheduleFilter) ([]*AnnotatedSchedule, error) {
	l := []*AnnotatedSchedule{}
	r, err := db.Query(filter.Query())
	if err != nil {
		return l, err
	}

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

func (db *DB) AnnotateSchedule(id uuid.UUID, name string, summary string) error {
	return db.Exec(
		`UPDATE schedules SET name = ?, summary = ? WHERE uuid = ?`,
		name, summary, id.String(),
	)
}

func (db *DB) CreateSchedule(timespec string) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO schedules (uuid, timespec) VALUES (?, ?)`,
		id.String(), timespec,
	)
}

func (db *DB) UpdateSchedule(id uuid.UUID, timespec string) error {
	return db.Exec(
		`UPDATE schedules SET timespec = ? WHERE uuid = ?`,
		timespec, id.String(),
	)
}

func (db *DB) DeleteSchedule(id uuid.UUID) (bool, error) {
	r, err := db.Query(
		`SELECT COUNT(uuid) FROM jobs WHERE jobs.schedule_uuid = ?`,
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
		return false, fmt.Errorf("Schedule %s is in used by %d (negative) Jobs", id.String(), numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.Exec(
		`DELETE FROM schedules WHERE uuid = ?`,
		id.String(),
	)
}
