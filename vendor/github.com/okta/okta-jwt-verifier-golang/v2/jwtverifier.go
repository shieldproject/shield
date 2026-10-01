/*******************************************************************************
 * Copyright 2018 - Present Okta, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 ******************************************************************************/

package jwtverifier

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/okta/okta-jwt-verifier-golang/v2/adaptors"
	"github.com/okta/okta-jwt-verifier-golang/v2/adaptors/lestrratGoJwx"
	"github.com/okta/okta-jwt-verifier-golang/v2/discovery"
	"github.com/okta/okta-jwt-verifier-golang/v2/discovery/oidc"
	"github.com/okta/okta-jwt-verifier-golang/v2/errors"
	"github.com/okta/okta-jwt-verifier-golang/v2/utils"
)

var (
	regx = regexp.MustCompile(`[a-zA-Z0-9-_]+\.[a-zA-Z0-9-_]+\.?([a-zA-Z0-9-_]+)[/a-zA-Z0-9-_]+?$`)
)

type JwtVerifier struct {
	Issuer string

	ClaimsToValidate map[string]string

	Discovery discovery.Discovery

	Adaptor adaptors.Adaptor

	Client *http.Client

	// Cache allows customization of the cache used to store resources
	Cache func(func(string) (interface{}, error), time.Duration, time.Duration) (utils.Cacher, error)

	metadataCache utils.Cacher

	leeway  int64
	Timeout time.Duration
	Cleanup time.Duration
}

type Jwt struct {
	Claims map[string]interface{}
}

