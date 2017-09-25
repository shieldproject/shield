package server

import (
	"crypto/rsa"
	"io/ioutil"

	"github.com/cloudfoundry/bosh-utils/errors"
	"github.com/dgrijalva/jwt-go"
)

const (
	expectedScope = "config_server.admin"
)

type JwtTokenValidator struct {
	verificationKey *rsa.PublicKey
}

func NewJwtTokenValidator(jwtVerificationKeyPath string) (JwtTokenValidator, error) {
	bytes, err := ioutil.ReadFile(jwtVerificationKeyPath)
	if err != nil {
		return JwtTokenValidator{}, errors.WrapError(err, "Failed to read JWT Verification key")
	}

	verificationKey, err := jwt.ParseRSAPublicKeyFromPEM(bytes)
	if err != nil {
		return JwtTokenValidator{}, errors.WrapError(err, "Failed to parse RSA public key from PEM")
	}

	return JwtTokenValidator{verificationKey: verificationKey}, nil
}

func NewJWTTokenValidatorWithKey(verificationKey *rsa.PublicKey) JwtTokenValidator {
	return JwtTokenValidator{verificationKey: verificationKey}
}

func (j JwtTokenValidator) Validate(tokenStr string) error {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if !j.isValidSigningMethod(t) {
			return nil, errors.Error("Invalid signing method")
		}
		return j.verificationKey, nil
	})
	if err != nil {
		return errors.WrapError(err, "Validating token")
	}

	scopes := token.Claims.(jwt.MapClaims)["scope"].([]interface{})

	for _, el := range scopes {
		if el == expectedScope {
			return nil
		}
	}

	return errors.Errorf("Missing required scope: %s", expectedScope)
}

func (JwtTokenValidator) isValidSigningMethod(token *jwt.Token) bool {
	switch token.Method {
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		return true
	default:
		return false
	}
}
