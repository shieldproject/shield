package db

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

type Session struct {
	UUID      uuid.UUID
	UserUUID  uuid.UUID
	CreatedAt timestamp.Timestamp
	LastSeen  *timestamp.Timestamp
	Token     uuid.UUID
	Name      string
}

func (db *DB) GetSession(id string) (*Session, error) {
	r, err := db.Query(`
	   SELECT uuid, user_uuid, created_at, last_seen, token, name
	     FROM sessions
	    WHERE uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var (
		this, user, token  NullUUID
		created, last_seen *int64
	)
	s := &Session{}
	if err := r.Scan(&this, &user, &created, &last_seen, &token, &s.Name); err != nil {
		return nil, err
	}

	if last_seen != nil {
		ts := parseEpochTime(*last_seen)
		s.LastSeen = &ts
	}
	s.UUID = this.UUID
	s.Token = token.UUID
	s.UserUUID = user.UUID
	s.CreatedAt = parseEpochTime(*created)

	return s, nil
}

func (db *DB) GetUserForSession(id string) (*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend, u.sysrole
	      FROM sessions s INNER JOIN users u ON u.uuid = s.user_uuid
	     WHERE s.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this NullUUID
	u := &User{}
	if err := r.Scan(&this, &u.Name, &u.Account, &u.Backend, &u.SysRole); err != nil {
		return nil, err
	}
	u.UUID = this.UUID

	return u, nil
}

func (db *DB) CreateSessionFor(user *User) (*Session, error) {
	if user == nil {
		return nil, fmt.Errorf("cannot create an empty (user-less) session.")
	}

	id := uuid.NewRandom()
	err := db.Exec(`
	   INSERT INTO sessions (uuid, user_uuid, created_at)
	                 VALUES (?,    ?,         ?)`,
		id.String(), user.UUID.String(), time.Now().Unix())
	if err != nil {
		return nil, err
	}

	return db.GetSession(id.String())
}

func (db *DB) ClearAllSessions() error {
	return db.Exec(`DELETE FROM sessions`)
}

func (db *DB) ClearSession(id string) error {
	return db.Exec(`DELETE FROM sessions WHERE token_uuid IS NULL AND uuid = ?`, id)
}

func (db *DB) PokeSession(id string) error {
	return db.Exec(`UPDATE sessions SET last_seen = ? WHERE uuid = ?`, time.Now().Unix(), id)
}
