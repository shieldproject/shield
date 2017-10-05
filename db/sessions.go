package db

import (
	"fmt"

	"github.com/pborman/uuid"
)

type Session struct {
	UUID     uuid.UUID
	UserUUID uuid.UUID
}

func (db *DB) GetSession(sid interface{}) (*Session, error) {
	var id uuid.UUID
	if x, ok := sid.(uuid.UUID); ok {
		id = x
	} else {
		id = uuid.Parse(fmt.Sprintf("%s", sid))
	}

	r, err := db.Query(`SELECT uuid, user_uuid FROM sessions WHERE uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this, user NullUUID
	if err := r.Scan(&this, &user); err != nil {
		return nil, err
	}

	return &Session{
		UUID:     this.UUID,
		UserUUID: user.UUID,
	}, nil
}

func (db *DB) GetUserForSession(sid string) (*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend, u.sysrole
	      FROM sessions s INNER JOIN users u ON u.uuid = s.user_uuid
	     WHERE s.uuid = ?`, sid)
	// note: we specifically skip u.pwhash...
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
	err := db.Exec(`INSERT INTO sessions (uuid, user_uuid) VALUES (?, ?)`,
		id.String(), user.UUID.String())
	if err != nil {
		return nil, err
	}
	return db.GetSession(id)
}

func (db *DB) ClearAllSessions() error {
	return db.Exec(`DELETE FROM sessions`)
}

func (db *DB) ClearSession(sid uuid.UUID) error {
	return db.Exec(`DELETE FROM sessions WHERE uuid = ?`, sid.String())
}
