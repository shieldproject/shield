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
	wheres := []string{}
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

	if len(wheres) == 0 {
		wheres = []string{"1"}
	} else if len(wheres) > 1 {
		wheres = []string{strings.Join(wheres, " OR ")}
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
	r, err := db.Query(query, args...)
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
	r, err := db.Query(`
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
	err := db.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, tenant.UUID, tenant.Name)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func (db *DB) UpdateTenant(tenant *Tenant) (*Tenant, error) {
	err := db.Exec(`
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
	rows, err := db.Query(`SELECT tenant_uuid, role FROM org_team_tenant_role WHERE org = ? AND team = ?`, org, team)
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

func (db *DB) DeleteTenant(tenant *Tenant, recurse bool) error {
	if recurse {
		/* delete all non-running tasks for this tenant */
		err := db.Exec(`
		   DELETE FROM tasks
		         WHERE stopped_at IS NOT NULL 
		           AND tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to delete tenant tasks: %s", err)
		}

		/* delete all tenant memberships for this tenant */
		err = db.Exec(`
		   DELETE FROM memberships 
		         WHERE tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to delete tenant memberships: %s", err)
		}

		/* delete all backup jobs for this tenant */
		err = db.Exec(`
		   DELETE FROM jobs
		         WHERE tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to delete tenant jobs: %s", err)
		}

		/* detach all data systems from this tenant */
		err = db.Exec(`
		  UPDATE targets
		     SET tenant_uuid = ''
		   WHERE tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to clear tenant targets: %s", err)
		}

		/* detach all cloud storage systems from this tenant */
		err = db.Exec(`
		   UPDATE stores
		      SET tenant_uuid = ''
		    WHERE tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to clear tenant stores: %s", err)
		}

		/* detach and expire all archives from this tenant */
		err = db.Exec(`
		   UPDATE archives
		      SET tenant_uuid = '', status = 'tenant deleted'
		    WHERE tenant_uuid = ?`, tenant.UUID)
		if err != nil {
			return fmt.Errorf("unable to mark tenant archives for deletion: %s", err)
		}

	} else {
		if n, _ := db.Count(`SELECT uuid FROM jobs WHERE tenant_uuid = ?`, tenant.UUID); n > 0 {
			return fmt.Errorf("unable to delete tenant: tenant has outstanding jobs")
		}

		if n, _ := db.Count(`SELECT uuid FROM stores WHERE tenant_uuid = ?`, tenant.UUID); n > 0 {
			return fmt.Errorf("unable to delete tenant: tenant has outstanding stores")
		}

		if n, _ := db.Count(`SELECT uuid FROM targets WHERE tenant_uuid = ?`, tenant.UUID); n > 0 {
			return fmt.Errorf("unable to delete tenant: tenant has outstanding targets")
		}

		if n, _ := db.Count(`SELECT uuid FROM archives WHERE tenant_uuid = ? and status NOT IN ("purged")`, tenant.UUID); n > 0 {
			return fmt.Errorf("unable to delete tenant: tenant has outstanding archives")
		}

		if n, _ := db.Count(`SELECT uuid FROM tasks WHERE tenant_uuid = ?`, tenant.UUID); n > 0 {
			return fmt.Errorf("unable to delete tenant: tenant has outstanding tasks")
		}
	}

	return db.Exec(`
	   DELETE FROM tenants
	         WHERE uuid = ?`, tenant.UUID)
}
