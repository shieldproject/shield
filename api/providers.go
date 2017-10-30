package api

import (
	"fmt"
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
