package db

import (
	"fmt"
	"strings"
)

type Tenant struct {
	UUID          string  `json:"uuid"              mbus:"uuid"`
	Name          string  `json:"name"              mbus:"name"`
	Members       []*User `json:"members,omitempty"`
	DailyIncrease int64   `json:"daily_increase"    mbus:"daily_increase"`
	StorageUsed   int64   `json:"storage_used"      mbus:"storage_used"`
	ArchiveCount  int     `json:"archive_count"     mbus:"archive_count"`
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
		var dailyIncrease, storageUsed *int64
		var archiveCount *int
		t := &Tenant{}
		if err := r.Scan(&t.UUID, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
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

	var dailyIncrease, storageUsed *int64
	var archiveCount *int
	t := &Tenant{}
	if err := r.Scan(&t.UUID, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
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

	var dailyIncrease, storageUsed *int64
	var archiveCount *int
	t := &Tenant{}
	if err := r.Scan(&t.UUID, &t.Name, &dailyIncrease, &storageUsed, &archiveCount); err != nil {
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
	return t, nil
}

func (db *DB) EnsureTenant(name string) (*Tenant, error) {
	if t, err := db.GetTenantByName(name); t != nil {
		return t, err
	}
	return db.CreateTenant("", name)
}

func (db *DB) CreateTenant(id, name string) (*Tenant, error) {
	if id == "" {
		id = randomID()
	}
	t := &Tenant{
		UUID: id,
		Name: name,
	}
	err := db.exec(`INSERT INTO tenants (uuid, name) VALUES (?, ?)`, t.UUID, t.Name)
	if err != nil {
		return nil, err
	}

	fmt.Printf("SENDING create-object MESSAGES via MBUS...\n")
	db.sendCreateObjectEvent(t, "admins")
	return t, nil
}

func (db *DB) UpdateTenant(t *Tenant) (*Tenant, error) {
	err := db.exec(`
		UPDATE tenants 
			SET name = ?,
			daily_increase = ?,
			archive_count  = ?,
			storage_used   = ? 
			WHERE uuid = ?`, t.Name, t.DailyIncrease, t.ArchiveCount, t.StorageUsed, t.UUID)
	if err != nil {
		return nil, err
	}

	db.sendUpdateObjectEvent(t, "admins", t.UUID)
	return t, nil
}

func (db *DB) GetTenantRole(org string, team string) (string, string, error) {
	rows, err := db.query(`SELECT tenant_uuid, role FROM org_team_tenant_role WHERE org = ? AND team = ?`, org, team)
	if err != nil {
		return "", "", err
	}

	defer rows.Close()
	if !rows.Next() {
		return "", "", nil
	}

	var id, role string
	err = rows.Scan(&id, &role)
	if err != nil {
		return "", "", err
	}
	return id, role, nil
}

func (db *DB) DeleteTenant(t *Tenant) error {
	err := db.exec(`DELETE FROM tenants WHERE uuid = ?`, t.UUID)
	if err != nil {
		return err
	}

	db.sendDeleteObjectEvent(t, "admins", t.UUID)
	return nil
}
