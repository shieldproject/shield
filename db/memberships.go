package db

type Membership struct {
	TenantUUID string `json:"tenant_uuid" mbus:"tenant_uuid"`
	TenantName string `json:"tenant_name" mbus:"tenant_name"`
	Role       string `json:"role"        mbus:"role"`
}

func (db *DB) GetMembershipsForUser(user string) ([]*Membership, error) {
	r, err := db.Query(`
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

func (db *DB) AddUserToTenant(user, tenant, role string) error {
	r, err := db.Query(`
	    SELECT m.role
	      FROM memberships m
	     WHERE m.user_uuid = ?
	       AND m.tenant_uuid = ?`, user, tenant)
	if err != nil {
		return err
	}

	exists := r.Next()
	r.Close() /* so we can run another query... */

	if exists {
		err = db.Exec(`
		    UPDATE memberships
		       SET role = ?
		     WHERE user_uuid = ?
		       AND tenant_uuid = ?`,
			role, user, tenant)

	} else {
		err = db.Exec(`
		    INSERT INTO memberships (user_uuid, tenant_uuid, role)
		                     VALUES (?, ?, ?)`,
			user, tenant, role)
	}

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

//GetTenantsForUser given a user's uuid returns a slice of Tenants that the user has membership with
func (db *DB) GetTenantsForUser(user string) ([]*Tenant, error) {
	r, err := db.Query(`
	    SELECT t.uuid, t.name
	      FROM tenants t INNER JOIN memberships m ON m.tenant_uuid = t.uuid
	     WHERE m.user_uuid = ?`, user)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*Tenant, 0)
	for r.Next() {
		t := &Tenant{}

		if err := r.Scan(&t.UUID, &t.Name); err != nil {
			return l, err
		}

		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetUsersForTenant(tenant string) ([]*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend,
	           m.role
	      FROM users u INNER JOIN memberships m
	        ON u.uuid = m.user_uuid
	     WHERE m.tenant_uuid = ?`, tenant)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	l := make([]*User, 0)
	for r.Next() {
		u := &User{}

		if err = r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.Role); err != nil {
			return nil, err
		}

		l = append(l, u)
	}

	return l, nil
}
