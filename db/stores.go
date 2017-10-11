package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type Store struct {
	UUID       uuid.UUID `json:"uuid"`
	Name       string    `json:"name"`
	Summary    string    `json:"summary"`
	Plugin     string    `json:"plugin"`
	Endpoint   string    `json:"endpoint"`
	TenantUUID uuid.UUID `json:"tenant_uuid"`
}

type StoreFilter struct {
	SkipUsed   bool
	SkipUnused bool
	SearchName string
	ForPlugin  string
	ForTenant  string
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
	if f.ForTenant != "" {
		wheres = append(wheres, "s.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint, s.tenant_uuid, -1 AS n
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
		SELECT DISTINCT s.uuid, s.name, s.summary, s.plugin, s.endpoint, s.tenant_uuid, COUNT(j.uuid) AS n
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
		var this, tenant NullUUID
		if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &tenant, &n); err != nil {
			return l, err
		}
		ann.UUID = this.UUID
		ann.TenantUUID = tenant.UUID
		l = append(l, ann)
	}

	return l, nil
}

func (db *DB) GetStore(id uuid.UUID) (*Store, error) {
	r, err := db.Query(`
		SELECT s.uuid, s.name, s.summary, s.plugin, s.endpoint, s.tenant_uuid
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
	var this, tenant NullUUID
	if err = r.Scan(&this, &ann.Name, &ann.Summary, &ann.Plugin, &ann.Endpoint, &tenant); err != nil {
		return nil, err
	}
	ann.UUID = this.UUID
	ann.TenantUUID = tenant.UUID

	return ann, nil
}

func (db *DB) CreateStore(new_store *Store) (uuid.UUID, error) {
	id := uuid.NewRandom()
	return id, db.Exec(
		`INSERT INTO stores (uuid, plugin, endpoint, tenant_uuid, name, summary) VALUES (?, ?, ?, ?, ?, ?)`,
		id.String(), new_store.Plugin, new_store.Endpoint, new_store.TenantUUID.String(), new_store.Name, new_store.Summary,
	)
}

func (db *DB) UpdateStore(update *Store) error {
	return db.Exec(
		`UPDATE stores SET plugin = ?, endpoint = ?, name = ?, summary = ? WHERE uuid = ?`,
		update.Plugin, update.Endpoint, update.Name, update.Summary, update.UUID.String(),
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
