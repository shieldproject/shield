package db

type v7Schema struct{}

func (s v7Schema) Deploy(db *DB) error {
	var err error

	// rename tenant1 -> 'Default Tenant'
	err = db.Exec(`UPDATE tenants SET name = 'Default Tenant' WHERE name = 'tenant1'`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 7`)
	if err != nil {
		return err
	}

	return nil
}
