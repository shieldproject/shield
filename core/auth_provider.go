package core

import (
	"net/http"
)

type AuthProvider interface {
	DisplayName() string

	Configure(map[interface{}]interface{}) error
	Initiate(http.ResponseWriter, *http.Request)
	HandleRedirect(http.ResponseWriter, *http.Request)
}
