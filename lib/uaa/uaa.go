package uaa

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type Client struct {
	ID       string
	Secret   string
	Endpoint string

	http *http.Client
}

func NewClient(c Client, verifytls bool) *Client {
	c.http = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !verifytls,
			},
		},
	}
	return &c
}

const DefaultScopes = "openid"

func (c *Client) AuthorizationURL(scope string) string {
	u, _ := url.Parse(c.Endpoint)
	u.Path += "/oauth/authorize"
	q := u.Query()

	q.Add("response_type", "code")
	q.Add("client_id", c.ID)
	q.Add("scope", scope)

	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) GetAccessToken(code string) (string, error) {
	u := url.Values{}
	u.Set("client_id", c.ID)
	u.Set("client_secret", c.Secret)
	u.Set("response_type", "token")
	u.Set("grant_type", "authorization_code")
	u.Set("code", code)

	res, err := c.http.PostForm(c.Endpoint+"/oauth/token", u)
	if err != nil {
		return "", fmt.Errorf("POST /oauth/token failed: %s", err)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("POST /oauth/token: failed to decode response body: %s", err)
	}

	data := struct {
		Token string `json:"access_token"`
	}{}
	if err = json.Unmarshal(b, &data); err != nil {
		return "", fmt.Errorf("POST /oauth/token: failed to unmarshal JSON [%s]: %s", string(b), err)
	}
	if data.Token == "" {
		return "", fmt.Errorf("POST /oauth/token: no access token found in response body [%s]", string(b))
	}

	return data.Token, nil
}

func (c *Client) Lookup(token string) (string, string, []string, error) {
	req, err := http.NewRequest("GET", c.Endpoint+"/userinfo", nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /userinfo failed to create request: %s", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /userinfo failed: %s", err)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /userinfo: failed to decode response body: %s", err)
	}

	u := struct {
		ID       string `json:"user_id"`
		Username string `json:"user_name"`
		Name     string `json:"name"`
	}{}
	if err = json.Unmarshal(b, &u); err != nil {
		return "", "", nil, fmt.Errorf("GET /userinfo: failed to unmarshal JSON [%s]: %s", string(b), err)
	}
	if u.ID == "" {
		return "", "", nil, fmt.Errorf("GET /userinfo: no user_id found in response body [%s]", string(b))
	}
	if u.Username == "" {
		return "", "", nil, fmt.Errorf("GET /userinfo: no user_name found in response body [%s]", string(b))
	}

	if u.Name == " " {
		u.Name = u.Username
	}

	req, err = http.NewRequest("GET", c.Endpoint+"/Users/"+u.ID, nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /Users/%s failed to create request: %s", u.ID, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err = c.http.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /Users/%s failed: %s", u.ID, err)
	}

	b, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("GET /Users/%s: failed to decode response body: %s", u.ID, err)
	}

	info := struct {
		Groups []struct {
			Type    string `json:"type"`
			Display string `json:"display"`
			Value   string `json:"value"`
		} `json:"groups"`
	}{}
	if err = json.Unmarshal(b, &info); err != nil {
		return "", "", nil, fmt.Errorf("GET /Users/%s failed to unmarshal JSON [%s]: %s", u.ID, string(b), err)
	}

	rights := make([]string, 0)
	for _, right := range info.Groups {
		if right.Type == "DIRECT" {
			rights = append(rights, right.Display)
			rights = append(rights, right.Value) /* in case someone likes uuids */
		}
	}

	return u.Name, u.Username, rights, nil
}
