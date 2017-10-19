package api

import "encoding/json"

//Token is a v2 authentication token
type Token struct {
	ID         string `json:"id,omitempty"`
	Token      string `json:"token,omitempty"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

//ListTokens returns a list of tokens created by the currently authenticated user
func ListTokens() (t []Token, err error) {
	uri, err := ShieldURI("/v2/auth/tokens")
	if err != nil {
		return
	}

	t = []Token{}

	err = uri.Get(&t)
	return
}

//CreateToken makes a new token for the authenticated user. The created token is
// returned. This is the only call in which you can see the sessionID of the
// created Token.
func CreateToken(name string) (t *Token, err error) {
	uri, err := ShieldURI("/v2/auth/tokens")
	if err != nil {
		return
	}

	body, err := json.Marshal(&struct {
		Name string `json:"name"`
	}{
		Name: name,
	})
	if err != nil {
		panic("Could not marshal token creation body")
	}

	t = &Token{}

	err = uri.Post(t, string(body))
	return
}

//DeleteToken makes a call to delete the token with the given identifier. Note
// that this does not take the token UUID itself, but instead the UUID which is
// the id of the token.
func DeleteToken(tokenID string) (err error) {
	uri, err := ShieldURI("/v2/auth/tokens/%s", tokenID)
	if err != nil {
		return
	}

	return uri.Delete(nil)
}
