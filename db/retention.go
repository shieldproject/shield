package db

import (
	"fmt"
	"strings"
)

type RetentionPolicy struct {
	UUID       string `json:"uuid"    mbus:"uuid"`
	TenantUUID string `json:"-"       mbus:"tennt_uuid"`
	Name       string `json:"name"    mbus:"name"`
	Summary    string `json:"summary" mbus:"summary"`
	Expires    uint   `json:"expires" mbus:"expires"`
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
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		p := &RetentionPolicy{}

		var n int
		if err = r.Scan(&p.UUID, &p.TenantUUID, &p.Name, &p.Summary, &p.Expires, &n); err != nil {
			return l, err
		}

		l = append(l, p)
	}

	return l, nil
}

func (db *DB) GetRetentionPolicy(id string) (*RetentionPolicy, error) {
	r, err := db.query(`
	   SELECT uuid, tenant_uuid, name, summary, expiry
	     FROM retention
	    WHERE uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	p := &RetentionPolicy{}
	if err = r.Scan(&p.UUID, &p.TenantUUID, &p.Name, &p.Summary, &p.Expires); err != nil {
		return nil, err
	}

	return p, nil
}

func (db *DB) CreateRetentionPolicy(p *RetentionPolicy) (*RetentionPolicy, error) {
	p.UUID = randomID()
	return p, db.exec(`
	   INSERT INTO retention (uuid, tenant_uuid, name, summary, expiry)
	                  VALUES (?,    ?,           ?,    ?,       ?)`,
		p.UUID, p.TenantUUID, p.Name, p.Summary, p.Expires)
}

func (db *DB) UpdateRetentionPolicy(p *RetentionPolicy) error {
	return db.exec(`
	   UPDATE retention
	      SET name    = ?,
	          summary = ?,
	          expiry  = ?
	    WHERE uuid = ?`,
		p.Name, p.Summary, p.Expires, p.UUID)
}

func (db *DB) DeleteRetentionPolicy(id string) (bool, error) {
	r, err := db.query(`SELECT COUNT(uuid) FROM jobs WHERE jobs.retention_uuid = ?`, id)
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
		return false, fmt.Errorf("Retention policy %s is in used by %d (negative) Jobs", id, numJobs)
	}
	if numJobs > 0 {
		return false, nil
	}

	r.Close()
	return true, db.exec(`DELETE FROM retention WHERE uuid = ?`, id)
}

func (db *DB) InheritRetentionTemplates(tenant *Tenant) error {
	policies, err := db.GetAllRetentionPolicies(&RetentionFilter{ForTenant: GlobalTenantUUID})
	if err != nil {
		return err
	}

	for _, policy := range policies {
		policy.TenantUUID = tenant.UUID
		db.CreateRetentionPolicy(policy)
	}
	return nil
}
