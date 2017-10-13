package api

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

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

	r, err := makeRequest(req)
	if err != nil {
		return
	}

	if r.StatusCode == 200 {
		sessionID = r.Header.Get("X-Shield-Session")
		var body []byte
		body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}

		user = &AuthIDOutput{}
		err = json.Unmarshal(body, user)
	} else {
		err = getAPIError(r)
	}

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
		Sysrole string `json:"admin"`
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