func (j *JwtVerifier) fetchMetaData(url string) (interface{}, error) {
	resp, err := j.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request for metadata was not successful: %w", err)
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !ok {
		return nil, fmt.Errorf("request for metadata %q was not HTTP 2xx OK, it was: %d", url, resp.StatusCode)
	}

	metadata := make(map[string]interface{})
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

func (j *JwtVerifier) New() (*JwtVerifier, error) {
	// Default to OIDC discovery if none is defined
	if j.Discovery == nil {
		disc := oidc.Oidc{}
		j.Discovery = disc.New()
	}

	if j.Timeout == 0 {
		j.Timeout = 5 * time.Minute
	}

	if j.Cleanup == 0 {
		j.Cleanup = 10 * time.Minute
	}

	if j.Client == nil {
		j.Client = http.DefaultClient
	}

	if j.Cache == nil {
		j.Cache = utils.NewDefaultCache
	}

	// Default to LestrratGoJwx Adaptor if none is defined
	if j.Adaptor == nil {
		adaptor := &lestrratGoJwx.LestrratGoJwx{Cache: j.Cache, Timeout: j.Timeout, Cleanup: j.Cleanup, Client: j.Client}
		adp, err := adaptor.New()
		if err != nil {
			return nil, err
		}
		j.Adaptor = adp
	}

	// Default to PT2M Leeway
	j.leeway = 120
	var err error
	metadataCache, err := j.Cache(j.fetchMetaData, j.Timeout, j.Cleanup)
	if err != nil {
		return nil, err
	}
	j.metadataCache = metadataCache
	return j, nil
}

func (j *JwtVerifier) SetLeeway(duration string) {
	dur, _ := time.ParseDuration(duration)
	j.leeway = int64(dur.Seconds())
}

func (j *JwtVerifier) SetTimeOut(duration time.Duration) {
	j.Timeout = duration
}

func (j *JwtVerifier) SetCleanUp(duration time.Duration) {
	j.Cleanup = duration
}

func (j *JwtVerifier) VerifyAccessToken(jwt string) (*Jwt, error) {
	validJwt, err := j.isValidJwt(jwt)
	if !validJwt {
		return nil, fmt.Errorf("token is not valid: %w", err)
	}

	resp, err := j.decodeJwt(jwt)
	if err != nil {
		return nil, err
	}

	token := resp.(map[string]interface{})

	myJwt := Jwt{
		Claims: token,
	}

	err = j.validateIss(token["iss"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Issuer` was not able to be validated. %w", err)
	}

	err = j.validateAudience(token["aud"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Audience` was not able to be validated. %w", err)
	}

	err = j.validateClientId(token["cid"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Client Id` was not able to be validated. %w", err)
	}

	err = j.validateExp(token["exp"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Expiration` was not able to be validated. %w", err)
	}

	err = j.validateIat(token["iat"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Issued At` was not able to be validated. %w", err)
	}

	return &myJwt, nil
}

func (j *JwtVerifier) decodeJwt(jwt string) (interface{}, error) {
	metaData, err := j.getMetaData()
	if err != nil {
		return nil, err
	}
	jwksURI, ok := metaData["jwks_uri"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to decode JWT: missing 'jwks_uri' from metadata")
	}
	resp, err := j.Adaptor.Decode(jwt, jwksURI)
	if err != nil {
		return nil, fmt.Errorf("could not decode token: %w", err)
	}

	return resp, nil
}

func (j *JwtVerifier) VerifyIdToken(jwt string) (*Jwt, error) {
	validJwt, err := j.isValidJwt(jwt)
	if !validJwt {
		return nil, fmt.Errorf("token is not valid: %w", err)
	}

	resp, err := j.decodeJwt(jwt)
	if err != nil {
		return nil, err
	}

	token := resp.(map[string]interface{})

	myJwt := Jwt{
		Claims: token,
	}

	err = j.validateIss(token["iss"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Issuer` was not able to be validated. %w", err)
	}

	err = j.validateAudience(token["aud"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Audience` was not able to be validated. %w", err)
	}

	err = j.validateExp(token["exp"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Expiration` was not able to be validated. %w", err)
	}

	err = j.validateIat(token["iat"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Issued At` was not able to be validated. %w", err)
	}

	err = j.validateNonce(token["nonce"])
	if err != nil {
		return &myJwt, fmt.Errorf("the `Nonce` was not able to be validated. %w", err)
	}

	return &myJwt, nil
}

func (j *JwtVerifier) GetDiscovery() discovery.Discovery {
	return j.Discovery
}

func (j *JwtVerifier) GetAdaptor() adaptors.Adaptor {
	return j.Adaptor
}

func (j *JwtVerifier) validateNonce(nonce interface{}) error {
	if nonce == nil {
		nonce = ""
	}

	if nonce != j.ClaimsToValidate["nonce"] {
		return fmt.Errorf("nonce: %s does not match %s", nonce, j.ClaimsToValidate["nonce"])
	}
	return nil
}

func (j *JwtVerifier) validateAudience(audience interface{}) error {
	// Audience is optional, it will be validated if it is present in the ClaimsToValidate array
	if expectedAudience, exists := j.ClaimsToValidate["aud"]; !exists || audience == expectedAudience {
		return nil
	}

	switch v := audience.(type) {
	case string:
		if v != j.ClaimsToValidate["aud"] {
			return fmt.Errorf("aud: %s does not match %s", v, j.ClaimsToValidate["aud"])
		}
	case []string:
		for _, element := range v {
			if element == j.ClaimsToValidate["aud"] {
				return nil
			}
		}
		return fmt.Errorf("aud: %s does not match %s", v, j.ClaimsToValidate["aud"])
	case []interface{}:
		for _, e := range v {
			element, ok := e.(string)
			if !ok {
				return fmt.Errorf("unknown type for audience validation")
			}
			if element == j.ClaimsToValidate["aud"] {
				return nil
			}
		}
		return fmt.Errorf("aud: %s does not match %s", v, j.ClaimsToValidate["aud"])
	default:
		return fmt.Errorf("unknown type for audience validation")
	}

	return nil
}

func (j *JwtVerifier) validateClientId(clientId interface{}) error {
	// Client Id can be optional, it will be validated if it is present in the ClaimsToValidate array
	if cid, exists := j.ClaimsToValidate["cid"]; exists && clientId != cid {
		switch v := clientId.(type) {
		case string:
			if v != cid {
				return fmt.Errorf("cid: %s does not match %s", v, cid)
			}
		case []string:
			for _, element := range v {
				if element == cid {
					return nil
				}
			}
			return fmt.Errorf("cid: %s does not match %s", v, cid)
		default:
			return fmt.Errorf("unknown type for clientId validation")
		}
	}
	return nil
}

func (j *JwtVerifier) validateExp(exp interface{}) error {
	expf, ok := exp.(float64)
	if !ok {
		return fmt.Errorf("exp: missing")
	}
	if float64(time.Now().Unix()-j.leeway) > expf {
		return fmt.Errorf("the token is expired")
	}
	return nil
}

func (j *JwtVerifier) validateIat(iat interface{}) error {
	iatf, ok := iat.(float64)
	if !ok {
		return fmt.Errorf("iat: missing")
	}
	if float64(time.Now().Unix()+j.leeway) < iatf {
		return fmt.Errorf("the token was issued in the future")
	}
	return nil
}

func (j *JwtVerifier) validateIss(issuer interface{}) error {
	if issuer != j.Issuer {
		return fmt.Errorf("iss: %s does not match %s", issuer, j.Issuer)
	}
	return nil
}

func (j *JwtVerifier) getMetaData() (map[string]interface{}, error) {
	metaDataUrl := j.Issuer + j.Discovery.GetWellKnownUrl()

	value, err := j.metadataCache.Get(metaDataUrl)
	if err != nil {
		return nil, err
	}

	metadata, ok := value.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unable to cast %v to metadata", value)
	}
	return metadata, nil
}

func (j *JwtVerifier) isValidJwt(jwt string) (bool, error) {
	if jwt == "" {
		return false, errors.JwtEmptyStringError()
	}

	// Verify that the JWT Follows correct JWT encoding.
	jwtRegex := regx.MatchString
	if !jwtRegex(jwt) {
		return false, fmt.Errorf("token must contain at least 1 period ('.') and only characters 'a-Z 0-9 _'")
	}

	parts := strings.Split(jwt, ".")
	header := parts[0]
	header = padHeader(header)
	headerDecoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return false, fmt.Errorf("the tokens header does not appear to be a base64 encoded string")
	}

	var jsonObject map[string]interface{}
	isHeaderJson := json.Unmarshal([]byte(headerDecoded), &jsonObject) == nil
	if !isHeaderJson {
		return false, fmt.Errorf("the tokens header is not a json object")
	}

	_, algExists := jsonObject["alg"]
	_, kidExists := jsonObject["kid"]

	if !algExists {
		return false, fmt.Errorf("the tokens header must contain an 'alg'")
	}

	if !kidExists {
		return false, fmt.Errorf("the tokens header must contain a 'kid'")
	}

	if jsonObject["alg"] != "RS256" {
		return false, fmt.Errorf("the only supported alg is RS256")
	}

	return true, nil
}

func padHeader(header string) string {
	if i := len(header) % 4; i != 0 {
		header += strings.Repeat("=", 4-i)
	}
	return header
}
