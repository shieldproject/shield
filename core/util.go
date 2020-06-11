package core

func IsValidTenantRole(role string) bool {
	return role == "admin" || role == "engineer" || role == "operator"
}

func IsValidSystemRole(role string) bool {
	return role == "admin" || role == "manager" || role == "engineer"
}
