package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type Tenant struct {
	UUID    uuid.UUID `json:"uuid"`
	Name    string    `json:"name"`
	Members []*User   `json:"members,omitempty"`
}

type TenantFilter struct {
	Name       string
	ExactMatch bool
	UUID       string
	Limit      string
}

func (f *TenantFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	var args []interface{}

	if f.UUID != "" {
		wheres = append(wheres, "t.uuid = ?")
		args = append(args, f.UUID)
	}

	if f.Name != "" {
		comparator := "LIKE"
		toAdd := Pattern(f.Name)
		if f.ExactMatch {
			comparator = "="
			toAdd = f.Name
		}
		wheres = append(wheres, fmt.Sprintf("t.name %s ?", comparator))
		args = append(args, toAdd)
	}

	limit := ""
	if f.Limit != "" {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
	    SELECT t.uuid, t.name
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
		var id NullUUID
		t := &Tenant{}

		if err := r.Scan(&id, &t.Name); err != nil {
			return l, err
		}
		t.UUID = id.UUID
		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetTenant(id string) (*Tenant, error) {
	r, err := db.Query(`SELECT t.uuid, t.name FROM tenants t WHERE t.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this NullUUID
	t := &Tenant{}

	if err := r.Scan(&this, &t.Name); err != nil {
		return t, err
	}
	t.UUID = this.UUID
	return t, nil
}

func (db *DB) GetTenantByName(name string) (*Tenant, error) {
	r, err := db.Query(`SELECT t.uuid, t.name FROM tenants t WHERE t.name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this NullUUID
	t := &Tenant{}

	if err := r.Scan(&this, &t.Name); err != nil {
		return t, err
	}
	t.UUID = this.UUID
	return t, nil
}

func (db *DB) EnsureTenant(name string) (*Tenant, error) {
	if t, err := db.GetTenantByName(name); t != nil {
		return t, err
	}
	return db.CreateTenant("", name)
}

func (db *DB) CreateTenant(given_uuid string, given_name string) (*Tenant, error) {
	var id uuid.UUID
	if given_uuid != "" {
		id = uuid.Parse(given_uuid)
	} else {
		id = uuid.NewRandom()
	}
	if id == nil {
		return nil, fmt.Errorf("uuid must be of format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	}

	err := db.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, id.String(), given_name)
	if err != nil {
		return nil, err
	}

	return &Tenant{
		UUID: id,
		Name: given_name,
	}, nil
}

func (db *DB) UpdateTenant(given_uuid string, given_name string) (*Tenant, error) {
	err := db.Exec(`UPDATE tenants SET name = ? WHERE uuid = ?`, given_name, given_uuid)
	if err != nil {
		return nil, err
	}

	t := &Tenant{}
	t.UUID = uuid.Parse(given_uuid)
	t.Name = given_name
	return t, nil
}

func (db *DB) GetTenantRole(org string, team string) (uuid.UUID, string, error) {
	rows, err := db.Query(`SELECT tenant_uuid, role FROM org_team_tenant_role WHERE org = ? AND team = ?`, org, team)
	if err != nil {
		return nil, "", err
	}

	defer rows.Close()
	if !rows.Next() {
		return nil, "", nil
	}

	var tenantUUID string
	var role string
	err = rows.Scan(&tenantUUID, &role)
	if err != nil {
		return nil, "", err
	}
	return uuid.Parse(tenantUUID), role, nil
}

func (db *DB) DeleteTenant(tenant *Tenant) error {
	return db.Exec(`
		DELETE FROM tenants
		      WHERE uuid = ?`, tenant.UUID.String())
}
