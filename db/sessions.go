package db

import (
	"fmt"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

type Session struct {
	UUID       uuid.UUID
	UserUUID   uuid.UUID
	LastUsedAt *timestamp.Timestamp //nil if never used
}

func (db *DB) GetSession(id uuid.UUID) (*Session, error) {
	r, err := db.Query(`SELECT uuid, user_uuid, last_used FROM sessions WHERE uuid = ?`, id.String())
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	var this, user NullUUID
	var lastUsed *int64
	if err := r.Scan(&this, &user); err != nil {
		return nil, err
	}

	var lastUsedTimestamp *timestamp.Timestamp
	if lastUsed != nil {
		tmpLastUsedTimestamp := parseEpochTime(*lastUsed)
		lastUsedTimestamp = &tmpLastUsedTimestamp
	}

	return &Session{
		UUID:       this.UUID,
		UserUUID:   user.UUID,
		LastUsedAt: lastUsedTimestamp,
	}, nil
}

//GetUserForSession returns the User struct associated with the session with the
// given UUID. If no such session exists, the User pointer returned is nil. An
// error is only thrown if a database error occurs.
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

//UpdateSessionLastUsed sets the last_used field for the session with the given
// UUID to the current time
func (db *DB) UpdateSessionLastUsed(sid uuid.UUID) (err error) {
	session, err := db.GetSession(sid)
	if err != nil {
		return
	}
	if session == nil {
		err = fmt.Errorf("No session exists with UUID: %s", sid)
	}
	now := time.Now().Unix()

	return db.Exec(`UPDATE tokens SET last_used_at = ? WHERE uuid = ?`, now, sid.String())
}
