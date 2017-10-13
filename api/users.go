package api

import (
	"errors"
	"fmt"

	"github.com/pborman/uuid"
)

type v2LocalTenant struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type User struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account string `json:"account"`
	Backend string `json:"backend"`
	SysRole string `json:"sysrole"`

	Tenants []v2LocalTenant `json:"tenants"`
	pwhash  string
}

type UserFilter struct {
	UUID       string
	Backend    string
	Account    string
	Limit      string
	SysRole    string
	ExactMatch YesNo
}

func CreateUser(contentJSON string) (User, error) {
	uri, err := ShieldURI("/v2/auth/local/users")
	if err != nil {
		return User{}, err
	}

	respMap := make(map[string]string)
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		if create_error, present := respMap["error"]; present {
			return User{}, errors.New(create_error)
		}
		return User{}, err
	}

	return GetUser(uuid.Parse(respMap["uuid"]))
}

func GetUser(id uuid.UUID) (User, error) {
	var data User
	uri, err := ShieldURI("/v2/auth/local/users/%s", id)
	if err != nil {
		return User{}, err
	}
	return data, uri.Get(&data)
}

func GetUsers(filter UserFilter) ([]User, error) {
	uri, err := ShieldURI("/v2/auth/local/users")
	if err != nil {
		return []User{}, err
	}
	uri.MaybeAddParameter("uuid", filter.UUID)
	uri.MaybeAddParameter("account", filter.Account)
	uri.MaybeAddParameter("sysrole", filter.SysRole)
	uri.MaybeAddParameter("limit", filter.Limit)
	uri.MaybeAddParameter("exact", filter.ExactMatch)

	var data []User
	return data, uri.Get(&data)
}

func DeleteUser(id uuid.UUID) error {
	uri, err := ShieldURI("/v2/auth/local/users/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}

func UpdateUser(id uuid.UUID, contentJSON string) (User, error) {
	uri, err := ShieldURI("/v2/auth/local/users/%s", id)
	if err != nil {
		return User{}, err
	}
	respMap := make(map[string]string)
	if err := uri.Patch(&respMap, contentJSON); err != nil {
		if update_error, present := respMap["error"]; present {
			return User{}, errors.New(update_error)
		}
		return User{}, err
	}
	return GetUser(id)
}

func LocalTenantsToString(tennants []v2LocalTenant, showTennantUUID bool) string {
	tennantString := ""
	for _, tennant := range tennants {
		if showTennantUUID {
			tennantString += fmt.Sprintf("%s  %s (%s)\n", tennant.UUID, tennant.Name, tennant.Role)
		} else {
			tennantString += fmt.Sprintf("%s (%s)\n", tennant.Name, tennant.Role)
		}
	}
	return tennantString
}
