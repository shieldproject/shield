package db

import (
	"errors"
	"fmt"

	"github.com/pborman/uuid"
)

type Tenant struct {
	UUID uuid.UUID `json:"uuid"`
	Name string    `json:"name"`
}

func (db *DB) GetAllTenants() ([]*Tenant, error) {
	r, err := db.Query(`SELECT t.uuid, t.name FROM tenants t`)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*Tenant, 0)
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

func (db *DB) CreateTenant(given_uuid string, given_name string) (*Tenant, error) {

	id := uuid.NewRandom()
	if given_uuid != "" {
		id = uuid.Parse(given_uuid)
	}
	if id == nil {
		return nil, errors.New("uuid must be of format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	}
	fmt.Printf("%s, %s \n", id, given_uuid)
	err := db.Exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, id.String(), given_name)
	if err != nil {
		return nil, err
	}
	t := &Tenant{}
	t.UUID = id
	t.Name = given_name
	return t, nil
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
