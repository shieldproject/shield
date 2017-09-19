package core

import (
	"fmt"
	"net/http"

	"github.com/starkandwayne/shield/db"
)

// create a session for the user and returns a pointer to said session
func (core *Core) createSession(user *db.User) (*db.Session, error) {

	username := user.Name
	session, err := core.DB.CreateSessionFor(user)
	if err != nil {
		return nil, fmt.Errorf("failed to create session for local user '%s' (database error: %s)", username, err)
	}
	if session == nil {
		return nil, fmt.Errorf("failed to create session for local user '%s' (session was nil)", username)
	}
	return session, nil
}

func SessionCookie(value string, valid bool) *http.Cookie {
	c := &http.Cookie{
		Name:  "shield7",
		Value: value,
		Path:  "/",
	}
	if !valid {
		c.MaxAge = 0
	}
	return c
}
