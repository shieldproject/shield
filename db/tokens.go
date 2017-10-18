package db

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"github.com/starkandwayne/goutils/timestamp"
)

//Token is an access token, giving access on behalf of a user
type Token struct {
	SessionUUID uuid.UUID
	Label       string
	CreatedAt   timestamp.Timestamp
	UserUUID    uuid.UUID
}

//TokenFilter is a set of query limiters to use in a where clause
type TokenFilter struct {
	SessionUUID string
	UserUUID    string
}

func (t TokenFilter) doSelect(db *DB) (ret []*Token, err error) {
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
	wheres := []string{"t.session_uuid = t.session_uuid"}
	if t.SessionUUID != "" {
		wheres = append(wheres, "t.session_uuid = ?")
		args = append(args, t.SessionUUID)
	}
	if t.UserUUID != "" {
		wheres = append(wheres, `s.user_uuid = ?`)
		args = append(args, t.UserUUID)
	}

	query = fmt.Sprintf(`
	SELECT DISTINCT t.session_uuid, t.label, t.created_at, s.uuid
		FROM tokens t
		INNER JOIN sessions s ON t.session_uuid = s.uuid
		WHERE %s
		ORDER BY t.session_uuid`, strings.Join(wheres, " AND "))
	return
}

func parseToken(r *sql.Rows) (ret *Token, err error) {
	var sessionUUID, userUUID NullUUID
	var label sql.NullString
	var created *int64
	if err = r.Scan(&sessionUUID, &label, &created, &userUUID); err != nil {
		return
	}

	ret = &Token{}

	if sessionUUID.Valid {
		ret.SessionUUID = sessionUUID.UUID
	}

	if userUUID.Valid {
		ret.UserUUID = userUUID.UUID
	}

	if label.Valid {
		ret.Label = label.String
	}

	if created != nil {
		ret.CreatedAt = parseEpochTime(*created)
	}

	return
}

//GetToken retrieves the token referencing the given session UUID from the
//database
func (db *DB) GetToken(sid uuid.UUID) (ret *Token, err error) {
	tokens, err := TokenFilter{SessionUUID: sid.String()}.doSelect(db)
	if err != nil {
		return
	}

	if len(tokens) > 0 {
		ret = tokens[0]
	}

	return
}

//GetAllTokens returns the list of all Tokens in the database
func (db *DB) GetAllTokens() (ret []*Token, err error) {
	return TokenFilter{}.doSelect(db)
}

//GetTokensForUser returns the list of Tokens associated with the given user
//UUID
func (db *DB) GetTokensForUser(user uuid.UUID) (ret []*Token, err error) {
	return TokenFilter{UserUUID: user.String()}.doSelect(db)
}

//CreateToken creates a new Session entry in the database, and then creates a
//new Token entry associated with it. The Token's UserUUID must already exist
//in the users table. If the Token's SessionID value is an empty UUID, then
//a new Session is created, and the SessionID value is populated if successful.
//Otherwise, only a token entry is created, referencing that UUID (in which
//case, UserUUID is ignored).
func (db *DB) CreateToken(t *Token) (ret *Token, err error) {
	if reflect.DeepEqual(t.CreatedAt, timestamp.Timestamp{}) {
		t.CreatedAt = parseEpochTime(time.Now().Unix())
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

	if reflect.DeepEqual(t.SessionUUID, uuid.UUID{}) {
		//Make a new session with the given UserUUID
		t.SessionUUID = uuid.NewRandom()
		err = db.Exec(`INSERT INTO sessions (uuid, user_uuid) VALUES (?, ?)`,
			t.SessionUUID.String(), t.UserUUID.String())
		if err != nil {
			return
		}
	}

	err = db.Exec(`INSERT INTO tokens (session_uuid, label, created_at) VALUES (?, ?, ?)`,
		t.SessionUUID, t.Label, t.CreatedAt)

	if err != nil {
		return
	}

	err = tx.Commit()
	if err != nil {
		return
	}

	return db.GetToken(t.SessionUUID)
}

//DeleteToken deletes the token referencing the session with the given uuid, and
// then deletes the session itself
func (db *DB) DeleteToken(sid uuid.UUID) (err error) {
	tx, err := db.connection.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	err = db.Exec(`DELETE FROM tokens WHERE session_uuid = ?`, sid.String())
	if err != nil {
		return
	}

	err = db.Exec(`DELETE FROM sessions WHERE uuid = ?`, sid.String())
	return
}
