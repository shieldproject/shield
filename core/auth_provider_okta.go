package core

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	// "github.com/jhunt/go-log"
	"github.com/shieldproject/shield/db"
	"github.com/shieldproject/shield/route"
	"github.com/shieldproject/shield/util"

	"github.com/pborman/uuid"
	"github.com/thanhpk/randstr"

	verifier "github.com/okta/okta-jwt-verifier-golang"
)

type OktaAuthProvider struct {
	AuthProviderBase

	ClientID            string `json:"client_id"`
	ClientSecret        string `json:"client_secret"`
	AuthorizationServer string `json:"authorization_server"`
	OktaDomain          string `json:"okta_domain"`
	DeploymentURI       string `json:"deployment_uri"`
	TokenVerification   bool   `json:"token_verification"`

	OktaEnterprise bool `json:"okta_enterprise"`
	Mapping        []struct {
		Okta   string `json:"okta"`
		Tenant string `json:"tenant"`
		Rights []struct {
			Group string `json:"group"`
			Role  string `json:"role"`
		} `json:"rights"`
	} `json:"mapping"`
}

type Exchange struct {
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
	Scope            string `json:"scope,omitempty"`
	IdToken          string `json:"id_token,omitempty"`
}

func (p *OktaAuthProvider) Configure(raw map[interface{}]interface{}) error {
	b, err := json.Marshal(util.StringifyKeys(raw))
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, p)
	if err != nil {
		return err
	}

	if p.ClientID == "" {
		return fmt.Errorf("invalid configuration for Okta OAuth Provider: missing `client_id' value")
	}

	if p.ClientSecret == "" {
		return fmt.Errorf("invalid configuration for Okta OAuth Provider: missing `client_secret' value")
	}

	if p.OktaDomain == "" {
		// ex: p.OktaDomain = "https://dev-xyz.okta.com/"
		// p.OktaIssuer = "https://dev-xyz.okta.com/oauth2/default"
		return fmt.Errorf("invalid configuration for Okta OAuth Provider: missing `okta_domain' value")
	}

	if p.DeploymentURI == "" {
		// ex: p.DeploymentURI = "https://shield.starkandwayne.com"
		return fmt.Errorf("invalid configuration for Okta OAuth Provider: missing `deployment_uri' value")
	}

	if p.AuthorizationServer == "" {
		// ex: p.AuthorizationServer = "default"
		return fmt.Errorf("invalid configuration for Okta OAuth Provider: missing `authorization_server' value")
	}

	p.OktaDomain = strings.TrimSuffix(p.OktaDomain, "/")
	p.DeploymentURI = strings.TrimSuffix(p.DeploymentURI, "/")

	p.properties = util.StringifyKeys(raw).(map[string]interface{})

	return nil
}

func (p *OktaAuthProvider) WireUpTo(c *Core) {
	p.core = c
}

func (p *OktaAuthProvider) ReferencedTenants() []string {
	ll := make([]string, 0)
	for _, m := range p.Mapping {
		ll = append(ll, m.Tenant)
	}
	return ll
}

func (p *OktaAuthProvider) Initiate(r *route.Request) {

	var state = randstr.Hex(16)

	var redirectPath string

	q := r.Req.URL.Query()
	q.Add("client_id", p.ClientID)
	q.Add("response_type", "code")
	q.Add("response_mode", "query")
	q.Add("scope", "openid profile email groups")
	q.Add("redirect_uri", p.DeploymentURI+"/auth/okta/redir") //ex: https://shield.starkandwayne.com/auth/okta/redir
	q.Add("state", state)
	// q.Add("nonce", nonce)

	redirectPath = p.OktaDomain + "/oauth2/" + p.AuthorizationServer + "/v1/authorize?" + q.Encode()
	//ex: "https://dev-xyz.okta.com/oauth2/default/v1/authorize?"

	r.Redirect(302, redirectPath)
}

