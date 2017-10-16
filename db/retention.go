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

	TenantUUID uuid.UUID `json:"-"`
}

type RetentionFilter struct {
	ForTenant  string
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

	if f.ForTenant != "" {
		wheres = append(wheres, "r.tenant_uuid = ?")
		args = append(args, f.ForTenant)
	}

	if !f.SkipUsed && !f.SkipUnused {
		return `
			SELECT r.uuid, r.tenant_uuid, r.name, r.summary, r.expiry, -1 AS n
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
		SELECT DISTINCT r.uuid, r.tenant_uuid, r.name, r.summary, r.expiry, COUNT(j.uuid) AS n
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
		p := &RetentionPolicy{}
		var n int
		var this NullUUID
		var tenant NullUUID

		if err = r.Scan(&this, &tenant, &p.Name, &p.Summary, &p.Expires, &n); err != nil {
			return l, err
		}
		p.UUID = this.UUID
		p.TenantUUID = tenant.UUID

		l = append(l, p)
	}

	return l, nil
}

func (db *DB) GetRetentionPolicy(id uuid.UUID) (*RetentionPolicy, error) {
	r, err := db.Query(`
		SELECT uuid, tenant_uuid, name, summary, expiry
			FROM retention WHERE uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}
	p := &RetentionPolicy{}
	var this NullUUID
	var tenant NullUUID
	if err = r.Scan(&this, &tenant, &p.Name, &p.Summary, &p.Expires); err != nil {
		return nil, err
	}
	p.UUID = this.UUID
	p.TenantUUID = tenant.UUID

	return p, nil
}

func (db *DB) CreateRetentionPolicy(p *RetentionPolicy) (*RetentionPolicy, error) {
	p.UUID = uuid.NewRandom()
	return p, db.Exec(`
	   INSERT INTO retention (uuid, tenant_uuid, name, summary, expiry)
	                  VALUES (?,    ?,           ?,    ?,       ?)`,
		p.UUID.String(), p.TenantUUID.String(), p.Name, p.Summary, p.Expires)
}

func (db *DB) UpdateRetentionPolicy(p *RetentionPolicy) error {
	return db.Exec(`
	   UPDATE retention
	      SET name    = ?,
	          summary = ?,
	          expiry  = ?
	    WHERE uuid = ?`,
		p.Name, p.Summary, p.Expires, p.UUID.String())
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

//InheritRetentionTemplates gives a tenant the global (templated) retention policies
func (db *DB) InheritRetentionTemplates(tenantUUID uuid.UUID) error {

	policies, err := db.GetAllRetentionPolicies(&RetentionFilter{ForTenant: "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		return err
	}

	for _, policy := range policies {
		policy.TenantUUID = tenantUUID
		db.CreateRetentionPolicy(policy)
	}
	return nil
}
