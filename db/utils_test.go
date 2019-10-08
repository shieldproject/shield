package db

type q struct {
	query string
	args  []interface{}
}

func database(queries ...q) (*DB, error) {
	db, err := Connect(":memory:")
	if err != nil {
		return nil, err
	}

	if _, err := db.Setup(0); err != nil {
		db.Disconnect()
		return nil, err
	}

	for _, q := range queries {
		if err := db.Exec(q.query, q.args...); err != nil {
			db.Disconnect()
			return nil, err
		}
	}

	return db, nil
}

func SQL(query string, args ...interface{}) q {
	return q{query: query, args: args}
}
