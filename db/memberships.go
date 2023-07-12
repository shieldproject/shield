package db

import (
	"fmt"
)

type Membership struct {
	TenantUUID string `json:"tenant_uuid" mbus:"tenant_uuid"`
	TenantName string `json:"tenant_name" mbus:"tenant_name"`
	Role       string `json:"role"        mbus:"role"`
}

func (db *DB) GetMembershipsForUser(user string) ([]*Membership, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	r, err := db.query(`
	    SELECT t.uuid, t.name, m.role
	      FROM tenants t INNER JOIN memberships m ON m.tenant_uuid = t.uuid
	     WHERE m.user_uuid = ?`, user)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*Membership, 0)
	for r.Next() {
		m := &Membership{}

		if err := r.Scan(&m.TenantUUID, &m.TenantName, &m.Role); err != nil {
			return l, err
		}

		l = append(l, m)
	}

	return l, nil
}

func (db *DB) ClearMembershipsFor(user *User) error {
	return db.Exec(`DELETE FROM memberships WHERE user_uuid = ?`, user.UUID)
}

func (db *DB) AddUserToTenant(user, tenant_id, role string) error {
	tenant, err := db.GetTenant(tenant_id)
	if err != nil {
		return fmt.Errorf("unable to create tenant membership: %s", err)
	}
	err = db.exclusively(func() error {
		/* validate the user */
		if err := db.userShouldExist(user); err != nil {
			return fmt.Errorf("unable to create tenant membership: %s", err)
		}
		exists, err := db.exists(`
		    SELECT m.role
		      FROM memberships m
		     WHERE m.user_uuid = ?
		       AND m.tenant_uuid = ?`, user, tenant.UUID)
		if err != nil {
			return err
		}
		if exists {
			return db.exec(`
			    UPDATE memberships
			       SET role = ?
			     WHERE user_uuid = ?
			       AND tenant_uuid = ?`,
				role, user, tenant.UUID)

		} else {
			return db.exec(`
			    INSERT INTO memberships (user_uuid, tenant_uuid, role)
			                     VALUES (?, ?, ?)`,
				user, tenant.UUID, role)
		}
	})
	if err != nil {
		return err
	}

	db.sendTenantInviteEvent(user, tenant, role)
	return nil
}

func (db *DB) RemoveUserFromTenant(user, tenant string) error {
	err := db.Exec(`
	    DELETE FROM memberships
	          WHERE user_uuid = ?
	            AND tenant_uuid = ?`, user, tenant)
	if err != nil {
		return err
	}

	db.sendTenantBanishEvent(user, tenant)
	return nil
}

// GetTenantsForUser given a user's uuid returns a slice of Tenants that the user has membership with
func (db *DB) GetTenantsForUser(user string) ([]*Tenant, error) {
	l := make([]*Tenant, 0)
	return l, db.exclusively(func() error {
		r, err := db.query(`
		    SELECT t.uuid, t.name
		      FROM tenants t INNER JOIN memberships m ON m.tenant_uuid = t.uuid
		     WHERE m.user_uuid = ?`, user)
		if err != nil {
			return err
		}
		defer r.Close()

		for r.Next() {
			t := &Tenant{}

			if err := r.Scan(&t.UUID, &t.Name); err != nil {
				return err
			}

			l = append(l, t)
		}

		return nil
	})
}

func (db *DB) GetUsersForTenant(tenant string) ([]*User, error) {
	l := make([]*User, 0)
	return l, db.exclusively(func() error {
		r, err := db.query(`
		    SELECT u.uuid, u.name, u.account, u.backend,
		           m.role
		      FROM users u INNER JOIN memberships m
		        ON u.uuid = m.user_uuid
		     WHERE m.tenant_uuid = ?`, tenant)
		if err != nil {
			return err
		}
		defer r.Close()

		for r.Next() {
			u := &User{}

			if err = r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.Role); err != nil {
				return err
			}

			l = append(l, u)
		}

		return nil
	})
}
