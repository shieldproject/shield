package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

type AuthToken struct {
	UUID      uuid.UUID            `json:"uuid"`
	Session   uuid.UUID            `json:"session"`
	Name      string               `json:"name"`
	CreatedAt timestamp.Timestamp  `json:"created_at"`
	LastSeen  *timestamp.Timestamp `json:"last_seen"`
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
		args = append(args, t.User.UUID.String())
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
		var (
			this, session      NullUUID
			created, last_seen *int64
		)
		if err = r.Scan(&this, &session, &created, &last_seen, &t.Name); err != nil {
			return l, err
		}
		t.UUID = this.UUID
		t.Session = session.UUID

		if created != nil {
			t.CreatedAt = parseEpochTime(*created)
		}
		if last_seen != nil {
			ts := parseEpochTime(*last_seen)
			t.LastSeen = &ts
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

	id := uuid.NewRandom()
	token := uuid.NewRandom()
	err := db.Exec(`
	   INSERT INTO sessions (uuid, user_uuid, created_at, token, name)
	                 VALUES (?,    ?,         ?,          ?,     ?)`,
		id.String(), user.UUID.String(), time.Now().Unix(), token.String(), name)
	if err != nil {
		return nil, "", err
	}

	t, err := db.GetAuthToken(token.String())
	return t, token.String(), err
}

func (db *DB) DeleteAuthToken(id string) error {
	return db.Exec(`DELETE FROM sessions WHERE token = ?`, id)
}
