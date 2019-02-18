package core

import (
	"net/http"
)

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
