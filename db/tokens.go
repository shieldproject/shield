package db

import (
	"fmt"
	"strings"
	"time"
)

type AuthToken struct {
	UUID      string `json:"uuid"`
	Session   string `json:"session,omitempty"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	LastSeen  *int64 `json:"last_seen"`
}

type AuthTokenFilter struct {
	UUID string
	User *User
	Name string
}

func (t AuthTokenFilter) Query() (string, []interface{}) {
	wheres := []string{"s.token IS NOT NULL"}
	var args []interface{}
	if t.UUID != "" {
		wheres = append(wheres, "s.token = ?")
		args = append(args, t.UUID)
	}
	if t.Name != "" {
		wheres = append(wheres, "s.name = ?")
		args = append(args, t.Name)
	}
	if t.User != nil {
		wheres = append(wheres, "u.uuid = ?")
		args = append(args, t.User.UUID)
	}

	return `
		SELECT s.token, s.uuid, s.created_at, s.last_seen, s.name

		FROM sessions s INNER JOIN users u ON s.user_uuid = u.uuid

		WHERE ` + strings.Join(wheres, " AND ") + `
		ORDER BY s.name, s.uuid`, args
}

func (db *DB) GetAllAuthTokens(filter *AuthTokenFilter) ([]*AuthToken, error) {
	if filter == nil {
		filter = &AuthTokenFilter{}
	}

	l := []*AuthToken{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		t := &AuthToken{}
		if err = r.Scan(&t.UUID, &t.Session, &t.CreatedAt, &t.LastSeen, &t.Name); err != nil {
			return l, err
		}
		l = append(l, t)
	}

	return l, nil
}

func (db *DB) GetAuthToken(id string) (*AuthToken, error) {
	r, err := db.GetAllAuthTokens(&AuthTokenFilter{UUID: id})
	if err != nil {
		return nil, err
	}
	if len(r) == 0 {
		return nil, nil
	}
	return r[0], nil
}

func (db *DB) GenerateAuthToken(name string, user *User) (*AuthToken, string, error) {
	if user == nil {
		return nil, "", fmt.Errorf("cannot generate a token without a user")
	}

	id := RandomID()
	token := RandomID()
	err := db.Exec(`
	   INSERT INTO sessions (uuid, user_uuid, created_at, token, name)
	                 VALUES (?,    ?,         ?,          ?,     ?)`,
		id, user.UUID, time.Now().Unix(), token, name)
	if err != nil {
		return nil, "", err
	}

	t, err := db.GetAuthToken(token)
	return t, token, err
}

func (db *DB) DeleteAuthToken(id string, user *User) error {
	return db.Exec(`DELETE FROM sessions WHERE token = ? AND user_uuid = ?`, id, user.UUID)
}
