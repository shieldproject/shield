package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

//Token is an access token, giving access on behalf of a user
type Token struct {
	UUID      uuid.UUID
	Name      string
	CreatedAt timestamp.Timestamp
	//Session stuff
	SessionUUID uuid.UUID
	LastUsedAt  *timestamp.Timestamp //nil if never used
	UserUUID    uuid.UUID
}

//Create makes this Token in the database. The receiver Token will be updated
// as necessary for entry into the database. Only the name and user uuid are
// used for creation
func (t *Token) Create(db *DB) (err error) {
	t, err = db.CreateToken(t.Name, t.UserUUID)
	return
}

//Delete removes the token with this Token's UUID from the database
func (t *Token) Delete(db *DB) (err error) {
	return db.DeleteToken(t.UUID)
}

//UpdateLastUsed bumps the last used in the database to the current time and
// updates the given token.
func (t *Token) UpdateLastUsed(db *DB) (err error) {
	t, err = db.UpdateTokenLastUsed(t.UUID)
	return
}

//TokenFilter is a set of query limiters to use in a where clause
type TokenFilter struct {
	UUID        *uuid.UUID
	Name        *string
	SessionUUID *uuid.UUID
	UserUUID    *uuid.UUID
}

//Get runs a List and returns the first item in the list or nil if the list is
// empty. No err is returned if the item does not exist.
func (t TokenFilter) Get(db *DB) (ret *Token, err error) {
	r, err := db.Query(t.Query())
	if err != nil {
		return
	}
	defer r.Close()

	if !r.Next() {
		return
	}

	return parseToken(r)
}

//List runs a select on the tokens table with the given contraints. No
//constraints will select all.
func (t TokenFilter) List(db *DB) (ret []*Token, err error) {
	r, err := db.Query(t.Query())
	if err != nil {
		return
	}
	defer r.Close()

	for r.Next() {
		var thisToken *Token
		thisToken, err = parseToken(r)
		if err != nil {
			break
		}

		ret = append(ret, thisToken)
	}

	return
}

//Query returns the query string which selects Tokens from the table, adhering
//to the parameters given in the TokenFilter
func (t TokenFilter) Query() (query string, args []interface{}) {
	wheres := []string{"t.uuid = t.uuid"}
	if t.UUID != nil {
		wheres = append(wheres, "t.uuid = ?")
		args = append(args, *t.UUID)
	}
	if t.Name != nil {
		wheres = append(wheres, "t.name = ?")
		args = append(args, *t.Name)
	}
	if t.SessionUUID != nil {
		wheres = append(wheres, "t.session_uuid = ?")
		args = append(args, *t.SessionUUID)
	}
	if t.UserUUID != nil {
		wheres = append(wheres, `s.user_uuid = ?`)
		args = append(args, *t.UserUUID)
	}

	query = fmt.Sprintf(`
	SELECT DISTINCT t.uuid, t.session_uuid, t.name, t.created_at, s.last_used_at, s.uuid
		FROM tokens t
		INNER JOIN sessions s ON t.session_uuid = s.uuid
		WHERE %s
		ORDER BY t.name, t.uuid`, strings.Join(wheres, " AND "))
	return
}

func parseToken(r *sql.Rows) (ret *Token, err error) {
	var sessionUUID, userUUID NullUUID
	var name sql.NullString
	var created, lastUsed *int64
	if err = r.Scan(&sessionUUID, &name, &created, &lastUsed, &userUUID); err != nil {
		return
	}

	ret = &Token{
		SessionUUID: sessionUUID.UUID,
		UserUUID:    userUUID.UUID,
		Name:        name.String,
	}

	if created != nil {
		ret.CreatedAt = parseEpochTime(*created)
	}

	if lastUsed != nil {
		lastUsedTimestamp := parseEpochTime(*lastUsed)
		ret.LastUsedAt = &lastUsedTimestamp
	}

	return
}

//CreateToken creates a new Session entry in the database, and then a new Token
// entry associated with it.
func (db *DB) CreateToken(name string, userid uuid.UUID) (_ *Token, err error) {
	insert := &Token{}

	testtoken, err := TokenFilter{Name: &name, UserUUID: &userid}.Get(db)
	if err != nil {
		return
	}
	if testtoken != nil {
		err = fmt.Errorf("Refusing to create token with preexisting name and userid combination")
		return
	}

	insert.UUID = uuid.NewRandom()
	insert.SessionUUID = uuid.NewRandom()
	insert.CreatedAt = parseEpochTime(time.Now().Unix())

	tx, err := db.connection.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err == sql.ErrTxDone {
				err = nil
			}
		}
	}()

	//Make a new session with the given UserUUID
	err = db.Exec(`INSERT INTO sessions (uuid, user_uuid) VALUES (?, ?)`,
		insert.SessionUUID.String(), insert.UserUUID.String())
	if err != nil {
		return
	}

	err = db.Exec(`INSERT INTO tokens (uuid, session_uuid, label, created_at) VALUES (?, ?, ?)`,
		insert.UUID, insert.SessionUUID, insert.Name, insert.CreatedAt.Time().Unix())
	if err != nil {
		return
	}

	return TokenFilter{UUID: &insert.UUID}.Get(db)
}

//UpdateTokenLastUsed takes a token UUID and updates the last used time of that
//token to the current time. An error is returned if no such token exists
func (db *DB) UpdateTokenLastUsed(id uuid.UUID) (ret *Token, err error) {
	token, err := TokenFilter{UUID: &id}.Get(db)
	if err != nil {
		return
	}
	if token != nil {
		err = fmt.Errorf("No token exists with UUID: %s", id)
		return
	}

	err = db.UpdateSessionLastUsed(token.SessionUUID)
	if err != nil {
		return
	}

	return TokenFilter{UUID: &id}.Get(db)
}

//DeleteToken deletes the token referencing the session with the given uuid, and
// then deletes the session itself. If the token doesn't exist, then no error is
// returned
func (db *DB) DeleteToken(id uuid.UUID) (err error) {
	token, err := TokenFilter{UUID: &id}.Get(db)
	if err != nil || token == nil {
		return
	}

	tx, err := db.connection.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err == sql.ErrTxDone {
				err = nil
			}
		}
	}()

	err = db.Exec(`DELETE FROM tokens WHERE uuid = ?`, id.String())
	if err != nil {
		return
	}

	err = db.Exec(`DELETE FROM sessions WHERE uuid = ?`, token.SessionUUID.String())
	return
}
