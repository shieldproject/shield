package db

import (
	"github.com/pborman/uuid"
)

type Tenant struct {
	UUID uuid.UUID
	Name string
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
