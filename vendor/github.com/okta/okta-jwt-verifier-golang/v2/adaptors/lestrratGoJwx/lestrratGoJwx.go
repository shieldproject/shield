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

package lestrratGoJwx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/okta/okta-jwt-verifier-golang/v2/adaptors"
	"github.com/okta/okta-jwt-verifier-golang/v2/utils"
)

func (lgj *LestrratGoJwx) fetchJwkSet(jwkUri string) (interface{}, error) {
	return jwk.Fetch(context.Background(), jwkUri, jwk.WithHTTPClient(lgj.Client))
}

type LestrratGoJwx struct {
	JWKSet      jwk.Set
	Cache       func(func(string) (interface{}, error), time.Duration, time.Duration) (utils.Cacher, error)
	jwkSetCache utils.Cacher
	Timeout     time.Duration
	Cleanup     time.Duration
	Client      *http.Client
}

func (lgj *LestrratGoJwx) New() (adaptors.Adaptor, error) {
	var err error
	if lgj.Cache == nil {
		lgj.Cache = utils.NewDefaultCache
	}
	lgj.jwkSetCache, err = lgj.Cache(lgj.fetchJwkSet, lgj.Timeout, lgj.Cleanup)
	if err != nil {
		return nil, err
	}
	return lgj, nil
}

func (lgj *LestrratGoJwx) Decode(jwt string, jwkUri string) (interface{}, error) {
	value, err := lgj.jwkSetCache.Get(jwkUri)
	if err != nil {
		return nil, err
	}

	jwkSet, ok := value.(jwk.Set)
	if !ok {
		return nil, fmt.Errorf("could not cast %v to jwk.Set", value)
	}

	token, err := jws.Verify([]byte(jwt), jws.WithKeySet(jwkSet))
	if err != nil {
		return nil, err
	}

	var claims interface{}
	if err := json.Unmarshal(token, &claims); err != nil {
		return nil, fmt.Errorf("could not unmarshal claims: %w", err)
	}

	return claims, nil
}
