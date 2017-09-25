package store

import (
	"database/sql"
	"strconv"
)

type mysqlStore struct {
	dbProvider DbProvider
}

func NewMysqlStore(dbProvider DbProvider) Store {
	return mysqlStore{dbProvider}
}

func (ms mysqlStore) Put(name string, value string) (string, error) {

	db, err := ms.dbProvider.Db()
	if err != nil {
		return "", err
	}

	result, err := db.Exec("INSERT INTO configurations (name, value) VALUES(?,?)", name, value)

	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}

	return strconv.Itoa(int(id)), err
}

func (ms mysqlStore) GetByName(name string) (Configurations, error) {
	var results Configurations

	db, err := ms.dbProvider.Db()
	if err != nil {
		return results, err
	}

	rows, err := db.Query("SELECT id, name, value FROM configurations WHERE name = ? ORDER BY id DESC", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return results, nil
		}
		return results, err
	}

	defer rows.Close()

	for rows.Next() {
		var config Configuration
		if err := rows.Scan(&config.ID, &config.Name, &config.Value); err != nil {
			return results, err
		}
		results = append(results, config)
	}

	return results, err
}

func (ms mysqlStore) GetByID(id string) (Configuration, error) {
	result := Configuration{}

	db, err := ms.dbProvider.Db()
	if err != nil {
		return result, err
	}

	err = db.QueryRow("SELECT id, name, value FROM configurations WHERE id = ?", id).Scan(&result.ID, &result.Name, &result.Value)
	if err == sql.ErrNoRows {
		return result, nil
	}

	return result, err
}

func (ms mysqlStore) Delete(name string) (int, error) {
	deletedCount := 0

	db, err := ms.dbProvider.Db()
	if err != nil {
		return deletedCount, err
	}

	result, err := db.Exec("DELETE FROM configurations WHERE name = ?", name)
	if err != nil {
		return 0, err
	}

	if result != nil {
		rows, err := result.RowsAffected()
		return int(rows), err
	}

	return 0, err
}
