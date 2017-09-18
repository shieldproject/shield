package db

import (
	"github.com/pborman/uuid"
)

type Membership struct {
	TenantUUID uuid.UUID
	TenantName string
	Role       string
}

func (db *DB) GetMembershipsForUser(user uuid.UUID) ([]*Membership, error) {
	r, err := db.Query(`
	    SELECT t.uuid, t.name, m.role
	      FROM tenants t INNER JOIN memberships m ON m.tenant_uuid = t.uuid
	     WHERE m.user_uuid = ?`, user.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*Membership, 0)
	for r.Next() {
		var (
			id   NullUUID
			name string
			role string
		)

		if err := r.Scan(&id, &name, &role); err != nil {
			return l, err
		}

		l = append(l, &Membership{
			TenantUUID: id.UUID,
			TenantName: name,
			Role:       role,
		})
	}

	return l, nil
}

//Manual close of sql transaction necessary to avoid database lockup due to defer
func (db *DB) AddUserToTenant(user, tenant, role string) error {
	r, err := db.Query(`
	    SELECT m.role
	      FROM memberships m
	     WHERE m.user_uuid = ?
	       AND m.tenant_uuid = ?`, user, tenant)
	if err != nil {
		return err
	}

	if r.Next() {
		r.Close()
		return db.Exec(`
		    UPDATE memberships
		       SET role = ?
		     WHERE user_uuid = ?
		       AND tenant_uuid = ?`, role, user, tenant)
	}

	r.Close()
	return db.Exec(`
	    INSERT INTO memberships (user_uuid, tenant_uuid, role)
	                     VALUES (?, ?, ?)`, user, tenant, role)
}

func (db *DB) RemoveUserFromTenant(user, tenant string) error {
	return db.Exec(`
	    DELETE FROM memberships m
	          WHERE m.user_uuid = ?
	            AND m.tenant_uuid = ?`, user, tenant)
}

//GetTenantsForUser given a user's uuid returns a slice of Tenants that the user has membership with
func (db *DB) GetTenantsForUser(user uuid.UUID) ([]*Tenant, error) {
	r, err := db.Query(`
	    SELECT t.uuid, t.name
	      FROM tenants t INNER JOIN memberships m ON m.tenant_uuid = t.uuid
	     WHERE m.user_uuid = ?`, user.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*Tenant, 0)
	for r.Next() {
		var (
			id   NullUUID
			name string
		)

		if err := r.Scan(&id, &name); err != nil {
			return l, err
		}

		l = append(l, &Tenant{
			UUID: id.UUID,
			Name: name,
		})
	}

	return l, nil
}
