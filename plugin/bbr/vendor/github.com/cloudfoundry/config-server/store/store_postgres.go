package store

import (
	"database/sql"
	"strconv"
)

type postgresStore struct {
	dbProvider DbProvider
}

func NewPostgresStore(dbProvider DbProvider) Store {
	return postgresStore{dbProvider}
}

func (ps postgresStore) Put(name string, value string) (string, error) {

	db, err := ps.dbProvider.Db()
	if err != nil {
		return "", err
	}

	var id int
	err = db.QueryRow("INSERT INTO configurations (name, value) VALUES($1, $2) RETURNING id", name, value).Scan(&id)

	if err != nil {
		return "", err
	}

	return strconv.Itoa(int(id)), err
}

func (ps postgresStore) GetByName(name string) (Configurations, error) {
	var results Configurations

	db, err := ps.dbProvider.Db()
	if err != nil {
		return results, err
	}

	rows, err := db.Query("SELECT id, name, value FROM configurations WHERE name = $1 ORDER BY id DESC", name)
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

func (ps postgresStore) GetByID(id string) (Configuration, error) {
	result := Configuration{}

	_, err := strconv.Atoi(id)
	if err != nil {
		return result, nil
	}

	db, err := ps.dbProvider.Db()
	if err != nil {
		return result, err
	}

	err = db.QueryRow("SELECT id, name, value FROM configurations WHERE id = $1", id).Scan(&result.ID, &result.Name, &result.Value)
	if err == sql.ErrNoRows {
		return result, nil
	}

	return result, err
}

func (ps postgresStore) Delete(name string) (int, error) {

	db, err := ps.dbProvider.Db()
	if err != nil {
		return 0, err
	}

	result, err := db.Exec("DELETE FROM configurations WHERE name = $1", name)
	if err != nil {
		return 0, err
	}

	if result != nil {
		rows, err := result.RowsAffected()
		return int(rows), err
	}

	return 0, err
}
