package db

import (
	"fmt"
	"strings"
)

type Tenant struct {
	UUID          string  `json:"uuid"              mbus:"uuid"`
	Name          string  `json:"name"              mbus:"name"`
	Members       []*User `json:"members,omitempty"`
	DailyIncrease int64   `json:"daily_increase"    mbus:"daily_increase"`
	StorageUsed   int64   `json:"storage_used"      mbus:"storage_used"`
	ArchiveCount  int     `json:"archive_count"     mbus:"archive_count"`
}

type TenantFilter struct {
	Name       string
	ExactMatch bool
	UUID       string
	Limit      int
}

func (f *TenantFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	var args []interface{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.uuid = ?")
			args = append(args, f.UUID)
		} else {
			wheres = append(wheres, "t.uuid LIKE ? ESCAPE '/'")
			args = append(args, PatternPrefix(f.UUID))
		}
	}

	if f.Name != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.name = ?")
			args = append(args, f.Name)
		} else {
			wheres = append(wheres, "t.name LIKE ?")
			args = append(args, Pattern(f.Name))
		}
	}

	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
	    SELECT t.uuid, t.name, t.daily_increase, t.storage_used, t.archive_count
	      FROM tenants t
	     WHERE ` + strings.Join(wheres, " AND ") + `
	` + limit, args
}

func (db *DB) GetAllTenants(filter *TenantFilter) ([]*Tenant, error) {
	if filter == nil {
		filter = &TenantFilter{}
	}

	l := make([]*Tenant, 0)
	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		tenant := &Tenant{}

		var (
			daily, used *int64
			archives    *int
		)
		if err := r.Scan(&tenant.UUID, &tenant.Name, &daily, &used, &archives); err != nil {
			return l, err
		}
		if daily != nil {
			tenant.DailyIncrease = *daily
		}
		if used != nil {
			tenant.StorageUsed = *used
		}
		if archives != nil {
			tenant.ArchiveCount = *archives
		}

		l = append(l, tenant)
	}

	return l, nil
}

func (db *DB) GetTenant(id string) (*Tenant, error) {
	r, err := db.query(`
	     SELECT t.uuid, t.name,
	            t.daily_increase, t.storage_used, t.archive_count

	       FROM tenants t

	      WHERE t.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	tenant := &Tenant{}
	var (
		daily, used *int64
		archives    *int
	)
	if err := r.Scan(&tenant.UUID, &tenant.Name,
		&daily, &used, &archives); err != nil {
		return tenant, err
	}
	if daily != nil {
		tenant.DailyIncrease = *daily
	}
	if used != nil {
		tenant.StorageUsed = *used
	}
	if archives != nil {
		tenant.ArchiveCount = *archives
	}

	return tenant, nil
}

func (db *DB) EnsureTenant(name string) (*Tenant, error) {
	l, err := db.GetAllTenants(&TenantFilter{
		Name:       name,
		ExactMatch: true,
	})
	if err != nil {
		return nil, err
	}
	if len(l) == 1 {
		return l[0], nil
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("found %d tenants matching '%s'", len(l), name)
	}

	return db.CreateTenant(&Tenant{Name: name})
}

func (db *DB) CreateTenant(tenant *Tenant) (*Tenant, error) {
	if tenant.UUID == "" {
		tenant.UUID = RandomID()
	}
	err := db.exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, tenant.UUID, tenant.Name)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (db *DB) UpdateTenant(tenant *Tenant) (*Tenant, error) {
	err := db.exec(`
	   UPDATE tenants
	      SET name = ?,
	          daily_increase = ?,
	          archive_count  = ?,
	          storage_used   = ?
	    WHERE uuid = ?`,
		tenant.Name, tenant.DailyIncrease, tenant.ArchiveCount, tenant.StorageUsed,
		tenant.UUID)
	if err != nil {
		return nil, err
	}

	db.sendUpdateObjectEvent(tenant, "tenant:"+tenant.UUID)
	return tenant, nil
}

func (db *DB) GetTenantRole(org string, team string) (string, string, error) {
	rows, err := db.query(`SELECT tenant_uuid, role FROM org_team_tenant_role WHERE org = ? AND team = ?`, org, team)
	if err != nil {
		return "", "", err
	}

	defer rows.Close()
	if !rows.Next() {
		return "", "", nil
	}

	var id, role string
	err = rows.Scan(&id, &role)
	if err != nil {
		return "", "", err
	}
	return id, role, nil
}

func (db *DB) DeleteTenant(tenant *Tenant) error {
	err := db.exec(`DELETE FROM tenants WHERE uuid = ?`, tenant.UUID)
	if err != nil {
		return err
	}

	db.sendDeleteObjectEvent(tenant, "tenant:"+tenant.UUID)
	return nil
}
