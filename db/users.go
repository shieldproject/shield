package db

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	BcryptWorkFactor = 14
	LocalBackend     = `local`
)

type User struct {
	UUID    string `json:"uuid"`
	Name    string `json:"name"`
	Account string `json:"account"`
	Backend string `json:"backend"`
	SysRole string `json:"sysrole"`

	Role string `json:"role,omitempty"`

	DefaultTenant string `json:"default_tenant"`

	pwhash string
}

func (u *User) IsLocal() bool {
	return u.Backend == LocalBackend
}

func (u *User) Authenticate(password string) bool {
	/* always do this first, to avoid timing attacks */
	err := bcrypt.CompareHashAndPassword([]byte(u.pwhash), []byte(password))
	return u.IsLocal() && err == nil
}

func (u *User) SetPassword(password string) error {
	if !u.IsLocal() {
		return fmt.Errorf("%s is not a local user account", u.Account)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptWorkFactor)
	if err != nil {
		return err
	}
	u.pwhash = string(hash)
	return nil
}

type UserFilter struct {
	UUID       string
	Backend    string
	Account    string
	SysRole    string
	ExactMatch bool
	Search     string
	Limit      int
}

func (f *UserFilter) Query() (string, []interface{}) {
	wheres := []string{"u.uuid = u.uuid"}
	var args []interface{}

	if f.UUID != "" {
		if f.ExactMatch {
			wheres = append(wheres, "u.uuid = ?")
			args = append(args, f.UUID)
		} else {

			wheres = append(wheres, "u.uuid LIKE ? ESCAPE '/'")
			args = append(args, PatternPrefix(f.UUID))
		}
	}

	if f.Backend != "" {
		wheres = append(wheres, "u.backend = ?")
		args = append(args, f.Backend)
	}

	if f.Account != "" {
		if f.ExactMatch {
			wheres = append(wheres, "u.account = ?")
			args = append(args, f.Account)
		} else {
			wheres = append(wheres, "u.account LIKE ?")
			args = append(args, Pattern(f.Account))
		}
	}

	if f.SysRole != "" {
		wheres = append(wheres, "sysrole = ?")
		args = append(args, f.SysRole)
	}

	if f.Search != "" {
		wheres = append(wheres, "(u.account LIKE ? OR u.name LIKE ?)")
		args = append(args, Pattern(f.Search), Pattern(f.Search))
	}

	limit := ""
	if f.Limit > 0 {
		limit = " LIMIT ?"
		args = append(args, f.Limit)
	}

	return `
	    SELECT u.uuid, u.name, u.account, u.backend, sysrole, pwhash,
	           u.default_tenant
	      FROM users u
	     WHERE ` + strings.Join(wheres, " AND ") + `
	` + limit, args
}

func (db *DB) GetAllUsers(filter *UserFilter) ([]*User, error) {
	if filter == nil {
		filter = &UserFilter{}
	}

	l := []*User{}
	query, args := filter.Query()
	r, err := db.Query(query, args...)
	if err != nil {
		return l, err
	}
	defer r.Close()

	for r.Next() {
		u := &User{}
		if err = r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.SysRole, &u.pwhash, &u.DefaultTenant); err != nil {
			return l, err
		}
		l = append(l, u)
	}

	return l, nil
}

func (db *DB) GetUserByID(id string) (*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend, u.sysrole, u.pwhash,
	           u.default_tenant
	      FROM users u
	     WHERE u.uuid = ?`, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	u := &User{}
	if err = r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.SysRole, &u.pwhash, &u.DefaultTenant); err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) GetUser(account string, backend string) (*User, error) {
	r, err := db.Query(`
	    SELECT u.uuid, u.name, u.account, u.backend, u.sysrole, u.pwhash,
	           u.default_tenant
	      FROM users u
	     WHERE u.account = ? AND backend = ?`, account, backend)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if !r.Next() {
		return nil, nil
	}

	u := &User{}
	if err = r.Scan(&u.UUID, &u.Name, &u.Account, &u.Backend, &u.SysRole, &u.pwhash, &u.DefaultTenant); err != nil {
		return nil, err
	}
	return u, nil
}

func (db *DB) CreateUser(user *User) (*User, error) {
	if user.UUID == "" {
		user.UUID = RandomID()
	}
	err := db.Exec(`
	    INSERT INTO users (uuid, name, account, backend, sysrole, pwhash)
	               VALUES (?, ?, ?, ?, ?, ?)
	`, user.UUID, user.Name, user.Account, user.Backend, user.SysRole, user.pwhash)
	return user, err
}

func (db *DB) UpdateUser(user *User) error {
	return db.Exec(`
	   UPDATE users
	      SET name = ?, account = ?, backend = ?, sysrole = ?, pwhash = ?
	    WHERE uuid = ?
	`, user.Name, user.Account, user.Backend, user.SysRole, user.pwhash, user.UUID)
}

func (db *DB) UpdateUserSettings(user *User) error {
	return db.Exec(`
	   UPDATE users
	      SET default_tenant = ?
	    WHERE uuid = ?
	`, user.DefaultTenant, user.UUID)
}

func (db *DB) DeleteUser(user *User) error {
	return db.Exec(`
		DELETE FROM users
		      WHERE uuid = ?`, user.UUID)
}
