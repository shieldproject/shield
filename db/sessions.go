package db

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

type Session struct {
	UUID           uuid.UUID            `json:"uuid"`
	UserUUID       uuid.UUID            `json:"user_uuid"`
	CreatedAt      timestamp.Timestamp  `json:"created_at"`
	LastSeen       *timestamp.Timestamp `json:"last_seen_at"`
	Token          uuid.UUID            `json:"token_uuid"`
	Name           string               `json:"name"`
	IP             string               `json:"ip_addr"`
	UserAgent      string               `json:"user_agent"`
	UserAccount    string               `json:"user_account"`
	CurrentSession bool                 `json:"current_session"`
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
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		var (
			this, user, token  NullUUID
			created, last_seen *int64
			backend            string
		)
		s := &Session{}
		if err := r.Scan(&this, &user, &created, &last_seen, &token, &s.Name, &s.IP, &s.UserAgent, &s.UserAccount, &backend); err != nil {
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
		s.UserAccount = s.UserAccount + "@" + backend

		l = append(l, s)
	}

	return l, nil
}

func (db *DB) GetSession(id string) (*Session, error) {
	r, err := db.Query(`
	   SELECT s.uuid, s.user_uuid, s.created_at, s.last_seen, s.token, s.name, s.ip_addr, s.user_agent, u.account, u.backend
		 FROM sessions s
		 INNER JOIN users u   ON u.uuid = s.user_uuid
	    WHERE s.uuid = ?`, id)
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
		backend            string
	)
	s := &Session{}
	if err := r.Scan(&this, &user, &created, &last_seen, &token, &s.Name, &s.IP, &s.UserAgent, &s.UserAccount, &backend); err != nil {
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
	s.UserAccount = s.UserAccount + "@" + backend

	return s, nil
}

func (db *DB) GetUserForSession(id string) (*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend, u.sysrole, u.pwhash
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
	if err := r.Scan(&this, &u.Name, &u.Account, &u.Backend, &u.SysRole, &u.pwhash); err != nil {
		return nil, err
	}
	u.UUID = this.UUID

	return u, nil
}

func (db *DB) CreateSession(session *Session) (*Session, error) {
	if session == nil {
		return nil, fmt.Errorf("cannot create an empty (user-less) session")
	}

	id := uuid.NewRandom()
	err := db.Exec(`
	   INSERT INTO sessions (uuid, user_uuid, created_at, last_seen, ip_addr, user_agent)
	                 VALUES (?,    ?,         ?,        ?,        ?,        ?)`,
		id.String(), session.UserUUID.String(), time.Now().Unix(), time.Now().Unix(), stripIP(session.IP), session.UserAgent)
	if err != nil {
		return nil, err
	}

	return db.GetSession(id.String())
}

func (db *DB) ClearAllSessions() error {
	return db.Exec(`DELETE FROM sessions`)
}

func (db *DB) ClearSession(id string) error {
	return db.Exec(`DELETE FROM sessions WHERE token IS NULL AND uuid = ?`, id)
}

func (db *DB) PokeSession(session *Session) error {
	return db.Exec(`
		UPDATE sessions SET last_seen = ?, user_uuid = ?, ip_addr = ?, user_agent = ? 
		WHERE uuid = ?`, time.Now().Unix(), session.UUID, session.IP, session.UserAgent, session.UUID)
}

func stripIP(raw_ip string) string {
	return regexp.MustCompile(":[^:]+$").ReplaceAllString(raw_ip, "")
}
