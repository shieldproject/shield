package db_migrations

func PostgresMigrations() []string {
	migrations := []string{
		"CREATE TABLE configurations (id SERIAL NOT NULL PRIMARY KEY, name VARCHAR(255) NOT NULL, value TEXT NOT NULL)",
	}

	return migrations
}

func MysqlMigrations() []string {
	migrations := []string{
		"CREATE TABLE configurations (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, value TEXT NOT NULL)",
	}

	return migrations
}
