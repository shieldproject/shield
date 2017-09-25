package mockuaa

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/gomega"
)

const (
	UnauthorizedError            = "you are unauthorized"
	UnauthorizedErrorDescription = "please go away"
	MalformedResponseUser        = "MalformedResponse"
	InternalServerErrorUser      = "InternalServerError"
	InternalServerErrorMessage   = "Shut the F up, Donny"
)

type ClientCredentialsServer struct {
	*httptest.Server

	UAAClientID     string
	UAAClientSecret string
	TokenToReturn   string

	ValiditySecondsToReturn int
	TokensIssued            int
}

type UserCredentialsServer struct {
	*httptest.Server

	ClientID      string
	ClientSecret  string
	Username      string
	Password      string
	TokenToReturn string

	ValiditySecondsToReturn int
	TokensIssued            int
}

func (s ClientCredentialsServer) ExpectedAuthorizationHeader() string {
	return "Bearer " + s.TokenToReturn
}

func (s *ClientCredentialsServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	validateRequest(req, "")

	username, password, ok := req.BasicAuth()
	Expect(ok).To(BeTrue())

	// Default validity seconds = 599
	if s.ValiditySecondsToReturn == 0 {
		s.ValiditySecondsToReturn = 599
	}

	var status int
	var response map[string]interface{}

	if username == MalformedResponseUser {
		status, response = handleMalformedResponse(s.ValiditySecondsToReturn)
	} else if username == InternalServerErrorUser {
		handleInternalServerError(writer)
		return
	} else if (username == s.UAAClientID) && (password == s.UAAClientSecret) {
		status, response = handleAuthorised(s.TokenToReturn, s.ValiditySecondsToReturn)
		s.TokensIssued++
	} else {
		status, response = handleUnauthorised()
	}

	writeStatusAndResponse(status, response, writer)
}

func (s *UserCredentialsServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	validateRequest(req, "password")

	clientID, clientSecret, ok := req.BasicAuth()
	Expect(ok).To(BeTrue())
	username := req.PostFormValue("username")
	password := req.PostFormValue("password")

	// Default validity seconds = 599
	if s.ValiditySecondsToReturn == 0 {
		s.ValiditySecondsToReturn = 599
	}

	var status int
	var response map[string]interface{}

	if username == MalformedResponseUser {
		status, response = handleMalformedResponse(s.ValiditySecondsToReturn)
	} else if username == InternalServerErrorUser {
		handleInternalServerError(writer)
		return
	} else if (clientID == s.ClientID) && (clientSecret == s.ClientSecret) && (username == s.Username) && (password == s.Password) {
		status, response = handleAuthorised(s.TokenToReturn, s.ValiditySecondsToReturn)
		s.TokensIssued++
	} else {
		status, response = handleUnauthorised()
	}

	writeStatusAndResponse(status, response, writer)
}

func writeStatusAndResponse(status int, response map[string]interface{}, writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json; charset=UTF-8")
	writer.WriteHeader(status)
	err := json.NewEncoder(writer).Encode(response)
	Expect(err).NotTo(HaveOccurred())
}

func validateRequest(req *http.Request, grantType string) {
	Expect(req.Method).To(Equal(http.MethodPost))
	Expect(req.URL.Path).To(Equal("/oauth/token"))
	Expect(req.URL.Query().Get("grant_type")).To(Equal(grantType))
}

func handleMalformedResponse(validitySecondsToReturn int) (int, map[string]interface{}) {
	status := http.StatusOK
	response := map[string]interface{}{
		// "access_token": missing!
		"token_type": "bearer",
		"expires_in": validitySecondsToReturn,
		"scope":      "bosh.admin",
		"jti":        "some-random-uuid",
	}
	return status, response
}

func handleInternalServerError(writer http.ResponseWriter) {
	status := http.StatusInternalServerError
	writer.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	writer.WriteHeader(status)
	writer.Write([]byte(InternalServerErrorMessage))
}

func handleAuthorised(tokenToReturn string, validitySecondsToReturn int) (int, map[string]interface{}) {
	status := http.StatusOK
	response := map[string]interface{}{
		"access_token": tokenToReturn,
		"token_type":   "bearer",
		"expires_in":   validitySecondsToReturn,
		"scope":        "bosh.admin",
		"jti":          "some-random-uuid",
	}
	return status, response
}

func handleUnauthorised() (int, map[string]interface{}) {
	status := http.StatusUnauthorized
	response := map[string]interface{}{
		"error":             UnauthorizedError,
		"error_description": UnauthorizedErrorDescription,
	}
	return status, response
}

func NewClientCredentialsServer(uaaClientID, uaaClientSecret, tokenToReturn string) *ClientCredentialsServer {
	return startClientCredentialsServer(uaaClientID, uaaClientSecret, tokenToReturn, httptest.NewServer)
}

func NewClientCredentialsServerTLS(uaaClientID, uaaClientSecret, tokenToReturn string) *ClientCredentialsServer {
	return startClientCredentialsServer(uaaClientID, uaaClientSecret, tokenToReturn, httptest.NewTLSServer)
}

func NewUserCredentialsServer(clientID, clientSecret, username, password, tokenToReturn string) *UserCredentialsServer {
	return startUserCredentialsServer(clientID, clientSecret, username, password, tokenToReturn, httptest.NewServer)
}

func NewUserCredentialsServerTLS(clientID, clientSecret, username, password, tokenToReturn string) *UserCredentialsServer {
	return startUserCredentialsServer(clientID, clientSecret, username, password, tokenToReturn, httptest.NewTLSServer)
}

func startClientCredentialsServer(uaaClientID, uaaClientSecret, tokenToReturn string, serverStarter func(http.Handler) *httptest.Server) *ClientCredentialsServer {
	uaa := &ClientCredentialsServer{
		UAAClientID:     uaaClientID,
		UAAClientSecret: uaaClientSecret,
		TokenToReturn:   tokenToReturn,
	}
	uaa.Server = serverStarter(uaa)
	return uaa
}

func startUserCredentialsServer(clientID, clientSecret, username, password, tokenToReturn string, serverStarter func(http.Handler) *httptest.Server) *UserCredentialsServer {
	uaa := &UserCredentialsServer{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		Username:      username,
		Password:      password,
		TokenToReturn: tokenToReturn,
	}
	uaa.Server = serverStarter(uaa)
	return uaa
}
