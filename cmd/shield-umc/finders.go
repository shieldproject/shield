package main

import (
	"github.com/pborman/uuid"
	"github.com/starkandwayne/shield/db"
)

func finduser(database *db.DB, username_or_uuid string) string {
	if uuid.Parse(username_or_uuid) != nil {
		return username_or_uuid
	}

	users, err := database.GetAllUsers(&db.UserFilter{
		Account: username_or_uuid,
	})
	if err != nil || len(users) != 1 {
		return ""
	}
	return users[0].UUID.String()
}

func findtenant(database *db.DB, name_or_uuid string) string {
	if uuid.Parse(name_or_uuid) != nil {
		return name_or_uuid
	}

	tenants, err := database.GetAllTenants(&db.TenantFilter{})
	if err != nil {
		return ""
	}
	for _, tenant := range tenants {
		if tenant.Name == name_or_uuid {
			return tenant.UUID.String()
		}
	}
	return ""
}
