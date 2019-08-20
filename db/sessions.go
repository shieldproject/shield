package db

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Session struct {
	UUID           string `json:"uuid"`
	UserUUID       string `json:"user_uuid"`
	CreatedAt      int64  `json:"created_at"`
	LastSeen       int64  `json:"last_seen_at"`
	Token          string `json:"token_uuid"`
	Name           string `json:"name"`
	IP             string `json:"ip_addr"`
	UserAgent      string `json:"user_agent"`
	UserAccount    string `json:"user_account"`
	CurrentSession bool   `json:"current_session"`
}

type SessionFilter struct {
	Name       string
	ExactMatch bool
	UUID       string
	UserUUID   string
	Limit      int
	IP         string
	IsToken    bool
}

func (f *SessionFilter) Query() (string, []interface{}) {
	wheres := []string{"s.uuid = s.uuid"}
	var args []interface{}

	if f.UUID != "" {
		wheres = append(wheres, "s.uuid = ?")
		args = append(args, f.UUID)
	}

	if f.UserUUID != "" {
		wheres = append(wheres, "s.user_uuid = ?")
		args = append(args, f.UserUUID)
	}

	if f.Name != "" {
		if f.ExactMatch {
			wheres = append(wheres, "s.name = ?")
			args = append(args, Pattern(f.Name))
		} else {
			wheres = append(wheres, "s.name LIKE ?")
			args = append(args, f.Name)
		}
	}

	if f.IP != "" {
		wheres = append(wheres, "s.ip_addr = ?")
		args = append(args, f.IP)
	}

	if !f.IsToken {
		wheres = append(wheres, "s.token IS NULL")
	}

	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
	    SELECT s.uuid, s.user_uuid, s.created_at, s.last_seen, s.token, s.name, s.ip_addr, s.user_agent, u.account, u.backend
		  FROM sessions s
		  INNER JOIN users u   ON u.uuid = s.user_uuid
	     WHERE ` + strings.Join(wheres, " AND ") + `
	` + limit, args
}

func (db *DB) GetAllSessions(filter *SessionFilter) ([]*Session, error) {
	if filter == nil {
		filter = &SessionFilter{}
	}

	l := []*Session{}
	query, args := filter.Query()
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	r, err := db.query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		s := &Session{}

		var (
			backend string
			last    *int64
			token   sql.NullString
		)
		if err := r.Scan(&s.UUID, &s.UserUUID, &s.CreatedAt, &last, &token, &s.Name, &s.IP, &s.UserAgent, &s.UserAccount, &backend); err != nil {
			return nil, err
		}
		s.UserAccount = s.UserAccount + "@" + backend
		if last != nil {
			s.LastSeen = *last
		}
		if token.Valid {
			s.Token = token.String
		}

		l = append(l, s)
	}

	return l, nil
}

func (db *DB) GetSession(id string) (*Session, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.doGetSession(id)
}

//The caller must Lock the Mutex
func (db *DB) doGetSession(id string) (*Session, error) {
	r, err := db.query(`
	         SELECT s.uuid, s.user_uuid, s.created_at, s.last_seen, s.token,
	                s.name, s.ip_addr, s.user_agent, u.account, u.backend

	           FROM sessions s
	     INNER JOIN users u   ON u.uuid = s.user_uuid

	         WHERE s.uuid = ?`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session: %s", err)
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	s := &Session{}
	var (
		backend string
		last    *int64
		token   sql.NullString
	)
	if err := r.Scan(&s.UUID, &s.UserUUID, &s.CreatedAt, &last, &token,
		&s.Name, &s.IP, &s.UserAgent, &s.UserAccount, &backend); err != nil {
		return nil, err
	}
	s.UserAccount = s.UserAccount + "@" + backend
	if token.Valid {
		s.Token = token.String
	}
	if last != nil {
		s.LastSeen = *last
	}

	return s, nil
}

func (db *DB) GetUserForSession(id string) (*User, error) {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	r, err := db.query(`
	        SELECT u.uuid, u.name, u.account, u.backend, u.sysrole,
	               u.pwhash, u.default_tenant

	          FROM sessions s
	    INNER JOIN users u ON u.uuid = s.user_uuid
	         WHERE s.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	u := &User{}
	var pwhash sql.NullString
	if err := r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.SysRole,
		&pwhash, &u.DefaultTenant); err != nil {
		return nil, err
	}
	if pwhash.Valid {
		u.pwhash = pwhash.String
	}

	return u, nil
}

func (db *DB) CreateSession(session *Session) (*Session, error) {
	if session == nil {
		return nil, fmt.Errorf("cannot create an empty (user-less) session")
	}

	id := RandomID()
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	err := db.exec(`
	   INSERT INTO sessions (uuid, user_uuid, created_at, last_seen, ip_addr, user_agent)
	                 VALUES (   ?,         ?,          ?,         ?,       ?,          ?)`,
		id, session.UserUUID, time.Now().Unix(), time.Now().Unix(), stripIP(session.IP), session.UserAgent)
	if err != nil {
		return nil, err
	}

	return db.doGetSession(id)
}

func (db *DB) ClearAllSessions() error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exec(`DELETE FROM sessions`)
}

func (db *DB) ClearExpiredSessions(expiration_threshold time.Time) error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exec(`DELETE FROM sessions WHERE token IS NULL AND last_seen < ?`, expiration_threshold.Unix())
}

func (db *DB) ClearSession(id string) error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exec(`DELETE FROM sessions WHERE token IS NULL AND uuid = ?`, id)
}

func (db *DB) PokeSession(session *Session) error {
	db.exclusive.Lock()
	defer db.exclusive.Unlock()
	return db.exec(`
	   UPDATE sessions SET last_seen = ?, user_uuid = ?, ip_addr = ?, user_agent = ?
	    WHERE uuid = ?`, time.Now().Unix(), session.UserUUID, session.IP, session.UserAgent, session.UUID)
}

func stripIP(raw_ip string) string {
	return regexp.MustCompile(":[^:]+$").ReplaceAllString(raw_ip, "")
}

func (s *Session) Expired(lifetime int) bool {
	return s.Token == "" && time.Now().Unix() > s.LastSeen+(int64)(lifetime*3600)
}
