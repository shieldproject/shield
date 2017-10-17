package api

import (
	"fmt"
	"net/http"
)

//AuthProvider contains all the info about an auth provider (usually an oauth
// provider) in the targeted SHIELD backend
type AuthProvider struct {
	Name       string                 `json:"name"`
	Identifier string                 `json:"identifier"`
	Type       string                 `json:"type"`
	WebEntry   string                 `json:"web_entry"`
	CLIEntry   string                 `json:"cli_entry"`
	Redirect   string                 `json:"redirect"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

//GetProvider returns the auth provider with the given ID. ErrNotFound is
//returned if the provider requested was not found in the backend.
func GetProvider(provider string) (ret *AuthProvider, err error) {
	var uri *URL
	uri, err = ShieldURI(fmt.Sprintf("/v2/auth/providers/%s", provider))
	if err != nil {
		return
	}

	err = uri.Get(&ret)
	return
}

//ListProviders returns the list of auth providers registered with the SHIELD
// backend.
func ListProviders() (rets []AuthProvider, err error) {
	var uri *URL
	uri, err = ShieldURI("/v2/auth/providers")
	if err != nil {
		return
	}

	err = uri.Get(&rets)
	return
}

//TokenAuth attempts to perform the token auth flow on the given provider. If
// this provider is not a token provider, an error is returned. If the token
// provided is correct and the flow succeeds, a session id is returned.
func (p *AuthProvider) TokenAuth(token string) (sessionID string, user *AuthIDOutput, err error) {
	if p.Type != "token" {
		err = fmt.Errorf("Can't do token auth for non-token provider")
		return
	}

	var uri *URL
	uri, err = ShieldURI(p.CLIEntry)
	if err != nil {
		return
	}

	var req *http.Request
	req, err = http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return
	}

	req.Header.Set("X-Shield-Token", token)
	user = &AuthIDOutput{}

	var header http.Header
	header, err = uri.RequestWithHeaders(user, req)
	if err != nil {
		return
	}

	sessionID = header.Get("X-Shield-Session")
	return
}
