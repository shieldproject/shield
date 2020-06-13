package db

type v13Schema struct{}

func (s v13Schema) Deploy(db *DB) error {
	var err error

	err = db.Exec(`ALTER TABLE jobs DROP COLUMN store_uuid`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE jobs ADD COLUMN bucket TEXT NOT NULL DEFAULT 'ssg:///'`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives DROP COLUMN store_uuid`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE tasks DROP COLUMN store_uuid`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE tasks DROP COLUMN store_plugin`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE tasks DROP COLUMN store_endpoint`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE tasks ADD COLUMN stream TEXT DEFAULT '{}'`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE tasks ADD COLUMN bucket TEXT DEFAULT ''`)
	if err != nil {
		return err
	}

	err = db.Exec(`DROP TABLE stores`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives DROP COLUMN compression`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE targets DROP COLUMN compression`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives DROP COLUMN encryption_type`)
	if err != nil {
		return err
	}

	err = db.Exec(`ALTER TABLE archives DROP COLUMN tenant_uuid`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE jobs DROP COLUMN tenant_uuid`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE tasks DROP COLUMN tenant_uuid`)
	if err != nil {
		return err
	}
	err = db.Exec(`ALTER TABLE users DROP COLUMN default_tenant`)
	if err != nil {
		return err
	}
	err = db.Exec(`DROP TABLE memberships`)
	if err != nil {
		return err
	}
	err = db.Exec(`DROP TABLE tenants`)
	if err != nil {
		return err
	}

	err = db.Exec(`UPDATE schema_info set version = 13`)
	if err != nil {
		return err
	}

	return nil
}
