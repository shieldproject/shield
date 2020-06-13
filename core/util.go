package core

func IsValidSystemRole(role string) bool {
	return role == "admin" || role == "manager" || role == "engineer" || role == "operator"
}
