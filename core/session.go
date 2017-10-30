package core

import (
	"fmt"
	"net/http"
	"time"

	"github.com/jhunt/go-log"

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

func (core *Core) purgeExpiredSessions() {
	log.Debugf("purging expired user sessions from the database")
	err := core.DB.ClearExpiredSessions(time.Now().Add(0 - (time.Duration(core.sessionTimeout) * time.Hour)))
	if err != nil {
		log.Errorf("Failed to purge expired sessions from the database: %s", err.Error())
	}
}
