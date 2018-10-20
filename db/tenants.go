package db

import (
	"fmt"
	"strings"

	"github.com/pborman/uuid"
)

type Tenant struct {
	UUID          uuid.UUID `json:"uuid"              mbus:"uuid"`
	Name          string    `json:"name"              mbus:"name"`
	Members       []*User   `json:"members,omitempty"`
	DailyIncrease int64     `json:"daily_increase"`
	StorageUsed   int64     `json:"storage_used"`
	ArchiveCount  int       `json:"archive_count"`
}

type TenantFilter struct {
	Name       string
	ExactMatch bool
	UUID       string
	Limit      int
}

func (f *TenantFilter) Query() (string, []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	var args []interface{}

	if f.UUID != "" {
		wheres = append(wheres, "t.uuid = ?")
		args = append(args, f.UUID)
	}

	if f.Name != "" {
		if f.ExactMatch {
			wheres = append(wheres, "t.name = ?")
			args = append(args, f.Name)
		} else {
			wheres = append(wheres, "t.name LIKE ?")
			args = append(args, Pattern(f.Name))
		}
	}

	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
	    SELECT t.uuid, t.name, t.daily_increase, t.storage_used, t.archive_count
	      FROM tenants t
	     WHERE ` + strings.Join(wheres, " AND ") + `
	` + limit, args
}

func (db *DB) GetAllTenants(filter *TenantFilter) ([]*Tenant, error) {
	if filter == nil {
		filter = &TenantFilter{}
	}

	l := make([]*Tenant, 0)

	query, args := filter.Query()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		var id NullUUID
		var dailyIncrease, storageUsed *int64
		var archiveCount *int
		t := &Tenant{}

		if err := r.Scan(&id, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
			return l, err
		}
		if dailyIncrease != nil {
			t.DailyIncrease = *dailyIncrease
		}
		if storageUsed != nil {
			t.StorageUsed = *storageUsed
		}
		if archiveCount != nil {
			t.ArchiveCount = *archiveCount
		}
		t.UUID = id.UUID
		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetTenant(id string) (*Tenant, error) {
	r, err := db.query(`
		SELECT t.uuid, t.name, t.daily_increase, t.storage_used, t.archive_count
		FROM tenants t 
		WHERE t.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this NullUUID
	var dailyIncrease, storageUsed *int64
	var archiveCount *int
	t := &Tenant{}

	if err := r.Scan(&this, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
		return t, err
	}
	if dailyIncrease != nil {
		t.DailyIncrease = *dailyIncrease
	}
	if storageUsed != nil {
		t.StorageUsed = *storageUsed
	}
	if archiveCount != nil {
		t.ArchiveCount = *archiveCount
	}
	t.UUID = this.UUID
	return t, nil
}

func (db *DB) GetTenantByName(name string) (*Tenant, error) {
	r, err := db.query(`
		SELECT t.uuid, t.name, t.daily_increase, t.storage_used, t.archive_count
		FROM tenants t 
		WHERE t.name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this NullUUID
	var dailyIncrease, storageUsed *int64
	var archiveCount *int
	t := &Tenant{}

	if err := r.Scan(&this, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
		return t, err
	}
	if dailyIncrease != nil {
		t.DailyIncrease = *dailyIncrease
	}
	if storageUsed != nil {
		t.StorageUsed = *storageUsed
	}
	if archiveCount != nil {
		t.ArchiveCount = *archiveCount
	}
	t.UUID = this.UUID
	return t, nil
}

func (db *DB) EnsureTenant(name string) (*Tenant, error) {
	if t, err := db.GetTenantByName(name); t != nil {
		return t, err
	}
	return db.CreateTenant("", name)
}

func (db *DB) CreateTenant(given_uuid string, given_name string) (*Tenant, error) {
	var id uuid.UUID
	if given_uuid != "" {
		id = uuid.Parse(given_uuid)
	} else {
		id = uuid.NewRandom()
	}
	if id == nil {
		return nil, fmt.Errorf("uuid must be of format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx")
	}

	t := &Tenant{
		UUID: id,
		Name: given_name,
	}
	err := db.exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, t.UUID.String(), t.Name)
	if err != nil {
		return nil, err
	}

	fmt.Printf("SENDING create-object MESSAGES via MBUS...\n")
	db.sendCreateObjectEvent(toTenant(t.UUID), t)
	db.sendCreateObjectEvent(toAdmins(), t)
	return t, nil
}

func (db *DB) UpdateTenant(t *Tenant) (*Tenant, error) {
	err := db.exec(`
		UPDATE tenants 
			SET name = ?,
			daily_increase = ?,
			archive_count  = ?,
			storage_used   = ? 
			WHERE uuid = ?`, t.Name, t.DailyIncrease, t.ArchiveCount, t.StorageUsed, t.UUID.String())
	if err != nil {
		return nil, err
	}

	db.sendUpdateObjectEvent(toTenant(t.UUID), t)
	db.sendUpdateObjectEvent(toAdmins(), t)
	return t, nil
}

func (db *DB) GetTenantRole(org string, team string) (uuid.UUID, string, error) {
	rows, err := db.query(`SELECT tenant_uuid, role FROM org_team_tenant_role WHERE org = ? AND team = ?`, org, team)
	if err != nil {
		return nil, "", err
	}

	defer rows.Close()
	if !rows.Next() {
		return nil, "", nil
	}

	var id, role string
	err = rows.Scan(&id, &role)
	if err != nil {
		return nil, "", err
	}
	return uuid.Parse(id), role, nil
}

func (db *DB) DeleteTenant(t *Tenant) error {
	err := db.exec(`DELETE FROM tenants WHERE uuid = ?`, t.UUID.String())
	if err != nil {
		return err
	}

	db.sendDeleteObjectEvent(toTenant(t.UUID), t)
	db.sendDeleteObjectEvent(toAdmins(), t)
	return nil
}
