package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/jhunt/go-log"
)

type Fixup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Summary   string `json:"summary"`
	CreatedAt int64  `json:"created_at"`
	AppliedAt int64  `json:"applied_at"`
	fn        func(*DB) error
}

func (f *Fixup) ReApply(db *DB) error {
	if ff, found := fixups[f.ID]; found {
		return ff.Apply(db)
	}
	return fmt.Errorf("unrecognized fixup '%s'", f.ID)
}

func (f *Fixup) Apply(db *DB) error {
	err := f.fn(db)
	if err != nil {
		return fmt.Errorf("unable to apply fixup '%s': %s", f.ID, err)
	}

	err = db.Exec(`UPDATE fixups SET applied_at = ? WHERE id = ?`, time.Now().Unix(), f.ID)
	if err != nil {
		return fmt.Errorf("unable to track application of fixup '%s' in database: %s", f.ID, err)
	}

	return nil
}

type FixupFilter struct {
	ID          string
	SkipApplied bool
}

func (f *FixupFilter) Query() (string, []interface{}) {
	wheres := []string{"f.id = f.id"}
	var args []interface{}

	if f.SkipApplied {
		wheres = append(wheres, "f.applied_at IS NOT NULL")
	}

	if f.ID != "" {
		wheres = append(wheres, "f.id = ?")
		args = append(args, f.ID)
	}

	return `
		SELECT f.id, f.name, f.summary, f.created_at, f.applied_at
		  FROM fixups f
		 WHERE ` + strings.Join(wheres, " AND ") + `
		 ORDER BY created_at ASC`, args
}

var fixups map[string]*Fixup

func init() {
	fixups = make(map[string]*Fixup)
}

func (db *DB) GetAllFixups(filter *FixupFilter) ([]*Fixup, error) {
	if filter == nil {
		filter = &FixupFilter{}
	}

	l := []*Fixup{}
	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		f := &Fixup{}

		if err = r.Scan(&f.ID, &f.Name, &f.Summary, &f.CreatedAt, &f.AppliedAt); err != nil {
			return l, err
		}

		l = append(l, f)
	}

	return l, nil
}

func (db *DB) GetFixup(id string) (*Fixup, error) {
	r, err := db.GetAllFixups(&FixupFilter{ID: id})
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r[0], nil
}

func (db *DB) ApplyFixups() error {
	db.Exec(`DELETE FROM fixups WHERE id = ''`)

	for _, f := range fixups {
		if existing, err := db.GetFixup(f.ID); err != nil {
			return fmt.Errorf("unable to determine if fixup '%s' is in the database already: %s", f.ID, err)

		} else if existing == nil {
			log.Infof("INITIALIZING: creating fixup record for %s...", f.ID)

			err = db.Exec(`
				INSERT INTO fixups (id, name, summary, created_at, applied_at)
				              VALUES (?,  ?,    ?,       ?,          NULL)`,
				f.ID, f.Name, f.Summary, f.CreatedAt)
			if err != nil {
				return fmt.Errorf("unable to register fixup '%s': %s", f.ID, err)
			}

		} else {
			// in case we fix a typo in a name / summary / date...
			_ = db.Exec(`
				UPDATE fixups SET name = ?, summary = ?, created_at = ?
				            WHERE id = ?`, f.Name, f.Summary, f.CreatedAt, f.ID)

			if existing.AppliedAt > 0 {
				log.Infof("INITIALIZING: skipping fixup %s (already applied)...", f.ID)
				continue
			}
		}

		log.Infof("INITIALIZING: applying fixup %s...", f.ID)
		if err := f.Apply(db); err != nil {
			return err
		}
	}

	return nil
}