func (p *OktaAuthProvider) HandleRedirect(r *route.Request) *db.User {

	// The login is achieved through the authorization code flow, where the user
	// is redirected to the Okta-Hosted login page. After the user authenticates
	// they are redirected back to the application with an access code that is
	// then exchanged for an access_token.

	// log.Debugf("DEBUG::r *route.Request is [GET /auth/okta/redir]")

	q := r.Req.URL.Query()
	q.Add("grant_type", "authorization_code")
	q.Set("code", r.Param("code", ""))
	q.Add("redirect_uri", p.DeploymentURI+"/auth/okta/redir") //ex: https://shield.starkandwayne.com/auth/okta/redir
	q.Add("response_type", "token")

	url := p.OktaDomain + "/oauth2/" + p.AuthorizationServer + "/v1/token?" + q.Encode()

	authHeader := base64.StdEncoding.EncodeToString([]byte(p.ClientID + ":" + p.ClientSecret))

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("")))
	if err != nil {
		p.Errorf("failed to create NewRequest for Okta endpoint %s: %s", url, err)
		return nil
	}
	h := req.Header
	h.Add("Authorization", "Basic "+authHeader)
	h.Add("Accept", "application/json")
	h.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		p.Errorf("failed to POST authcode to the Okta endpoint %s: %s", url, err)
		return nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.Errorf("failed to read response from POST %s: %s", url, err)
		return nil
	}
	defer resp.Body.Close()
	var exchange Exchange
	json.Unmarshal(body, &exchange)

	if exchange.Error != "" {
		p.Errorf("exchange resulted in error %s: %s", exchange.Error, exchange.ErrorDescription)
		return nil
	}

	// Now we succussfully exhchanged the auth code for an openID token which is stored in our struct "exchange"
	// Next we verify the token(s) and extract our okta claims from it (i.e. groups)
	// Goal is to get {name, account, and orgs} from this token to do the mapping.

	id_token, err := p.verifyToken(exchange.IdToken, "id")
	if err != nil {
		p.Errorf("Verification Failed: %s", err)
		return nil
	}
	access_token, err := p.verifyToken(exchange.AccessToken, "access")
	if err != nil {
		p.Errorf("Verification Failed: %s", err)
		return nil
	}

	groups_claim := id_token.Claims["groups"]
	if groups_claim == nil {
		groups_claim = access_token.Claims["groups"] //ex: access_token.groups_claim - [[Everyone User Admin]]
	}
	// log.Debugf("DEBUG::groups_claim [%s]", groups_claim)

	// TODO: get orgs from okta endpoint
	org_names := make([]string, 0)
	org_names = append(org_names, "okta")

	orgs := make(map[string][]string)
	for _, org_name := range org_names {
		orgs[org_name] = make([]string, 0)
	} //ex: orgs- [map[okta:[]]] - [map[string][]string]

	g, err := json.Marshal(groups_claim) //ex: g-[["Everyone","User","Admin"]] - [[]uint8]
	if err != nil {
		p.Errorf("failed to marshal groups claim: %s", err)
		return nil
	}

	groups_list := make([]string, 0)
	err = json.Unmarshal(g, &groups_list) //ex: groups_list-[[Everyone User Admin]] - [[]string]
	if err != nil {
		p.Errorf("failed to unmarshal groups list: %s", err)
		return nil
	}

	for _, groups := range groups_list {
		orgs["okta"] = append(orgs["okta"], groups) //ex: orgs- [map[okta:[Everyone User Admin]]]
	}

	username_claim := id_token.Claims["preferred_username"]
	if username_claim == nil {
		username_claim = access_token.Claims["preferred_username"]
	}
	account := fmt.Sprintf("%v", username_claim) // okta username = shield account

	name_claim := id_token.Claims["name"] // okta name = shield name
	if name_claim == nil {
		name_claim = access_token.Claims["name"]
	}
	name := fmt.Sprintf("%v", name_claim)

	// check if the user that logged in via okta already exists
	if p.core.db == nil {
		p.Errorf("no handle for the core database found!")
		return nil
	}
	user, err := p.core.db.GetUser(account, p.Identifier)
	if err != nil {
		p.Errorf("failed to retrieve user %s@%s from database: %s", account, p.Identifier, err)
		return nil
	}

	if user == nil {
		user = &db.User{
			UUID:    uuid.NewRandom().String(),
			Name:    name,
			Account: account,
			Backend: p.Identifier,
			SysRole: "",
		}
		p.core.db.CreateUser(user)

	}

	p.ClearAssignments()

Mapping:
	for _, candidate := range p.Mapping {
		for org, groups := range orgs {
			if candidate.Okta == org {
				for _, match := range candidate.Rights {
					if match.Group == "" {
						if !p.Assign(user, candidate.Tenant, match.Role) {

							return nil
						}
						continue Mapping
					}

					for _, group := range groups {
						if match.Group == group {
							if !p.Assign(user, candidate.Tenant, match.Role) {

								return nil
							}
							continue Mapping
						}
					}
				}
			}
		}
	}
	if !p.SaveAssignments(p.core.db, user) {
		return nil
	}

	return user
}

func (p *OktaAuthProvider) verifyToken(t string, category string) (*verifier.Jwt, error) {
	toValidate := map[string]string{}

	if category == "id" {							//id_token
		// toValidate["nonce"] = "{NONCE}"
		toValidate["aud"] = p.ClientID
	} else {										//access_token
		toValidate["aud"] = "api://" + p.AuthorizationServer
		toValidate["cid"] = "" //p.ClientID
	}

	jwtVerifierSetup := verifier.JwtVerifier{
		Issuer:           p.OktaDomain + "/oauth2/" + p.AuthorizationServer,
		ClaimsToValidate: toValidate,
	}

	verifier := jwtVerifierSetup.New()

	if category == "id"{
		token, err := verifier.VerifyIdToken(t)
		if err != nil && p.TokenVerification {
			return nil, fmt.Errorf("%s", err)
		}
	
		if token != nil {
			return token, nil
		}
	}else {
		token, err := verifier.VerifyAccessToken(t)
		if err != nil && p.TokenVerification {
			return nil, fmt.Errorf("%s", err)
		}
	
		if token != nil {
			return token, nil
		}
	}

	

	return nil, fmt.Errorf("%s token could not be verified: %s", category, "")
}
