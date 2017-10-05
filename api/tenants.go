package api

import (
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
}

//GetTenant given a tenant uuid returns a struct containing the given tenants name and UUID
func GetTenant(id uuid.UUID) (Tenant, error) {
	var data Tenant
	uri, err := ShieldURI("/v2/tenant/%s", id)
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
	uri.MaybeAddParameter("name", filter.Name)
	uri.MaybeAddParameter("exact", filter.ExactMatch)
	var data []Tenant
	return data, uri.Get(&data)
}
