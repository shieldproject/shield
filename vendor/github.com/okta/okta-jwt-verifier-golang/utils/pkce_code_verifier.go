/*******************************************************************************
 * Copyright 2022 - Present Okta, Inc.
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

// based on https://datatracker.ietf.org/doc/html/rfc7636
package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
)

const (
	MinLength = 32
	MaxLength = 96
)

type PKCECodeVerifier struct {
	CodeVerifier string
}

func (v *PKCECodeVerifier) String() string {
	return v.CodeVerifier
}

// CodeChallengePlain generates a plain code challenge from a code verifier
func (v *PKCECodeVerifier) CodeChallengePlain() string {
	return v.CodeVerifier
}

// CodeChallengeS256 generates a Sha256 code challenge from a code verifier
func (v *PKCECodeVerifier) CodeChallengeS256() string {
	h := sha256.New()
	h.Write([]byte(v.CodeVerifier))
	return encode(h.Sum(nil))
}

// GenerateCodeVerifier generates a code verifier with the minimum length
func GenerateCodeVerifier() (*PKCECodeVerifier, error) {
	return GenerateCodeVerifierWithLength(MinLength)
}

// GenerateCodeVerifierWithLength generates a code verifier with the specified length
func GenerateCodeVerifierWithLength(length int) (*PKCECodeVerifier, error) {
	if length < MinLength || length > MaxLength {
		return nil, fmt.Errorf("invalid length: %v", length)
	}
	// create random bytes
	b, err := bytes(length)
	if err != nil {
		return nil, err
	}
	return &PKCECodeVerifier{
		CodeVerifier: encode(b),
	}, nil
}

// bytes generates n random bytes
func bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

// encode encodes a byte array to a base64 string with no padding
func encode(b []byte) string {
	encoded := base64.StdEncoding.EncodeToString(b)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}
