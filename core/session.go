package core

import (
	"fmt"
	"net/http"

	"github.com/starkandwayne/shield/db"
)

// create a session for the user and returns a pointer to said session
func (core *Core) createSession(session *db.Session) (*db.Session, error) {

	new_session, err := core.DB.CreateSession(session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session for local user '%s' (database error: %s)", session.UserUUID, err)
	}
	if new_session == nil {
		return nil, fmt.Errorf("failed to create session for local user '%s' (session was nil)", session.UserUUID)
	}
	return new_session, nil
}

func SessionCookie(value string, valid bool) *http.Cookie {
	c := &http.Cookie{
		Name:  SessionCookieName,
		Value: value,
		Path:  "/",
	}
	if !valid {
		c.MaxAge = 0
	}
	return c
}
