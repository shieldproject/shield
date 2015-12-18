package db

import (
	"fmt"
	"sort"
)

var CurrentSchema int = currentSchema()

var Schemas = map[int]Schema{
	1: v1Schema{},
	2: v2Schema{},
}

type Schema interface {
	Deploy(*DB) error
}

func (db *DB) Setup() error {
	current, err := db.SchemaVersion()
	if err != nil {
		return err
	}

	if current > CurrentSchema {
		err = fmt.Errorf("Schema version %d is newer than this version of SHIELD (%d)", current, CurrentSchema)
		if err != nil {
			return err
		}
	}

	versions := schemaVersions()

	for _, version := range versions {
		if current < version {
			err = Schemas[version].Deploy(db)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func schemaVersions() []int {
	var versions []int
	for k, _ := range Schemas {
		versions = append(versions, k)
	}
	sort.Ints(versions)
	return versions
}

func currentSchema() int {
	versions := schemaVersions()
	return int(versions[len(versions)-1:][0])
}

func (db *DB) SchemaVersion() (int, error) {
	r, err := db.Query(`SELECT version FROM schema_info LIMIT 1`)
	if err != nil {
		if err.Error() == "no such table: schema_info" {
			return 0, nil
		}
		if err.Error() == `pq: relation "schema_info" does not exist` {
			return 0, nil
		}
		return 0, err
	}
	defer r.Close()

	// no records = no schema
	if !r.Next() {
		return 0, nil
	}

	var v int
	err = r.Scan(&v)
	// failed unmarshall is an actual error
	if err != nil {
		return 0, err
	}

	// invalid (negative) schema version is an actual error
	if v < 0 {
		return 0, fmt.Errorf("Invalid schema version %d found", v)
	}

	return int(v), nil
}

func (db *DB) CheckCurrentSchema() error {
	v, err := db.SchemaVersion()
	if err != nil {
		return err
	}
	if v != CurrentSchema {
		return fmt.Errorf("wrong schema version (%d, but want to be at %d)", v, CurrentSchema)
	}
	return nil
}
