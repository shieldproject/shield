package db

import (
	"github.com/pborman/uuid"
)

//Role contains the fields in the role table in the core db
type Role struct {
	RoleID   int
	RoleName string
	Right    string
}

//GetRoleForUserTenant takes in a user uuid and a tenant uuid and returns the users role for that given tenant
func (db *DB) GetRoleForUserTenant(user uuid.UUID, tenant uuid.UUID) (*Role, error) {
	r, err := db.Query(` 
		SELECT role
		FROM memberships
		WHERE user_uuid = ?
		AND tenant_uuid = ?
		`, user.String(), tenant.String())
	if err != nil {
		return nil, err
	}

	defer r.Close()
	var roleID int
	var roleName string
	var roleRight string
	err = r.Scan(&roleID, &roleName, &roleRight)
	if err != nil {
		return nil, err
	}
	returnRole := Role{
		RoleID:   roleID,
		RoleName: roleName,
		Right:    roleRight,
	}
	return &returnRole, nil
}
