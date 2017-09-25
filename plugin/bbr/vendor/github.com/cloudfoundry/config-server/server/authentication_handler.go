package server

import (
	"net/http"
	"strings"

	"github.com/cloudfoundry/bosh-utils/errors"
)

type authenticationHandler struct {
	tokenValidator TokenValidator
	nextHandler    http.Handler
}

func NewAuthenticationHandler(tokenValidator TokenValidator, nextHandler http.Handler) http.Handler {
	return authenticationHandler{
		tokenValidator: tokenValidator,
		nextHandler:    nextHandler,
	}
}

func (handler authenticationHandler) ServeHTTP(resWriter http.ResponseWriter, req *http.Request) {
	if handler.authenticate(req) == nil {
		handler.nextHandler.ServeHTTP(resWriter, req)
	} else {
		http.Error(resWriter, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func (handler authenticationHandler) authenticate(req *http.Request) error {
	authHeader := req.Header.Get("Authorization")
	if len(authHeader) == 0 {
		return errors.Error("Missing Token")
	}

	jwtToken, err := handler.checkTokenFormat(authHeader)
	if err != nil {
		return err
	}

	return handler.tokenValidator.Validate(jwtToken)
}

func (handler authenticationHandler) checkTokenFormat(token string) (string, error) {
	tokenParts := strings.Split(token, " ")
	if len(tokenParts) != 2 {
		return "", errors.Error("Invalid token format")
	}

	tokenType, userToken := tokenParts[0], tokenParts[1]
	if !strings.EqualFold(tokenType, "bearer") {
		return "", errors.Error("Invalid token type: " + tokenType)
	}

	return userToken, nil
}
