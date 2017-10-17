package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pborman/uuid"
)

func Unlock(master string) error {
	uri, err := ShieldURI("/v2/unlock")
	if err != nil {
		return err
	}

	creds := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}
	contentJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	respMap := make(map[string]string)
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		return err
	}

	return nil
}

func Init(master string) error {
	uri, err := ShieldURI("/v2/init")
	if err != nil {
		return err
	}

	respMap := make(map[string]string)
	creds := struct {
		Master string `json:"master"`
	}{
		Master: master,
	}
	contentJSON, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err := uri.Post(&respMap, string(contentJSON)); err != nil {
		return err
	}

	return nil
}

func Rekey(current, proposed string) error {
	uri, err := ShieldURI("/v2/rekey")
	if err != nil {
		return err
	}

	respMap := make(map[string]string)
	creds := struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}{
		Current: current,
		New:     proposed,
	}
	b, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	if err := uri.Post(&respMap, string(b)); err != nil {
		return err
	}

	return nil
}

//Login hits the /v2/auth/login endpoint with the given username and password
// strings in an attempt to create an authenticated session. Returns a sessionID
// if successful and an error otherwise. If the error is due to bad credentials,
// the error returned will be of type ErrUnauthorized.
func Login(username, password string) (sessionID string, user *AuthIDOutput, err error) {
	uri, err := ShieldURI("/v2/auth/login")
	if err != nil {
		return
	}

	j, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	})

	if err != nil {
		panic("Could not marshal auth struct that we JUST made")
	}

	req, err := http.NewRequest("POST", uri.String(), bytes.NewReader(j))
	if err != nil {
		return
	}

	user = &AuthIDOutput{}
	var header http.Header
	header, err = uri.RequestWithHeaders(user, req)
	if err != nil {
		return
	}

	sessionID = header.Get("X-Shield-Session")
	return
}

//Logout contacts the /v2/auth/logout endpoint in an attempt to invalidate the
// current backend's session
func Logout() error {
	uri, err := ShieldURI("/v2/auth/logout")
	if err != nil {
		return err
	}

	return uri.Get(nil)
}

//LogoutSession contacts the /v2/auth/logout endpoint in an attempt to invalidate the
// requested session ID
func LogoutSession(sessionID string) error {
	uri, err := ShieldURI("/v2/auth/logout")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-Shield-Session", sessionID)
	r, err := makeRequest(req)
	if err != nil {
		return err
	}

	if r.StatusCode != 200 {
		err = getAPIError(r)
	}

	return err
}

//AuthIDOutput contains all the information from a call to /v2/auth/id
type AuthIDOutput struct {
	User struct {
		Name    string `json:"name"`
		Account string `json:"account"`
		Backend string `json:"backend"`
		Sysrole string `json:"sysrole"`
	} `json:"user"`
	Tenants []struct {
		UUID uuid.UUID `json:"uuid"`
		Name string    `json:"name"`
		Role string    `json:"role"`
	} `json:"tenants"`
}

//AuthID hits the /v2/auth/id endpoint to retrieve data about the current user
func AuthID() (out *AuthIDOutput, err error) {
	var uri *URL
	uri, err = ShieldURI("/v2/auth/id")
	if err != nil {
		return
	}

	out = &AuthIDOutput{}
	err = uri.Get(out)
	return
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
	//AuthV2Token represents a v2 backend in which the desired auth provider is a
	// token provider
	AuthV2Token
)

//FetchAuthType returns the auth type of the SHIELD backend auth provider that
//you request. If the backend is v1, the provided providerID is ignored, and
//the status endpoint is hit without authentication, and the headers are read
//to determine the auth type. If the backend is v2, if the providerID is empty,
//then AuthV2Local is returned, and otherwise the provider is looked up in the
//v2 API for its auth type. If there is no provider with the given identifier in
//the SHIELD backend, ErrNotFound is returned.
//Returns AuthProvider if requested type of auth has a provider.
func FetchAuthType(providerID string) (authType AuthType, provider *AuthProvider, err error) {
	if curBackend.APIVersion == 1 {
		authType, err = fetchV1AuthType()
		return
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

func fetchV2AuthType(providerID string) (authType AuthType, provider *AuthProvider, err error) {
	if providerID == "" {
		return AuthV2Local, nil, nil
	}

	provider, err = GetProvider(providerID)
	if err != nil {
		return
	}

	switch provider.Type {
	case "token":
		authType = AuthV2Token
	default:
		err = fmt.Errorf("Unknown auth type `%s'", provider.Type)
	}
	return
}
