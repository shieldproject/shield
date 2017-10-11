package api

import (
	"fmt"
	"net/http"
	"strings"
)

type Status struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func GetStatus() (Status, error) {
	uri, err := ShieldURI("/v1/status")
	if err != nil {
		return Status{}, err
	}

	data := Status{}
	return data, uri.Get(&data)
}

type JobsStatus map[string]JobHealth

type JobHealth struct {
	Name    string `json:"name"`
	LastRun int64  `json:"last_run"`
	NextRun int64  `json:"next_run"`
	Paused  bool   `json:"paused"`
	Status  string `json:"status"`
}

func GetJobsStatus() (JobsStatus, error) {
	uri, err := ShieldURI("/v1/status/jobs")
	if err != nil {
		return JobsStatus{}, err
	}

	var data JobsStatus
	return data, uri.Get(&data)
}

//Ping hits the /v1/ping endpoint and returns the APIVersion if present.
// Returns 1 if api_version is not in the response (aka v1 APIs)
func Ping() (apiVersion int, err error) {
	uri, err := ShieldURI("/v1/ping")
	if err != nil {
		return 0, err
	}

	pingBody := struct {
		APIVersion int `json:"api_version"`
	}{
		APIVersion: 1,
	}

	err = uri.Get(&pingBody)
	return pingBody.APIVersion, err
}

//AuthType is an enumeration type that represents different types of auth that
// which a provider may be
type AuthType int

const (
	//AuthUnknown is the zero value of AuthType
	AuthUnknown AuthType = iota
	//AuthV1Basic represents a v1 backend providing basic auth
	AuthV1Basic
	//AuthV1OAuth represents a v1 backend providing an OAuth authentication backend
	AuthV1OAuth
	//AuthV2Local represents a v2 backend, and you're targeting a local authentication
	//user
	AuthV2Local
)

//FetchAuthType returns the auth type of the SHIELD backend auth provider that
//you request. If the backend is v1, the provided providerID is ignored, and
//the status endpoint is hit without authentication, and the headers are read
//to determine the auth type. If the backend is v2, if the providerID is empty,
//then AuthV2Local is returned, and otherwise the provider is looked up in the
//v2 API for its auth type. An error is returned if an HTTP error occurs
//
//That last clause isn't true at the moment, as we haven't implemented things
//for v2 OAuth providers yet, but it will be. Right now, providerID is simply
//ignored.
func FetchAuthType(providerID string) (authType AuthType, err error) {
	if curBackend.APIVersion == 1 {
		return fetchV1AuthType()
	}
	return fetchV2AuthType(providerID)
}

func fetchV1AuthType() (authType AuthType, err error) {
	var uri *URL
	uri, err = ShieldURI("/v1/status")
	if err != nil {
		return
	}

	var r *http.Response
	r, err = curClient.Get(uri.String())
	if err != nil {
		return
	}

	auth := strings.Split(r.Header.Get("www-authenticate"), " ")
	var a string
	if len(auth) > 0 {
		a = strings.ToLower(auth[0])
	}

	switch a {
	case "basic":
		authType = AuthV1Basic
	case "bearer":
		authType = AuthV1OAuth
	default:
		err = fmt.Errorf("Unable to determine auth type from v1 backend")
	}

	return
}

func fetchV2AuthType(providerID string) (authType AuthType, err error) {
	return AuthV2Local, nil
}
