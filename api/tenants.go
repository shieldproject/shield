package api

import (
	"errors"

	"github.com/pborman/uuid"
)

//Tenant contains the uuid and name fields of a tenant
type Tenant struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type TenantFilter struct {
	Name       string
	ExactMatch YesNo
	UUID       uuid.UUID
	Limit      string
}

func CreateTenant(contentJSON string) (Tenant, error) {
	uri, err := ShieldURI("/v2/tenants")
	if err != nil {
		return Tenant{}, err
	}

	respMap := make(map[string]string)
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		if create_error, present := respMap["error"]; present {
			return Tenant{}, errors.New(create_error)
		}
		return Tenant{}, err
	}

	return GetTenant(uuid.Parse(respMap["uuid"]))
}

//GetTenant given a tenant uuid returns a struct containing the given tenants name and UUID
func GetTenant(id uuid.UUID) (Tenant, error) {
	var data Tenant
	uri, err := ShieldURI("/v2/tenants/%s", id)
	if err != nil {
		return Tenant{}, err
	}
	return data, uri.Get(&data)
}

func GetTenants(filter TenantFilter) ([]Tenant, error) {
	uri, err := ShieldURI("/v2/tenants")
	if err != nil {
		return []Tenant{}, err
	}
	uri.MaybeAddParameter("uuid", filter.UUID)
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("limit", filter.Limit)

	var data []Tenant
	return data, uri.Get(&data)
}

func DeleteTenant(id uuid.UUID) error {
	uri, err := ShieldURI("/v2/tenants/%s", id)
	if err != nil {
		return err
	}
	return uri.Delete(nil)
}

func UpdateTenant(id uuid.UUID, contentJSON string) (Tenant, error) {
	uri, err := ShieldURI("/v2/tenants/%s", id)
	if err != nil {
		return Tenant{}, err
	}
	respMap := make(map[string]string)
	if err := uri.Patch(&respMap, contentJSON); err != nil {
		if update_error, present := respMap["error"]; present {
			return Tenant{}, errors.New(update_error)
		}
		return Tenant{}, err
	}
	return GetTenant(id)
}

func Banish(id uuid.UUID, contentJSON string) error {
	data := struct {
		UserUUID    string `json:"uuid"`
		UserAccount string `json:"account"`
	}{}
	uri, err := ShieldURI("/v2/tenants/%s/banish", id)
	if err != nil {
		return err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return err
	}
	return nil
}

func Invite(id uuid.UUID, contentJSON string) error {
	data := struct {
		UserUUID    string `json:"uuid"`
		UserAccount string `json:"account"`
		Role        string `json:"role"`
	}{}
	uri, err := ShieldURI("/v2/tenants/%s/invite", id)
	if err != nil {
		return err
	}
	if err := uri.Post(&data, contentJSON); err != nil {
		return err
	}
	return nil
}
